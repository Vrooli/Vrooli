// Package autofix registers structure-health's deterministic remediations
// against the shared maturity-go/autofix orchestrator. Every concrete edit is a
// Fixer here; the orchestrator owns preview/apply/idempotency. service.json edits
// are structured and format-preserving (see internal/svcedit) — they never blind
// round-trip the document, so key order and unknown fields survive.
package autofix

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"structure-health/internal/intent"
	"structure-health/internal/rules"
	"structure-health/internal/svcedit"

	autofixcore "github.com/vrooli/maturity-go/autofix"
)

// Candidate is the shared auto-fix candidate (re-exported for callers).
type Candidate = autofixcore.Candidate

// Finding/rule codes structure-health can auto-remediate. These mirror the codes
// emitted by internal/rules and listed in .vrooli/maturity.json.
const (
	RuleServiceNameMismatch  = "SERVICE_NAME_MISMATCH"
	RuleHealthCheckMissing   = "HEALTH_CHECK_MISSING"
	RuleHealthCheckMalformed = "HEALTH_CHECK_MALFORMED"
	RuleFreshnessMissing     = "FRESHNESS_CHECK_MISSING"
	RuleSurfaceDirMissing    = "SURFACE_DIR_MISSING"
	RuleRequiredFileMissing  = "REQUIRED_FILE_MISSING"
	RulePortBand             = "PORT_BAND_NONCONFORMANT"
	RuleAPIBinaryName        = "API_BINARY_NAME_NONCONFORMANT"
	RuleProductionServe      = "PRODUCTION_SERVE_NONCONFORMANT"
	RuleProjectConfigSurface = "PROJECT_CONFIG_SURFACE"
)

var registry = autofixcore.NewRegistry(
	autofixcore.Fixer{RuleID: RuleServiceNameMismatch, Preview: previewServiceName, CanFix: canFix(previewServiceName)},
	autofixcore.Fixer{RuleID: RuleHealthCheckMissing, Preview: previewHealthCheck, CanFix: canFix(previewHealthCheck)},
	autofixcore.Fixer{RuleID: RuleHealthCheckMalformed, Preview: previewMalformedHealthCheck, CanFix: canFix(previewMalformedHealthCheck)},
	autofixcore.Fixer{RuleID: RuleFreshnessMissing, Preview: previewFreshnessChecks, CanFix: canFix(previewFreshnessChecks)},
	autofixcore.Fixer{RuleID: RuleSurfaceDirMissing, Preview: previewSurfaceDirs, CanFix: canFix(previewSurfaceDirs)},
	autofixcore.Fixer{RuleID: RuleRequiredFileMissing, Preview: previewRequiredFiles, CanFix: canFix(previewRequiredFiles)},
	autofixcore.Fixer{RuleID: RulePortBand, Preview: previewPortBand, CanFix: canFix(previewPortBand)},
	autofixcore.Fixer{RuleID: RuleAPIBinaryName, Preview: previewAPIBinaryName, CanFix: canFix(previewAPIBinaryName)},
	autofixcore.Fixer{RuleID: RuleProductionServe, Preview: previewProductionServe, CanFix: canFix(previewProductionServe)},
	autofixcore.Fixer{RuleID: RuleProjectConfigSurface, Preview: previewProjectConfigSurface, CanFix: canFix(previewProjectConfigSurface)},
)

// Preview returns the proposed edits for the requested rules (all when empty).
func Preview(root string, ruleIDs []string) ([]Candidate, error) {
	return registry.Preview(root, ruleIDs)
}

// Apply writes the requested edits one at a time, re-previewing from fresh disk
// state after every write. Several structure-health rules (SERVICE_NAME_MISMATCH,
// HEALTH_CHECK_MISSING, FRESHNESS_CHECK_MISSING) edit the *same* service.json, so
// re-previewing is what makes them compose: each later candidate is recomputed
// against the already-applied earlier one instead of clobbering it with a stale
// full-file snapshot. Idempotency holds because Preview re-checks state and emits
// nothing once the tree is fixed; a (ruleID,filePath) pair is applied at most
// once per call as a termination backstop.
func Apply(root string, ruleIDs []string) ([]Candidate, error) {
	var applied []Candidate
	seen := map[string]bool{}
	for {
		candidates, err := registry.Preview(root, ruleIDs)
		if err != nil {
			return applied, err
		}
		next := -1
		for i := range candidates {
			if !seen[candidateKey(candidates[i])] {
				next = i
				break
			}
		}
		if next < 0 {
			return applied, nil
		}
		c := candidates[next]
		seen[candidateKey(c)] = true
		if err := os.MkdirAll(filepath.Dir(c.FilePath), 0o755); err != nil {
			return applied, err
		}
		if err := os.WriteFile(c.FilePath, []byte(c.After), 0o644); err != nil {
			return applied, err
		}
		c.Applied = true
		applied = append(applied, c)
	}
}

func candidateKey(c Candidate) string { return c.RuleID + "\x00" + c.FilePath }

// CanFix reports whether the named rule can currently remediate the finding.
func CanFix(root, ruleID, findingPath string) bool {
	return registry.CanFix(root, ruleID, findingPath)
}

// FixClassFor returns the fix classification for a finding code: autofix when a
// fixer is registered for it, detection_only otherwise.
func FixClassFor(code string) autofixcore.FixClass {
	switch code {
	case RuleServiceNameMismatch, RuleHealthCheckMissing, RuleHealthCheckMalformed, RuleFreshnessMissing, RuleSurfaceDirMissing, RuleRequiredFileMissing,
		RulePortBand, RuleAPIBinaryName, RuleProductionServe:
		return autofixcore.FixClassAutofix
	case RuleProjectConfigSurface:
		return autofixcore.FixClassAutofix
	default:
		return autofixcore.FixClassDetectionOnly
	}
}

// previewProjectConfigSurface declares currently observed project metadata in
// the contract allowlist. It is intentionally an explicit apply operation:
// preview is the default so an operator can inspect the exact contract change
// before accepting a new project entry.
func previewProjectConfigSurface(root string) ([]Candidate, error) {
	projectConfig := filepath.Join(root, ".vrooli", "repo-contract.json")
	raw, err := os.ReadFile(projectConfig)
	if err != nil {
		return nil, nil
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, nil
	}
	layout, ok := doc["layout"].(map[string]any)
	if !ok {
		return nil, nil
	}
	allowlist := stringSet(layout["project_config_allowlist"])
	configDir, _ := layout["project_config_dir"].(string)
	if strings.TrimSpace(configDir) == "" {
		configDir = ".vrooli"
	}
	entries, err := os.ReadDir(filepath.Join(root, configDir))
	if err != nil {
		return nil, nil
	}
	var additions []string
	for _, entry := range entries {
		if !allowlist[entry.Name()] {
			additions = append(additions, entry.Name())
		}
	}
	if len(additions) == 0 {
		return nil, nil
	}
	sort.Strings(additions)
	for _, entry := range additions {
		allowlist[entry] = true
	}
	updated := make([]string, 0, len(allowlist))
	for entry := range allowlist {
		updated = append(updated, entry)
	}
	sort.Strings(updated)
	layout["project_config_allowlist"] = updated
	after, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	after = append(after, '\n')
	return []Candidate{{
		RuleID:      RuleProjectConfigSurface,
		FilePath:    projectConfig,
		Description: fmt.Sprintf("Declare observed project config entries in layout.project_config_allowlist: %s.", strings.Join(additions, ", ")),
		Before:      string(raw),
		After:       string(after),
	}}, nil
}

func stringSet(value any) map[string]bool {
	out := map[string]bool{}
	items, _ := value.([]any)
	for _, item := range items {
		if name, ok := item.(string); ok {
			out[name] = true
		}
	}
	return out
}

// canFix adapts a preview func into a CanFix predicate (the finding is fixable
// when the preview would produce at least one candidate).
func canFix(preview func(root string) ([]Candidate, error)) func(root, findingPath string) bool {
	return func(root, _ string) bool {
		c, err := preview(root)
		return err == nil && len(c) > 0
	}
}

// --- service.json fixers --------------------------------------------------

const (
	serviceJSONRel           = ".vrooli/service.json"
	canonicalAPIHealthTarget = "http://localhost:${API_PORT}/health"
)

// loadServiceJSON reads + parses the target service.json into an editable doc,
// returning the original bytes for the candidate's Before snapshot.
func loadServiceJSON(root string) (*svcedit.Doc, []byte, error) {
	path := filepath.Join(root, filepath.FromSlash(serviceJSONRel))
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	doc, err := svcedit.Parse(raw)
	if err != nil {
		return nil, nil, err
	}
	return doc, raw, nil
}

func isCanonicalHealthTarget(target string) bool {
	target = strings.TrimSpace(target)
	if target == canonicalAPIHealthTarget {
		return true
	}
	return strings.HasSuffix(target, "${API_PORT}/health") || strings.HasSuffix(target, ":${API_PORT}/health")
}

// previewServiceName rewrites service.name to the scenario directory name.
func previewServiceName(root string) ([]Candidate, error) {
	in, err := intent.Load(root)
	if err != nil {
		return nil, nil
	}
	dir := scenarioDir(root)
	current := strings.TrimSpace(in.Name)
	if dir == "" || current == "" || current == dir {
		return nil, nil
	}
	doc, before, err := loadServiceJSON(root)
	if err != nil {
		return nil, nil
	}
	svc := svcedit.EnsureMap(doc.Root(), "service")
	svc.Set("name", dir)
	after, err := doc.Bytes()
	if err != nil {
		return nil, err
	}
	return []Candidate{{
		RuleID:      RuleServiceNameMismatch,
		FilePath:    filepath.Join(root, filepath.FromSlash(serviceJSONRel)),
		Description: fmt.Sprintf("Set service.name to %q (must equal the scenario directory).", dir),
		Before:      string(before),
		After:       string(after),
	}}, nil
}

// previewHealthCheck adds an http health check when an API surface is declared
// but no http check exists.
func previewHealthCheck(root string) ([]Candidate, error) {
	in, err := intent.Load(root)
	if err != nil {
		return nil, nil
	}
	api, _, _ := declaredSurfaces(in)
	if !api {
		return nil, nil
	}
	for _, c := range in.Lifecycle.Health.Checks {
		if strings.EqualFold(c.Type, "http") {
			return nil, nil
		}
	}
	doc, before, err := loadServiceJSON(root)
	if err != nil {
		return nil, nil
	}
	health := svcedit.EnsureMap(svcedit.EnsureMap(doc.Root(), "lifecycle"), "health")
	svcedit.AppendToSlice(health, "checks", svcedit.NewObject(
		"name", "api_endpoint",
		"type", "http",
		"target", "http://localhost:${API_PORT}/health",
		"critical", true,
		"timeout", 10000,
		"interval", 30000,
	))
	after, err := doc.Bytes()
	if err != nil {
		return nil, err
	}
	return []Candidate{{
		RuleID:      RuleHealthCheckMissing,
		FilePath:    filepath.Join(root, filepath.FromSlash(serviceJSONRel)),
		Description: "Add an http lifecycle.health check targeting the API /health endpoint.",
		Before:      string(before),
		After:       string(after),
	}}, nil
}

// previewMalformedHealthCheck normalizes the first existing HTTP health check
// onto the canonical API readiness probe. It does not create checks; the missing
// check fixer owns that case.
func previewMalformedHealthCheck(root string) ([]Candidate, error) {
	in, err := intent.Load(root)
	if err != nil {
		return nil, nil
	}
	api, _, _ := declaredSurfaces(in)
	if !api {
		return nil, nil
	}
	hasHTTP := false
	for _, c := range in.Lifecycle.Health.Checks {
		if strings.EqualFold(c.Type, "http") {
			hasHTTP = true
			if isCanonicalHealthTarget(c.Target) && c.Critical {
				return nil, nil
			}
			break
		}
	}
	if !hasHTTP {
		return nil, nil
	}
	doc, before, err := loadServiceJSON(root)
	if err != nil {
		return nil, nil
	}
	health := svcedit.EnsureMap(svcedit.EnsureMap(doc.Root(), "lifecycle"), "health")
	check, ok := svcedit.FindMapInSlice(health, "checks", func(m *svcedit.Map) bool {
		return strings.EqualFold(strings.TrimSpace(svcedit.StringField(m, "type")), "http")
	})
	if !ok {
		return nil, nil
	}
	if strings.TrimSpace(svcedit.StringField(check, "name")) == "" {
		check.Set("name", "api_endpoint")
	}
	check.Set("type", "http")
	check.Set("target", canonicalAPIHealthTarget)
	check.Set("critical", true)
	after, err := doc.Bytes()
	if err != nil {
		return nil, err
	}
	return []Candidate{{
		RuleID:      RuleHealthCheckMalformed,
		FilePath:    filepath.Join(root, filepath.FromSlash(serviceJSONRel)),
		Description: "Normalize the API lifecycle health check to the canonical critical /health probe.",
		Before:      string(before),
		After:       string(after),
	}}, nil
}

// previewFreshnessChecks adds the missing per-surface freshness checks in a
// single edit (binaries for api, cli for cli, ui-bundle for ui).
func previewFreshnessChecks(root string) ([]Candidate, error) {
	in, err := intent.Load(root)
	if err != nil {
		return nil, nil
	}
	api, ui, cli := declaredSurfaces(in)
	scenario := scenarioName(root, in)

	type want struct {
		declared bool
		typ      string
		build    func() interface{}
	}
	wants := []want{
		{api, "binaries", func() interface{} {
			return svcedit.NewObject("type", "binaries", "targets", []interface{}{"api/" + scenario + "-api"})
		}},
		{cli, "cli", func() interface{} {
			return svcedit.NewObject("type", "cli", "command", scenario)
		}},
		{ui, "ui-bundle", func() interface{} {
			return svcedit.NewObject("type", "ui-bundle", "bundle_path", "ui/dist/index.html", "source_dir", "ui/src")
		}},
	}

	var missing []want
	for _, w := range wants {
		if w.declared && len(in.FreshCheckByType(w.typ)) == 0 {
			missing = append(missing, w)
		}
	}
	if len(missing) == 0 {
		return nil, nil
	}
	doc, before, err := loadServiceJSON(root)
	if err != nil {
		return nil, nil
	}
	condition := svcedit.EnsureMap(svcedit.EnsureMap(svcedit.EnsureMap(doc.Root(), "lifecycle"), "setup"), "condition")
	types := make([]string, 0, len(missing))
	for _, w := range missing {
		svcedit.AppendToSlice(condition, "checks", w.build())
		types = append(types, w.typ)
	}
	after, err := doc.Bytes()
	if err != nil {
		return nil, err
	}
	return []Candidate{{
		RuleID:      RuleFreshnessMissing,
		FilePath:    filepath.Join(root, filepath.FromSlash(serviceJSONRel)),
		Description: "Add missing lifecycle.setup freshness checks: " + strings.Join(types, ", ") + ".",
		Before:      string(before),
		After:       string(after),
	}}, nil
}

// previewPortBand rewrites each canonical listener port (api/ui/websocket) whose
// env_var or range is present but off its canonical band, in a single
// format-preserving edit. Missing allocations and out-of-band fixed ports are
// left untouched (the rule does not flag them) so the fix is always a safe
// in-place correction.
func previewPortBand(root string) ([]Candidate, error) {
	in, err := intent.Load(root)
	if err != nil {
		return nil, nil
	}
	type change struct{ name, env, band string }
	var changes []change
	for _, name := range sortedPortNames(in.Ports) {
		p := in.Ports[name]
		band, env, ok := rules.CanonicalPortBand(name, p.EnvVar)
		if !ok {
			continue
		}
		envWrong := strings.TrimSpace(p.EnvVar) != "" && strings.TrimSpace(p.EnvVar) != env
		rangeWrong := strings.TrimSpace(p.Range) != "" && strings.TrimSpace(p.Range) != band
		if envWrong || rangeWrong {
			changes = append(changes, change{name: name, env: env, band: band})
		}
	}
	if len(changes) == 0 {
		return nil, nil
	}
	doc, before, err := loadServiceJSON(root)
	if err != nil {
		return nil, nil
	}
	ports := svcedit.EnsureMap(doc.Root(), "ports")
	names := make([]string, 0, len(changes))
	for _, c := range changes {
		port, ok := svcedit.GetMap(ports, c.name)
		if !ok {
			continue
		}
		if v := strings.TrimSpace(svcedit.StringField(port, "env_var")); v != "" && v != c.env {
			port.Set("env_var", c.env)
		}
		if v := strings.TrimSpace(svcedit.StringField(port, "range")); v != "" && v != c.band {
			port.Set("range", c.band)
		}
		names = append(names, c.name)
	}
	after, err := doc.Bytes()
	if err != nil {
		return nil, err
	}
	return []Candidate{{
		RuleID:      RulePortBand,
		FilePath:    filepath.Join(root, filepath.FromSlash(serviceJSONRel)),
		Description: "Move canonical port(s) " + strings.Join(names, ", ") + " back onto their allocated env_var/band.",
		Before:      string(before),
		After:       string(after),
	}}, nil
}

// previewAPIBinaryName renames the start-api invocation (and its file_exists
// condition) to the canonical api/<scenario>-api binary, preserving the rest of
// the command. It only fires on the two lifecycle-recognized run shapes, so the
// rename is always surgical.
func previewAPIBinaryName(root string) ([]Candidate, error) {
	in, err := intent.Load(root)
	if err != nil {
		return nil, nil
	}
	expected := rules.ExpectedAPIBinaryName(scenarioName(root, in))
	if expected == "" {
		return nil, nil
	}
	var step *intent.Step
	for i := range in.Lifecycle.Develop.Steps {
		if strings.TrimSpace(in.Lifecycle.Develop.Steps[i].Name) == "start-api" {
			step = &in.Lifecycle.Develop.Steps[i]
			break
		}
	}
	if step == nil {
		return nil, nil
	}
	current, fixedRun, ok := rules.RewriteAPIBinary(step.Run, expected)
	if !ok {
		return nil, nil
	}
	doc, before, err := loadServiceJSON(root)
	if err != nil {
		return nil, nil
	}
	develop, ok := svcedit.GetMap(svcedit.EnsureMap(doc.Root(), "lifecycle"), "develop")
	if !ok {
		return nil, nil
	}
	stepMap, ok := svcedit.FindMapInSlice(develop, "steps", func(m *svcedit.Map) bool {
		return strings.TrimSpace(svcedit.StringField(m, "name")) == "start-api"
	})
	if !ok {
		return nil, nil
	}
	stepMap.Set("run", fixedRun)
	// Repoint the staleness condition at the renamed binary when it referenced
	// the old name (the condition.file_exists tracks the api binary).
	if cond, ok := svcedit.GetMap(stepMap, "condition"); ok {
		if fe := svcedit.StringField(cond, "file_exists"); strings.Contains(fe, current) {
			cond.Set("file_exists", strings.Replace(fe, current, expected, 1))
		}
	}
	after, err := doc.Bytes()
	if err != nil {
		return nil, err
	}
	return []Candidate{{
		RuleID:      RuleAPIBinaryName,
		FilePath:    filepath.Join(root, filepath.FromSlash(serviceJSONRel)),
		Description: fmt.Sprintf("Rename the start-api binary from %q to the canonical %q.", current, expected),
		Before:      string(before),
		After:       string(after),
	}}, nil
}

// previewProductionServe rewrites a UI develop step that runs a dev server into
// one that serves the built production bundle, preserving the hashed-asset
// caching guarantee. It edits only the step's run command.
func previewProductionServe(root string) ([]Candidate, error) {
	in, err := intent.Load(root)
	if err != nil {
		return nil, nil
	}
	if _, ui, _ := declaredSurfaces(in); !ui {
		return nil, nil
	}
	// Mirror productionServeRules' UI-step selection exactly (name lowercased,
	// run matched as-is) so the fixer rewrites the same step the rule flagged.
	var step *intent.Step
	for i := range in.Lifecycle.Develop.Steps {
		s := &in.Lifecycle.Develop.Steps[i]
		if strings.Contains(strings.ToLower(s.Name), "ui") || strings.Contains(s.Run, "ui") {
			step = s
			break
		}
	}
	if step == nil || !rules.IsDevServer(step.Run) {
		return nil, nil
	}
	doc, before, err := loadServiceJSON(root)
	if err != nil {
		return nil, nil
	}
	develop, ok := svcedit.GetMap(svcedit.EnsureMap(doc.Root(), "lifecycle"), "develop")
	if !ok {
		return nil, nil
	}
	stepName := strings.TrimSpace(step.Name)
	stepMap, ok := svcedit.FindMapInSlice(develop, "steps", func(m *svcedit.Map) bool {
		name := strings.TrimSpace(svcedit.StringField(m, "name"))
		if stepName != "" {
			return name == stepName
		}
		return rules.IsDevServer(svcedit.StringField(m, "run"))
	})
	if !ok {
		return nil, nil
	}
	const productionRun = "cd ui && NODE_ENV=production node server.js"
	stepMap.Set("run", productionRun)
	after, err := doc.Bytes()
	if err != nil {
		return nil, err
	}
	return []Candidate{{
		RuleID:      RuleProductionServe,
		FilePath:    filepath.Join(root, filepath.FromSlash(serviceJSONRel)),
		Description: "Serve the built ui/dist bundle (" + productionRun + ") instead of a dev server.",
		Before:      string(before),
		After:       string(after),
	}}, nil
}

// --- filesystem fixers ----------------------------------------------------

// previewSurfaceDirs creates a .gitkeep for each declared surface whose
// directory is absent (one candidate per missing surface directory).
func previewSurfaceDirs(root string) ([]Candidate, error) {
	in, err := intent.Load(root)
	if err != nil {
		return nil, nil
	}
	api, ui, cli := declaredSurfaces(in)
	var out []Candidate
	for _, s := range []struct {
		want bool
		dir  string
	}{{api, "api"}, {ui, "ui"}, {cli, "cli"}} {
		if !s.want || dirExists(filepath.Join(root, s.dir)) {
			continue
		}
		keep := filepath.Join(root, s.dir, ".gitkeep")
		out = append(out, Candidate{
			RuleID:      RuleSurfaceDirMissing,
			FilePath:    keep,
			Description: fmt.Sprintf("Create the %s/ surface directory.", s.dir),
			Before:      "",
			After:       "",
		})
	}
	return out, nil
}

// previewRequiredFiles creates a README.md stub when it is missing. The Makefile
// is intentionally NOT auto-generated: it must route real targets through vrooli
// and cannot be safely synthesized.
func previewRequiredFiles(root string) ([]Candidate, error) {
	readme := filepath.Join(root, "README.md")
	if fileExists(readme) {
		return nil, nil
	}
	scenario := scenarioDir(root)
	content := fmt.Sprintf("# %s\n\nProfile-aware structure/lifecycle validation scenario.\n\n"+
		"See `.vrooli/service.json` for the lifecycle contract and `docs/` for details.\n", scenario)
	return []Candidate{{
		RuleID:      RuleRequiredFileMissing,
		FilePath:    readme,
		Description: "Create a README.md stub at the scenario root.",
		Before:      "",
		After:       content,
	}}, nil
}

// --- helpers --------------------------------------------------------------

// declaredSurfaces mirrors reconcile's declared-surface derivation from intent
// alone (ports, health endpoints, cli.enabled) without needing code facts.
func declaredSurfaces(in intent.Intent) (api, ui, cli bool) {
	mark := func(name string) {
		switch strings.ToLower(name) {
		case "api":
			api = true
		case "ui":
			ui = true
		case "cli":
			cli = true
		}
	}
	for name := range in.Ports {
		mark(name)
	}
	for name := range in.Lifecycle.Health.Endpoints {
		mark(name)
	}
	if in.CLIEnabled {
		cli = true
	}
	return api, ui, cli
}

func scenarioDir(root string) string {
	return filepath.Base(strings.TrimRight(root, string(filepath.Separator)))
}

// sortedPortNames returns the declared port names in deterministic order.
func sortedPortNames(ports map[string]intent.Port) []string {
	out := make([]string, 0, len(ports))
	for name := range ports {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func scenarioName(root string, in intent.Intent) string {
	if n := strings.TrimSpace(in.Name); n != "" {
		return n
	}
	return scenarioDir(root)
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
