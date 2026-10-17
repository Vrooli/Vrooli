// Package gatedb owns the explicit-path database seam for calibration and
// scratch fixtures. Serving and production evidence gates use handed-in DBs.
package gatedb

import (
	"context"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
)

func Open(ctx context.Context, path string) (*database.RoutedDB, error) {
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		return nil, err
	}
	return database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: dsn, MaxOpenConns: 1, MaxIdleConns: 1})
}
