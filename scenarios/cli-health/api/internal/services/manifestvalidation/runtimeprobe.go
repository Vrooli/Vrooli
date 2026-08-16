package manifestvalidation

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"cli-health/internal/cliruntime"

	"github.com/vrooli/cli-core/cliapp"
)

// RuntimeProbe observes a scenario CLI binary's runtime behavior by resolving
// and exec'ing it (binary resolution + `--help`-tree walk both live in
// internal/cliruntime, shared with the AI-search index builder). It is the seam
// that turns cli-health's static manifest↔proto reconciliation into a static +
// runtime CLI authority.
//
// seam: RuntimeProbe — production wires cliRuntimeProbe (internal/cliruntime);
// tests inject a fake returning canned observations. Optional on Deps: a nil
// probe disables runtime probing entirely (the default static-only path). Even
// when wired, the probe runs only when the caller requests execution
// (include_execution) and the scenario declares a CLI surface.
type RuntimeProbe interface {
	Probe(ctx context.Context, scenario string) (RuntimeObservation, error)
}

// RuntimeObservation is what the probe saw when it ran the scenario CLI.
type RuntimeObservation struct {
	// Resolved reports whether the binary was found in this run context. When
	// false the scenario's CLI is simply not installed here (degrade, never a
	// hard error).
	Resolved bool
	// Binary is the resolved executable path (informational).
	Binary string
	// HelpFailed reports that the binary resolved but its root `--help` errored
	// or produced no output (a real defect — the CLI runs but its help is broken).
	HelpFailed bool
	// HelpError carries the help-failure detail when HelpFailed.
	HelpError string
	// Commands are the leaf commands the binary exposed at runtime, relative to
	// the scenario (origin stripped).
	Commands []RuntimeCommand
}

// RuntimeCommand is one leaf the binary exposed at runtime. Group is the first
// path segment ("" for a top-level command); Name is the leaf command.
type RuntimeCommand struct {
	Group string
	Name  string
}

// builtinCommands are commands cli-core injects into every CLI that are not
// part of the manifest's proto-bound surface. They must never be flagged as
// "undeclared" when observed at runtime.
var builtinCommands = map[string]bool{
	"help":       true,
	"version":    true,
	"configure":  true,
	"completion": true,
	"status":     true,
}

// cliRuntimeProbe is the production RuntimeProbe: it resolves the scenario's
// binary by name (matching the aisearch resolution rule) and walks its
// `--help` tree, both via internal/cliruntime.
type cliRuntimeProbe struct {
	timeout time.Duration
}

// NewCLIRuntimeProbe returns the production runtime probe. A non-positive
// timeout defaults inside cliruntime.ExecRunner.
func NewCLIRuntimeProbe(timeout time.Duration) RuntimeProbe {
	return cliRuntimeProbe{timeout: timeout}
}

func (p cliRuntimeProbe) Probe(ctx context.Context, scenario string) (RuntimeObservation, error) {
	binaryName := scenario
	origin := scenario
	if isProjectTarget(scenario) {
		binaryName = ProjectCLIBinary
		origin = ProjectCLIBinary
	}
	bin := cliruntime.ResolveBinary(binaryName, "")
	if bin == "" {
		return RuntimeObservation{Resolved: false}, nil
	}
	obs := RuntimeObservation{Resolved: true, Binary: bin}

	maxDepth := cliruntime.DefaultHelpMaxDepth
	if isProjectTarget(scenario) {
		// The root manifest catalogs the control-plane command tree. The root
		// CLI's immediate subcommands are the authority for this target; deeper
		// trees belong to the scenario/resource command surfaces and would make
		// a parent command look like a missing leaf after flattening.
		maxDepth = 2
	}
	cmds := cliruntime.ParseHelpTree(ctx, cliruntime.ExecRunner(p.timeout), bin,
		cliruntime.HelpTreeOptions{Origin: origin, MaxDepth: maxDepth})

	// A single help-failed stub means the root `--help` did not run cleanly.
	if len(cmds) == 1 && cmds[0].Source == cliruntime.SourceHelpFailed {
		obs.HelpFailed = true
		obs.HelpError = cmds[0].Description
		return obs, nil
	}
	for _, c := range cmds {
		if c.Source == cliruntime.SourceHelpFailed {
			continue
		}
		obs.Commands = append(obs.Commands, RuntimeCommand{Group: c.Group, Name: c.Name})
	}
	return obs, nil
}

// hasCLISurface reports whether the manifest declares any command (i.e. there is
// a CLI to probe at all). A manifest with no groups/commands has no runtime
// surface to validate.
func hasCLISurface(m *cliapp.Manifest) bool {
	if m == nil {
		return false
	}
	for _, g := range m.Groups {
		if len(g.Commands) > 0 {
			return true
		}
	}
	return false
}

// runtimeFindings turns a RuntimeObservation into findings, applying the
// degrade-don't-hard-fail policy:
//   - binary unresolved  -> warning (cli.binary_unrunnable): not installed here.
//   - root --help broken  -> error   (cli.help_failed): present but broken.
//   - command divergence  -> error   (cli.command_undeclared): runtime surface
//     diverges from the manifest SSOT (either direction).
func runtimeFindingsForTarget(obs RuntimeObservation, m *cliapp.Manifest, manifestPath, target string) []Finding {
	if !obs.Resolved {
		return []Finding{{
			Severity:   SeverityWarning,
			Code:       CodeCLIBinaryUnrunnable,
			Location:   manifestPath,
			Message:    "scenario declares a CLI surface but its binary could not be resolved in this run context (PATH lookup failed)",
			Suggestion: "install/expose the scenario CLI so runtime validation can exercise it, or confirm cli.invoke.command matches the installed binary name",
		}}
	}
	if obs.HelpFailed {
		msg := "scenario CLI binary resolves but `--help` failed to run cleanly"
		if strings.TrimSpace(obs.HelpError) != "" {
			msg = fmt.Sprintf("%s: %s", msg, strings.TrimSpace(obs.HelpError))
		}
		return []Finding{{
			Severity:   SeverityError,
			Code:       CodeCLIHelpFailed,
			Location:   obs.Binary,
			Message:    msg,
			Suggestion: "run `<cli> --help` manually and fix the error so the CLI is introspectable",
		}}
	}
	findings := commandSurfaceFindingsForTarget(obs, m, manifestPath, target)
	if isProjectTarget(target) && len(obs.Commands) == 0 {
		findings = append(findings, Finding{
			Severity:   SeverityWarning,
			Code:       CodeProjectCLIEmpty,
			Location:   manifestPath,
			Message:    "project CLI help emitted no registered commands, so project command coverage cannot be treated as clean",
			Suggestion: "ensure the root vrooli CLI registers its command tree before validating the project target",
		})
	}
	return append(findings, omissionContradictionFindings(obs, m, manifestPath)...)
}

// commandSurfaceFindings reconciles the binary's runtime command surface against
// the manifest's declared commands in both directions. Runtime-only groups are
// contract gaps: ignoring them made a partial manifest look complete.
func commandSurfaceFindings(obs RuntimeObservation, m *cliapp.Manifest, manifestPath string) []Finding {
	return commandSurfaceFindingsForTarget(obs, m, manifestPath, "")
}

func commandSurfaceFindingsForTarget(obs RuntimeObservation, m *cliapp.Manifest, manifestPath, target string) []Finding {
	// Commands declared as legitimate special cases in the manifest's top-level
	// exceptions[] are not "undeclared": they live outside the binding path on
	// purpose. Treat them like framework built-ins so declaring an exception
	// silences the undeclared-command finding for that command.
	declaredExceptions := exceptionCommandPaths(m)
	manifestByGroup := map[string]map[string]bool{}
	for _, g := range m.Groups {
		group := runtimeManifestGroup(m, g.Name, g.Flat, target)
		set := manifestByGroup[group]
		if set == nil {
			set = map[string]bool{}
			manifestByGroup[group] = set
		}
		for _, c := range g.Commands {
			set[c.Name] = true
		}
	}
	runtimeByGroup := map[string]map[string]bool{}
	for _, rc := range obs.Commands {
		set := runtimeByGroup[rc.Group]
		if set == nil {
			set = map[string]bool{}
			runtimeByGroup[rc.Group] = set
		}
		set[rc.Name] = true
	}

	var findings []Finding
	groups := map[string]bool{}
	for group := range manifestByGroup {
		groups[group] = true
	}
	for group := range runtimeByGroup {
		groups[group] = true
	}
	for _, group := range sortedKeys(groups) {
		runtime, present := runtimeByGroup[group]
		if !present {
			// Group not observed at runtime — likely a help-parse gap or a
			// capability-gated group; do not flag its whole command set missing.
			continue
		}
		declared := manifestByGroup[group]

		// Declared-but-missing: the manifest claims a command the binary's help
		// tree does not expose under a group it otherwise does expose.
		for _, name := range sortedKeys(declared) {
			if isProjectTarget(target) && group == "" && projectParentCommand(m, name) {
				// The root manifest catalogs parent commands as governance
				// entries, but the flattened runtime observation reports the
				// command's registered children. Parent absence is therefore not
				// a manifest/runtime contradiction.
				continue
			}
			if runtime[name] {
				continue
			}
			findings = append(findings, Finding{
				Severity:   SeverityError,
				Code:       CodeCLICommandUndeclared,
				Location:   commandLocation(manifestPath, group, name),
				Message:    fmt.Sprintf("manifest declares command %q but the binary does not expose it at runtime under group %q", groupCmd(group, name), group),
				Suggestion: "rebuild/reinstall the CLI, or remove the stale command from the manifest",
			})
		}

		// Undeclared-accepted: the binary exposes a command under a manifest group
		// that the manifest neither binds nor (for built-ins) needs to.
		for _, name := range sortedKeys(runtime) {
			if declared[name] || builtinCommands[name] || declaredExceptions[normalizeCommandPath(groupCmd(group, name))] {
				continue
			}
			findings = append(findings, Finding{
				Severity:   SeverityError,
				Code:       CodeCLICommandUndeclared,
				Location:   commandLocation(manifestPath, group, name),
				Message:    fmt.Sprintf("binary exposes command %q at runtime but the manifest does not declare it (manifest is the CLI SSOT)", groupCmd(group, name)),
				Suggestion: "add the command to the manifest under its group/binding, declare it in exceptions[] if it is a legitimate special case (streaming/upload/passthrough/durable run), or remove it from the CLI",
			})
		}
	}
	if len(runtimeByGroup) > 0 && len(findings) > 0 {
		findings = append(findings, Finding{
			Severity: SeverityError, Code: CodeCLIDiscoveryCoverage, Location: manifestPath,
			Message:    "manifest command coverage is below the observed runtime CLI surface",
			Suggestion: "declare every runtime leaf command in cli/manifest.json so discovery indexes the executable CLI contract",
		})
	}
	return findings
}

// runtimeManifestGroup maps governance-only synthetic groups back onto the
// first runtime help-tree segment. The manifest can keep a primitive-only
// group (for example scenario-primitives) without making the runtime surface
// appear to contain a second, fictional command tree.
func runtimeManifestGroup(m *cliapp.Manifest, name string, flat bool, target string) string {
	if flat {
		return ""
	}
	if !isProjectTarget(target) {
		return name
	}
	// Primitive migration groups are source/governance partitions, not extra
	// runtime help nodes. Other hyphenated groups (resource-archive,
	// runtime-supervisor, credentials-store, ...) describe deeper help trees;
	// flattening them at the project's bounded depth would falsely require
	// grandchildren that were intentionally not probed.
	if name == "scenario-primitives" {
		return "scenario"
	}
	return name
}

func projectParentCommand(m *cliapp.Manifest, name string) bool {
	name = normalizeCommandPath(name)
	for _, g := range m.Groups {
		if g.Name == name && !g.Flat {
			return true
		}
		if strings.HasPrefix(g.Name, name+"-") {
			return true
		}
	}
	return false
}

var handRegisteredCommand = regexp.MustCompile(`(?i)hand-registered (?:as|through) ['\x60]([^'\x60]+)['\x60]`)

func omissionContradictionFindings(obs RuntimeObservation, m *cliapp.Manifest, manifestPath string) []Finding {
	live := map[string]bool{}
	for _, command := range obs.Commands {
		live[normalizeCommandPath(groupCmd(command.Group, command.Name))] = true
	}
	var findings []Finding
	for _, omission := range m.Omitted {
		for _, match := range handRegisteredCommand.FindAllStringSubmatch(omission.Reason, -1) {
			path := normalizeCommandPath(match[1])
			if !live[path] {
				continue
			}
			findings = append(findings, Finding{
				Severity: SeverityError, Code: CodeOmissionContradictsCommand, Location: manifestPath + "#/omitted",
				Message:    fmt.Sprintf("omitted RPC %s.%s says live command %q is absent from the CLI surface", omission.Service, omission.Method, path),
				Suggestion: "replace the omission with the command binding or state a reason that does not contradict the runtime CLI",
			})
		}
	}
	return findings
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func groupCmd(group, name string) string {
	if group == "" {
		return name
	}
	return group + " " + name
}

func commandLocation(manifestPath, group, name string) string {
	return fmt.Sprintf("%s#/groups/%s/commands/%s", manifestPath, group, name)
}
