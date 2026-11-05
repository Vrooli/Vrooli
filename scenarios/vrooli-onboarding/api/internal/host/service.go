// Package host owns the transport-free host requirements, facts, fleet, and
// host notification configuration contract.
package host

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/api-core/targetmodel"
)

type Requirement struct {
	Name         string
	Required     bool
	Reason       string
	Notes        string
	Description  string
	Risk         string
	Privilege    string
	Bundling     string
	Platforms    []string
	Commands     []string
	ConfigSchema map[string]any
	Config       map[string]any
	Status       string
	Disposition  string
	Detail       string
	Remediation  string
}

type Facts struct {
	Available            bool
	Reason               string
	CPUCount             *int
	MemoryTotalBytes     *uint64
	MemoryAvailableBytes *uint64
	DiskFreeBytes        *uint64
	GPUs                 []string
	Platform             string
}

type Service struct {
	ListRequirementsFn       func(context.Context, string) ([]Requirement, []Requirement, error)
	FactsFn                  func(context.Context, string) (Facts, error)
	TargetsFn                func(context.Context) ([]targetmodel.Target, string, error)
	PatchSafeguardConfigFn   func(context.Context, string, string, string, any) error
	SetNotificationRecipient func(context.Context, string, string) error
}

func (s Service) ListRequirements(ctx context.Context, target string) ([]Requirement, []Requirement, error) {
	if s.ListRequirementsFn == nil {
		return nil, nil, fmt.Errorf("host requirements provider is not configured")
	}
	return s.ListRequirementsFn(ctx, strings.TrimSpace(target))
}

func (s Service) Facts(ctx context.Context, target string) (Facts, error) {
	if s.FactsFn == nil {
		return Facts{}, fmt.Errorf("host facts provider is not configured")
	}
	return s.FactsFn(ctx, strings.TrimSpace(target))
}

func (s Service) Targets(ctx context.Context) ([]targetmodel.Target, string, error) {
	if s.TargetsFn == nil {
		return nil, "", fmt.Errorf("target provider is not configured")
	}
	return s.TargetsFn(ctx)
}

func (s Service) PatchSafeguardConfig(ctx context.Context, target, name, key string, value any) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(key) == "" {
		return fmt.Errorf("safeguard name and config key are required")
	}
	if s.PatchSafeguardConfigFn == nil {
		return fmt.Errorf("host safeguard configuration provider is not configured")
	}
	return s.PatchSafeguardConfigFn(ctx, strings.TrimSpace(target), strings.TrimSpace(name), strings.TrimSpace(key), value)
}

func (s Service) SetRecipient(ctx context.Context, target, subject string) error {
	if strings.TrimSpace(subject) == "" {
		return fmt.Errorf("notification recipient is required")
	}
	if s.SetNotificationRecipient == nil {
		return fmt.Errorf("notification recipient provider is not configured")
	}
	return s.SetNotificationRecipient(ctx, strings.TrimSpace(target), strings.TrimSpace(subject))
}
