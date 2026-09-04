# Решение

## Задание 1. Фильтр по placement

### Что было сломано

Причина оказалась в одной строчке. Репозиторий доставал значение фильтра из query-строки по ключу `placement`, а дашборд и контракт из README передают `placement_id`:

```php
if (!empty($query['placement'])) {          // такого ключа в запросе нет
    $where[] = 'ds.placement_id = :placement_id';
    $params['placement_id'] = $query['placement'];
}
```

Условие не выполнялось ни разу, кусок с WHERE по placement в итоговый SQL не попадал, и запрос всегда возвращал все placement за дату

### Как исправлено

```php
$placementId = trim((string) ($query['placement_id'] ?? ''));

if ($placementId !== '') {
    $where[] = 'ds.placement_id = :placement_id';
    $params['placement_id'] = $placementId;
}
```

### Как проверялось

До правки оба запроса возвращали одно и то же — это было три строки. То есть фильтр действительно ни на что не влиял.

После правки я прогнал шесть проверок. Запрос без фильтра по-прежнему отдаёт три строки. Запрос с `placement-video-main` отдаёт одну нужную. То же самое с `placement-banner-sidebar`. Запрос с пустым `placement_id=` отдаёт все три строки, не сломался. Запрос с несуществующим id отдаёт ноль строк, а не ошибку



