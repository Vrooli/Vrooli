package monitor

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/validationrunner"
)

const (
	defaultInterval = 24 * time.Hour
	defaultTemplate = "react-vite"
	schedulerSource = "scheduler"
)

type Repository interface {
	GetMonitorStatus(ctx context.Context) (catalog.MonitorStatus, error)
	SaveMonitorStatus(ctx context.Context, status catalog.MonitorStatus) error
	ListTemplates(ctx context.Context, kind catalog.TemplateKind) ([]catalog.TemplateRecord, error)
	DeepValidateGreenStreak(ctx context.Context, templateID string) (int64, error)
}

type ValidationService interface {
	RunValidation(ctx context.Context, req validationrunner.ValidateRequest) (catalog.ValidationRun, error)
}

type Config struct {
	Enabled     bool
	Interval    time.Duration
	RunOnStart  bool
	TemplateIDs []string
}

func ConfigFromEnv() Config {
	return Config{
		Enabled:    envBool("TEMPLATE_MANAGER_MONITOR_ENABLED", true),
		Interval:   envDuration("TEMPLATE_MANAGER_MONITOR_INTERVAL", defaultInterval),
		RunOnStart: envBool("TEMPLATE_MANAGER_MONITOR_RUN_ON_START", false),
	}
}

type Service struct {
	repo       Repository
	validator  ValidationService
	config     Config
	logger     *log.Logger
	now        func() time.Time
	mu         sync.Mutex
	running    bool
	cancelLoop context.CancelFunc
}

func NewService(repo Repository, validator ValidationService, config Config, logger *log.Logger) *Service {
	if config.Interval <= 0 {
		config.Interval = defaultInterval
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Service{
		repo:      repo,
		validator: validator,
		config:    config,
		logger:    logger,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Status(ctx context.Context) (catalog.MonitorStatus, error) {
	status, err := s.repo.GetMonitorStatus(ctx)
	if err != nil {
		return catalog.MonitorStatus{}, err
	}
	s.mu.Lock()
	status.InFlight = status.InFlight || s.running
	s.mu.Unlock()
	status.Enabled = s.config.Enabled
	status.IntervalSeconds = int64(s.config.Interval.Seconds())
	if status.NextRunAt.IsZero() {
		status.NextRunAt = s.now().Add(s.config.Interval)
	}
	return status, nil
}

func (s *Service) Start(ctx context.Context) {
	if !s.config.Enabled {
		_ = s.persistBaseStatus(ctx, false, "disabled")
		return
	}
	if s.validator == nil {
		s.logger.Printf("template monitor disabled: validation service is nil")
		return
	}
	loopCtx, cancel := context.WithCancel(ctx)
	s.cancelLoop = cancel
	go s.loop(loopCtx)
}

func (s *Service) Stop() {
	if s.cancelLoop != nil {
		s.cancelLoop()
	}
}

func (s *Service) RunDue(ctx context.Context) (catalog.MonitorStatus, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		status, err := s.Status(ctx)
		if err != nil {
			return catalog.MonitorStatus{}, err
		}
		status.LastStatus = "skipped_busy"
		status.UpdatedAt = s.now()
		_ = s.repo.SaveMonitorStatus(ctx, status)
		return status, nil
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	started := s.now()
	status := catalog.MonitorStatus{
		ID:              "default",
		Enabled:         s.config.Enabled,
		IntervalSeconds: int64(s.config.Interval.Seconds()),
		InFlight:        true,
		LastStatus:      "running",
		LastStartedAt:   started,
		NextRunAt:       started.Add(s.config.Interval),
		UpdatedAt:       started,
	}
	_ = s.repo.SaveMonitorStatus(ctx, status)

	templates, err := s.templateIDs(ctx)
	if err != nil {
		return s.finish(ctx, status, "", "failed", started, err)
	}
	var lastRun catalog.ValidationRun
	runStatus := "passed"
	for _, templateID := range templates {
		run, err := s.validator.RunValidation(ctx, validationrunner.ValidateRequest{
			TemplateID: templateID,
			Mode:       catalog.ModeDeep,
			Trigger:    schedulerSource,
		})
		if err != nil {
			return s.finish(ctx, status, lastRun.ID, "failed", started, err)
		}
		lastRun = run
		if run.Status != "passed" {
			runStatus = "failed"
		}
	}
	return s.finish(ctx, status, lastRun.ID, runStatus, started, nil)
}

func (s *Service) loop(ctx context.Context) {
	next := s.now().Add(s.config.Interval)
	if s.config.RunOnStart {
		next = s.now()
	}
	status, err := s.repo.GetMonitorStatus(ctx)
	if err != nil {
		status = catalog.MonitorStatus{ID: "default", LastStatus: "scheduled"}
	}
	status.Enabled = true
	status.IntervalSeconds = int64(s.config.Interval.Seconds())
	status.InFlight = false
	status.NextRunAt = next
	status.UpdatedAt = s.now()
	if status.LastStatus == "" || status.LastStatus == "never-run" {
		status.LastStatus = "scheduled"
	}
	_ = s.repo.SaveMonitorStatus(ctx, status)
	for {
		delay := time.Until(next)
		if delay < 0 {
			delay = 0
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if _, err := s.RunDue(ctx); err != nil {
				s.logger.Printf("template monitor run failed: %v", err)
			}
			next = s.now().Add(s.config.Interval)
		}
	}
}

func (s *Service) finish(ctx context.Context, status catalog.MonitorStatus, runID, lastStatus string, started time.Time, cause error) (catalog.MonitorStatus, error) {
	finished := s.now()
	streak, err := s.repo.DeepValidateGreenStreak(ctx, defaultTemplate)
	if err != nil && cause == nil {
		cause = err
	}
	status.InFlight = false
	status.LastRunID = runID
	status.LastStatus = lastStatus
	status.LastStartedAt = started
	status.LastFinishedAt = finished
	status.NextRunAt = finished.Add(s.config.Interval)
	status.GreenStreak = streak
	status.UpdatedAt = finished
	if err := s.repo.SaveMonitorStatus(ctx, status); err != nil && cause == nil {
		cause = err
	}
	if cause != nil {
		return status, fmt.Errorf("monitor deep validation: %w", cause)
	}
	return status, nil
}

func (s *Service) templateIDs(ctx context.Context) ([]string, error) {
	if len(s.config.TemplateIDs) > 0 {
		return s.config.TemplateIDs, nil
	}
	records, err := s.repo.ListTemplates(ctx, catalog.KindScenario)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(records))
	for _, record := range records {
		// Quarantined templates are intentionally excluded from scheduled
		// validation. Their known failures must not turn the active-template
		// monitor red; a manual validation is still available for diagnosis.
		if record.Status == "active" && strings.TrimSpace(record.ID) != "" {
			out = append(out, record.ID)
		}
	}
	if len(out) == 0 {
		out = append(out, defaultTemplate)
	}
	return out, nil
}

func (s *Service) persistBaseStatus(ctx context.Context, enabled bool, lastStatus string) error {
	now := s.now()
	return s.repo.SaveMonitorStatus(ctx, catalog.MonitorStatus{
		ID:              "default",
		Enabled:         enabled,
		IntervalSeconds: int64(s.config.Interval.Seconds()),
		LastStatus:      lastStatus,
		NextRunAt:       now.Add(s.config.Interval),
		UpdatedAt:       now,
	})
}

func envBool(name string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envDuration(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	if seconds, err := strconv.Atoi(raw); err == nil {
		return time.Duration(seconds) * time.Second
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}
