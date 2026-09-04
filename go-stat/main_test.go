package main

import (
	"testing"
	"time"
)

func TestParsePriceCents(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    int
		wantErr bool
	}{
		{name: "кейс из задания", raw: "12.34", want: 1234},
		{name: "целые рубли", raw: "8.00", want: 800},
		{name: "целые рубли без дробной части", raw: "34", want: 3400},
		{name: "один цент", raw: "0.01", want: 1},
		{name: "меньше рубля", raw: "0.99", want: 99},
		{name: "старый код давал тут ноль", raw: "0.29", want: 29},
		{name: "ноль", raw: "0.00", want: 0},
		{name: "крупная сумма", raw: "99999.99", want: 9999999},
		{name: "пустая цена это ноль", raw: "", want: 0},
		{name: "пробелы это тоже ноль", raw: "   ", want: 0},
		{name: "не число", raw: "abc", wantErr: true},
		{name: "отрицательная", raw: "-5", wantErr: true},
		{name: "NaN", raw: "NaN", wantErr: true},
		{name: "бесконечность", raw: "Inf", wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parsePriceCents(c.raw)

			if c.wantErr {
				if err == nil {
					t.Fatalf("parsePriceCents(%q) = %d, ожидалась ошибка", c.raw, got)
				}
				return
			}

			if err != nil {
				t.Fatalf("parsePriceCents(%q) вернул ошибку: %v", c.raw, err)
			}

			if got != c.want {
				t.Errorf("parsePriceCents(%q) = %d, ожидалось %d", c.raw, got, c.want)
			}
		})
	}
}

func TestParseOccurredAtValid(t *testing.T) {
	// Один и тот же момент, записанный в UTC и в московском времени.
	// Полночь по Москве — это 21:00 предыдущего дня по UTC.
	utc, err := parseOccurredAt("2026-08-06T21:05:00+00:00")
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	msk, err := parseOccurredAt("2026-08-07T00:05:00+03:00")
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if !utc.Equal(msk) {
		t.Errorf("одно и то же время разошлось: %s и %s", utc, msk)
	}

	if utc.Location() != time.UTC {
		t.Errorf("время должно приводиться к UTC, получено %s", utc.Location())
	}
}

func TestParseOccurredAtEmptyMeansNow(t *testing.T) {
	got, err := parseOccurredAt("")
	if err != nil {
		t.Fatalf("пустое время должно означать «сейчас», получена ошибка: %v", err)
	}

	if time.Since(got) > time.Minute {
		t.Errorf("ожидалось текущее время, получено %s", got)
	}
}

func TestParseOccurredAtInvalid(t *testing.T) {
	// Раньше битое время молча подменялось на now() и событие уезжало не в свой день.
	for _, raw := range []string{"вчера", "2026-08-06 21:05:00", "07.08.2026", "-1"} {
		if _, err := parseOccurredAt(raw); err == nil {
			t.Errorf("parseOccurredAt(%q) должен был вернуть ошибку", raw)
		}
	}
}

func TestAllowedActionTypes(t *testing.T) {
	for _, action := range []string{"impression", "click"} {
		if !allowedActionTypes[action] {
			t.Errorf("%q должен приниматься: агрегация умеет его считать", action)
		}
	}

	for _, action := range []string{"unknown_action", "hover", "", "IMPRESSION"} {
		if allowedActionTypes[action] {
			t.Errorf("%q не должен приниматься", action)
		}
	}
}
