# Tools

Каждый каталог с `go.mod` — отдельный Go-модуль и выпускается своим тегом `<путь>/vX.Y.Z`.

| Модуль | Назначение |
| --- | --- |
| `background` | Управление фоновыми задачами |
| `http/protocol` | HTTP-ответы и проверка запросов |
| `http/server` | Настройка HTTP-сервера |
| `logging` | Логирование |
| `postgres` | Подключение к PostgreSQL |
| `postgres/migrate` | Применение SQL-миграций через `golang-migrate` |

Пример импорта: `github.com/neutral-jungle-creep/tools/postgres/migrate`.

Для миграций нужны файлы вида `000001_db_schema.up.sql` в каталоге сервиса. Модуль применяет только `up`-миграции; переданный пул PostgreSQL остаётся открытым. Проверки всех модулей запускаются через `task`.
