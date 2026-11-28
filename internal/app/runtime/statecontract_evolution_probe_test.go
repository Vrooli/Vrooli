package runtimeapp

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/statecontract"
)

// goPackageDirs maps import paths to source directories for the dependency
// closure of the given package patterns.
func goPackageDirs(t *testing.T, root string, patterns ...string) map[string]string {
	t.Helper()
	dirs := map[string]string{}
	// Include supported native targets as well as the host, because go list's
	// default closure omits imports behind other platform build constraints.
	targets := []struct{ os, arch string }{{"", ""}, {"linux", "amd64"}, {"linux", "arm64"}, {"darwin", "amd64"}, {"darwin", "arm64"}, {"windows", "amd64"}, {"windows", "arm64"}}
	for _, target := range targets {
		cmd := exec.Command("go", append([]string{"list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}\t{{.Dir}}{{end}}"}, patterns...)...)
		cmd.Dir = root
		if target.os != "" {
			cmd.Env = append(os.Environ(), "GOOS="+target.os, "GOARCH="+target.arch, "CGO_ENABLED=0")
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go list %s/%s: %v: %s", target.os, target.arch, err, out)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if importPath, dir, ok := strings.Cut(line, "\t"); ok && strings.HasPrefix(importPath, "github.com/vrooli/") {
				dirs[importPath] = dir
			}
		}
	}
	return dirs
}

// versionedEnvelopeExemptions are versioned documents that are not durable
// control-plane state the native supervisor writes. Keys are type keys or
// prefixes ending in "/" or ".".
var versionedEnvelopeExemptions = map[string]string{
	"github.com/vrooli/vrooli/internal/setup.SetupStatusReport":        "setup status JSON is a live command response, never written as cross-activation state",
	"github.com/vrooli/vrooli/internal/hostinventory.KernelInfo":       "integrity observation; version is the observed kernel version, not a stored envelope",
	"github.com/vrooli/vrooli/internal/hostinventory.RuntimeToolInfo":  "integrity observation; version is the observed tool version, not a stored envelope",
	"github.com/vrooli/vrooli/internal/hostinventory.PackageInfo":      "integrity observation; version is the observed package version, not a stored envelope",
	"github.com/vrooli/vrooli/internal/hostinventory.PackageCandidate": "integrity observation; version is the observed package version, not a stored envelope",

	"github.com/vrooli/vrooli/internal/setup/assets.Plan":                               "source-owned setup plan input, read only; persisted image manifest and receipt are declared",
	"github.com/vrooli/vrooli/internal/tools/blender-renderer.PythonCacheRepairReceipt": "write-only quarantine audit receipt; no runtime reader consumes it",
	"github.com/vrooli/vrooli/internal/rootmetrics.rootAttemptRecord":                   "versioned forensic observations; no activation, authorization or recovery code reads them",
	"github.com/vrooli/vrooli/internal/hostreqkit.ToolManifest":                         "source-owned tool.json declaration loaded by host requirements; never written as runtime state (durable OpenRGB permission records are resource-owned)",
	"github.com/vrooli/vrooli/internal/hostreqkit.SafeguardManifest":                    "source-owned safeguard.json declaration loaded by host requirements; never written as runtime state",
	"github.com/vrooli/vrooli/internal/hostreqkit.ItemStatus":                           "in-memory inspection/application response, whose version is observed tool version rather than a persistence envelope; never read back as control-plane state",

	"github.com/vrooli/vrooli/internal/setpoint.Document":                       "source-owned setpoint document read from the repository, never written",
	"github.com/vrooli/vrooli/packages/resource-deployment.ReleaseManifest":     "release-owned metadata read from a published release directory",
	"github.com/vrooli/vrooli/packages/resource-deployment.ReleaseSignature":    "signed release metadata verified in memory",
	"github.com/vrooli/vrooli/internal/resources.ArtifactVersionDecision":       "artifact retention plan rows, printed by the retention command and never stored",
	"github.com/vrooli/vrooli/internal/app/runtime.rebaselineReceipt":           "rebaseline audit evidence; never read back by any binary",
	"github.com/vrooli/vrooli/internal/app/runtime.rebaselinePause":             "read only by the rebaseline command of the same checkout, never on the activation path",
	"github.com/vrooli/vrooli/internal/privilegebroker.smartctlResponse":        "parsed smartctl output (its json_format_version), never stored",
	"github.com/vrooli/vrooli/internal/operatorcapability.VerificationRequest":  "verification request sent to a provider adapter, not stored by the control plane",
	"github.com/vrooli/vrooli/internal/resources.ArtifactRetentionPlan":         "artifact retention plan printed by the retention command and never stored",
	"github.com/vrooli/vrooli/internal/resources.ManagedServiceUpdateAdmission": "update admission options supplied by the caller; the stored update records its target separately",
	"github.com/vrooli/vrooli/internal/scenario.ServiceManifest":                "service.json declaration read from a scenario directory and never written by the control plane; only its CLI contract types are stored, inside resource state",
	"github.com/vrooli/vrooli/internal/scenario.NodeRuntime":                    "Node version selector nested in the source-owned service.json declaration; not persisted by the control plane",
	"github.com/vrooli/vrooli/internal/scenario.ServiceMetadata":                "service.json declaration read from a scenario directory and never written by the control plane; only its CLI contract types are stored, inside resource state",
	"github.com/vrooli/vrooli/internal/scenario.Lifecycle":                      "service.json declaration read from a scenario directory and never written by the control plane; only its CLI contract types are stored, inside resource state",
	"github.com/vrooli/vrooli/internal/scenario.Dependency":                     "service.json declaration read from a scenario directory and never written by the control plane; only its CLI contract types are stored, inside resource state",
	"github.com/vrooli/vrooli/internal/scenario.AuthenticationProfile":          "service.json declaration read from a scenario directory and never written by the control plane; only its CLI contract types are stored, inside resource state",
	"github.com/vrooli/vrooli/internal/scenario.TierFeasibility":                "service.json declaration read from a scenario directory and never written by the control plane; only its CLI contract types are stored, inside resource state",
	"github.com/vrooli/vrooli/internal/scenario.GenerationTemplate":             "service.json declaration read from a scenario directory and never written by the control plane; only its CLI contract types are stored, inside resource state",
	"github.com/vrooli/vrooli/internal/scenario.GenerationDesign":               "service.json declaration read from a scenario directory and never written by the control plane; only its CLI contract types are stored, inside resource state",
	"github.com/vrooli/vrooli/packages/proto/gen/go/":                           "generated wire and CLI-output messages, not stored by the control plane",
	"github.com/vrooli/vrooli/packages/proto/genmanifest.":                      "proto generation manifest, written only by the proto generation owner",
	"github.com/vrooli/vrooli/packages/proto/protogen.":                         "proto generation receipts and leases, written only by the proto generation owner",
}

func exemptEnvelope(key string) bool {
	for prefix := range versionedEnvelopeExemptions {
		if key == prefix || (strings.HasSuffix(prefix, "/") || strings.HasSuffix(prefix, ".")) && strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// A struct with a schema_version or format_version JSON field is, by this
// codebase's convention, a stored envelope. Every one in the root CLI and
// watchdog dependency closures must be declared, covered by a whole-source
// owner, or exempted with a reason. Inside owner packages any *version* key counts.
func TestPersistenceOwnersDeclareEveryVersionedEnvelope(t *testing.T) {
	root := withRepositoryCodecStamp(t)
	manifest, err := NativePersistentStateManifest()
	if err != nil {
		t.Fatal(err)
	}
	declared := map[string]bool{}
	for _, owner := range manifest.Owners {
		for key := range owner.Types {
			declared[key] = true
		}
	}
	ownerPackages := map[string]bool{}
	for _, owner := range nativePersistenceOwners() {
		for _, r := range owner.Roots {
			ownerPackages[r.PkgPath()] = true
		}
	}
	wholeDirs := map[string]bool{}
	for _, dirs := range statecontract.WholeSources {
		for _, dir := range dirs {
			if !strings.Contains(dir, "*") {
				wholeDirs[filepath.Join(root, filepath.FromSlash(dir))] = true
			}
		}
	}
	for importPath, dir := range goPackageDirs(t, root, "./cmd/vrooli", "./cmd/vrooli-watchdog") {
		if wholeDirs[dir] {
			continue
		}
		for _, name := range versionedStructs(t, dir, ownerPackages[importPath]) {
			key := importPath + "." + name
			if !declared[key] && !exemptEnvelope(key) {
				t.Errorf("%s is a versioned envelope on the native path that no persistence owner declares (declare it as a root or exempt it with a reason)", key)
			}
		}
	}
}

// unauditedWriters are packages on the native path that call a durable
// writer but sit outside the persistence inventory. Neither the original
// whole-source contract nor this one covered them; they are listed so that a
// new writer entering the native closure fails this test instead of passing
// silently. Moving one into the inventory removes it here.
var unauditedWriters = map[string]string{
	"github.com/vrooli/vrooli/internal/cli/runtimehandlers":           "root_updates.go copies the rootdecision locator into retained recovery state; prepared-root-bundle covers the copy and paths, root-rebaseline-hub-decision covers the record codec",
	"github.com/vrooli/vrooli/internal/cli/scenariocli":               "wait-attempts cache records recent wait timestamps solely for a diagnostic warning; unreadable entries are ignored",
	"github.com/vrooli/vrooli/internal/clifreshness":                  "writes the published command-surface digest, its temporary file and a build lock inside the repository build directory; the binary and runtime state are owned elsewhere",
	"github.com/vrooli/vrooli/internal/packagegov":                    "detachFile copies and replaces existing governed package files without changing their content or format",
	"github.com/vrooli/api-core/trustposture":                         "breakglass.go is a whole-source owner; posture.go has no persistent writer",
	"github.com/vrooli/vrooli/internal/tools/blender-renderer":        "writes quarantine audit receipts; no runtime reader consumes them",
	"github.com/vrooli/vrooli/internal/cli/vroolicli":                 "host_watchdog.go is covered by the host-watchdog-state whole-source owner",
	"github.com/vrooli/vrooli/internal/shell/shelltest":               "test fixture script writer, no production runtime state",
	"github.com/vrooli/vrooli/internal/setupprivilege":                "bounded temporary apt-key checks, package staging and onboarding sudoers files; durable broker, package and onboarding state are owned by those owners",
	"github.com/vrooli/platform-go/rlimitexec":                        "syscall.Exec replaces the current process; it writes no durable state",
	"github.com/vrooli/vrooli/internal/cli/metrics":                   "append-only CLI usage metrics and README, no control-plane reader consumes them",
	"github.com/vrooli/vrooli/internal/volumeremediation":             "append-only remediation receipts for audit, not reloaded to execute remediation",
	"github.com/vrooli/vrooli/internal/cli/capacityhandlers":          "exports caller-requested reports to an exclusive output file; capacity owner holds the ledger",
	"github.com/vrooli/vrooli/internal/app/credentials":               "exports caller-requested credential recovery data; its format belongs to credential-recovery owner",
	"github.com/vrooli/vrooli/internal/safeguards/coding-agent-shims": "installs generated executable shims; installation is converged on each application",
	"github.com/vrooli/vrooli/internal/safeguards/path-hygiene":       "writes host shell configuration, which is separately interpreted by the shell",
	"github.com/vrooli/vrooli/internal/deployability":                 "updates source-owned capability manifests and generated source declarations",

	"github.com/vrooli/vrooli/internal/rootmetrics":               "best-effort, versioned forensic observations only; no activation, authorization or recovery reader consumes them",
	"github.com/vrooli/vrooli/internal/runtimeprofile":            "Capture publishes caller-requested binary pprof forensic output using a temporary file and exclusive hard link; no control-plane reader reloads it across activation and Result is returned rather than persisted",
	"github.com/vrooli/vrooli/packages/capacity/companion":        "Exec invokes capacity and lifecycle commands; the capacity owner persists the ledger, and this package writes no local files",
	"github.com/vrooli/vrooli/packages/capacity/activityedge":     "Exec invokes resource start; its sockets are process-lifetime IPC and resource lifecycle state belongs to the resources owner",
	"github.com/vrooli/vrooli/internal/resources/activitynetwork": "owner-specific RootlessKit helper executables are regenerated before each launch; child PID state belongs to the live external process, not a cross-activation serialized record",

	"github.com/vrooli/api-core/database":                             "shared database helpers; callers own their schemas",
	"github.com/vrooli/api-core/retention":                            "scenario retention helpers; not written by the root supervisor's own stores",
	"github.com/vrooli/api-core/scopecatalog":                         "scope catalog cache; unaudited",
	"github.com/vrooli/api-core/storage":                              "write helpers; callers own the stored formats",
	"github.com/vrooli/cli-core/cliapp":                               "CLI framework files; unaudited",
	"github.com/vrooli/platform-go":                                   "native operation, service and containment identities are a whole-source owner; other platform files write host configuration",
	"github.com/vrooli/vrooli/internal/app/runtime":                   "rebaseline audit receipt (never read back) and pause marker; a rebaseline of another version that cannot read the marker treats the pause as not in effect and leaves enrollment disabled for the operator, never re-enabling on a guess",
	"github.com/vrooli/vrooli/cmd/vrooli-watchdog":                    "pressure report and probe caches written by the watchdog executable itself",
	"github.com/vrooli/vrooli/internal/safeguards/runtime-supervisor": "native supervisor unit definition (host configuration converged by its owner)",
	"github.com/vrooli/vrooli/internal/buildinfo":                     "build outputs and rebuild sidecars; not read across an activation",
	"github.com/vrooli/vrooli/internal/buildsnapshot":                 "temporary compiler-input scratch for one build; it reads the checkout, owns no durable state, and deletes the copy after publishing",
	"github.com/vrooli/vrooli/internal/config":                        "write helpers; callers own the stored formats",
	"github.com/vrooli/vrooli/internal/privilegebroker":               "requests, local admissions and the audit log are the protocol of the separately installed broker daemon, which root activation neither swaps nor rolls back; its installer writes the unit and binary as host configuration",
	"github.com/vrooli/vrooli/internal/artifactledger":                "append-only removal receipts that name their schema; no binary reads them back (operator forensic evidence)",
	"github.com/vrooli/vrooli/internal/shell":                         "opens the null device for child stdio; writes nothing durable",
	"github.com/vrooli/vrooli/internal/hostreqkit":                    "temporary scripts in the OS temp directory for one privileged request; not durable state",
	"github.com/vrooli/api-core/operatorsession":                      "operator-session enrollment shared with every scenario API binary, which update independently of root activation; its format is already a cross-version contract with them",
	"github.com/vrooli/api-core/filerouting":                          "test-mode throwaway roots for scenario file storage; not control-plane state",
	"github.com/vrooli/vrooli/internal/templatevalidation":            "marker.go is covered by the template-validation-markers owner",
	"github.com/vrooli/vrooli/packages/proto/genmanifest":             "proto generation manifest, written only by the proto generation owner",
	"github.com/vrooli/vrooli/packages/proto/protogen":                "proto generation receipts and leases, written only by the proto generation owner",
}

// durableWriter matches the calls that persist state: the repository's write
// helpers, whole-file writes, files created or opened for writing (append
// included), and SQL statements.
var durableWriter = regexp.MustCompile(`WriteOwnedFile|WriteFileAtomic|os\.WriteFile\(|os\.Create(Temp)?\(|os\.OpenFile\([^\n]*O_(WRONLY|RDWR|APPEND|CREATE)|\.ExecContext\(|\.Exec\(`)

// Every package on the native path that writes durable state is either a
// declared persistence owner or a listed, reasoned unaudited writer.
func TestNativePathWritersAreOwnedOrListed(t *testing.T) {
	root := withRepositoryCodecStamp(t)
	owned := map[string]bool{}
	for _, owner := range nativePersistenceOwners() {
		for _, r := range owner.Roots {
			owned[r.PkgPath()] = true
		}
	}
	// Inventory directories are owned whole; a file-pattern owner
	// (native-operation-identities) owns only some files of its package, so
	// that package stays listed.
	ownedDirs := map[string]bool{}
	for _, inventory := range []map[string][]string{statecontract.PreciseSources, statecontract.ConstantSources, statecontract.WholeSources} {
		for _, dirs := range inventory {
			for _, dir := range dirs {
				if !strings.HasSuffix(dir, "*") {
					ownedDirs[filepath.Join(root, filepath.FromSlash(dir))] = true
				}
			}
		}
	}
	for importPath, dir := range goPackageDirs(t, root, "./cmd/vrooli", "./cmd/vrooli-watchdog") {
		if ownedDirs[dir] && unauditedWriters[importPath] != "" {
			t.Errorf("%s is a persistence owner but is still listed as an unaudited writer", importPath)
		}
		if owned[importPath] || ownedDirs[dir] {
			continue
		}
		if unauditedWriters[importPath] != "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if durableWriter.Match(data) {
				t.Errorf("%s writes durable state (%s) on the native path but is neither a persistence owner nor a listed unaudited writer", importPath, entry.Name())
				break
			}
		}
	}
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

// versionedStructs names struct types carrying a format-version JSON key:
// schema_version, format_version or *_schema_version / *_format_version
// anywhere, and any *version* key inside an owner package.
func versionedStructs(t *testing.T, dir string, ownerPackage bool) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			spec, ok := node.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := spec.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range st.Fields.List {
				if field.Tag == nil {
					continue
				}
				tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
				key, _, _ := strings.Cut(tag.Get("json"), ",")
				format := key == "schema_version" || key == "format_version" || strings.HasSuffix(key, "_schema_version") || strings.HasSuffix(key, "_format_version")
				if format || ownerPackage && strings.Contains(key, "version") {
					names = append(names, spec.Name.Name)
					break
				}
			}
			return true
		})
	}
	return names
}

// A real stored-format change must be refused with the owner and the exact
// fact or field named. These fixtures change one declared fact and one
// compiled stored field of the actual inventory.
func TestStoredFormatChangesAreRefusedWithAPreciseReason(t *testing.T) {
	withRepositoryCodecStamp(t)
	prior, err := NativePersistentStateManifest()
	if err != nil {
		t.Fatal(err)
	}
	owners := nativePersistenceOwners()
	for i := range owners {
		if owners[i].Name == "runtime-registry" {
			facts := map[string]any{}
			for name, value := range owners[i].Facts {
				facts[name] = value
			}
			facts["SchemaVersion"] = scenarioruntime.SchemaVersion + 1
			owners[i].Facts = facts
		}
	}
	bumped, err := statecontract.Compute(owners)
	if err != nil {
		t.Fatal(err)
	}
	if bumped.Contract == prior.Contract {
		t.Fatal("a schema_version bump kept the persistent-state contract")
	}
	if got := statecontract.Diff(prior, bumped); len(got) != 1 || got[0] != "owner runtime-registry: format fact SchemaVersion changed" {
		t.Fatalf("schema bump explanation = %q", got)
	}

	candidate := prior
	candidate.Owners = append([]statecontract.OwnerManifest(nil), prior.Owners...)
	key := statecontract.TypeKey(reflect.TypeFor[resources.ManagedServiceState]())
	for i, owner := range candidate.Owners {
		shape, ok := owner.Types[key]
		if !ok {
			continue
		}
		types := map[string]statecontract.TypeShape{}
		for name, value := range owner.Types {
			types[name] = value
		}
		shape.Fields = append(append([]string(nil), shape.Fields...), "staged_artifact|staged_artifact,omitempty|string|false")
		types[key] = shape
		owner.Types = types
		owner.Digest = "candidate-with-added-field"
		candidate.Owners[i] = owner
	}
	got := strings.Join(statecontract.Diff(prior, candidate), "\n")
	want := `owner resources: stored type internal/resources.ManagedServiceState field staged_artifact added (json "staged_artifact,omitempty", type string)`
	if got != want {
		t.Fatalf("stored field explanation:\n%s\nwant:\n%s", got, want)
	}
}

// A stored vocabulary value (a state, status, kind or operation that an older
// binary reads by value) written as a string literal would change without the
// contract noticing. The build stamp refuses it inside owner packages; this
// test also holds callers anywhere in the repository to it, and surfaces both
// during development instead of as a refused activation.
func TestStoredVocabularyIsNeverWrittenAsALiteral(t *testing.T) {
	root := withRepositoryCodecStamp(t)
	var ownerDirs []string
	for _, inventory := range []map[string][]string{statecontract.PreciseSources, statecontract.ConstantSources} {
		for _, dirs := range inventory {
			ownerDirs = append(ownerDirs, dirs...)
		}
	}
	candidates, err := buildinfo.StoredFieldLiteralCandidates(root, ownerDirs, "internal", "cmd", "packages/resource-deployment", "packages/binaryfetch", "packages/platform-go")
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for _, candidate := range candidates {
		if len(statecontract.LiteralViolations(nativePersistenceOwners(), []string{candidate})) > 0 {
			found = append(found, candidate)
		}
	}
	if len(found) > 0 {
		t.Fatalf("stored vocabulary written as literals (use a named constant of the owner):\n%s", strings.Join(found, "\n"))
	}
}
