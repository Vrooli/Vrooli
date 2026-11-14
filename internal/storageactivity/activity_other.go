//go:build !linux

package storageactivity

import (
	"context"
	"fmt"
	"time"
)

type Snapshot struct {
	UID        uint32    `json:"uid"`
	ObservedAt time.Time `json:"observed_at"`
	Complete   bool      `json:"complete"`
	Paths      []string  `json:"paths"`
	Roots      []string  `json:"roots"`
	Error      string    `json:"error,omitempty"`
}

func Observe(context.Context, uint32) (Snapshot, error) {
	return Snapshot{}, fmt.Errorf("temporary process observation requires Linux")
}
