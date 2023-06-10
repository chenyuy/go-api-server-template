package database

import (
	"context"
	"embed"
	"fmt"
	"io"

	"github.com/jackc/pgx/v4/pgxpool"
)

//go:embed migrations
var migrationsFs embed.FS

const _schemaVersion = 1

const _versionTable = "schema_version"

func createVersionTable(dbpgx *pgxpool.Pool) error {
	if _, err := dbpgx.Exec(context.Background(), fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s(version INT NOT NULL);
		INSERT INTO %s(version)
		SELECT 0 WHERE NOT EXISTS (SELECT * FROM %s);
	`, _versionTable, _versionTable, _versionTable)); err != nil {
		return err
	}
	return nil
}

func Migrate(dbpgx *pgxpool.Pool) error {
	if err := createVersionTable(dbpgx); err != nil {
		return err
	}

	rows, err := dbpgx.Query(context.Background(), fmt.Sprintf("select version from %s", _versionTable))
	if err != nil {
		return err
	}
	defer rows.Close()
	version := -1
	for rows.Next() {
		if version != -1 {
			continue
		}
		var v int
		if err := rows.Scan(&v); err != nil {
			return err
		}
		version = v
	}
	if version == -1 {
		return fmt.Errorf("cannot find schema version")
	}
	if version == _schemaVersion {
		return nil
	}

	ctx := context.Background()
	tx, err := dbpgx.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for i := version + 1; i <= _schemaVersion; i++ {
		filename := fmt.Sprintf("migrations/%d_step.sql", i)
		f, err := migrationsFs.Open(filename)
		if err != nil {
			return err
		}
		content, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(content)); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(
		ctx,
		fmt.Sprintf("update %s set version = %d", _versionTable, _schemaVersion),
	); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}
