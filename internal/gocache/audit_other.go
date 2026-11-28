//go:build !linux

package gocache

import (
	"errors"
	"time"
)

const auditSupported = false

type auditor struct{}

func newAuditor(Store, string, func() time.Time) (*auditor, error) {
	return nil, errors.New("auditing live go commands requires Linux /proc; reclaim is unavailable on this platform")
}

func (*auditor) run() (AuditReport, error) {
	return AuditReport{}, errors.New("auditing live go commands requires Linux /proc")
}
