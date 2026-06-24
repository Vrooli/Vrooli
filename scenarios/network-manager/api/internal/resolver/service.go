package resolver

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

const AdGuardHomeBackend = "adguard-home"

type Service struct {
	repo   Repository
	client AdGuardClient
}

type Config struct {
	Repo   Repository
	Client AdGuardClient
}

func NewService(cfg Config) *Service {
	s := &Service{repo: cfg.Repo, client: cfg.Client}
	if s.client == nil {
		s.client = ConservativeAdGuardClient{}
	}
	return s
}

func (s *Service) Status(ctx context.Context) (Status, error) {
	cfg, err := s.repo.GetBackend(ctx, AdGuardHomeBackend)
	if errors.Is(err, ErrNotFound) {
		return Status{
			Backend:  AdGuardHomeBackend,
			Status:   "not_configured",
			Warnings: []string{"No governed AdGuard Home backend is configured."},
		}, nil
	}
	if err != nil {
		return Status{}, err
	}
	return s.statusFromClient(ctx, cfg)
}

func (s *Service) ConfigureAdGuardHome(ctx context.Context, baseURL, username, tokenRef string, dryRun bool) (Status, []string, error) {
	cfg, err := normalizeConfig(baseURL, username, tokenRef)
	if err != nil {
		return Status{}, nil, err
	}
	if dryRun {
		return Status{
			Backend:  AdGuardHomeBackend,
			Status:   "dry_run",
			BaseURL:  cfg.BaseURL,
			Warnings: []string{"Dry run only; credentials were not stored and resolver state was not changed."},
		}, []string{"Configuration shape is valid.", "Store the credential in the Vrooli secret system and pass only its token_ref."}, nil
	}
	saved, err := s.repo.SaveBackend(ctx, cfg)
	if err != nil {
		return Status{}, nil, err
	}
	status, err := s.statusFromClient(ctx, saved)
	if err != nil {
		return Status{}, nil, err
	}
	return status, []string{"Stored AdGuard Home backend using a secret reference.", "Filtering remains unclaimed until health confirms it."}, nil
}

func (s *Service) UpdateUpstreams(ctx context.Context, upstreams []string, dryRun bool) (Status, []string, error) {
	cleaned, err := normalizeUpstreams(upstreams)
	if err != nil {
		return Status{}, nil, err
	}
	cfg, err := s.repo.GetBackend(ctx, AdGuardHomeBackend)
	if errors.Is(err, ErrNotFound) {
		return Status{}, nil, fmt.Errorf("configure AdGuard Home before updating upstreams")
	}
	if err != nil {
		return Status{}, nil, err
	}
	if dryRun {
		changes, err := s.client.PreviewUpstreams(ctx, cfg, cleaned)
		if err != nil {
			return Status{}, nil, err
		}
		status, err := s.statusFromClient(ctx, cfg)
		if err != nil {
			return Status{}, nil, err
		}
		return status, changes, nil
	}
	clientStatus, changes, err := s.client.UpdateUpstreams(ctx, cfg, cleaned)
	if err != nil {
		return Status{}, nil, err
	}
	if err := s.repo.UpdateUpstreams(ctx, AdGuardHomeBackend, cleaned); err != nil {
		return Status{}, nil, err
	}
	return fromClientStatus(cfg, clientStatus), changes, nil
}

func (s *Service) Health(ctx context.Context) (Status, []string, error) {
	cfg, err := s.repo.GetBackend(ctx, AdGuardHomeBackend)
	if errors.Is(err, ErrNotFound) {
		status := Status{Backend: AdGuardHomeBackend, Status: "not_configured", Warnings: []string{"No AdGuard Home backend is configured."}}
		return status, []string{"No backend configuration found."}, nil
	}
	if err != nil {
		return Status{}, nil, err
	}
	clientStatus, err := s.client.Check(ctx, cfg)
	if err != nil {
		return Status{}, nil, err
	}
	return fromClientStatus(cfg, clientStatus), clientStatus.Checks, nil
}

func (s *Service) statusFromClient(ctx context.Context, cfg BackendConfig) (Status, error) {
	clientStatus, err := s.client.Check(ctx, cfg)
	if err != nil {
		return Status{}, err
	}
	status := fromClientStatus(cfg, clientStatus)
	if len(status.Upstreams) == 0 {
		upstreams, err := s.repo.GetUpstreams(ctx, AdGuardHomeBackend)
		if err != nil {
			return Status{}, err
		}
		status.Upstreams = upstreams
	}
	return status, nil
}

func normalizeConfig(baseURL, username, tokenRef string) (BackendConfig, error) {
	baseURL = firstNonEmpty(baseURL, os.Getenv("ADGUARD_HOME_BASE_URL"), os.Getenv("ADGUARD_HOME_URL"))
	username = firstNonEmpty(username, os.Getenv("ADGUARD_HOME_USERNAME"))
	tokenRef = firstNonEmpty(tokenRef, os.Getenv("ADGUARD_HOME_CREDENTIAL_REF"))
	if baseURL == "" {
		return BackendConfig{}, fmt.Errorf("base_url is required")
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return BackendConfig{}, fmt.Errorf("base_url must be an absolute URL")
	}
	if tokenRef == "" {
		return BackendConfig{}, fmt.Errorf("token_ref is required; plaintext resolver tokens are not accepted")
	}
	return BackendConfig{Backend: AdGuardHomeBackend, BaseURL: strings.TrimRight(baseURL, "/"), Username: username, TokenRef: tokenRef}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func normalizeUpstreams(values []string) ([]string, error) {
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one upstream is required")
	}
	return out, nil
}

func fromClientStatus(cfg BackendConfig, clientStatus ClientStatus) Status {
	status := clientStatus.Status
	if status == "" {
		status = "unknown"
	}
	return Status{
		Backend:          cfg.Backend,
		Status:           status,
		BaseURL:          cfg.BaseURL,
		Upstreams:        clientStatus.Upstreams,
		FilteringEnabled: clientStatus.FilteringEnabled,
		Warnings:         clientStatus.Warnings,
	}
}

func joinUpstreams(upstreams []string) string {
	if len(upstreams) == 0 {
		return "(none)"
	}
	return strings.Join(upstreams, ", ")
}
