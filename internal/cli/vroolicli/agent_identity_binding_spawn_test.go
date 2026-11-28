package vroolicli

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/agentsession"
)

// A `vrooli agent launch` session waits for its agent instead of exec'ing it:
// the launcher (the lease's PID) keeps the cgroup it started in and the agent is
// its direct child inside the session scope. Every Web Console Claude launch
// takes this shape, and the proof refused all of them with
// scope_membership_unverified, so none could request authority.
const (
	spawnLauncherPID = 4086
	spawnAgentPID    = 4100
	spawnLauncherCG  = "0::/user.slice/user-1000.slice/user@1000.service/tmux-spawn-1.scope"
)

// spawnProofHost is a kernel fixture: one fact per process and the scope's members.
type spawnProofHost struct {
	members []int
	facts   map[int]agentsession.ProcessFact
	started time.Time
}

func (h *spawnProofHost) Process(pid int) (agentsession.ProcessFact, error) {
	fact, ok := h.facts[pid]
	if !ok {
		return agentsession.ProcessFact{}, errors.New("no such process")
	}
	return fact, nil
}
func (h *spawnProofHost) StartedAt(int) (time.Time, error) { return h.started, nil }
func (h *spawnProofHost) ScopeMembers(unit string) ([]int, error) {
	if unit != proofUnit {
		return nil, errors.New("unexpected scope unit")
	}
	return h.members, nil
}

func spawnProofFixture() (identityBindingDeps, *spawnProofHost) {
	deps, _, _, _ := proofFixture()
	host := &spawnProofHost{
		members: []int{spawnAgentPID, 4101, 4102},
		facts: map[int]agentsession.ProcessFact{
			spawnLauncherPID: {PID: spawnLauncherPID, PPID: 100, StartTime: "100"},
			spawnAgentPID:    {PID: spawnAgentPID, PPID: spawnLauncherPID, StartTime: "101"},
			// Descendants of the agent share its scope but are not the launcher's children.
			4101: {PID: 4101, PPID: spawnAgentPID, StartTime: "102"},
			4102: {PID: 4102, PPID: 4101, StartTime: "103"},
		},
		started: time.Date(2026, 10, 18, 10, 0, 0, 0, time.UTC),
	}
	deps.host = host
	deps.observe = func(pid int) (cliutil.ProcessBinding, error) {
		fact, ok := host.facts[pid]
		if !ok {
			return cliutil.ProcessBinding{}, errors.New("no such process")
		}
		cgroup := "0::" + proofPath
		if pid == spawnLauncherPID {
			cgroup = spawnLauncherCG
		}
		return cliutil.ProcessBinding{PID: pid, UID: 1000, Start: fact.StartTime, Boot: "boot-1", Cgroup: cgroup}, nil
	}
	return deps, host
}

func TestProveAgentIdentityBindingAcceptsSpawnAndWaitLauncher(t *testing.T) {
	deps, _ := spawnProofFixture()
	got, err := proveAgentIdentityBinding(context.Background(), deps, "", proofHarnessSessionID, proofWCID, "codex")
	if err != nil {
		t.Fatalf("a launcher that waits for its in-scope agent child could not be proved: %v", err)
	}
	// The proof binds the lease's own process: that is the PID the grant request
	// carries and the incarnation a grant is later matched against.
	if got.Binding.PID != spawnLauncherPID || got.Binding.Start != "100" || got.Binding.Cgroup != spawnLauncherCG {
		t.Fatalf("proof must bind the launcher incarnation the lease recorded, got %+v", got.Binding)
	}
	if got.WebConsoleSessionID != proofWCID || got.OwnerKind != agentsession.KindWebConsoleSession {
		t.Fatalf("proof lost the Web Console owner: %+v", got)
	}
}

func TestProveAgentIdentityBindingRefusesUnprovenSpawnAndWaitScope(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*identityBindingDeps, *spawnProofHost)
		want  string
	}{
		{name: "no scope member is the launcher's child", want: "scope_membership_unverified", setup: func(_ *identityBindingDeps, h *spawnProofHost) {
			h.members = []int{4101, 4102}
		}},
		{name: "empty scope", want: "scope_membership_unverified", setup: func(_ *identityBindingDeps, h *spawnProofHost) { h.members = nil }},
		{name: "two children in scope is ambiguous", want: "scope_membership_unverified", setup: func(_ *identityBindingDeps, h *spawnProofHost) {
			h.members = append(h.members, 4103)
			h.facts[4103] = agentsession.ProcessFact{PID: 4103, PPID: spawnLauncherPID, StartTime: "104"}
		}},
		{name: "child reparented away from the launcher", want: "scope_membership_unverified", setup: func(_ *identityBindingDeps, h *spawnProofHost) {
			h.facts[spawnAgentPID] = agentsession.ProcessFact{PID: spawnAgentPID, PPID: 1, StartTime: "101"}
		}},
		{name: "child outside the scope cgroup", want: "kernel_binding_unverified", setup: func(d *identityBindingDeps, h *spawnProofHost) {
			inner := d.observe
			d.observe = func(pid int) (cliutil.ProcessBinding, error) {
				binding, err := inner(pid)
				if pid == spawnAgentPID {
					binding.Cgroup = "0::/user.slice/other.scope"
				}
				return binding, err
			}
		}},
		{name: "launcher from another boot", want: "kernel_binding_unverified", setup: func(d *identityBindingDeps, h *spawnProofHost) {
			inner := d.observe
			d.observe = func(pid int) (cliutil.ProcessBinding, error) {
				binding, err := inner(pid)
				if pid == spawnLauncherPID {
					binding.Boot = "boot-2"
				}
				return binding, err
			}
		}},
		{name: "launcher process facts disagree with its binding", want: "process_incarnation_unverified", setup: func(d *identityBindingDeps, h *spawnProofHost) {
			inner := d.observe
			d.observe = func(pid int) (cliutil.ProcessBinding, error) {
				binding, err := inner(pid)
				if pid == spawnLauncherPID {
					binding.Start = "a-reused-pid"
				}
				return binding, err
			}
		}},
		{name: "launcher started after the lease", want: "process_started_after_lease", setup: func(_ *identityBindingDeps, h *spawnProofHost) {
			h.started = time.Date(2026, 10, 18, 11, 30, 0, 0, time.UTC)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps, host := spawnProofFixture()
			tc.setup(&deps, host)
			_, err := proveAgentIdentityBinding(context.Background(), deps, "", proofHarnessSessionID, proofWCID, "codex")
			if err == nil || err.Error() != tc.want {
				t.Fatalf("error = %v, want %s", err, tc.want)
			}
		})
	}
}

// The scope root is part of what both reads must agree on, not only the
// launcher the proof binds.
func TestProveAgentIdentityBindingRefusesSpawnRootSwappedWithMatchingFacts(t *testing.T) {
	deps, host := spawnProofFixture()
	inner := deps.observe
	childReads := 0
	deps.observe = func(pid int) (cliutil.ProcessBinding, error) {
		binding, err := inner(pid)
		if pid == spawnAgentPID {
			childReads++
			if childReads == 2 {
				binding.Start = "replaced"
				host.facts[spawnAgentPID] = agentsession.ProcessFact{PID: spawnAgentPID, PPID: spawnLauncherPID, StartTime: "replaced"}
			}
		}
		return binding, err
	}
	if _, err := proveAgentIdentityBinding(context.Background(), deps, "", proofHarnessSessionID, proofWCID, "codex"); err == nil || !strings.Contains(err.Error(), "process_changed") {
		t.Fatalf("a scope root swapped between reads with consistent facts was accepted: %v", err)
	}
}
