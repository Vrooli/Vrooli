package reconcile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/coreset"
	repocontract "github.com/vrooli/repo-contract-go"
)

type CommandRunner func(context.Context, string, ...string) ([]byte, error)

type Provider interface {
	Expected(context.Context) ([]string, error)
}

type CoreSetProvider struct{ run CommandRunner }

func NewCoreSetProvider() Provider { return &CoreSetProvider{run: execRunner} }

func NewCoreSetProviderWithRunner(run CommandRunner) Provider { return &CoreSetProvider{run: run} }

func (p *CoreSetProvider) Expected(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := p.run(ctx, "vrooli", "supervision-set", "--json")
	if err != nil {
		return nil, fmt.Errorf("load supervision set: %w", err)
	}
	var response coreset.Report
	if err := json.Unmarshal(out, &response); err != nil {
		return nil, fmt.Errorf("decode supervision set: %w", err)
	}
	if response.Source != "computed" || len(response.Members) == 0 {
		return nil, fmt.Errorf("supervision set is unavailable or not computed")
	}
	scenarios := make([]string, 0, len(response.Members))
	for _, member := range response.Members {
		if member.Kind == coreset.MemberKindScenario {
			scenarios = append(scenarios, member.Name)
		}
	}
	if len(scenarios) == 0 {
		return nil, fmt.Errorf("supervision set contains no scenarios")
	}
	return normalize(scenarios), nil
}

// Diff separates the three distinct reconcile answers. GhostChecks and
// OutOfScopeChecks were previously one field, which meant a check for a live
// scenario that simply sits outside the core-set closure was reported as a
// ghost — and ghost readings are excluded from every aggregate. That silently
// dropped real plant from uptime accounting.
// InstalledProvider answers "does this target still exist". It is deliberately
// separate from Provider: existence and should-be-supervised are different
// questions with different failure modes, and one being unavailable must not
// be answered with the other.
type InstalledProvider interface {
	Installed(context.Context) ([]string, error)
}

// FilesystemInstalledProvider reads the installed scenario set from the repo
// working tree. It is the most direct available answer to "does the target
// exist" and, unlike a CLI or API probe, cannot be wedged by a stopped
// control plane — which matters because that is exactly when reconcile runs.
type FilesystemInstalledProvider struct {
	// Root is the repository root. When empty it is resolved on first use.
	Root string
}

func NewFilesystemInstalledProvider() InstalledProvider { return &FilesystemInstalledProvider{} }

func (p *FilesystemInstalledProvider) Installed(_ context.Context) ([]string, error) {
	root := p.Root
	if root == "" {
		resolved, err := repocontract.FindRepoRootFromEnvOrCWD()
		if err != nil {
			return nil, fmt.Errorf("resolve repo root: %w", err)
		}
		root = resolved
	}
	entries, err := os.ReadDir(filepath.Join(root, "scenarios"))
	if err != nil {
		return nil, fmt.Errorf("read installed scenarios: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("installed scenario set is empty")
	}
	return normalize(names), nil
}

type Diff struct {
	// GhostChecks target something that no longer exists.
	GhostChecks []string
	// OutOfScopeChecks target something that exists but is not in the derived
	// should-be-supervised set. Their readings stay in every aggregate.
	OutOfScopeChecks []string
	// UnsupervisedPlant are expected members with no registered check.
	UnsupervisedPlant []string
	// GhostDetectionAvailable is false when the installed-target set could not
	// be read. Ghost and out-of-scope are then both empty, because absence
	// from an unreadable set proves nothing.
	GhostDetectionAvailable bool
	GhostUnavailableReason  string
}

// Compare classifies registered scenario checks against two independent sets:
// `installed` answers "does the target still exist" and `expected` answers
// "should the target be supervised". They are different questions and a check
// may fail either one without failing the other.
func Compare(registered, installed, expected []string) Diff {
	registeredSet := registeredScenarioNames(registered)
	expectedSet := plainNameSet(expected)
	diff := Diff{GhostDetectionAvailable: installed != nil}
	if installed == nil {
		diff.GhostUnavailableReason = "installed scenario set is unavailable; no check can be classified as a ghost"
	}
	installedSet := plainNameSet(installed)
	for id := range registeredSet {
		if id == "" {
			continue
		}
		if diff.GhostDetectionAvailable {
			if _, exists := installedSet[id]; !exists {
				diff.GhostChecks = append(diff.GhostChecks, "scenario-"+id)
				continue
			}
			if _, wanted := expectedSet[id]; !wanted {
				diff.OutOfScopeChecks = append(diff.OutOfScopeChecks, "scenario-"+id)
			}
		}
	}
	for name := range expectedSet {
		if _, ok := registeredSet[name]; !ok {
			diff.UnsupervisedPlant = append(diff.UnsupervisedPlant, name)
		}
	}
	sort.Strings(diff.GhostChecks)
	sort.Strings(diff.OutOfScopeChecks)
	sort.Strings(diff.UnsupervisedPlant)
	return diff
}

// registeredScenarioNames keeps only check ids that target a scenario. Host,
// infra, resource and system checks have no scenario target, so they are
// never candidates for either classification. The `scenario-` prefix is the
// only reliable marker: a scenario may itself be named `vrooli-events`, so
// prefix-matching the bare name would drop real targets.
func registeredScenarioNames(ids []string) map[string]struct{} {
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if !strings.HasPrefix(id, "scenario-") {
			continue
		}
		if name := strings.TrimPrefix(id, "scenario-"); name != "" {
			set[name] = struct{}{}
		}
	}
	return set
}

// plainNameSet normalizes a list of bare scenario names. It deliberately does
// NOT strip a `scenario-` prefix: several scenarios are genuinely named
// `scenario-authenticator`, `scenario-stack-governor` and so on, and stripping
// turned those names into ones that match nothing — which reported every one
// of them as both a ghost check and unsupervised plant at the same time.
func plainNameSet(names []string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name = strings.TrimSpace(name); name != "" {
			set[name] = struct{}{}
		}
	}
	return set
}

func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(name); err != nil {
		return nil, err
	}
	return exec.CommandContext(ctx, name, args...).Output()
}

func normalize(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
