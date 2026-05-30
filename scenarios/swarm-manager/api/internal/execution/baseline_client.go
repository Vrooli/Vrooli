package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
	"github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines/baselines_v1connect"
)

// BaselineClient captures and compares git-control-tower baselines so that
// finalization can distinguish regressions introduced by a backlog item from
// failures that pre-existed it. A nil BaselineClient disables the before/after
// baseline feature entirely (graceful degradation to the absolute review path).
//
// DOC: docs/internal/SEAMS.md#baseline-client
type BaselineClient interface {
	// EnsureSnapshot guarantees a baseline named `name` exists for `scenario`.
	// When a baseline of that name already exists it is reused (cached=true)
	// and no fresh capture runs; otherwise a new snapshot is captured across
	// all review surfaces (cached=false). Capturing is the expensive path —
	// callers key `name` on working-tree state so steady-state hits the cache.
	EnsureSnapshot(ctx context.Context, scenario, name string) (cached bool, err error)
	// Diff compares the named baseline against the current working tree and
	// returns the structured new-vs-pre-existing delta.
	Diff(ctx context.Context, scenario, name string) (BaselineDiffResult, error)
	// Delete removes the named baseline and unpins its test-genie runs.
	Delete(ctx context.Context, scenario, name string) error
	// Ping checks whether git-control-tower's baseline service is reachable.
	Ping(ctx context.Context) error
}

// SurfaceFinding is one finding (test name, violation, etc.) attributed to a
// specific review surface (tests / standards / structure / visuals / workflows).
type SurfaceFinding struct {
	Surface string `json:"surface"`
	Detail  string `json:"detail"`
}

// BaselineDiffResult is the execution package's neutral view of a baseline diff.
//
// Verdict is git-control-tower's overall verdict string: one of
// "clean", "regression", "new-failure", "preexisting", "not-comparable".
// ExitCode mirrors the CLI's process exit semantics (0 safe, 1 regression,
// 2 not-comparable) so callers can reason about severity without re-deriving it.
//
// The four finding buckets follow git-control-tower's surface-diff split:
//   - Regressions: passing in the baseline, failing now — caused by this change.
//   - NewFailures: absent in the baseline, failing now — NOT caused by this change.
//   - PreExisting: failing in both the baseline and now — pre-existing debt.
//   - Cleared: failing in the baseline, passing now — fixed by this change.
type BaselineDiffResult struct {
	ScenarioName      string           `json:"scenario_name"`
	Verdict           string           `json:"verdict"`
	ExitCode          int              `json:"exit_code"`
	Comparable        bool             `json:"comparable"`
	Stale             bool             `json:"stale,omitempty"`
	RegressedSurfaces []string         `json:"regressed_surfaces,omitempty"`
	Regressions       []SurfaceFinding `json:"regressions,omitempty"`
	NewFailures       []SurfaceFinding `json:"new_failures,omitempty"`
	PreExisting       []SurfaceFinding `json:"preexisting,omitempty"`
	Cleared           []SurfaceFinding `json:"cleared,omitempty"`
}

const (
	baselineVerdictClean         = "clean"
	baselineVerdictRegression    = "regression"
	baselineVerdictNewFailure    = "new-failure"
	baselineVerdictPreExisting   = "preexisting"
	baselineVerdictNotComparable = "not-comparable"
)

// exitCodeForBaselineVerdict mirrors git-control-tower's CLI mapping:
// regression → 1, not-comparable → 2, everything else → 0.
func exitCodeForBaselineVerdict(verdict string) int {
	switch verdict {
	case baselineVerdictRegression:
		return 1
	case baselineVerdictNotComparable:
		return 2
	default:
		return 0
	}
}

// ConnectBaselineClient implements BaselineClient against git-control-tower's
// BaselinesService over Connect-RPC. The base URL is re-resolved on every call
// via discovery so the client survives git-control-tower restarts.
type ConnectBaselineClient struct {
	httpClient *http.Client
	resolveURL func(ctx context.Context) (string, error)
	createdBy  string
}

// NewConnectBaselineClient creates a baseline client. If httpClient is nil a
// default client with a 30-minute timeout is used (snapshot capture runs the
// full test-genie surface set and can take several minutes).
func NewConnectBaselineClient(httpClient *http.Client) *ConnectBaselineClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Minute}
	}
	return &ConnectBaselineClient{
		httpClient: httpClient,
		resolveURL: func(ctx context.Context) (string, error) {
			return resolveGitControlTowerBaseURL(ctx)
		},
		createdBy: "swarm-manager",
	}
}

func (c *ConnectBaselineClient) client(ctx context.Context) (baselines_v1connect.BaselinesServiceClient, error) {
	baseURL, err := c.resolveURL(ctx)
	if err != nil {
		return nil, err
	}
	return baselines_v1connect.NewBaselinesServiceClient(c.httpClient, baseURL), nil
}

// EnsureSnapshot reuses an existing baseline of the given name when present,
// otherwise captures a fresh one across all surfaces.
func (c *ConnectBaselineClient) EnsureSnapshot(ctx context.Context, scenario, name string) (bool, error) {
	cl, err := c.client(ctx)
	if err != nil {
		return false, err
	}

	// Cache check: a baseline of this name already pins the current
	// working-tree state, so reuse it rather than recapturing.
	getResp, getErr := cl.GetBaseline(ctx, connect.NewRequest(&baselinesv1.GetBaselineRequest{
		Scenario: scenario,
		Name:     name,
	}))
	if getErr == nil && getResp != nil && getResp.Msg.GetBaseline() != nil {
		return true, nil
	}
	if getErr != nil && connect.CodeOf(getErr) != connect.CodeNotFound {
		return false, fmt.Errorf("check baseline %s/%s: %w", scenario, name, getErr)
	}

	_, snapErr := cl.SnapshotForBaseline(ctx, connect.NewRequest(&baselinesv1.SnapshotForBaselineRequest{
		Scenario:  scenario,
		Name:      name,
		CreatedBy: c.createdBy,
		Reason:    "pre-execution baseline for before/after regression diff",
	}))
	if snapErr != nil {
		// A concurrent execution may have captured the same-named baseline
		// between our Get and Snapshot — treat AlreadyExists as a cache hit.
		if connect.CodeOf(snapErr) == connect.CodeAlreadyExists {
			return true, nil
		}
		return false, fmt.Errorf("snapshot baseline %s/%s: %w", scenario, name, snapErr)
	}
	return false, nil
}

// Diff compares the named baseline against the working tree.
func (c *ConnectBaselineClient) Diff(ctx context.Context, scenario, name string) (BaselineDiffResult, error) {
	cl, err := c.client(ctx)
	if err != nil {
		return BaselineDiffResult{}, err
	}
	resp, err := cl.DiffBaseline(ctx, connect.NewRequest(&baselinesv1.DiffBaselineRequest{
		Scenario: scenario,
		Name:     name,
	}))
	if err != nil {
		return BaselineDiffResult{}, fmt.Errorf("diff baseline %s/%s: %w", scenario, name, err)
	}
	return baselineDiffResultFromProto(scenario, resp.Msg), nil
}

// Delete removes the named baseline.
func (c *ConnectBaselineClient) Delete(ctx context.Context, scenario, name string) error {
	cl, err := c.client(ctx)
	if err != nil {
		return err
	}
	_, err = cl.DeleteBaseline(ctx, connect.NewRequest(&baselinesv1.DeleteBaselineRequest{
		Scenario: scenario,
		Name:     name,
	}))
	if err != nil {
		// Deleting an already-absent baseline is not an error for cleanup.
		if connect.CodeOf(err) == connect.CodeNotFound {
			return nil
		}
		return fmt.Errorf("delete baseline %s/%s: %w", scenario, name, err)
	}
	return nil
}

// Ping verifies the baseline service is reachable with a bounded timeout.
func (c *ConnectBaselineClient) Ping(ctx context.Context) error {
	cl, err := c.client(ctx)
	if err != nil {
		return err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	// ListBaselines for a sentinel scenario is a cheap reachability probe;
	// an empty list (or NotFound) still proves the service answered.
	_, err = cl.ListBaselines(pingCtx, connect.NewRequest(&baselinesv1.ListBaselinesRequest{
		Scenario: "swarm-manager",
	}))
	if err != nil && connect.CodeOf(err) != connect.CodeNotFound {
		return fmt.Errorf("baseline service unreachable: %w", err)
	}
	return nil
}

// baselineDiffResultFromProto folds the proto DiffBaselineResponse into the
// execution package's neutral BaselineDiffResult, attributing every per-surface
// finding to its surface and deriving the CLI-equivalent exit code.
func baselineDiffResultFromProto(scenario string, msg *baselinesv1.DiffBaselineResponse) BaselineDiffResult {
	out := BaselineDiffResult{
		ScenarioName: scenario,
		Verdict:      baselineVerdictClean,
		Comparable:   true,
	}
	if msg == nil {
		return out
	}
	out.Verdict = msg.GetVerdict()
	out.ExitCode = exitCodeForBaselineVerdict(out.Verdict)
	out.Comparable = out.Verdict != baselineVerdictNotComparable
	if st := msg.GetStaleness(); st != nil {
		out.Stale = st.GetLikelyStale()
	}

	for _, sd := range msg.GetSurfaces() {
		if sd == nil {
			continue
		}
		surface := sd.GetSurfaceId()
		if sd.GetVerdict() == baselineVerdictRegression {
			out.RegressedSurfaces = append(out.RegressedSurfaces, surface)
		}
		out.Regressions = appendFindings(out.Regressions, surface, sd.GetRegressions())
		out.NewFailures = appendFindings(out.NewFailures, surface, sd.GetNewFailures())
		out.PreExisting = appendFindings(out.PreExisting, surface, sd.GetPreexisting())
		out.Cleared = appendFindings(out.Cleared, surface, sd.GetCleared())
	}
	return out
}

func appendFindings(dst []SurfaceFinding, surface string, items []string) []SurfaceFinding {
	for _, item := range items {
		dst = append(dst, SurfaceFinding{Surface: surface, Detail: item})
	}
	return dst
}

// HasNewRegressions reports whether the diff surfaced any regression this
// change is responsible for (the agent's priority signal).
func (r BaselineDiffResult) HasNewRegressions() bool {
	return len(r.Regressions) > 0
}

// MarshalBaselineDiffResults serializes a per-scenario diff map to JSON for use
// as a review-agent context attachment. Returns "" for an empty map.
func MarshalBaselineDiffResults(results map[string]BaselineDiffResult) string {
	if len(results) == 0 {
		return ""
	}
	data, err := json.Marshal(results)
	if err != nil {
		return ""
	}
	return string(data)
}
