-- используется, чтобы удалить уже существующие дубликаты в таблице daily_stats, чтобы уникальный индекс мог быть создан без ошибок
DELETE a 
FROM daily_stats a
JOIN daily_stats b ON a.stat_date = b.stat_date AND a.placement_id = b.placement_id
WHERE a.id < b.id;

CREATE UNIQUE INDEX IF NOT EXISTS daily_stats_date_placement_idx ON daily_stats (stat_date, placement_id);
