package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	gomigrate "github.com/golang-migrate/migrate/v4"
	postgresdriver "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Регистрируем файловый источник миграций.
)

var ErrNoMigrations = errors.New("нет SQL-миграций для применения")

// Up применяет SQL-миграции по возрастанию версии и не закрывает переданный пул.
func Up(ctx context.Context, db *sql.DB, dir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("путь к миграциям: %w", err)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("каталог миграций: %w", err)
	}
	found := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			found = true
			break
		}
	}
	if !found {
		return ErrNoMigrations
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("соединение для миграций: %w", err)
	}
	driver, err := postgresdriver.WithConnection(ctx, conn, &postgresdriver.Config{MultiStatementEnabled: true})
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("драйвер миграций: %w", err)
	}
	sourceURL := (&url.URL{Scheme: "file", Path: path}).String()
	m, err := gomigrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		_ = driver.Close()
		return fmt.Errorf("источник миграций: %w", err)
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Up(); err != nil && !errors.Is(err, gomigrate.ErrNoChange) {
		return fmt.Errorf("применение миграций: %w", err)
	}
	return nil
}
