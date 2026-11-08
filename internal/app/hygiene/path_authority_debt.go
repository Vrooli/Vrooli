package hygiene

// pathAuthorityDebt is the ratchet for hand-built runtime-home paths that
// predate the repository-wide scan: file -> the number of bypass lines it may
// still hold. A file not listed here may hold none, and a listed file may not
// gain any. Fixing a site means lowering its count (or deleting the entry);
// the repository test fails while a count is higher than the file needs, so
// the ledger can only shrink. Do not add entries: resolve new paths through
// api-core/storage or the repo contract's runtime_home entries instead.
//
// Grouped by owner, with why each site is not a mechanical fix.
var pathAuthorityDebt = map[string]int{
	// Control plane. The proto generation lock (~/.vrooli/locks) has no
	// runtime_home entry and is shared with packages/proto; the native
	// messaging host directory and the go-work temp root have no entry either;
	// storageactivity runs for another uid, possibly before any contract is
	// reachable.
	"internal/credentialextension/extension.go":  1,
	"internal/lifecycle/shared_package_lock.go":  1,
	"internal/storageactivity/activity_linux.go": 1,

	// Shared packages: bootstrap or dependency-light code. repo-contract-go's
	// source-root pointer is how a checkout is found before the contract can
	// be loaded; the others would need a contract or storage dependency, or a
	// runtime_home entry that does not exist (identity/break-glass, harness
	// state written by resources/claude-code).
	"packages/api-core/trustposture/breakglass.go":            1,
	"packages/capabilityprobe/probe.go":                       1,
	"packages/cli-core/agentcatalog/harness/contextwindow.go": 1,
	"packages/cliresolve/resolve.go":                          1,
	"packages/proto/protogen/artifact_store.go":               1,
	"packages/repo-contract-go/source_root.go":                1,

	// Resources. statusline.go restates the api-core/storage class-root
	// default because Claude Code runs it from any directory with no
	// discoverable contract; samples_path_test.go proves it agrees with the
	// storage resolver.
	"resources/android-sdk/cli/internal/androidsdk/implementation.go": 3,
	"resources/claude-code/cli/internal/statusline/statusline.go":     1,

	// Scenarios, each owned by its scenario.
	"scenarios/cli-health/api/internal/cliruntime/cliruntime.go":              1,
	"scenarios/command-center/cli/domains/domains.go":                         1,
	"scenarios/deployment-manager/api/server/releasefacts.go":                 1,
	"scenarios/deployment-manager/cli/domains/evidence/register.go":           1,
	"scenarios/performance-health/api/main.go":                                1,
	"scenarios/scenario-to-android/api/internal/builds/builder.go":            2,
	"scenarios/scenario-to-cloud/api/authz/policy.go":                         1,
	"scenarios/scenario-to-cloud/api/vps/actions.go":                          1,
	"scenarios/scenario-to-desktop/api/storagemigrate/migrate.go":             1,
	"scenarios/scenario-to-desktop/cli/domains/evidence/commands.go":          1,
	"scenarios/storage-manager/api/handlers/cleanup/module.go":                3,
	"scenarios/storage-manager/api/hostpaths/roots.go":                        1,
	"scenarios/storage-manager/api/internal/providers/lease_unix.go":          1,
	"scenarios/storage-manager/cli/domains/cleanup/chaos.go":                  1,
	"scenarios/storage-manager/cli/domains/cleanup/handlers.go":               1,
	"scenarios/swarm-manager/api/internal/backlog/handler.go":                 1,
	"scenarios/swarm-manager/api/internal/pathredact/redactor.go":             1,
	"scenarios/system-monitor/api/internal/server/runtime.go":                 1,
	"scenarios/test-genie/api/internal/execution/evidence_producer.go":        1,
	"scenarios/test-genie/api/internal/orchestrator/targetruntime/manager.go": 1,
	"scenarios/test-genie/api/internal/scenarios/testing_runner.go":           1,
	"scenarios/unit-health/api/internal/executor/executor.go":                 1,
	"scenarios/vrooli-bridge/agent/internal/channel/companion.go":             1,
	"scenarios/vrooli-bridge/agent/internal/exec/command.go":                  1,
	"scenarios/vrooli-bridge/agent/internal/health/host.go":                   1,
	"scenarios/vrooli-bridge/agent/internal/privsep/cleanup.go":               3,
	"scenarios/vrooli-bridge/agent/internal/privsep/principal.go":             1,
	"scenarios/vrooli-memory/api/internal/harness/adapters.go":                1,
}
