// Package rules turns a reconcile.Model into structure-health findings. The
// rules are profile-aware but the universal checks here apply to every scenario;
// profile-keyed conformance packs layer on in a later phase.
//
// Each finding's Code matches a code in .vrooli/maturity.json so the assessment
// builder can map it to a maturity level and severity.
package rules

import (
	"os"
	"path/filepath"
	"strings"

	"structure-health/internal/intent"
	"structure-health/internal/reconcile"
)

// Finding is a structure-health rule finding (mapped to a native/proto finding
// and a maturity assessment finding downstream).
type Finding struct {
	Code        string
	Severity    string
	Title       string
	Message     string
	Location    string
	Remediation string
	Surface     string
}

const (
	sevError   = "error"
	sevWarning = "warning"
)

// Input carries everything the rules need.
type Input struct {
	Model reconcile.Model
	// ServiceJSONReadable is false when service.json could not be parsed.
	ServiceJSONReadable bool
}

// Evaluate runs every universal structure + lifecycle-wiring rule and returns
// the findings in a stable order.
func Evaluate(in Input) []Finding {
	var out []Finding
	out = append(out, serviceJSONRules(in)...)
	out = append(out, requiredFileRules(in)...)
	out = append(out, surfaceDirRules(in)...)
	out = append(out, lifecycleWiringRules(in)...)
	out = append(out, freshnessRules(in)...)
	out = append(out, healthCheckRules(in)...)
	out = append(out, dependencyRules(in)...)
	out = append(out, productionServeRules(in)...)
	out = append(out, reconcileRules(in)...)
	return out
}

// serviceJSONRules checks the service.json contract: present, valid, and
// service.name matching the scenario directory.
func serviceJSONRules(in Input) []Finding {
	root := in.Model.RootPath
	if !exists(filepath.Join(root, ".vrooli", "service.json")) {
		return []Finding{{
			Code:        "SERVICE_JSON_MISSING",
			Severity:    sevError,
			Title:       "service.json is missing",
			Message:     "The scenario has no .vrooli/service.json; the lifecycle engine cannot start it.",
			Location:    ".vrooli/service.json",
			Remediation: "Add a .vrooli/service.json declaring the service, ports, and lifecycle.",
		}}
	}
	if !in.ServiceJSONReadable {
		return []Finding{{
			Code:        "SERVICE_JSON_INVALID",
			Severity:    sevError,
			Title:       "service.json is invalid",
			Message:     "The scenario's .vrooli/service.json could not be parsed.",
			Location:    ".vrooli/service.json",
			Remediation: "Fix the JSON syntax / schema so service.json parses.",
		}}
	}
	var out []Finding
	declaredName := strings.TrimSpace(in.Model.Intent.Name)
	dirName := filepath.Base(strings.TrimRight(root, string(filepath.Separator)))
	if declaredName != "" && dirName != "" && declaredName != dirName {
		out = append(out, Finding{
			Code:        "SERVICE_NAME_MISMATCH",
			Severity:    sevError,
			Title:       "service.name does not match the directory",
			Message:     "service.name is " + declaredName + " but the scenario directory is " + dirName + ".",
			Location:    ".vrooli/service.json",
			Remediation: "Set service.name to " + dirName + " (it must equal the scenario directory name).",
		})
	}
	return out
}

// requiredFileRules asserts the universal top-level files every scenario needs.
func requiredFileRules(in Input) []Finding {
	root := in.Model.RootPath
	var out []Finding
	for _, f := range []struct{ rel, why string }{
		{"Makefile", "the canonical lifecycle entrypoint (make start/test/logs/stop)"},
		{"README.md", "the scenario overview and documentation map"},
	} {
		if !exists(filepath.Join(root, filepath.FromSlash(f.rel))) {
			out = append(out, Finding{
				Code:        "REQUIRED_FILE_MISSING",
				Severity:    sevError,
				Title:       "required file missing: " + f.rel,
				Message:     f.rel + " is required — " + f.why + ".",
				Location:    f.rel,
				Remediation: "Add a " + f.rel + " to the scenario root.",
			})
		}
	}
	return out
}

// surfaceDirRules asserts each declared surface has a directory on disk.
func surfaceDirRules(in Input) []Finding {
	root := in.Model.RootPath
	var out []Finding
	for _, s := range in.Model.Surfaces {
		if !s.Declared {
			continue
		}
		dir := surfaceDir(s.Surface)
		if dir == "" {
			continue
		}
		if !dirExists(filepath.Join(root, dir)) {
			out = append(out, Finding{
				Code:        "SURFACE_DIR_MISSING",
				Severity:    sevError,
				Title:       "declared surface has no directory: " + s.Surface,
				Message:     "Surface " + s.Surface + " is declared but its " + dir + "/ directory is missing.",
				Location:    dir + "/",
				Remediation: "Create the " + dir + "/ directory and its entrypoint, or remove the declaration.",
				Surface:     s.Surface,
			})
		}
	}
	return out
}

// lifecycleWiringRules asserts each declared+buildable surface has the
// build→start chain plus a port/env binding.
func lifecycleWiringRules(in Input) []Finding {
	lc := in.Model.Intent.Lifecycle
	setup := stepRuns(lc.Setup.Steps)
	develop := stepRuns(lc.Develop.Steps)
	var out []Finding
	for _, s := range in.Model.Surfaces {
		if !s.Declared {
			continue
		}
		switch strings.ToLower(s.Kind) {
		case "api", "ui":
			if !mentions(setup, s.Surface) && !mentions(setup, "build") {
				out = append(out, Finding{
					Code:        "LIFECYCLE_STEP_MISSING",
					Severity:    sevError,
					Title:       "no setup/build step for surface " + s.Surface,
					Message:     "Surface " + s.Surface + " is declared but no lifecycle.setup step builds it.",
					Location:    ".vrooli/service.json",
					Remediation: "Add a lifecycle.setup step that builds the " + s.Surface + " surface.",
					Surface:     s.Surface,
				})
			}
			if !mentions(develop, s.Surface) && !mentions(develop, "start") {
				out = append(out, Finding{
					Code:        "LIFECYCLE_STEP_MISSING",
					Severity:    sevError,
					Title:       "no develop/start step for surface " + s.Surface,
					Message:     "Surface " + s.Surface + " is declared but no lifecycle.develop step starts it.",
					Location:    ".vrooli/service.json",
					Remediation: "Add a background lifecycle.develop step that starts the " + s.Surface + " surface.",
					Surface:     s.Surface,
				})
			}
			if _, ok := in.Model.Intent.Ports[s.Surface]; !ok {
				out = append(out, Finding{
					Code:        "LIFECYCLE_STEP_MISSING",
					Severity:    sevError,
					Title:       "no port binding for surface " + s.Surface,
					Message:     "Surface " + s.Surface + " is declared but has no ports." + s.Surface + " env_var/range binding.",
					Location:    ".vrooli/service.json",
					Remediation: "Add a ports." + s.Surface + " entry with an env_var and range.",
					Surface:     s.Surface,
				})
			}
		}
	}
	return out
}

// freshnessRules asserts each buildable surface declares a freshness check so
// the lifecycle skips rebuilds when sources are unchanged.
func freshnessRules(in Input) []Finding {
	intentDoc := in.Model.Intent
	var out []Finding
	for _, s := range in.Model.Surfaces {
		if !s.Declared {
			continue
		}
		var checkType, hint string
		switch strings.ToLower(s.Kind) {
		case "api":
			checkType, hint = "binaries", "a binaries check listing api/<scenario>-api"
		case "ui":
			checkType, hint = "ui-bundle", "a ui-bundle check with bundle_path and source_dir"
		case "cli":
			checkType, hint = "cli", "a cli check naming the installed command"
		default:
			continue
		}
		if len(intentDoc.FreshCheckByType(checkType)) == 0 {
			out = append(out, Finding{
				Code:        "FRESHNESS_CHECK_MISSING",
				Severity:    sevWarning,
				Title:       "missing freshness check for surface " + s.Surface,
				Message:     "Surface " + s.Surface + " has no lifecycle.setup " + checkType + " freshness check; it rebuilds on every start.",
				Location:    ".vrooli/service.json",
				Remediation: "Add " + hint + " under lifecycle.setup.condition.checks.",
				Surface:     s.Surface,
			})
		}
	}
	return out
}

// healthCheckRules asserts a critical http health check exists for the api.
func healthCheckRules(in Input) []Finding {
	health := in.Model.Intent.Lifecycle.Health
	if !apiDeclared(in.Model) {
		return nil
	}
	for _, c := range health.Checks {
		if strings.EqualFold(c.Type, "http") {
			return nil
		}
	}
	return []Finding{{
		Code:        "HEALTH_CHECK_MISSING",
		Severity:    sevError,
		Title:       "no http health check",
		Message:     "The scenario declares an API surface but no lifecycle.health http check; the lifecycle cannot confirm readiness.",
		Location:    ".vrooli/service.json",
		Remediation: "Add a lifecycle.health.checks entry of type http targeting the API /health endpoint.",
	}}
}

// dependencyRules asserts dependency declarations use valid enums and no edge
// names the scenario itself (a trivial cycle).
func dependencyRules(in Input) []Finding {
	var out []Finding
	check := func(kind string, deps map[string]intent.Dependency) {
		for name, dep := range deps {
			if name == in.Model.Scenario {
				out = append(out, Finding{
					Code:        "DEPENDENCY_DECLARATION_INVALID",
					Severity:    sevError,
					Title:       "self-dependency declared",
					Message:     "The scenario declares a " + kind + " dependency on itself (" + name + ").",
					Location:    ".vrooli/service.json",
					Remediation: "Remove the self-referential dependency.",
				})
			}
			if p := dep.StartupPolicy; p != "" && !validStartupPolicy(p) {
				out = append(out, Finding{
					Code:        "DEPENDENCY_DECLARATION_INVALID",
					Severity:    sevError,
					Title:       "invalid startup_policy for " + name,
					Message:     "Dependency " + name + " has startup_policy " + p + " (must be must_start|try_start|ignore).",
					Location:    ".vrooli/service.json",
					Remediation: "Set a valid startup_policy.",
				})
			}
			if p := dep.FreshnessPolicy; p != "" && !validFreshnessPolicy(p) {
				out = append(out, Finding{
					Code:        "DEPENDENCY_DECLARATION_INVALID",
					Severity:    sevError,
					Title:       "invalid freshness_policy for " + name,
					Message:     "Dependency " + name + " has freshness_policy " + p + " (must be restart_when_stale|reuse_running|rebuild_only).",
					Location:    ".vrooli/service.json",
					Remediation: "Set a valid freshness_policy.",
				})
			}
		}
	}
	check("scenario", in.Model.Intent.Deps.Scenarios)
	check("resource", in.Model.Intent.Deps.Resources)
	return out
}

// productionServeRules asserts the UI develop step serves the built bundle
// rather than a dev server, preserving the hashed-asset caching guarantee.
func productionServeRules(in Input) []Finding {
	if !uiDeclared(in.Model) {
		return nil
	}
	develop := in.Model.Intent.Lifecycle.Develop.Steps
	var uiStep *intent.Step
	for i := range develop {
		s := &develop[i]
		if strings.Contains(strings.ToLower(s.Name), "ui") || strings.Contains(s.Run, "ui") {
			uiStep = s
			break
		}
	}
	if uiStep == nil {
		return nil // covered by lifecycleWiringRules
	}
	run := uiStep.Run
	if isDevServer(run) {
		return []Finding{{
			Code:        "PRODUCTION_SERVE_NONCONFORMANT",
			Severity:    sevWarning,
			Title:       "UI develop step runs a dev server",
			Message:     "The UI develop step appears to run a dev server (" + strings.TrimSpace(run) + ") instead of serving the built production bundle.",
			Location:    ".vrooli/service.json",
			Remediation: "Serve the built ui/dist bundle (e.g. node server.js) with NODE_ENV=production.",
			Surface:     "ui",
		}}
	}
	return nil
}

// reconcileRules flags declared-but-not-detected and detected-but-not-declared
// surfaces.
func reconcileRules(in Input) []Finding {
	var out []Finding
	for _, s := range in.Model.Surfaces {
		switch {
		case s.Declared && !s.Actual:
			out = append(out, Finding{
				Code:        "SURFACE_RECONCILE_MISMATCH",
				Severity:    sevWarning,
				Title:       "declared surface not detected: " + s.Surface,
				Message:     "Surface " + s.Surface + " is declared in service.json but code facts detected no implementation.",
				Location:    surfaceDir(s.Surface) + "/",
				Remediation: "Implement the " + s.Surface + " surface or remove its declaration.",
				Surface:     s.Surface,
			})
		case !s.Declared && s.Actual:
			out = append(out, Finding{
				Code:        "SURFACE_RECONCILE_MISMATCH",
				Severity:    sevWarning,
				Title:       "detected surface not declared: " + s.Surface,
				Message:     "Code facts detected a " + s.Surface + " surface that service.json does not declare (no port/health/cli wiring).",
				Location:    surfaceDir(s.Surface) + "/",
				Remediation: "Declare the " + s.Surface + " surface in service.json (ports/health/cli) or remove the code.",
				Surface:     s.Surface,
			})
		}
	}
	return out
}

// --- helpers ---

func surfaceDir(surface string) string {
	switch strings.ToLower(surface) {
	case "api", "ui", "cli":
		return strings.ToLower(surface)
	default:
		return ""
	}
}

func apiDeclared(m reconcile.Model) bool { return surfaceDeclared(m, "api") }
func uiDeclared(m reconcile.Model) bool  { return surfaceDeclared(m, "ui") }

func surfaceDeclared(m reconcile.Model, kind string) bool {
	for _, s := range m.Surfaces {
		if strings.EqualFold(s.Kind, kind) && s.Declared {
			return true
		}
	}
	return false
}

func stepRuns(steps []intent.Step) []string {
	out := make([]string, 0, len(steps))
	for _, s := range steps {
		out = append(out, strings.ToLower(s.Name+" "+s.Run))
	}
	return out
}

func mentions(haystacks []string, needle string) bool {
	needle = strings.ToLower(needle)
	for _, h := range haystacks {
		if strings.Contains(h, needle) {
			return true
		}
	}
	return false
}

func isDevServer(run string) bool {
	r := strings.ToLower(run)
	for _, marker := range []string{"vite dev", "vite serve", "run dev", "pnpm dev", "npm run dev", "vite\"", "vite "} {
		if strings.Contains(r, marker) {
			return true
		}
	}
	// "vite" alone (the bare dev binary) but not "vite build" / "vite preview".
	if strings.Contains(r, "vite") && !strings.Contains(r, "vite build") && !strings.Contains(r, "vite preview") && !strings.Contains(r, "dist") && !strings.Contains(r, "server.js") {
		return true
	}
	return false
}

func validStartupPolicy(p string) bool {
	switch p {
	case "must_start", "try_start", "ignore":
		return true
	}
	return false
}

func validFreshnessPolicy(p string) bool {
	switch p {
	case "restart_when_stale", "reuse_running", "rebuild_only":
		return true
	}
	return false
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
