package resolver

import (
	"context"
	"errors"
	"time"
)

const TimeFormat = time.RFC3339Nano

var ErrNotFound = errors.New("resolver backend not found")

type BackendConfig struct {
	Backend   string
	BaseURL   string
	Username  string
	TokenRef  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Status struct {
	Backend             string
	Status              string
	BaseURL             string
	Upstreams           []string
	FilteringEnabled    bool
	Warnings            []string
	EnforcementStatus   string
	EnforcementEvidence []string
}

type RolloutCheck struct {
	ID              string
	Title           string
	Status          string
	Evidence        string
	Recommendations []string
}

type RolloutReport struct {
	Status         string
	Summary        string
	DNSBindIP      string
	ResolverStatus Status
	Checks         []RolloutCheck
	RouterSettings []string
	NextSteps      []string
	Warnings       []string
}

type DNSInspection struct {
	Servers  []string
	Evidence []string
	Warnings []string
}

type HostDNSInspector interface {
	InspectHostDNS(ctx context.Context, targetDNS string) DNSInspection
}

type ClientStatus struct {
	Status              string
	Upstreams           []string
	FilteringEnabled    bool
	Warnings            []string
	Checks              []string
	EnforcementStatus   string
	EnforcementEvidence []string
}

type Repository interface {
	SaveBackend(ctx context.Context, cfg BackendConfig) (BackendConfig, error)
	GetBackend(ctx context.Context, backend string) (BackendConfig, error)
	UpdateUpstreams(ctx context.Context, backend string, upstreams []string) error
	GetUpstreams(ctx context.Context, backend string) ([]string, error)
}

type AdGuardClient interface {
	Check(ctx context.Context, cfg BackendConfig) (ClientStatus, error)
	PreviewUpstreams(ctx context.Context, cfg BackendConfig, upstreams []string) ([]string, error)
	UpdateUpstreams(ctx context.Context, cfg BackendConfig, upstreams []string) (ClientStatus, []string, error)
}
