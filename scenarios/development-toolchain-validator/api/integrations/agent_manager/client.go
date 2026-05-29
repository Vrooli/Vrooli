// Package agent_manager hosts the outbound HTTP client to agent-manager.
// It resolves agent-manager's base URL via api-core/discovery and
// implements the validation_run worker's AgentManagerClient seam by
// composing three agent-manager surfaces:
//
//   - POST /api/v1/profiles/reconcile-scenario   (Initialize)
//   - POST /api/v1/tasks + POST /api/v1/runs     (StartSandboxedRun)
//   - GET  /api/v1/runs/{id} + .../diff          (WaitForTerminal)
//
// The profile is owned by DTV on disk at .vrooli/agent-profiles/*.json
// and reconciled into agent-manager at startup; runs reference the
// profile by its stable key ("development-toolchain-validator/default")
// so UI edits on the agent-manager side are not clobbered by code
// defaults. See docs/internal/SEAMS.md for the seam contract.
package agent_manager

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	manifest "development-toolchain-validator/internal/manifest"
	vrun "development-toolchain-validator/internal/validation_run"

	"development-toolchain-validator/internal/httpc"

	"github.com/vrooli/api-core/discovery"
	repocontract "github.com/vrooli/repo-contract-go"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ScenarioName is the slug DTV registers under in discovery /
// agent-manager scenario lookups. Exported so the main wiring can
// pass the same string to Initialize without redeclaring it.
const ScenarioName = "development-toolchain-validator"

// ProfileKey is the stable profile key declared in
// .vrooli/agent-profiles/default.json. Run requests reference the
// profile by this key so UI edits to the persisted row are preserved
// across restarts.
const ProfileKey = "development-toolchain-validator/default"

// pollInterval is how often WaitForTerminal asks agent-manager for the
// latest run status. Kept short because skill validations are usually
// short-lived (single skill, single golden); a longer interval would
// dominate observed end-to-end latency.
const defaultPollInterval = 2 * time.Second

// Options configures the adapter.
type Options struct {
	Resolver    *discovery.Resolver
	Doer        httpc.Doer
	MaxAttempts int
	// PollInterval overrides the default WaitForTerminal cadence (tests
	// use a much shorter value; production keeps defaultPollInterval).
	PollInterval time.Duration
}

// Client is the HTTP adapter implementing vrun.AgentManagerClient.
type Client struct {
	opts Options
}

// New constructs a Client. Empty fields fall back to production
// defaults: discovery via the default resolver, a 30s HTTP timeout,
// three transport retries, and a 2s poll cadence.
func New(opts Options) *Client {
	if opts.Resolver == nil {
		opts.Resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	if opts.Doer == nil {
		opts.Doer = &http.Client{Timeout: 30 * time.Second}
	}
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 3
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = defaultPollInterval
	}
	return &Client{opts: opts}
}

var _ vrun.AgentManagerClient = (*Client)(nil)

var (
	protoJSONMarshal   = protojson.MarshalOptions{UseProtoNames: false}
	protoJSONUnmarshal = protojson.UnmarshalOptions{DiscardUnknown: true}
)

// Initialize asks agent-manager to reconcile DTV's profile sources
// declared in .vrooli/service.json. Call once at startup before the
// validation_run worker dispatches its first skill run. Failures here
// are surfaced to main; the operator decides whether the scenario
// should boot without a reconciled profile (today: it should not, the
// dep is required).
func (c *Client) Initialize(ctx context.Context) (*apipb.ReconcileScenarioProfilesResponse, error) {
	req := &apipb.ReconcileScenarioProfilesRequest{Scenario: ScenarioName}
	resp := &apipb.ReconcileScenarioProfilesResponse{}
	if err := c.doProto(ctx, http.MethodPost, "/api/v1/profiles/reconcile-scenario", req, resp); err != nil {
		return nil, fmt.Errorf("reconcile scenario profiles: %w", err)
	}
	found := false
	for _, item := range resp.Results {
		if item.ProfileKey == ProfileKey {
			found = true
			break
		}
	}
	if !found {
		return resp, fmt.Errorf("reconcile scenario profiles: profile %q not returned (failed=%d)", ProfileKey, resp.Failed)
	}
	return resp, nil
}

// StartSandboxedRun creates one Task + one sandboxed Run in
// agent-manager and returns the run id. The task and run are
// independent entities per agent-manager's contract — DTV does not
// reuse tasks across runs because each (skill, golden) validation is
// conceptually a fresh execution against a pristine golden.
func (c *Client) StartSandboxedRun(ctx context.Context, spec vrun.SandboxedRunSpec) (string, error) {
	// The sandbox overlays ProjectRoot as the writable lower layer and
	// joins a relative ScopePath onto it. agent-manager resolves a
	// relative ProjectRoot against ITS OWN cwd (not DTV's), so passing the
	// repo-relative golden path previously produced an empty, doubled
	// lowerdir and the golden was never seeded into the workspace. We
	// resolve the golden to an absolute host path and point ProjectRoot at
	// the golden tree itself with ScopePath "." so the overlay lower layer
	// IS the golden and the agent's edits become the observed diff.
	goldenRoot, err := resolveGoldenRoot(spec.GoldenPath)
	if err != nil {
		return "", err
	}
	task := &domainpb.Task{
		Title:       fmt.Sprintf("Validate skill %q against golden %q", spec.SkillID, spec.GoldenSlug),
		Description: buildSkillPrompt(spec),
		ScopePath:   ".",
		ProjectRoot: goldenRoot,
		CreatedBy:   ScenarioName,
	}
	createTaskReq := &apipb.CreateTaskRequest{Task: task}
	createTaskResp := &apipb.CreateTaskResponse{}
	if err := c.doProto(ctx, http.MethodPost, "/api/v1/tasks", createTaskReq, createTaskResp); err != nil {
		return "", err
	}
	if createTaskResp.Task == nil || strings.TrimSpace(createTaskResp.Task.Id) == "" {
		return "", fmt.Errorf("agent-manager returned empty task id")
	}

	tag := fmt.Sprintf("%s:skill:%s:%s", ScenarioName, spec.SkillID, spec.GoldenSlug)
	profileKey := ProfileKey
	runReq := &apipb.CreateRunRequest{
		TaskId:     createTaskResp.Task.Id,
		ProfileRef: &apipb.ProfileRef{ProfileKey: profileKey},
		Tag:        &tag,
		Force:      true,
	}
	runResp := &apipb.CreateRunResponse{}
	if err := c.doProto(ctx, http.MethodPost, "/api/v1/runs", runReq, runResp); err != nil {
		return "", err
	}
	if runResp.Run == nil || strings.TrimSpace(runResp.Run.Id) == "" {
		return "", fmt.Errorf("agent-manager returned empty run id")
	}
	return runResp.Run.Id, nil
}

// WaitForTerminal polls GetRun until the run reaches a terminal status
// (complete / failed / cancelled / needs_review), then fetches the
// run's diff so the evaluator can apply manifest path-globs against
// the changed files. The needs_review status is treated as terminal
// for DTV because the diff is final once it appears; whether an
// operator later approves the apply does not change the manifest
// verdict over the diff content.
func (c *Client) WaitForTerminal(ctx context.Context, runID string, timeout time.Duration) (vrun.RunSummary, error) {
	if strings.TrimSpace(runID) == "" {
		return vrun.RunSummary{}, fmt.Errorf("agent-manager wait: empty run id")
	}
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(c.opts.PollInterval)
	defer ticker.Stop()

	for {
		run, err := c.getRun(ctx, runID)
		if err != nil {
			return vrun.RunSummary{AgentManagerRunID: runID}, err
		}
		if isTerminal(run.Status) {
			return c.buildSummary(ctx, run)
		}
		if time.Now().After(deadline) {
			return vrun.RunSummary{AgentManagerRunID: runID}, fmt.Errorf("agent-manager wait: timed out after %s (last status=%s)", timeout, run.Status)
		}
		select {
		case <-ctx.Done():
			return vrun.RunSummary{AgentManagerRunID: runID}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func isTerminal(s domainpb.RunStatus) bool {
	switch s {
	case domainpb.RunStatus_RUN_STATUS_COMPLETE,
		domainpb.RunStatus_RUN_STATUS_FAILED,
		domainpb.RunStatus_RUN_STATUS_CANCELLED,
		domainpb.RunStatus_RUN_STATUS_NEEDS_REVIEW:
		return true
	}
	return false
}

func (c *Client) buildSummary(ctx context.Context, run *domainpb.Run) (vrun.RunSummary, error) {
	summary := vrun.RunSummary{AgentManagerRunID: run.Id}
	if run.StartedAt != nil {
		summary.StartedAt = run.StartedAt.AsTime().UTC()
	}
	if run.EndedAt != nil {
		summary.EndedAt = run.EndedAt.AsTime().UTC()
	}
	if run.Summary != nil {
		summary.TokensUsed = int64(run.Summary.TokensUsed)
		// CostEstimate is USD as float64; convert to micro-USD (int64)
		// for the integer-only record column without surprising the
		// reader at the database boundary.
		summary.CostUSDMicro = int64(run.Summary.CostEstimate * 1_000_000)
	}
	// FAILED runs may still carry a diff when the failure happened
	// after some files were mutated; ask for it but tolerate absence.
	diff, err := c.getRunDiff(ctx, run.Id)
	if err == nil && diff != nil {
		// DiffHash is a content hash so that identical diffs across
		// distinct runs collapse to the same value — that equality is
		// what lets DTV recognise a stable, converged skill behaviour.
		// A no-mutation run hashes the empty string (a stable constant),
		// which reads as "ran, produced no changes".
		summary.DiffHash = sha256Hex(diff.Content)
		summary.DiffPaths = make([]manifest.DiffFile, 0, len(diff.Files))
		for _, file := range diff.Files {
			summary.DiffPaths = append(summary.DiffPaths, manifest.DiffFile{
				Path: file.Path,
			})
		}
	}
	return summary, nil
}

func (c *Client) getRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	out := &apipb.GetRunResponse{}
	if err := c.doProto(ctx, http.MethodGet, "/api/v1/runs/"+url.PathEscape(runID), nil, out); err != nil {
		return nil, err
	}
	if out.Run == nil {
		return nil, fmt.Errorf("agent-manager GetRun: empty run")
	}
	return out.Run, nil
}

func (c *Client) getRunDiff(ctx context.Context, runID string) (*domainpb.RunDiff, error) {
	out := &apipb.GetRunDiffResponse{}
	if err := c.doProto(ctx, http.MethodGet, "/api/v1/runs/"+url.PathEscape(runID)+"/diff", nil, out); err != nil {
		return nil, err
	}
	return out.Diff, nil
}

// doProto centralizes URL resolution, retry on transport failure, proto
// JSON marshal/unmarshal, and error translation into vrun's typed
// ErrDependencyUnavailable.
func (c *Client) doProto(ctx context.Context, method, path string, in, out proto.Message) error {
	var body []byte
	if in != nil {
		buf, err := protoJSONMarshal.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = buf
	}

	var lastErr error
	for attempt := 0; attempt < c.opts.MaxAttempts; attempt++ {
		base, err := c.opts.Resolver.ResolveScenarioURLDefault(ctx, "agent-manager")
		if err != nil {
			if discovery.IsScenarioNotRunning(err) {
				return vrun.ErrDependencyUnavailable{Dependency: "agent-manager", Reason: "scenario not running"}
			}
			lastErr = err
			continue
		}
		respBytes, status, err := c.do(ctx, method, base+path, body)
		if err != nil {
			lastErr = err
			if isRetriable(err) {
				continue
			}
			return err
		}
		if status >= 500 {
			lastErr = fmt.Errorf("agent-manager upstream %d: %s", status, truncate(string(respBytes)))
			continue
		}
		if status >= 400 {
			return fmt.Errorf("agent-manager request failed: status %d: %s", status, truncate(string(respBytes)))
		}
		if out != nil && len(respBytes) > 0 {
			if err := protoJSONUnmarshal.Unmarshal(respBytes, out); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
		}
		return nil
	}
	if lastErr == nil {
		lastErr = errors.New("exhausted retries")
	}
	return vrun.ErrDependencyUnavailable{Dependency: "agent-manager", Reason: lastErr.Error()}
}

func (c *Client) do(ctx context.Context, method, fullURL string, body []byte) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reader)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.opts.Doer.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return respBytes, resp.StatusCode, nil
}

func isRetriable(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "upstream 5")
}

// buildSkillPrompt assembles the agent's prompt for a skill run. The
// sandbox working directory is the pristine golden scenario; the agent's
// job is to apply the steer skill to it exactly as it would during normal
// agent-driven development, so the resulting filesystem diff is the
// artifact DTV evaluates against the skill's expected-diff manifest.
//
// When the skill content could not be fetched, we fall back to a generic
// description that at least names the skill — the run will still execute
// but the operator should treat its diff with suspicion (the worker logs
// the fetch failure).
func buildSkillPrompt(spec vrun.SandboxedRunSpec) string {
	if strings.TrimSpace(spec.SkillPrompt) == "" {
		return fmt.Sprintf("Sandboxed execution of skill %s on golden %s for DTV manifest evaluation.", spec.SkillID, spec.GoldenSlug)
	}
	var b strings.Builder
	b.WriteString("You are validating a Vrooli steer skill against a pristine golden scenario.\n\n")
	b.WriteString("Your working directory is the golden scenario ")
	b.WriteString(fmt.Sprintf("%q (generated from its template, unmodified). ", spec.GoldenSlug))
	b.WriteString("Apply the skill below to THIS scenario exactly as you would during normal ")
	b.WriteString("agent-driven development: make only the file changes the skill actually implies, ")
	b.WriteString("and make no other changes. If the skill's intent is already satisfied by the ")
	b.WriteString("current scenario, make no changes at all — an empty diff is a valid, expected outcome. ")
	b.WriteString("Do not run builds, tests, or formatters unless the skill explicitly requires it; ")
	b.WriteString("the only artifact that matters is the filesystem diff your edits produce.\n\n")
	b.WriteString(fmt.Sprintf("=== BEGIN SKILL: %s ===\n", spec.SkillID))
	b.WriteString(spec.SkillPrompt)
	b.WriteString(fmt.Sprintf("\n=== END SKILL: %s ===\n", spec.SkillID))
	return b.String()
}

// resolveGoldenRoot turns a (possibly repo-relative) golden path into an
// absolute host path. Goldens are registered with repo-relative paths
// (e.g. "scenarios/reference-react-vite") for portability, but the
// sandbox needs an absolute path it can overlay. The repo root is
// resolved from VROOLI_SOURCE_ROOT/VROOLI_ROOT (set by the lifecycle)
// with a cwd/executable fallback.
func resolveGoldenRoot(goldenPath string) (string, error) {
	p := strings.TrimSpace(goldenPath)
	if p == "" {
		return "", fmt.Errorf("golden path is empty")
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return "", fmt.Errorf("resolve repo root for golden path %q: %w", p, err)
	}
	return filepath.Join(root, filepath.FromSlash(p)), nil
}

// sha256Hex returns the lowercase hex SHA-256 digest of s. Used to
// hash a run's unified diff content so identical diffs across runs are
// recognisably equal. The empty string hashes to a stable constant.
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func truncate(s string) string {
	const max = 512
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
