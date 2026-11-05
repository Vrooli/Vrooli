package hostapp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/operatorstate"
	hostruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/securestore"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/workloadowner"
	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"
)

const (
	hostBytesPerMegabyte = 1024
	hostWorkloadLimit    = 2700
)

// Service owns host observations and delegates host mutations to the
// control-plane registries. It has no command-line parsing or transport state.
type Service struct{}

// WorkloadOptions selects the host workload posture.
type WorkloadOptions struct {
	Root            string
	Posture         workloadowner.Posture
	PostureExplicit bool
}

// InstallOptions controls one host-tool observation or installation.
type InstallOptions struct {
	DryRun   bool
	SudoMode string
}

// SafeguardOptions controls one declared host safeguard.
type SafeguardOptions struct {
	DryRun            bool
	MaintenanceWindow bool
	SudoMode          string
}

// StorageCandidates is the typed read model for host storage discovery.
type StorageCandidates struct {
	Candidates      []hostinventory.StorageCandidate `json:"candidates"`
	ProtectedRoots  []string                         `json:"protected_roots"`
	RepositoryRoots []string                         `json:"repository_roots"`
}

// DiscoverStorageCandidates finds metadata-safe storage candidates. It never
// selects, formats, repairs, or adopts a destination.
func (Service) DiscoverStorageCandidates(ctx context.Context, allowUnknownPhysicalDevice bool) (StorageCandidates, error) {
	if ctx == nil {
		return StorageCandidates{}, fmt.Errorf("host storage context is nil")
	}
	home, err := config.VrooliHome()
	if err != nil {
		return StorageCandidates{}, fmt.Errorf("resolve Vrooli runtime home: %w", err)
	}
	protected := []string{home}
	if store, storeErr := securestore.DescribeStore(); storeErr == nil && store.Path != "" {
		protected = append(protected, filepath.Dir(store.Path))
	}
	repositoryRoots := []string{}
	registry := kopiaregistry.New(kopiaregistry.RegistryPath())
	if entries, loadErr := registry.Load(); loadErr == nil {
		for _, entry := range entries {
			if entry.Backend == kopiaregistry.BackendFilesystem && strings.TrimSpace(entry.Path) != "" {
				repositoryRoots = append(repositoryRoots, entry.Path)
			}
		}
	}
	candidates, err := hostinventory.DiscoverStorageCandidates(hostinventory.StoragePolicy{
		ProtectedRoots: protected, RepositoryRoots: repositoryRoots,
		RequirePhysicalSeparation: !allowUnknownPhysicalDevice,
	})
	if err != nil {
		return StorageCandidates{}, err
	}
	return StorageCandidates{Candidates: candidates, ProtectedRoots: protected, RepositoryRoots: repositoryRoots}, nil
}

// Install ensures one declared host tool and returns its typed result.
func (Service) Install(ctx context.Context, tool string, opts InstallOptions) (hostreqkit.ItemStatus, error) {
	if ctx == nil {
		return hostreqkit.ItemStatus{}, fmt.Errorf("host install context is nil")
	}
	return hostruntime.EnsureTool(tool, hostruntime.EnsureOptions{
		AutoInstall: true, IncludeOptional: true, DryRun: opts.DryRun, SudoMode: opts.SudoMode,
	})
}

// Safeguard ensures one declared safeguard and returns the domain-specific
// observation for read-only report names. The caller chooses the wire format.
func (Service) Safeguard(ctx context.Context, name string, opts SafeguardOptions) (any, error) {
	if ctx == nil {
		return nil, fmt.Errorf("host safeguard context is nil")
	}
	name = strings.TrimSpace(name)
	switch {
	case strings.EqualFold(name, "list"):
		return listSafeguards()
	case strings.EqualFold(name, "portability-backlog"), strings.EqualFold(name, "portability_backlog"):
		return hostruntime.SafeguardPortabilityBacklog()
	case strings.EqualFold(name, "invariant-coverage"), strings.EqualFold(name, "invariant_coverage"):
		return hostruntime.SafeguardInvariantCoverage()
	case strings.EqualFold(name, "audit"), strings.EqualFold(name, "host-state-audit"):
		return hostruntime.SafeguardHostStateAudit()
	default:
		status, err := hostruntime.EnsureSafeguard(strings.ReplaceAll(strings.ToLower(name), "-", "_"), hostruntime.EnsureOptions{
			AutoInstall: true, DryRun: opts.DryRun, MaintenanceWindow: opts.MaintenanceWindow, SudoMode: opts.SudoMode,
		})
		return status, err
	}
}

// SafeguardListEntry is the stable read model for declared safeguard status.
type SafeguardListEntry struct {
	Name           string   `json:"name"`
	Capability     string   `json:"capability"`
	CapabilityRole string   `json:"capability_role"`
	Platforms      []string `json:"platforms"`
	ObservedState  string   `json:"observed_state"`
	SupportClass   string   `json:"support_class"`
	ObservedAt     string   `json:"observed_at"`
	ObservedNotes  []string `json:"observed_notes,omitempty"`
}

func listSafeguards() ([]SafeguardListEntry, error) {
	observed, err := hostruntime.ListObservedSafeguardsAt(".", nil)
	if err != nil {
		return nil, err
	}
	entries := make([]SafeguardListEntry, 0, len(observed))
	for _, item := range observed {
		entries = append(entries, SafeguardListEntry{
			Name: item.Name, Capability: item.Capability, CapabilityRole: item.CapabilityRole,
			Platforms: append([]string(nil), item.Platforms...), ObservedState: string(item.ExecutionState),
			SupportClass: string(item.SupportClass), ObservedAt: item.ObservedAt.Format(time.RFC3339Nano),
			ObservedNotes: append([]string(nil), item.Notes...),
		})
	}
	return entries, nil
}

// CollectWorkloads observes and classifies host workloads using the caller's
// context. It does not render or mutate command output.
func (Service) CollectWorkloads(ctx context.Context, opts WorkloadOptions) (workloadowner.LiveReport, error) {
	if ctx == nil {
		return workloadowner.LiveReport{}, fmt.Errorf("host workload context is nil")
	}
	posture := opts.Posture
	if !opts.PostureExplicit {
		state, err := operatorstate.New(operatorstate.Config{RepoRoot: opts.Root}).Load(ctx)
		if err != nil {
			return workloadowner.LiveReport{}, fmt.Errorf("load host workload posture: %w", err)
		}
		if state.HostWorkloadPosture == string(workloadowner.WholeHost) {
			posture = workloadowner.WholeHost
		}
	}
	census, err := hostinventory.CollectWorkloads(ctx)
	if err != nil {
		return workloadowner.LiveReport{}, err
	}
	observed := workloadowner.LiveReport{
		Unread:       census.Unread,
		EvidenceNote: "classification is computed from hostinventory observations and enabled resource manifests",
	}
	observed.Observed = append(observed.Observed, census.Containers...)
	observed.Observed = append(observed.Observed, census.ServiceUnits...)
	observed.Observed = append(observed.Observed, census.ScheduledTasks...)
	declarations, declarationErr := workloadowner.DeclarationsFromRoot(opts.Root)
	if declarationErr != nil {
		observed.Unread = append(observed.Unread, "declarations: "+declarationErr.Error())
	}
	observed.Report = workloadowner.Classify(observed.Observed, declarations, posture, hostWorkloadLimit)
	workloadowner.RedactForPosture(&observed.Report)
	return observed, nil
}

// AuditCron observes repository-owned user-cron declarations.
func (Service) AuditCron(ctx context.Context, root string) (maintenance.CronAudit, error) {
	if ctx == nil {
		return maintenance.CronAudit{}, fmt.Errorf("host cron context is nil")
	}
	return maintenance.AuditUserCron(ctx, root, shell.OSRunner{})
}

// CollectInventory observes the complete host inventory snapshot.
func (Service) CollectInventory(ctx context.Context) (hostinventory.Snapshot, error) {
	if ctx == nil {
		return hostinventory.Snapshot{}, fmt.Errorf("host inventory context is nil")
	}
	return hostinventory.Collect(ctx)
}

// InventoryField returns the stable shell-friendly projection used by the
// host inventory command. Formatting remains a caller concern; this method
// only selects an observed value.
func (Service) InventoryField(snapshot hostinventory.Snapshot, field string) (string, error) {
	switch field {
	case "has_nvidia_gpu":
		return fmt.Sprintf("%t", snapshot.HasNvidiaGPU()), nil
	case "has_docker_nvidia_runtime":
		return fmt.Sprintf("%t", snapshot.HasDockerNvidiaRuntime()), nil
	case "has_docker_addressable_nvidia_gpu":
		return fmt.Sprintf("%t", snapshot.HasDockerAddressableNvidiaGPU()), nil
	case "gpu_count":
		return fmt.Sprintf("%d", len(snapshot.GPUs)), nil
	case "first_gpu_summary":
		for _, gpu := range snapshot.GPUs {
			if gpu.Name != "" {
				return fmt.Sprintf("%s,%d,%d", gpu.Name, gpu.VRAMUsedBytes/hostBytesPerMegabyte, gpu.VRAMBytes/hostBytesPerMegabyte), nil
			}
		}
		return "", nil
	case "cpu_cores":
		return fmt.Sprintf("%d", snapshot.CPU.Cores), nil
	case "memory_total_mb":
		return fmt.Sprintf("%d", snapshot.Memory.TotalBytes/hostBytesPerMegabyte), nil
	case "memory_available_mb":
		return fmt.Sprintf("%d", snapshot.Memory.AvailableBytes/hostBytesPerMegabyte), nil
	default:
		return "", fmt.Errorf("unknown host inventory field %q", field)
	}
}
