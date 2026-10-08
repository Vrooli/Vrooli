package adapters

import (
	"context"
	"strings"
	"time"
)

type Service struct {
	repo     Repository
	registry Registry
}

type Config struct {
	Repo     Repository
	Registry Registry
}

func NewService(cfg Config) *Service {
	s := &Service{repo: cfg.Repo, registry: cfg.Registry}
	if s.registry == nil {
		s.registry = NewStaticRegistry()
	}
	return s
}

func (s *Service) ListCapabilities(ctx context.Context) ([]Capability, error) {
	report, err := s.registry.Report(ctx)
	if err != nil {
		return nil, err
	}
	stampReport(&report)
	if s.repo != nil {
		if err := s.repo.SaveReport(ctx, report); err != nil {
			return nil, err
		}
	}
	return report.Capabilities, nil
}

func (s *Service) PlatformSummary(ctx context.Context) (PlatformSummary, error) {
	report, err := s.registry.Report(ctx)
	if err != nil {
		return PlatformSummary{}, err
	}
	stampReport(&report)
	if s.repo != nil {
		if err := s.repo.SaveReport(ctx, report); err != nil {
			return PlatformSummary{}, err
		}
	}
	return report.Platform, nil
}

func (s *Service) ExplainUnsupportedAction(ctx context.Context, action string) (Capability, []string, error) {
	action = strings.TrimSpace(action)
	if action == "" {
		action = "router_dns_enforcement"
	}
	caps, err := s.ListCapabilities(ctx)
	if err != nil {
		return Capability{}, nil, err
	}
	for _, cap := range caps {
		if cap.Action == action {
			return cap, manualSteps(cap), nil
		}
	}
	cap := Capability{
		Adapter:    "manual",
		Action:     action,
		Supported:  false,
		Reason:     "No adapter reports this action for the current platform.",
		ObservedAt: time.Now().UTC(),
	}
	return cap, manualSteps(cap), nil
}

func stampReport(report *Report) {
	if report.ObservedAt.IsZero() {
		report.ObservedAt = time.Now().UTC()
	}
	if report.Platform.ObservedAt.IsZero() {
		report.Platform.ObservedAt = report.ObservedAt
	}
	for i := range report.Capabilities {
		if report.Capabilities[i].ObservedAt.IsZero() {
			report.Capabilities[i].ObservedAt = report.ObservedAt
		}
	}
}

func manualSteps(cap Capability) []string {
	if cap.Supported {
		return []string{"This action is supported by the reported adapter; use the owning workflow command instead of manual steps."}
	}
	if cap.Action == "router_dns_enforcement" {
		return []string{
			"Open the router administration interface manually.",
			"Review current DNS settings before changing them.",
			"Apply only changes that have a Network Manager preview and rollback plan.",
		}
	}
	return []string{
		"Keep using read-only diagnostics for this action.",
		"Configure a supported adapter or resource before attempting persistent network changes.",
	}
}
