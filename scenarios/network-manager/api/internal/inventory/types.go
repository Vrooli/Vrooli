package inventory

import (
	"context"
	"errors"
	"time"
)

const TimeFormat = time.RFC3339Nano

var (
	ErrNotFound    = errors.New("device not found")
	ErrUnsupported = errors.New("device discovery source unsupported")
)

type Device struct {
	ID                 string
	Hostname           string
	IPAddress          string
	MACAddress         string
	StableID           string
	ResolverClientID   string
	Group              string
	IdentityConfidence string
	Notes              []string
	LastSeen           time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Observation struct {
	Source           string
	Hostname         string
	IPAddress        string
	MACAddress       string
	StableID         string
	ResolverClientID string
	LastSeen         time.Time
}

type Repository interface {
	SaveDevice(ctx context.Context, device Device) (Device, error)
	GetDevice(ctx context.Context, id string) (Device, error)
	ListDevices(ctx context.Context, group string) ([]Device, error)
	UpdateGroup(ctx context.Context, id, group string) (Device, error)
}

type DeviceDiscoverySource interface {
	Discover(ctx context.Context) ([]Observation, []string, error)
}
