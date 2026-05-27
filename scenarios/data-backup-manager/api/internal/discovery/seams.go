package discovery

import "context"

// The discovery service composes everything it needs from these consumer-owned
// seams. Production wires sysmounts + composition-root adapters; unit tests wire
// the fakes in mocks/. Keeping the interfaces here (the consumer) means the
// scanners, catalogs, and dismissal store stay decoupled from this domain.

// VolumeScanner enumerates mounted volumes.
//
// seam: VolumeScanner. Production wires *sysmounts.Scanner; tests wire
// mocks.FakeVolumeScanner.
type VolumeScanner interface {
	Scan(ctx context.Context) ([]Volume, error)
}

// TargetSourceScanner enumerates well-known sources worth protecting.
//
// seam: TargetSourceScanner. Production wires *discovery.WellKnownScanner;
// tests wire mocks.FakeTargetSourceScanner.
type TargetSourceScanner interface {
	Scan(ctx context.Context) ([]TargetCandidate, error)
}

// ResourceEnumerator lists the platform's resources so the ResourceDataScanner
// can read each one's declared durable host state. It is the single place the
// `vrooli` CLI is shelled (wrap-not-use): DBM never hardcodes the repo
// `resources/` layout, it trusts the CLI's reported manifest_path.
//
// seam: ResourceEnumerator. Production wires *discovery.CLIResourceEnumerator
// (shells `vrooli resource list --json`); tests wire mocks.FakeResourceEnumerator.
type ResourceEnumerator interface {
	Enumerate(ctx context.Context) ([]ResourceRef, error)
}

// TargetCatalog reads the live target catalog so already-registered sources are
// filtered out of suggestions.
//
// seam: TargetCatalog. Production wires the composition-root adapter over
// targets.Service; tests wire mocks.FakeTargetCatalog.
type TargetCatalog interface {
	ListAll(ctx context.Context) ([]ExistingTarget, error)
}

// DestinationCatalog reads the live destination catalog so already-used
// locations are filtered out of suggestions.
//
// seam: DestinationCatalog. Production wires the composition-root adapter over
// destinations.Service; tests wire mocks.FakeDestinationCatalog.
type DestinationCatalog interface {
	ListAll(ctx context.Context) ([]ExistingDestination, error)
}

// ProtectedPaths provides the set of paths a destination must not overlap. The
// destinations service's own protectedRoot (just SCENARIO_DATA_DIR) is too
// narrow for destination filtering, so discovery computes its own set (runtime
// root + existing destination locations + registered target locators) in the
// composition root — see Contract Decision D4.
//
// seam: ProtectedPaths. Production wires the composition-root adapter; tests
// wire mocks.FakeProtectedPaths.
type ProtectedPaths interface {
	ProtectedPaths(ctx context.Context) ([]string, error)
}

// DismissalStore persists which suggestions the operator has hidden.
//
// seam: DismissalStore. Production wires the sqlite store (sqlite.go); tests
// wire mocks.FakeDismissalStore.
type DismissalStore interface {
	IsDismissed(ctx context.Context, id string) (bool, error)
	Dismiss(ctx context.Context, id, kind string) error
}
