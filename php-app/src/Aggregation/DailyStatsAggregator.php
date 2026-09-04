<?php

declare(strict_types=1);

namespace App\Aggregation;

use App\Timezone;
use DateInterval;
use DateTimeImmutable;
use InvalidArgumentException;
use PDO;
use Throwable;

final class DailyStatsAggregator
{
    public function __construct(private PDO $pdo)
    {
    }
    public function aggregate(string $date): int
    {
        $start = $this->startOfBusinessDay($date);
        $end = $start->add(new DateInterval('P1D'));

        $select = $this->pdo->prepare(
            "SELECT
                placement_id,
                SUM(CASE WHEN action_type = 'impression' THEN 1 ELSE 0 END) AS impressions,
                SUM(CASE WHEN action_type = 'click' THEN 1 ELSE 0 END) AS clicks,
                SUM(price_cents) AS revenue_cents
             FROM raw_events
             WHERE occurred_at >= :start_at AND occurred_at < :end_at
             GROUP BY placement_id
             ORDER BY placement_id"
        );

        $select->execute([
            'start_at' => $start->format(DATE_ATOM),
            'end_at' => $end->format(DATE_ATOM),
        ]);

        $rows = $select->fetchAll();

        $upsert = $this->pdo->prepare(
            'INSERT INTO daily_stats (stat_date, placement_id, impressions, clicks, revenue_cents, updated_at)
             VALUES (:stat_date, :placement_id, :impressions, :clicks, :revenue_cents, now())
             ON CONFLICT (stat_date, placement_id) DO UPDATE SET
                 impressions = EXCLUDED.impressions,
                 clicks = EXCLUDED.clicks,
                 revenue_cents = EXCLUDED.revenue_cents,
                 updated_at = now()'
        );

        $this->pdo->beginTransaction();

        try {
            foreach ($rows as $row) {
                $upsert->execute([
                    'stat_date' => $date,
                    'placement_id' => $row['placement_id'],
                    'impressions' => (int) $row['impressions'],
                    'clicks' => (int) $row['clicks'],
                    'revenue_cents' => (int) $row['revenue_cents'],
                ]);
            }

            $this->deleteStaleRows($date, array_column($rows, 'placement_id'));

            $this->pdo->commit();
        } catch (Throwable $exception) {
            $this->pdo->rollBack();

            throw $exception;
        }

        return count($rows);
    }

    private function startOfBusinessDay(string $date): DateTimeImmutable
    {
        $start = DateTimeImmutable::createFromFormat('!Y-m-d', $date, Timezone::app());

        if ($start === false || $start->format('Y-m-d') !== $date) {
            throw new InvalidArgumentException(
                sprintf('Ожидалась дата в формате YYYY-MM-DD, получено: "%s"', $date)
            );
        }

        return $start;
    }

    private function deleteStaleRows(string $date, array $placementIds): void
    {
        if ($placementIds === []) {
            $statement = $this->pdo->prepare('DELETE FROM daily_stats WHERE stat_date = :stat_date');
            $statement->execute(['stat_date' => $date]);

            return;
        }

        $placeholders = implode(', ', array_fill(0, count($placementIds), '?'));

        $statement = $this->pdo->prepare(
            "DELETE FROM daily_stats
             WHERE stat_date = ?
               AND placement_id NOT IN ($placeholders)"
        );

        $statement->execute([$date, ...$placementIds]);
    }
}
