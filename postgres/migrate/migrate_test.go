package migrate_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/lib/pq"

	"github.com/neutral-jungle-creep/tools/postgres/migrate"
)

func TestUpEmptyDirectory(t *testing.T) {
	if err := migrate.Up(context.Background(), nil, t.TempDir()); !errors.Is(err, migrate.ErrNoMigrations) {
		t.Fatalf("ожидали ErrNoMigrations, получили %v", err)
	}
}

func TestUpMissingDirectory(t *testing.T) {
	err := migrate.Up(context.Background(), nil, filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ожидали os.ErrNotExist, получили %v", err)
	}
}

func TestUpCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := migrate.Up(ctx, nil, "unused"); !errors.Is(err, context.Canceled) {
		t.Fatalf("ожидали context.Canceled, получили %v", err)
	}
}

func TestUpConnectionError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "000001_db_schema.up.sql"), []byte("SELECT 1;"), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("postgres", "postgres://invalid host")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := migrate.Up(context.Background(), db, dir); err == nil {
		t.Fatal("ожидали ошибку подключения")
	}
}
