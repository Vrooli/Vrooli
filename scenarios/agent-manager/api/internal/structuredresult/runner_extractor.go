package structuredresult

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/rolepolicy"

	"github.com/google/uuid"
)

// RunnerExtractor executes the portable extraction role through the existing
// coding-runner and resource-policy seams. The returned bytes remain untrusted;
// Resolver always performs the authoritative local schema validation.
type RunnerExtractor struct {
	Roles      *rolepolicy.State
	Resolver   rolepolicy.Resolver
	Runners    runner.Registry
	WorkingDir string
	Timeout    time.Duration
}

func (e *RunnerExtractor) Extract(ctx context.Context, request ExtractRequest) (ExtractResponse, error) {
	if e == nil || e.Roles == nil || e.Resolver == nil || e.Runners == nil {
		return ExtractResponse{Abstained: true}, errors.New("structured extraction dependencies are unavailable")
	}
	resolution, err := e.Roles.Resolve(ctx, e.Resolver, request.RoleRef)
	if err != nil {
		return ExtractResponse{Abstained: true}, err
	}
	policy := resolution.Snapshot()
	for index, candidate := range policy.Candidates {
		if !candidate.Available {
			continue
		}
		backend, getErr := e.Runners.Get(candidate.RunnerType)
		if getErr != nil {
			continue
		}
		available, _ := backend.IsAvailable(ctx)
		if !available {
			continue
		}
		snapshot := *policy
		snapshot.SelectedIndex, snapshot.SelectedCandidate = index, candidate
		config := domain.DefaultRunConfig()
		config.RunnerType, config.Model, config.RoleRef = candidate.RunnerType, candidate.Model, request.RoleRef
		config.MaxTurns, config.NetworkAccess = 1, domain.NetworkAccessNone
		config.Timeout = e.Timeout
		if config.Timeout <= 0 {
			config.Timeout = 2 * time.Minute
		}
		config.PolicySnapshot = &snapshot
		config.SandboxConfig = &domain.SandboxConfig{Mode: domain.SandboxModeOff, NetworkMode: domain.NetworkAccessNone}
		runID := uuid.New()
		sink := &extractEventSink{}
		result, executeErr := backend.Execute(ctx, runner.ExecuteRequest{
			RunID: runID, Tag: "structured-extract-" + runID.String(), ResolvedConfig: config,
			WorkingDir: e.WorkingDir, EventSink: sink,
			SystemPrompt: "Extract exactly one value matching the supplied JSON Schema. Return only the JSON value. Do not use tools, modify files, or add commentary.",
			Prompt:       extractionPrompt(request),
		})
		if executeErr != nil || result == nil || !result.Success {
			continue
		}
		final := result.Result
		if final == nil {
			final = domain.ResolveRunResult(sink.eventsSnapshot(), true, result.ExitCode, "structured extraction")
		}
		if final == nil || final.Selection.Status != domain.FinalOutputSelectionSelected {
			continue
		}
		candidateJSON, _, status, _ := deterministicCandidate(final.FinalOutput)
		if status != domain.StructuredResultSuccess {
			continue
		}
		return ExtractResponse{Candidate: candidateJSON, Provider: string(candidate.RunnerType), Model: candidate.Model, PolicySnapshot: &snapshot}, nil
	}
	return ExtractResponse{Abstained: true, PolicySnapshot: policy}, errors.New("no portable extraction candidate completed successfully")
}

func extractionPrompt(request ExtractRequest) string {
	return fmt.Sprintf("JSON Schema:\n%s\n\nSource text:\n%s", strings.TrimSpace(string(request.Schema)), request.Source)
}

type extractEventSink struct {
	mu     sync.Mutex
	events []*domain.RunEvent
}

func (s *extractEventSink) Emit(event *domain.RunEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *extractEventSink) Close() error { return nil }

func (s *extractEventSink) eventsSnapshot() []*domain.RunEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*domain.RunEvent(nil), s.events...)
}

var _ Extractor = (*RunnerExtractor)(nil)
