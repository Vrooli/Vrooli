// Package cache owns bounded derived-result cache policy and storage.
package cache

import (
	"context"
	"time"
)

type Key struct {
	Kind       string
	Identity   string
	Generation string
	Policy     string
}

type Entry struct {
	Key        Key
	Payload    []byte
	Bytes      int64
	CreatedAt  time.Time
	AccessedAt time.Time
	ExpiresAt  time.Time
}

type Stats struct {
	Rows       int64
	Bytes      int64
	QuotaBytes int64
	Orphans    int64
}

type Repository interface {
	Get(context.Context, Key) (Entry, bool, error)
	Put(context.Context, Entry) error
	Delete(context.Context, []Key) error
	Collect(context.Context, time.Time) (Stats, error)
	Stats(context.Context) (Stats, error)
}

type Clock interface{ Now() time.Time }
