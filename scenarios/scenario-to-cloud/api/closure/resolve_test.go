package closure

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-cloud/domain"
)

const (
	fixtureShaStore = "8c18753fd1965bb4eaf099ad88412be883b25396b6a5e3b97ad610ddfa3855f2"
	fixtureShaCache = "e495022187d70554a26e83311146b847ee3477e972a97ae4224658e52ab299ef"
	mib             = uint64(1 << 20)
)

func fixtureCatalog(t *testing.T, repo string) *FileCatalog {
	t.Helper()
	root := filepath.Join("testdata", repo)
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("fixture %s: %v", repo, err)
	}
	return NewLayoutCatalog(filepath.Join(root, "scenarios"), filepath.Join(root, "resources"))
}

func fixtureInputs(t *testing.T, repo, scenario string) Inputs {
	t.Helper()
	return Inputs{
		ScenarioID:       scenario,
		Environment:      "production",
		Platform:         Platform{OS: "linux", Arch: "amd64"},
		Catalog:          fixtureCatalog(t, repo),
		HostRequirements: DeclaredHostRequirements{},
	}
}

func component(t *testing.T, closure domain.Closure, kind domain.ClosureComponentKind, id string) domain.ClosureComponent {
	t.Helper()
	for _, candidate := range closure.Components {
		if candidate.Kind == kind && candidate.ID == id {
			return candidate
		}
	}
	t.Fatalf("component %s/%s not in closure; have %v", kind, id, componentIDs(closure))
	return domain.ClosureComponent{}
}

func hasComponent(closure domain.Closure, kind domain.ClosureComponentKind, id string) bool {
	for _, candidate := range closure.Components {
		if candidate.Kind == kind && candidate.ID == id {
			return true
		}
	}
	return false
}

func componentIDs(closure domain.Closure) []string {
	ids := make([]string, 0, len(closure.Components))
	for _, candidate := range closure.Components {
		ids = append(ids, string(candidate.Kind)+"/"+candidate.ID)
	}
	return ids
}

func hasReason(component domain.ClosureComponent, kind domain.ClosureReasonKind, from string) bool {
	for _, reason := range component.Reasons {
		if reason.Kind == kind && reason.From == from {
			return true
		}
	}
	return false
}

func unsupportedFor(closure domain.Closure, component string) (domain.ClosureUnsupported, bool) {
	for _, entry := range closure.Unsupported {
		if entry.Component == component {
			return entry, true
		}
	}
	return domain.ClosureUnsupported{}, false
}

// TestResolveTransitiveClosureWithReasons proves every transitive component
// carries an inclusion reason, including a scenario dependency that owns its
// own resource and credential. [REQ:STC-P0-021]
func TestResolveTransitiveClosureWithReasons(t *testing.T) {
	closure, err := Resolve(context.Background(), fixtureInputs(t, "basic", "app"))
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if closure.SchemaVersion != domain.ClosureSchemaVersion {
		t.Fatalf("schema_version = %q", closure.SchemaVersion)
	}
	if len(closure.Unsupported) != 0 {
		t.Fatalf("unexpected unsupported entries: %+v", closure.Unsupported)
	}

	app := component(t, closure, domain.ClosureKindScenario, "app")
	if !app.Required || !hasReason(app, domain.ClosureReasonDeclaredBy, "selection") {
		t.Fatalf("root scenario must be required with a selection reason: %+v", app)
	}
	if app.Version != "1.2.0" || !strings.HasPrefix(app.ContentIdentity, "sha256:") {
		t.Fatalf("root identity missing: %+v", app)
	}
	if app.Recovery == nil || app.Recovery.CodeRollback != "compatible_predecessor_only" || app.Recovery.SchemaStrategy != "expand_contract" {
		t.Fatalf("recovery contract not carried: %+v", app.Recovery)
	}

	records := component(t, closure, domain.ClosureKindScenario, "records-service")
	if !records.Required || !hasReason(records, domain.ClosureReasonDeclaredBy, "app") {
		t.Fatalf("records-service must be required via app: %+v", records)
	}

	// The dependency scenario's own resource and credential are in the closure
	// with reasons that name the dependency, not the root.
	store := component(t, closure, domain.ClosureKindResource, "store")
	if !store.Required {
		t.Fatalf("store must be required (transitively via records-service): %+v", store)
	}
	if !hasReason(store, domain.ClosureReasonDeclaredBy, "records-service") || !hasReason(store, domain.ClosureReasonTransitiveVia, "records-service") {
		t.Fatalf("store reasons must name records-service: %+v", store.Reasons)
	}
	var transitive string
	for _, reason := range store.Reasons {
		if reason.Kind == domain.ClosureReasonTransitiveVia {
			transitive = reason.Detail
		}
	}
	if transitive != "path app -> records-service -> store" {
		t.Fatalf("transitive path = %q", transitive)
	}
	storeCredential := component(t, closure, domain.ClosureKindCredentialDescriptor, "fixture/store:password")
	if !hasReason(storeCredential, domain.ClosureReasonCredentialOf, "resource:store") {
		t.Fatalf("store credential reasons: %+v", storeCredential.Reasons)
	}
	if storeCredential.Credential == nil || storeCredential.Credential.LogicalID != "fixture/store" || storeCredential.Credential.Field != "password" || storeCredential.Credential.Env != "STORE_PASSWORD" || !storeCredential.Credential.Required {
		t.Fatalf("credential descriptor was normalised lossily: %+v", storeCredential.Credential)
	}
	recordsCredential := component(t, closure, domain.ClosureKindCredentialDescriptor, "fixture/records:api-token")
	if !hasReason(recordsCredential, domain.ClosureReasonCredentialOf, "scenario:records-service") {
		t.Fatalf("records credential reasons: %+v", recordsCredential.Reasons)
	}
	// The same address declared by two owners merges to one component with
	// both reasons and required ORed.
	sessionSecret := component(t, closure, domain.ClosureKindCredentialDescriptor, "fixture/app:session-secret")
	if !sessionSecret.Required || !hasReason(sessionSecret, domain.ClosureReasonCredentialOf, "scenario:app") || !hasReason(sessionSecret, domain.ClosureReasonCredentialOf, "scenario:optional-helper") {
		t.Fatalf("shared credential must merge owners: %+v", sessionSecret)
	}

	cache := component(t, closure, domain.ClosureKindResource, "cache")
	if !cache.Required || !hasReason(cache, domain.ClosureReasonDeclaredBy, "app") {
		t.Fatalf("cache: %+v", cache)
	}
	if hasReason(cache, domain.ClosureReasonTransitiveVia, "app") {
		t.Fatalf("a direct dependency of the root is not transitive: %+v", cache.Reasons)
	}

	// Optional dependencies stay optional; disabled ones are absent.
	helper := component(t, closure, domain.ClosureKindScenario, "optional-helper")
	if helper.Required || !helper.OptionalSelected || !hasReason(helper, domain.ClosureReasonSelectedBy, "app") {
		t.Fatalf("optional-helper must be optional and selected by manifest default: %+v", helper)
	}
	if hasComponent(closure, domain.ClosureKindScenario, "disabled-extra") {
		t.Fatalf("disabled optional dependency must not be in the closure")
	}
	if hasComponent(closure, domain.ClosureKindScenario, "system-core") {
		t.Fatalf("bundle scope must not pull system-required scenarios that no edge reaches")
	}

	// Host tools and safeguards carry their declaring owners; platform
	// mismatches are excluded.
	curl := component(t, closure, domain.ClosureKindTool, "curl")
	if !curl.Required || !hasReason(curl, domain.ClosureReasonSafeguardOf, "scenario:app") || !hasReason(curl, domain.ClosureReasonSafeguardOf, "resource:store") {
		t.Fatalf("curl reasons: %+v", curl.Reasons)
	}
	if hasComponent(closure, domain.ClosureKindTool, "powershell") {
		t.Fatalf("windows-only tool must not be in a linux closure")
	}
	firewall := component(t, closure, domain.ClosureKindSafeguard, "firewall")
	if !firewall.Required {
		t.Fatalf("firewall: %+v", firewall)
	}
	privileges := map[string]domain.ClosurePrivilege{}
	for _, privilege := range closure.Privileges {
		privileges[privilege.Subject] = privilege
	}
	if privileges["safeguard:firewall"].Effect != "elevated" || privileges["safeguard:firewall"].Safeguard != "firewall" {
		t.Fatalf("firewall privilege: %+v", closure.Privileges)
	}
	if privileges["resource:cache"].Effect != "user" {
		t.Fatalf("cache privilege: %+v", closure.Privileges)
	}

	// Native artifacts: the control plane plus each bundled resource server.
	vrooli := component(t, closure, domain.ClosureKindNativeArtifact, "vrooli")
	if vrooli.Artifact == nil || vrooli.Artifact.Eligibility != domain.ClosureArtifactBuildable || vrooli.Artifact.Platform != "linux-amd64" {
		t.Fatalf("control plane artifact: %+v", vrooli.Artifact)
	}
	storeServer := component(t, closure, domain.ClosureKindNativeArtifact, "store:server")
	if storeServer.Artifact == nil || storeServer.Artifact.Digest != "sha256:"+fixtureShaStore || storeServer.Artifact.Name != "store_linux_amd64" || storeServer.Artifact.Eligibility != domain.ClosureArtifactEligible {
		t.Fatalf("store server artifact: %+v", storeServer.Artifact)
	}
	if store.Artifact == nil || store.Artifact.Digest != storeServer.Artifact.Digest {
		t.Fatalf("resource artifact must match its native artifact: %+v", store.Artifact)
	}
	cacheServer := component(t, closure, domain.ClosureKindNativeArtifact, "cache:server")
	if cacheServer.Artifact == nil || cacheServer.Artifact.Digest != "sha256:"+fixtureShaCache {
		t.Fatalf("cache server artifact: %+v", cacheServer.Artifact)
	}
	for _, root := range []string{"internal", "packages"} {
		pkg := component(t, closure, domain.ClosureKindPackage, root)
		if !strings.Contains(pkg.Reasons[0].Detail, "not minimised") {
			t.Fatalf("package %s must say it is not minimised: %+v", root, pkg.Reasons)
		}
	}

	// Listeners and persistent data carry declared visibility, readiness and owners.
	listeners := map[string]domain.ClosureListener{}
	for _, listener := range closure.Listeners {
		listeners[listener.ID] = listener
	}
	if listeners["app/api"].Visibility != "public_via_edge" || listeners["app/api"].Readiness == nil || listeners["app/api"].Readiness.Path != "/health" {
		t.Fatalf("app/api listener: %+v", listeners["app/api"])
	}
	if listeners["app/ui"].Visibility != "private" || listeners["records-service/api"].Visibility != "private" || listeners["store/store"].Owner != "resource:store" {
		t.Fatalf("listeners: %+v", closure.Listeners)
	}
	data := map[string]domain.ClosurePersistentData{}
	for _, entry := range closure.PersistentData {
		data[entry.ID] = entry
	}
	if data["application-records"].Owner != "store" || data["application-records"].MigrationOwner != "scenario" || data["application-records"].BackupProvider != "data-backup-manager" || data["application-records"].DeclaredBy != "scenario:app" {
		t.Fatalf("application-records: %+v", data["application-records"])
	}
	if data["store-data"].Owner != "store" || data["store-data"].MigrationOwner != "resource" {
		t.Fatalf("store-data: %+v", data["store-data"])
	}

	// Capacity aggregates declared requirements plus transient headroom.
	if closure.Capacity.CPU != 2.75 {
		t.Fatalf("cpu = %v", closure.Capacity.CPU)
	}
	if closure.Capacity.MemoryBytes != (512+2048+256)*mib || closure.Capacity.DiskBytes != (1024+8192+512)*mib {
		t.Fatalf("capacity = %+v", closure.Capacity)
	}
	if closure.Capacity.TransientUpdateHeadroomBytes != 2*8192*mib+StagingHeadroomBytes || closure.Capacity.HeadroomBasis != "largest_declared_disk_footprint" {
		t.Fatalf("headroom = %+v", closure.Capacity)
	}
	if closure.Capacity.Fit != "unknown" {
		t.Fatalf("fit without a target must be unknown, got %q", closure.Capacity.Fit)
	}

	if closure.Sources.AnalyzerUsed || closure.Sources.HostRequirements != "declarations" {
		t.Fatalf("sources: %+v", closure.Sources)
	}
	ok, err := Verify(closure)
	if err != nil || !ok {
		t.Fatalf("digest must verify: ok=%v err=%v", ok, err)
	}
}

// TestResolveIsDeterministic proves identical inputs produce identical bytes
// and digests, the ground for API/CLI/UI parity.
func TestResolveIsDeterministic(t *testing.T) {
	first, err := Resolve(context.Background(), fixtureInputs(t, "basic", "app"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Resolve(context.Background(), fixtureInputs(t, "basic", "app"))
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Fatalf("closure bytes differ between runs")
	}
	if first.Digest != second.Digest || !strings.HasPrefix(first.Digest, "sha256:") {
		t.Fatalf("digests differ: %s vs %s", first.Digest, second.Digest)
	}
}

// TestResolveNamesMissingArchitectureArtifact proves an unsupported
// architecture dependency is named, never silently omitted. [REQ:STC-P0-022]
func TestResolveNamesMissingArchitectureArtifact(t *testing.T) {
	inputs := fixtureInputs(t, "basic", "app")
	inputs.Platform = Platform{OS: "linux", Arch: "arm64"}
	closure, err := Resolve(context.Background(), inputs)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	entry, ok := unsupportedFor(closure, "store")
	if !ok || entry.ReasonCode != ReasonMissingPlatformArtifact {
		t.Fatalf("store must be unsupported with %s: %+v", ReasonMissingPlatformArtifact, closure.Unsupported)
	}
	if !strings.Contains(entry.Detail, "store") || !strings.Contains(entry.Detail, "arm64") {
		t.Fatalf("detail must name the dependency and architecture: %q", entry.Detail)
	}
	store := component(t, closure, domain.ClosureKindResource, "store")
	if store.Artifact == nil || store.Artifact.Eligibility != domain.ClosureArtifactIneligible {
		t.Fatalf("store artifact must be ineligible: %+v", store.Artifact)
	}
	if hasComponent(closure, domain.ClosureKindNativeArtifact, "store:server") {
		t.Fatalf("no native artifact may be claimed for a missing architecture")
	}
	cache := component(t, closure, domain.ClosureKindNativeArtifact, "cache:server")
	if cache.Artifact.Eligibility != domain.ClosureArtifactEligible || cache.Artifact.Name != "cache_linux_arm64" {
		t.Fatalf("cache remains eligible on arm64: %+v", cache.Artifact)
	}
	if closure.Supported() {
		t.Fatalf("closure with unsupported entries must not report supported")
	}
}

// TestResolveUnsupportedControlPlanePlatform proves a platform the control
// plane is not built for is reported per component. [REQ:STC-P0-022]
func TestResolveUnsupportedControlPlanePlatform(t *testing.T) {
	inputs := fixtureInputs(t, "basic", "app")
	inputs.Platform = Platform{OS: "darwin", Arch: "arm64"}
	closure, err := Resolve(context.Background(), inputs)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if closure.Platform.OS != "macos" {
		t.Fatalf("darwin must canonicalise to macos: %+v", closure.Platform)
	}
	if entry, ok := unsupportedFor(closure, "vrooli"); !ok || entry.ReasonCode != ReasonUnsupportedControlPlanePlatform {
		t.Fatalf("control plane must be unsupported on macos: %+v", closure.Unsupported)
	}
	if entry, ok := unsupportedFor(closure, "store"); !ok || entry.ReasonCode != ReasonUnsupportedPlatform || !strings.Contains(entry.Detail, "no macos archive") {
		t.Fatalf("store must carry its declared unsupported reason: %+v", closure.Unsupported)
	}
	if entry, ok := unsupportedFor(closure, "app"); !ok || entry.ReasonCode != ReasonUnsupportedPlatform {
		t.Fatalf("app declares linux-only supported targets: %+v", closure.Unsupported)
	}
}

// TestResolveDetectsCycle proves a dependency cycle is a typed error naming
// the path. [REQ:STC-P0-022]
func TestResolveDetectsCycle(t *testing.T) {
	_, err := Resolve(context.Background(), fixtureInputs(t, "cycle", "alpha"))
	typed, ok := As(err)
	if !ok || typed.Code != CodeCycle {
		t.Fatalf("expected %s, got %v", CodeCycle, err)
	}
	path, _ := typed.Details["path"].([]string)
	if strings.Join(path, ">") != "alpha>beta>gamma>alpha" {
		t.Fatalf("cycle path = %v", path)
	}
}

// TestResolveOptionalCycleIsNotAnError proves a cycle of optional
// (try_start) edges is a declared degradation pattern: both scenarios are
// included, neither is required through the other, and a required edge into
// the cycle does not turn it into an error. [REQ:STC-P0-022]
func TestResolveOptionalCycleIsNotAnError(t *testing.T) {
	closure, err := Resolve(context.Background(), fixtureInputs(t, "cycle", "opt-a"))
	if err != nil {
		t.Fatalf("optional cycle must resolve: %v", err)
	}
	b := component(t, closure, domain.ClosureKindScenario, "opt-b")
	if b.Required || !b.OptionalSelected || !hasReason(b, domain.ClosureReasonDeclaredBy, "opt-a") {
		t.Fatalf("opt-b: %+v", b)
	}
	a := component(t, closure, domain.ClosureKindScenario, "opt-a")
	if !a.Required || !hasReason(a, domain.ClosureReasonDeclaredBy, "opt-b") {
		t.Fatalf("opt-a keeps its root reason and gains the back-edge reason: %+v", a)
	}
	mixed, err := Resolve(context.Background(), fixtureInputs(t, "cycle", "mixed"))
	if err != nil {
		t.Fatalf("required edge into an optional cycle must resolve: %v", err)
	}
	if !component(t, mixed, domain.ClosureKindScenario, "opt-a").Required || component(t, mixed, domain.ClosureKindScenario, "opt-b").Required {
		t.Fatalf("requiredness must stop at the optional edge: %v", componentIDs(mixed))
	}
}

// TestResolveMissingCatalogEntryIsUnavailable proves missing catalog data is
// a typed closure_unavailable naming the component, never an empty closure.
// [REQ:STC-P0-021]
func TestResolveMissingCatalogEntryIsUnavailable(t *testing.T) {
	_, err := Resolve(context.Background(), fixtureInputs(t, "missing", "app"))
	typed, ok := As(err)
	if !ok || typed.Code != CodeUnavailable {
		t.Fatalf("expected %s, got %v", CodeUnavailable, err)
	}
	if typed.Details["component"] != "absent-store" || typed.Details["declared_by"] != "app" {
		t.Fatalf("details must name the missing dependency and its declarer: %+v", typed.Details)
	}
	if _, err := Resolve(context.Background(), fixtureInputs(t, "missing", "no-such-scenario")); !Is(err, CodeUnavailable) {
		t.Fatalf("missing root scenario must be unavailable, got %v", err)
	}
}

// TestResolveRemovedCatalogEntryIsUnavailable removes an expected catalog
// entry from a copy of the fixture and asserts unavailable readiness
// (P05-V04). [REQ:STC-P0-021]
func TestResolveRemovedCatalogEntryIsUnavailable(t *testing.T) {
	root := copyFixture(t, "basic")
	if err := os.Remove(filepath.Join(root, "resources", "cache", "resource.json")); err != nil {
		t.Fatal(err)
	}
	inputs := fixtureInputs(t, "basic", "app")
	inputs.Catalog = NewLayoutCatalog(filepath.Join(root, "scenarios"), filepath.Join(root, "resources"))
	closure, err := Resolve(context.Background(), inputs)
	typed, ok := As(err)
	if !ok || typed.Code != CodeUnavailable {
		t.Fatalf("expected %s, got closure=%+v err=%v", CodeUnavailable, closure, err)
	}
	if typed.Details["component"] != "cache" {
		t.Fatalf("must name the removed entry: %+v", typed.Details)
	}
	if len(closure.Components) != 0 {
		t.Fatalf("an unavailable closure must not carry components")
	}
}

func copyFixture(t *testing.T, repo string) string {
	t.Helper()
	dest := t.TempDir()
	source := filepath.Join("testdata", repo)
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(source, path)
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dest
}

type fakeAnalyzer struct {
	results map[string]AnalyzerResult
	err     error
	calls   []string
}

func (f *fakeAnalyzer) Analyze(_ context.Context, scenarioID string) (AnalyzerResult, error) {
	f.calls = append(f.calls, scenarioID)
	if f.err != nil {
		return AnalyzerResult{}, f.err
	}
	return f.results[scenarioID], nil
}

// TestResolveAnalyzerFailureIsUnavailable proves an analyzer outage never
// degrades to a declarations-only closure. [REQ:STC-P0-021]
func TestResolveAnalyzerFailureIsUnavailable(t *testing.T) {
	inputs := fixtureInputs(t, "basic", "app")
	inputs.Analyzer = &fakeAnalyzer{err: errors.New("connection refused")}
	_, err := Resolve(context.Background(), inputs)
	typed, ok := As(err)
	if !ok || typed.Code != CodeUnavailable || typed.Details["source"] != "analyzer" {
		t.Fatalf("expected analyzer closure_unavailable, got %v", err)
	}
}

// TestResolveAnalyzerEdgesCarryDetectionReason proves analyzer-detected
// edges join the closure with a reason naming the analyzer and that every
// included scenario is analyzed. [REQ:STC-P0-021]
func TestResolveAnalyzerEdgesCarryDetectionReason(t *testing.T) {
	inputs := fixtureInputs(t, "basic", "app")
	analyzer := &fakeAnalyzer{results: map[string]AnalyzerResult{
		"app": {Tool: "scenario-dependency-analyzer", Resources: []AnalyzedDependency{{Name: "store", Required: true, Enabled: true}}},
	}}
	inputs.Analyzer = analyzer
	closure, err := Resolve(context.Background(), inputs)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	store := component(t, closure, domain.ClosureKindResource, "store")
	var detected bool
	for _, reason := range store.Reasons {
		if reason.Kind == domain.ClosureReasonDeclaredBy && reason.From == "app" && strings.Contains(reason.Detail, "detected by scenario-dependency-analyzer") {
			detected = true
		}
	}
	if !detected {
		t.Fatalf("store must carry the analyzer detection reason: %+v", store.Reasons)
	}
	if !closure.Sources.AnalyzerUsed || closure.Sources.AnalyzerTool != "scenario-dependency-analyzer" {
		t.Fatalf("sources must record the analyzer: %+v", closure.Sources)
	}
	for _, id := range []string{"app", "records-service", "optional-helper"} {
		if !containsFold(analyzer.calls, id) {
			t.Fatalf("analyzer must be consulted for %s; calls=%v", id, analyzer.calls)
		}
	}
	plain, err := Resolve(context.Background(), fixtureInputs(t, "basic", "app"))
	if err != nil {
		t.Fatal(err)
	}
	if plain.Digest == closure.Digest {
		t.Fatalf("a closure derived with the analyzer must not share a digest with one derived without it")
	}
}

// TestResolveConflictingOperatingModes proves contradictory modes are typed
// closure_conflict errors. [REQ:STC-P0-022]
func TestResolveConflictingOperatingModes(t *testing.T) {
	_, err := Resolve(context.Background(), fixtureInputs(t, "conflict", "batch"))
	typed, ok := As(err)
	if !ok || typed.Code != CodeConflict || typed.Details["component"] != "batch" {
		t.Fatalf("one_shot with auto-restart must conflict: %v", err)
	}

	inputs := fixtureInputs(t, "basic", "app")
	inputs.Overrides.OperatingMode = map[string]string{"store": "remote-vrooli"}
	_, err = Resolve(context.Background(), inputs)
	typed, ok = As(err)
	if !ok || typed.Code != CodeConflict || typed.Details["component"] != "store" || typed.Details["requested_mode"] != "remote-vrooli" {
		t.Fatalf("disallowed operating mode must conflict: %v", err)
	}

	inputs.Overrides.OperatingMode = map[string]string{"store": "attach-only"}
	if _, err := Resolve(context.Background(), inputs); err != nil {
		t.Fatalf("allowed operating mode must resolve: %v", err)
	}
}

// TestSupervisionAndAutoRestartAreIndependent proves supervision membership
// and auto-restart are carried and overridden separately. [REQ:STC-P0-023]
func TestSupervisionAndAutoRestartAreIndependent(t *testing.T) {
	base, err := Resolve(context.Background(), fixtureInputs(t, "basic", "app"))
	if err != nil {
		t.Fatal(err)
	}
	records := component(t, base, domain.ClosureKindScenario, "records-service").Supervision
	if records == nil || !records.Member || records.StartupPolicy != "must_start" || !records.AutoRestart || records.AutoRestartSource != "declared" || records.RuntimeKind != "long_running" {
		t.Fatalf("records-service supervision: %+v", records)
	}
	helper := component(t, base, domain.ClosureKindScenario, "optional-helper").Supervision
	if !helper.Member || helper.StartupPolicy != "try_start" || helper.AutoRestart {
		t.Fatalf("optional-helper supervision: %+v", helper)
	}

	// Changing supervision membership leaves auto-restart untouched.
	inputs := fixtureInputs(t, "basic", "app")
	inputs.Overrides.SupervisionMember = map[string]bool{"records-service": false}
	changed, err := Resolve(context.Background(), inputs)
	if err != nil {
		t.Fatal(err)
	}
	records = component(t, changed, domain.ClosureKindScenario, "records-service").Supervision
	if records.Member || !records.AutoRestart || records.AutoRestartSource != "declared" {
		t.Fatalf("membership change must not touch auto-restart: %+v", records)
	}

	// Changing auto-restart leaves membership untouched.
	inputs = fixtureInputs(t, "basic", "app")
	inputs.Overrides.AutoRestart = map[string]bool{"app": true, "records-service": false}
	changed, err = Resolve(context.Background(), inputs)
	if err != nil {
		t.Fatal(err)
	}
	app := component(t, changed, domain.ClosureKindScenario, "app").Supervision
	if !app.Member || !app.AutoRestart || app.AutoRestartSource != "override" {
		t.Fatalf("app auto-restart override: %+v", app)
	}
	records = component(t, changed, domain.ClosureKindScenario, "records-service").Supervision
	if !records.Member || records.AutoRestart || records.AutoRestartSource != "override" {
		t.Fatalf("records-service auto-restart override must keep membership: %+v", records)
	}
	if base.Digest == changed.Digest {
		t.Fatalf("a different auto-restart choice is a different closure")
	}
}

// TestOptionalSelectionRules proves optional choices stay optional, can be
// deselected, and required dependencies cannot be deselected. [REQ:STC-P0-021]
func TestOptionalSelectionRules(t *testing.T) {
	inputs := fixtureInputs(t, "basic", "app")
	inputs.Overrides.DeselectOptional = []string{"optional-helper", "records-service"}
	closure, err := Resolve(context.Background(), inputs)
	if err != nil {
		t.Fatal(err)
	}
	if hasComponent(closure, domain.ClosureKindScenario, "optional-helper") {
		t.Fatalf("deselected optional dependency must be absent")
	}
	if !component(t, closure, domain.ClosureKindScenario, "records-service").Required {
		t.Fatalf("a required dependency cannot be deselected")
	}
	if secret := component(t, closure, domain.ClosureKindCredentialDescriptor, "fixture/app:session-secret"); !secret.Required || len(secret.Reasons) != 1 {
		t.Fatalf("credential must remain with its remaining owner only: %+v", secret)
	}

	// Selecting a disabled optional dependency walks its declarations, so a
	// missing catalog entry beneath it is reported, not hidden.
	inputs = fixtureInputs(t, "basic", "app")
	inputs.Overrides.SelectOptional = []string{"disabled-extra"}
	_, err = Resolve(context.Background(), inputs)
	typed, ok := As(err)
	if !ok || typed.Code != CodeUnavailable || typed.Details["component"] != "missing-resource" || typed.Details["declared_by"] != "disabled-extra" {
		t.Fatalf("selected optional dependency's missing resource must be named: %v", err)
	}
}

// TestRepositoryScopeIncludesSystemRequired proves scope rules: repository
// scope adds system-required scenarios with a system_required reason; bundle
// scope does not. [REQ:STC-P0-021]
func TestRepositoryScopeIncludesSystemRequired(t *testing.T) {
	inputs := fixtureInputs(t, "basic", "app")
	inputs.Scope = ScopeRepository
	closure, err := Resolve(context.Background(), inputs)
	if err != nil {
		t.Fatal(err)
	}
	core := component(t, closure, domain.ClosureKindScenario, "system-core")
	if !core.Required || !hasReason(core, domain.ClosureReasonSystemRequired, "catalog") {
		t.Fatalf("system-core: %+v", core)
	}
	if hasComponent(closure, domain.ClosureKindPackage, "packages") {
		t.Fatalf("repository scope does not ship the bundle package set")
	}
	if closure.Sources.Scope != "repository" {
		t.Fatalf("sources.scope = %q", closure.Sources.Scope)
	}
}

// TestInsufficientCapacityIsUnsupported proves capacity shortfalls including
// transient update headroom are reported as unsupported. [REQ:STC-P0-022]
func TestInsufficientCapacityIsUnsupported(t *testing.T) {
	inputs := fixtureInputs(t, "basic", "app")
	inputs.ReleaseArtifactBytes = []uint64{100 * mib, 400 * mib}
	inputs.TargetCapacity = &TargetCapacity{CPU: 4, MemoryBytes: 8 * 1024 * mib, DiskBytes: (1024+8192+512)*mib + 800*mib + StagingHeadroomBytes - 1}
	closure, err := Resolve(context.Background(), inputs)
	if err != nil {
		t.Fatal(err)
	}
	if closure.Capacity.HeadroomBasis != "release_artifacts" || closure.Capacity.TransientUpdateHeadroomBytes != 800*mib+StagingHeadroomBytes {
		t.Fatalf("headroom from release artifacts: %+v", closure.Capacity)
	}
	entry, ok := unsupportedFor(closure, "capacity")
	if !ok || entry.ReasonCode != ReasonInsufficientCapacity || !strings.Contains(entry.Detail, "transient update headroom") || closure.Capacity.Fit != "insufficient" {
		t.Fatalf("one byte short of headroom must be insufficient: %+v %+v", entry, closure.Capacity)
	}

	inputs.TargetCapacity.DiskBytes++
	closure, err = Resolve(context.Background(), inputs)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := unsupportedFor(closure, "capacity"); ok || closure.Capacity.Fit != "fits" {
		t.Fatalf("exact headroom must fit: %+v", closure.Capacity)
	}
}

// TestResolveRejectsInvalidInputs proves the typed invalid_request path.
func TestResolveRejectsInvalidInputs(t *testing.T) {
	inputs := fixtureInputs(t, "basic", "app")
	inputs.Platform = Platform{OS: "linux"}
	if _, err := Resolve(context.Background(), inputs); !Is(err, CodeInvalidRequest) {
		t.Fatalf("missing architecture must be invalid_request, got %v", err)
	}
	inputs = fixtureInputs(t, "basic", "../app")
	if _, err := Resolve(context.Background(), inputs); !Is(err, CodeInvalidRequest) {
		t.Fatalf("path-like scenario id must be invalid_request, got %v", err)
	}
	inputs = fixtureInputs(t, "basic", "app")
	inputs.Catalog = nil
	if _, err := Resolve(context.Background(), inputs); !Is(err, CodeUnavailable) {
		t.Fatalf("no catalog must be unavailable, got %v", err)
	}
}

// A target scenario with no deployment.listeners declaration keeps the
// launch contract: its ui port is public_via_edge, everything else private.
// Dependencies never inherit that default.
func TestUndeclaredTargetScenarioExposesOnlyUI(t *testing.T) {
	decl := &ScenarioDeclaration{ID: "legacy", Ports: map[string]PortDeclaration{"ui": {}, "api": {}}}
	closure := &domain.Closure{ScenarioID: "legacy"}
	r := &resolver{}
	r.collectScenarioListeners(closure, decl)
	got := map[string]string{}
	for _, l := range closure.Listeners {
		got[l.ID] = l.Visibility
	}
	if got["legacy/ui"] != "public_via_edge" || got["legacy/api"] != "private" {
		t.Fatalf("listeners = %+v", got)
	}
	dep := &domain.Closure{ScenarioID: "other"}
	r.collectScenarioListeners(dep, decl)
	for _, l := range dep.Listeners {
		if l.Visibility != "private" {
			t.Fatalf("dependency listener %s must stay private, got %s", l.ID, l.Visibility)
		}
	}
}
