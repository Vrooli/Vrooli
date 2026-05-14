package adoptions_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/adoptions"
	adoptmocks "react-component-library/internal/adoptions/mocks"
	"react-component-library/internal/components"
	"react-component-library/internal/testutil/mocks"
)

// fakeReporter records each Report call and returns a canned ref. It
// is the test seam standing in for the production swarm-manager CLI
// reporter.
type fakeReporter struct {
	mu     sync.Mutex
	events []adoptions.DriftEvent
	ref    string
	err    error
}

func (f *fakeReporter) Report(_ context.Context, ev adoptions.DriftEvent) (adoptions.DriftReport, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, ev)
	if f.err != nil {
		return adoptions.DriftReport{}, f.err
	}
	return adoptions.DriftReport{Ref: f.ref}, nil
}

func (f *fakeReporter) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

func sha256hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// TestService_Refresh_FilesDriftBacklogOnce verifies the IS-001
// idempotency contract: a fresh drift triggers a reporter call and
// stores the returned ref; a second Refresh with the same drift skips
// the reporter entirely.
func TestService_Refresh_FilesDriftBacklogOnce(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-btn": {ID: "cmp-btn", LibraryID: "rcl:Button", Version: "2.0.0", LatestVersion: "2.0.0"},
		},
		body: map[string]string{"cmp-btn": "BODY-V20"},
	}
	files := &fakeFiles{bytes: map[string][]byte{
		// Adopted bytes equal the snapshot but not the library — behind.
		"swarm-manager::adopted.tsx": []byte("BODY-V10"),
	}}
	clk := mocks.NewFakeClock(time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC))

	repo.Seed(adoptions.Adoption{
		ID: "row-1", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
		Scenario: "swarm-manager", AdoptedPath: "adopted.tsx",
		AdoptedVersion: "1.0.0", AdoptedSnapshotSHA256: sha256hex("BODY-V10"),
		CreatedAt: clk.Now(),
	})

	reporter := &fakeReporter{ref: "fix/rcl-button-drift-swarm-manager"}
	svc := adoptions.NewService(repo, lib, files, clk)
	adoptions.SetDriftReporter(svc, reporter, nil)

	// First refresh: status goes to behind, reporter fires.
	rows, summary, err := svc.Refresh(context.Background(), "")
	require.NoError(t, err)
	require.Equal(t, 1, summary.LibraryBehind)
	require.Equal(t, 1, reporter.calls())
	require.Equal(t, "fix/rcl-button-drift-swarm-manager", rows[0].DriftBacklogRef)
	// Payload sanity: scenario + ids carried through.
	ev := reporter.events[0]
	require.Equal(t, "row-1", ev.AdoptionID)
	require.Equal(t, "swarm-manager", ev.Scenario)
	require.Equal(t, "rcl:Button", ev.LibraryID)
	require.Equal(t, "2.0.0", ev.LibraryVersion)
	require.Equal(t, adoptions.LibraryVersionStatusBehind, ev.LibraryVersionStatus)
	require.Equal(t, adoptions.LocalStatusClean, ev.LocalStatus)

	// Second refresh: still behind, but ref already stored → no fire.
	_, _, err = svc.Refresh(context.Background(), "")
	require.NoError(t, err)
	require.Equal(t, 1, reporter.calls(), "second refresh must not refile")
}

// TestService_Refresh_ClearsRefOnReturnToCurrent verifies that going
// back to current wipes the stored ref so a future drift files a new
// backlog item rather than being silently skipped.
func TestService_Refresh_ClearsRefOnReturnToCurrent(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-btn": {ID: "cmp-btn", LibraryID: "rcl:Button", Version: "1.0.0", LatestVersion: "1.0.0"},
		},
		body: map[string]string{"cmp-btn": "BODY-CURRENT"},
	}
	files := &fakeFiles{bytes: map[string][]byte{
		"swarm-manager::adopted.tsx": []byte("BODY-CURRENT"),
	}}
	clk := mocks.NewFakeClock(time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC))

	// Seed with a pre-existing drift ref (simulates a prior drift cycle).
	repo.Seed(adoptions.Adoption{
		ID: "row-1", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
		Scenario: "swarm-manager", AdoptedPath: "adopted.tsx",
		AdoptedVersion: "1.0.0", AdoptedSnapshotSHA256: sha256hex("BODY-CURRENT"),
		CreatedAt: clk.Now(), DriftBacklogRef: "fix/old-drift",
	})

	reporter := &fakeReporter{ref: "should-not-be-called"}
	svc := adoptions.NewService(repo, lib, files, clk)
	adoptions.SetDriftReporter(svc, reporter, nil)

	rows, summary, err := svc.Refresh(context.Background(), "")
	require.NoError(t, err)
	require.Equal(t, 1, summary.LibraryCurrent)
	require.Equal(t, 0, reporter.calls())
	require.Empty(t, rows[0].DriftBacklogRef)
}

// TestService_Refresh_ReporterErrorDoesNotFailRefresh ensures a CLI
// invocation failure is logged but does not propagate — the rest of
// the refresh batch still completes and the row stays without a ref so
// a future refresh can retry.
func TestService_Refresh_ReporterErrorDoesNotFailRefresh(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-btn": {ID: "cmp-btn", LibraryID: "rcl:Button", Version: "2.0.0", LatestVersion: "2.0.0"},
		},
		body: map[string]string{"cmp-btn": "BODY-V20"},
	}
	files := &fakeFiles{bytes: map[string][]byte{
		"swarm-manager::adopted.tsx": []byte("BODY-V10"),
	}}
	clk := mocks.NewFakeClock(time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC))

	repo.Seed(adoptions.Adoption{
		ID: "row-1", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
		Scenario: "swarm-manager", AdoptedPath: "adopted.tsx",
		AdoptedVersion:        "1.0.0",
		AdoptedSnapshotSHA256: sha256hex("BODY-V10"),
		CreatedAt:             clk.Now(),
	})

	reporter := &fakeReporter{err: errors.New("swarm-manager not available")}
	svc := adoptions.NewService(repo, lib, files, clk)
	adoptions.SetDriftReporter(svc, reporter, nil)

	rows, summary, err := svc.Refresh(context.Background(), "")
	require.NoError(t, err)
	require.Equal(t, 1, summary.LibraryBehind)
	require.Equal(t, 1, reporter.calls())
	require.Empty(t, rows[0].DriftBacklogRef, "ref must stay empty so a future refresh can retry")
}

// --- SwarmManagerCLIReporter unit tests ---

type fakeRunner struct {
	binary string
	args   []string
	out    []byte
	err    error
}

func (f *fakeRunner) Run(_ context.Context, binary string, args []string) ([]byte, error) {
	f.binary = binary
	f.args = args
	return f.out, f.err
}

func TestSwarmManagerCLIReporter_ShapesArgsAndParsesResponse(t *testing.T) {
	runner := &fakeRunner{
		out: []byte(`{"item":{"kind":"fix","name":"rcl-button-drift-swarm-manager"}}`),
	}
	r := adoptions.NewSwarmManagerCLIReporter(runner)
	r.BinaryPath = "swarm-manager"

	rep, err := r.Report(context.Background(), adoptions.DriftEvent{
		AdoptionID: "row-1", ComponentID: "cmp", LibraryID: "rcl:Button",
		Scenario: "swarm-manager", AdoptedPath: "ui/Button.tsx",
		AdoptedVersion: "1.0.0", LibraryVersion: "2.0.0",
		LibraryVersionStatus: adoptions.LibraryVersionStatusBehind,
		LocalStatus:          adoptions.LocalStatusClean,
		StatusDetail:         "library at 2.0.0",
	})
	require.NoError(t, err)
	require.Equal(t, "fix/rcl-button-drift-swarm-manager", rep.Ref)
	require.Equal(t, "swarm-manager", runner.binary)
	require.Equal(t, []string{"backlog", "create", "--json", "--data"}, runner.args[:4])
	// Payload contains the expected fields.
	payload := runner.args[4]
	require.Contains(t, payload, `"kind":"fix"`)
	require.Contains(t, payload, `"name":"rcl-button-drift-swarm-manager"`)
	require.Contains(t, payload, `react-component-library`)
	require.Contains(t, payload, `drift`)
}

func TestSwarmManagerCLIReporter_PropagatesCLIError(t *testing.T) {
	runner := &fakeRunner{out: []byte("connection refused"), err: errors.New("exit 1")}
	r := adoptions.NewSwarmManagerCLIReporter(runner)

	_, err := r.Report(context.Background(), adoptions.DriftEvent{
		AdoptionID: "x", ComponentID: "c", Scenario: "s",
	})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "exit 1"))
}

func TestSwarmManagerCLIReporter_RejectsResponseWithoutItem(t *testing.T) {
	runner := &fakeRunner{out: []byte(`{}`)}
	r := adoptions.NewSwarmManagerCLIReporter(runner)

	_, err := r.Report(context.Background(), adoptions.DriftEvent{ComponentID: "c", Scenario: "s"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "item.kind")
}
