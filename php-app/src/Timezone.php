<?php

declare(strict_types=1);

namespace App;

use DateTimeZone;
use Exception;
use RuntimeException;

final class Timezone
{
    public static function app(): DateTimeZone
    {
        $name = getenv('APP_TIMEZONE') ?: 'UTC';

        try {
            return new DateTimeZone($name);
        } catch (Exception $exception) {
            throw new RuntimeException(
                sprintf('Некорректная таймзона в APP_TIMEZONE: "%s"', $name),
                0,
                $exception
            );
        }
    }
}
