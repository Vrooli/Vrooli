package database

import (
	"context"
	"database/sql"
)

// Pinger is the narrow connectivity seam required by database-backed health
// checks. Both *sql.DB and api-core's routed databases satisfy this contract.
type Pinger interface {
	PingContext(context.Context) error
}

var _ Pinger = (*sql.DB)(nil)
