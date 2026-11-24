package maintenance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/agentscope"
	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/supervision"
)

// ExecutorScopeRequest is a read-only exclusion request. Labels must come from
// the durable run, never from an empty live-run list. It does not authorize
// recovery: the caller must hold its admission/continuation fence across this
// observation and dispatch. All fields are non-secret process identities.
type ExecutorScopeRequest struct {
	Executors []ExecutorScopeRef `json:"executors"`
	// Transcripts optionally names exact harness transcript paths the caller
	// already records. Each present executor reports the subset its processes
	// hold open (ExecutorScopeEvidence.OpenTranscripts). Older owners ignore
	// the field, which yields no claim rather than a negative.
	Transcripts []string `json:"transcripts,omitempty"`
}

type ExecutorScopeRef struct {
	RunID            string                            `json:"runId"`
	HarnessSessionID string                            `json:"harnessSessionId,omitempty"`
	HarnessKind      string                            `json:"harnessKind,omitempty"`
	BindingProof     *cliutil.AgentSessionBindingProof `json:"bindingProof,omitempty"`
	// HomePaths opts into exact process-environment matching for harness homes.
	// In-scope unreadable identities make a negative home verdict unknown.
	HomePaths []string   `json:"homePaths,omitempty"`
	Tag       string     `json:"tag"`
	LegacyTag string     `json:"legacyTag"`
	PID       int        `json:"pid"`
	PGID      int        `json:"pgid"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
}

func validExecutorScopeBindingProof(ref ExecutorScopeRef) bool {
	proof := ref.BindingProof
	if proof == nil || ref.PID != 0 || ref.PGID != 0 ||
		proof.ProofVersion != cliutil.AgentSessionBindingProofVersion ||
		proof.RunID != ref.RunID || proof.HarnessSessionID != ref.HarnessSessionID ||
		proof.LeaseSessionID == "" || proof.WebConsoleSessionID == "" ||
		proof.HarnessKind != ref.HarnessKind || proof.OwnerKind != agentsession.KindWebConsoleSession {
		return false
	}
	binding := proof.Binding
	if binding.PID <= 1 || binding.UID < 0 || len(binding.Cgroup) > 1024 ||
		!strings.HasPrefix(binding.Cgroup, "0::/user.slice/user-") ||
		!strings.Contains(binding.Cgroup, "/vrooli.slice/vrooli-agents.slice/") ||
		strings.ContainsAny(binding.Cgroup, "\r\n\t ") {
		return false
	}
	start, err := strconv.ParseUint(binding.Start, 10, 64)
	if err != nil || start == 0 {
		return false
	}
	boot, err := uuid.Parse(binding.Boot)
	return err == nil && boot != uuid.Nil
}

func executorScopeReferencePID(ref ExecutorScopeRef) int {
	if ref.PID > 1 {
		return ref.PID
	}
	if proof := ref.BindingProof; proof != nil {
		return proof.Binding.PID
	}
	return ref.PID
}

func executorScopeLeaseSessionID(ref ExecutorScopeRef) string {
	if proof := ref.BindingProof; proof != nil && proof.LeaseSessionID != "" {
		return proof.LeaseSessionID
	}
	return ref.HarnessSessionID
}

type ExecutorScopeEvidence struct {
	RunID   string   `json:"runId"`
	State   string   `json:"state"`
	PIDs    []int    `json:"pids"`
	Reasons []string `json:"reasons"`
	// OpenTranscripts is positive evidence only: requested transcript paths
	// held open by this present executor's processes or their descendants.
	OpenTranscripts []string `json:"openTranscripts,omitempty"`
}

type ExecutorScopeReport struct {
	IdentityVersion string                  `json:"identityVersion"`
	SchemaVersion   string                  `json:"schemaVersion"`
	Complete        bool                    `json:"complete"`
	Executors       []ExecutorScopeEvidence `json:"executors"`
}

// Only these labels survive reading a process environment. In particular no
// identity token, credential, command line or arbitrary environment is retained.
type scopeIdentity struct {
	RunID, Scenario, Variant, InstanceID string
	Tags                                 []string
	HomePaths                            []string
	TagLabels                            map[string]string
}

var (
	scopeProcessTable        = listProcessTableContext
	scopeReadIdentity        = readScopeIdentity
	scopeReadProcessBinding  = cliutil.ObserveProcessBinding
	scopeReadProcessFact     = func(pid int) (agentsession.ProcessFact, error) { return agentsession.NewHostFacts().Process(pid) }
	scopeProcessBirths       = (supervision.NativeProcessTableSource{}).ProcessesContext
	scopeEditorLeaseBindings = listScopeEditorLeaseBindings
	scopeEditorScopeMembers  = func(unit string) ([]int, error) {
		return agentsession.NewHostFacts().ScopeMembers(unit)
	}
	scopeSessionCgroupPath                                                                                               = agentsession.ScopeCgroupPath
	scopeEditorScopeOccupancy                                                                                            = platformgo.ScopeOccupancy
	scopeEditorScopeProcesses                                                                                            = platformgo.ScopeProcessesContext
	scopeEditorScopeProcessesBatch                                                                                       = platformgo.ScopeProcessesBatchContext
	scopeEditorLeaseProvenance      func(context.Context, string, string) (scenarioruntime.EditorLeaseProvenance, error) = getScopeEditorLeaseProvenance
	scopeEditorLeaseProvenanceBatch                                                                                      = getScopeEditorLeaseProvenances
	scopeEditorScopeGone                                                                                                 = func(ctx context.Context, unit string) (bool, error) {
		cgroupPath, err := scopeSessionCgroupPath(unit)
		if err != nil {
			return false, err
		}
		ref := agentscope.Ref(cgroupPath)
		ref.Name = strings.TrimSuffix(unit, agentscope.ScopeSuffix)
		_, occupancyErr := scopeEditorScopeOccupancy(ref)
		if occupancyErr == nil {
			return false, nil
		}
		if !errors.Is(occupancyErr, os.ErrNotExist) {
			return false, occupancyErr
		}
		// A missing cgroup is only a positive gone result when the platform
		// owner also proves the exact unit is not-found/inactive. This second
		// bounded probe rejects empty-but-active and inconsistent scope states.
		members, err := scopeEditorScopeProcesses(ctx, ref)
		if err != nil {
			return false, err
		}
		if len(members) != 0 {
			return false, nil
		}
		// The unit probe may observe an empty readable cgroup after the first
		// ENOENT. Recheck the exact cgroup so a reappeared scope is not absence.
		_, finalOccupancyErr := scopeEditorScopeOccupancy(ref)
		if finalOccupancyErr == nil {
			return false, nil
		}
		if !errors.Is(finalOccupancyErr, os.ErrNotExist) {
			return false, finalOccupancyErr
		}
		return true, nil
	}
)
var scopeCurrentBootID = hostsession.CurrentBootID

// Kept separate from the exported wire-contract timeout so focused tests can
// exercise budget exhaustion without waiting for a full observation window.
var executorScopeObservationTimeout = ExecutorScopeTimeout

const (
	// Retained terminal identities are inspected with the same single host
	// observation as live runs. Requests are bounded by bytes; reference work is
	// paged so a large history does not require a larger fixed count ceiling.
	ExecutorScopeBatchSize = 8192
	ExecutorScopeMaxBytes  = 32 << 20
	// A complete scope proof can include tens of thousands of retained runs and
	// exact attached-session checks. Keep it bounded, but allow normal large
	// inventories enough time to finish and return per-run evidence.
	ExecutorScopeTimeout             = 20 * time.Second
	ExecutorScopeMaxProcesses        = 262144
	ExecutorScopeIdentityWorkers     = 16
	executorScopeProcessTablePasses  = 3
	ExecutorScopeLeaseCheckBatchSize = 128

	// Older v1 readers silently ignore unknown request fields. The consumer
	// requires this version before trusting immutable attached-key evidence.
	ExecutorScopeIdentityVersion = "attached-key-v1"
	executorScopeSampleBudget    = 65536
)

// Index only requested identities. Samples are bounded independently of host
// size and shared tags: matching is O(references + processes + labels), plus a
// fixed total evidence-sample budget, not references times host processes.
type executorScopeIndex struct {
	runs        map[string][]int
	tags        map[string][]int
	homes       map[string][]int
	serviceTags map[string][]int
	groups      map[int]int
}

func normalizeExecutorHomePath(value string) (string, error) {
	if value == "" || len(value) > 4096 || strings.ContainsAny(value, "\x00\r\n") || !path.IsAbs(value) {
		return "", fmt.Errorf("executor home path must be a bounded absolute path")
	}
	return path.Clean(value), nil
}

func normalizeExecutorHomePaths(paths []string) ([]string, error) {
	if len(paths) > 8 {
		return nil, fmt.Errorf("executor home path limit exceeded")
	}
	seen := make(map[string]struct{}, len(paths))
	normalized := make([]string, 0, len(paths))
	for _, candidate := range paths {
		value, err := normalizeExecutorHomePath(candidate)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	slices.Sort(normalized)
	return normalized, nil
}

// executorScopeHomePathsFromEnvironment retains only the two harness-home
// values relevant to exact native-home inventory. Arbitrary environment
// entries are consumed by the caller but never stored in scopeIdentity.
func executorScopeHomePathsFromEnvironment(env map[string]string) []string {
	paths := make([]string, 0, 2)
	for _, key := range []string{"CODEX_HOME", "GROK_HOME"} {
		value := env[key]
		if value == "" {
			continue
		}
		clean, err := normalizeExecutorHomePath(value)
		if err != nil || slices.Contains(paths, clean) {
			continue
		}
		paths = append(paths, clean)
	}
	return paths
}

func indexExecutorScopes(ctx context.Context, refs []ExecutorScopeRef, entries map[int]processTableEntry, identities map[int]scopeIdentity, inherited map[int]map[string]bool, sampleLimit int) (executorScopeIndex, error) {
	index := executorScopeIndex{runs: make(map[string][]int, len(refs)), tags: make(map[string][]int, 2*len(refs)), homes: make(map[string][]int), serviceTags: make(map[string][]int, 2*len(refs)), groups: make(map[int]int)}
	for i := range refs {
		normalized, err := normalizeExecutorHomePaths(refs[i].HomePaths)
		if err != nil {
			return index, err
		}
		refs[i].HomePaths = normalized
		ref := refs[i]
		index.runs[ref.RunID] = nil
		index.tags[ref.Tag] = nil
		index.tags[ref.LegacyTag] = nil
		index.serviceTags[ref.Tag] = nil
		index.serviceTags[ref.LegacyTag] = nil
		for _, home := range ref.HomePaths {
			index.homes[home] = nil
		}
		if ref.PGID > 0 {
			index.groups[ref.PGID] = 0
		}
	}
	for pid, entry := range entries {
		if err := ctx.Err(); err != nil {
			return index, err
		}
		if strings.HasPrefix(entry.State, "Z") {
			continue
		}
		identity := identities[pid]
		for _, home := range identity.HomePaths {
			if pids, wanted := index.homes[home]; wanted && len(pids) < sampleLimit {
				index.homes[home] = append(pids, pid)
			}
		}
		if pids, wanted := index.runs[identity.RunID]; wanted && len(pids) < sampleLimit {
			index.runs[identity.RunID] = append(pids, pid)
		}
		for _, tag := range identity.Tags {
			if err := ctx.Err(); err != nil {
				return index, err
			}
			// Classify before sampling: many inherited service labels cannot
			// crowd a true executor out of the bounded positive evidence.
			tags := index.tags
			if inherited[pid][tag] {
				tags = index.serviceTags
			}
			if pids, wanted := tags[tag]; wanted && len(pids) < sampleLimit {
				tags[tag] = append(pids, pid)
			}
		}
		if member, wanted := index.groups[entry.PGID]; wanted && member == 0 {
			index.groups[entry.PGID] = pid
		}
	}
	return index, nil
}
