package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var allowedActionTypes = map[string]bool{
	"impression": true,
	"click":      true,
}

type statResponse struct {
	Status string `json:"status"`
	ID     int64  `json:"id,omitempty"`
	Error  string `json:"error,omitempty"`
}

func main() {
	dsn := getenv("DATABASE_URL", "postgres://app:app@127.0.0.1:15432/stat_pipeline?sslmode=disable")
	addr := getenv("HTTP_ADDR", ":7011")

	db, err := sql.Open("pgx", dsn)

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	if err := waitForDB(db); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/health", handleHealth(db))
	http.HandleFunc("/stat", handleStat(db))

	log.Printf("go-stat listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleHealth(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, statResponse{Status: "error", Error: "db_unavailable"})
			return
		}

		writeJSON(w, http.StatusOK, statResponse{Status: "ok"})
	}
}

func handleStat(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, statResponse{Status: "error", Error: "method_not_allowed"})
			return
		}

		query := r.URL.Query()
		requestID := query.Get("requestId")

		placementID := strings.TrimSpace(query.Get("placement"))
		if placementID == "" {
			writeJSON(w, http.StatusBadRequest, statResponse{Status: "error", Error: "missing_placement"})
			return
		}

		actionType := strings.TrimSpace(query.Get("actionType"))
		if !allowedActionTypes[actionType] {
			writeJSON(w, http.StatusBadRequest, statResponse{Status: "error", Error: "unknown_action_type"})
			return
		}

		priceCents, err := parsePriceCents(query.Get("price"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, statResponse{Status: "error", Error: "invalid_price"})
			return
		}

		occurredAt, err := parseOccurredAt(query.Get("occurredAt"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, statResponse{Status: "error", Error: "invalid_occurred_at"})
			return
		}

		known, err := placementExists(r.Context(), db, placementID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, statResponse{Status: "error", Error: "placement_lookup_failed"})
			return
		}

		if !known {
			writeJSON(w, http.StatusBadRequest, statResponse{Status: "error", Error: "unknown_placement"})
			return
		}

		var eventID int64
		err = db.QueryRowContext(
			r.Context(),
			`INSERT INTO raw_events (placement_id, action_type, price_cents, occurred_at, request_id)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id`,
			placementID,
			actionType,
			priceCents,
			occurredAt,
			nullableString(requestID),
		).Scan(&eventID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, statResponse{Status: "error", Error: "insert_failed"})
			return
		}

		writeJSON(w, http.StatusAccepted, statResponse{Status: "accepted", ID: eventID})
	}
}

func placementExists(ctx context.Context, db *sql.DB, placementID string) (bool, error) {
	var exists bool

	err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM placements WHERE id = $1)`,
		placementID,
	).Scan(&exists)

	return exists, err
}

func parsePriceCents(raw string) (int, error) {
	raw = strings.TrimSpace(raw)

	if raw == "" {
		return 0, nil
	}

	price, err := strconv.ParseFloat(raw, 64)

	if err != nil {
		return 0, err
	}

	if math.IsNaN(price) || math.IsInf(price, 0) || price < 0 {
		return 0, errors.New("price must be a non-negative number")
	}

	return int(math.Round(price * 100)), nil
}

func parseOccurredAt(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)

	if raw == "" {
		return time.Now().UTC(), nil
	}

	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, err
	}

	return parsed.UTC(), nil
}

func nullableString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func waitForDB(db *sql.DB) error {
	var lastErr error

	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		lastErr = db.PingContext(ctx)
		cancel()

		if lastErr == nil {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	if lastErr == nil {
		lastErr = errors.New("unknown database error")
	}

	return lastErr
}

func writeJSON(w http.ResponseWriter, status int, payload statResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func getenv(key string, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
