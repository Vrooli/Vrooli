package adapters

import (
	"context"
	"errors"
	"runtime"
	"time"

	"network-manager/internal/resolver"
)

type StaticRegistry struct {
	OS   string
	Arch string
	Now  func() time.Time
}

func NewStaticRegistry() StaticRegistry {
	return StaticRegistry{OS: runtime.GOOS, Arch: runtime.GOARCH}
}

var _ Registry = StaticRegistry{}

type ResolverBackendRepository interface {
	GetBackend(ctx context.Context, backend string) (resolver.BackendConfig, error)
}

type ResolverAwareRegistry struct {
	Base             Registry
	ResolverBackends ResolverBackendRepository
}

var _ Registry = ResolverAwareRegistry{}

func (r ResolverAwareRegistry) Report(ctx context.Context) (Report, error) {
	base := r.Base
	if base == nil {
		base = NewStaticRegistry()
	}
	report, err := base.Report(ctx)
	if err != nil {
		return Report{}, err
	}
	if r.ResolverBackends == nil {
		return report, nil
	}
	cfg, err := r.ResolverBackends.GetBackend(ctx, resolver.AdGuardHomeBackend)
	if errors.Is(err, resolver.ErrNotFound) {
		return report, nil
	}
	if err != nil {
		return Report{}, err
	}
	if cfg.BaseURL == "" || cfg.TokenRef == "" {
		return report, nil
	}
	for i := range report.Capabilities {
		cap := &report.Capabilities[i]
		if cap.Adapter != "adguard-home" {
			continue
		}
		switch cap.Action {
		case "resolver_status":
			cap.Supported = true
			cap.Reason = "A governed AdGuard Home resolver backend is configured by secret reference."
		case "resolver_client_inventory":
			cap.Supported = true
			cap.Reason = "A governed AdGuard Home resolver backend is configured; client evidence can be imported without query-level DNS logs."
		case "manage_dns_filtering":
			cap.Supported = true
			cap.RollbackSupported = true
			cap.Reason = "AdGuard Home resolver is configured; global filtering rules and protection changes are gated by Network Manager policy approval and rollback ledgers."
		}
	}
	return report, nil
}

func (r StaticRegistry) Report(context.Context) (Report, error) {
	osName := r.OS
	if osName == "" {
		osName = runtime.GOOS
	}
	arch := r.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}
	now := time.Now().UTC()
	if r.Now != nil {
		now = r.Now().UTC()
	}
	return Report{
		ObservedAt: now,
		Platform: PlatformSummary{
			OS:         osName,
			Arch:       arch,
			Profile:    profileForOS(osName),
			Notes:      notesForOS(osName),
			ObservedAt: now,
		},
		Capabilities: []Capability{
			{Adapter: "host-" + osName, Action: "read_network_status", Supported: true, Reason: "Host runtime can report OS and architecture for read-only diagnostics.", ObservedAt: now},
			{Adapter: "host-" + osName, Action: "privileged_packet_probe", Supported: false, RequiresAdmin: true, Reason: "Privileged packet probes require platform-specific elevated permissions and are not enabled by default.", ObservedAt: now},
			{Adapter: "adguard-home", Action: "resolver_status", Supported: false, RollbackSupported: false, Reason: "AdGuard Home backend is not configured yet.", ObservedAt: now},
			{Adapter: "adguard-home", Action: "resolver_client_inventory", Supported: false, RollbackSupported: false, Reason: "AdGuard Home client inventory requires a governed resolver backend and secret reference.", ObservedAt: now},
			{Adapter: "adguard-home", Action: "manage_dns_filtering", Supported: false, RollbackSupported: true, Reason: "Resolver adapter is planned but no governed AdGuard Home resource or secret reference is configured.", ObservedAt: now},
			{Adapter: "manual-router", Action: "router_dns_enforcement", Supported: false, RollbackSupported: false, Reason: "P0 does not perform unsupported router writes; use manual router instructions until a router adapter is selected.", ObservedAt: now},
		},
	}, nil
}

func profileForOS(osName string) string {
	switch osName {
	case "linux", "darwin", "windows":
		return "host-diagnostics"
	default:
		return "manual"
	}
}

func notesForOS(osName string) []string {
	switch osName {
	case "linux":
		return []string{"Linux host diagnostics can run read-only probes without router writes.", "Privileged packet probes remain disabled unless explicitly supported later."}
	case "darwin":
		return []string{"macOS host diagnostics can run portable read-only probes.", "Some low-level probes require privileges and are reported unsupported by default."}
	case "windows":
		return []string{"Windows host diagnostics can run portable read-only probes.", "Privilege-sensitive probes are reported unsupported until a Windows adapter implements them."}
	default:
		return []string{"Unknown platform; Network Manager limits behavior to manual-safe diagnostics."}
	}
}
