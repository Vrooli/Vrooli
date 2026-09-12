package agentharness

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Filesystem removal is floor policy: it is decided the same way under every
// rollout profile. The floor protects what no provider can open up (the
// filesystem root, the home and repository roots, .git) and supplies safe
// defaults. Providers such as storage-manager refine it with PathRules that
// name the locations they own.

const floorSource = "policy-floor"

// RemovalContext is the host state one removal decision reads. It is a value
// so the decision is testable and identical in and out of a hook process.
type RemovalContext struct {
	WorkingDirectory string
	Home             string
	RepoRoot         string
	// ProjectConfigDir and ScratchDir are repository-relative, from the repo
	// contract layout. Empty means the repository declares none.
	ProjectConfigDir string
	ScratchDir       string
	TempRoots        []string
	Lookup           func(string) (string, bool)
	CaseInsensitive  bool
	Rules            []PathRule
	// Stat reports whether an existing target is a directory; nil skips the
	// check, treating every recursive flag as reaching a tree.
	Stat func(string) (fs.FileInfo, error)
}

// HostRemovalContext resolves the context from this process: the user's home,
// the repository the working directory (or VROOLI_ROOT) belongs to, and the
// operating system's temporary directories.
func HostRemovalContext(workingDirectory string) RemovalContext {
	ctx := RemovalContext{
		WorkingDirectory: workingDirectory,
		TempRoots:        hostTempRoots(),
		Lookup:           os.LookupEnv,
		CaseInsensitive:  runtime.GOOS == "windows" || runtime.GOOS == "darwin",
		Stat:             os.Lstat,
	}
	if home, err := os.UserHomeDir(); err == nil {
		ctx.Home = home
	}
	for _, start := range []string{workingDirectory, os.Getenv("VROOLI_ROOT")} {
		if strings.TrimSpace(start) == "" {
			continue
		}
		root, err := repocontract.FindRepoRoot(start)
		if err != nil {
			continue
		}
		contract, err := repocontract.LoadDefault(root)
		if err != nil {
			continue
		}
		layout := contract.Layout()
		ctx.RepoRoot, ctx.ProjectConfigDir, ctx.ScratchDir = root, layout.ProjectConfigDir, layout.ScratchDir
		break
	}
	return ctx
}

func hostTempRoots() []string {
	roots := []string{os.TempDir()}
	if runtime.GOOS != "windows" {
		roots = append(roots, "/tmp", "/var/tmp")
	}
	return roots
}

// removalTargetsFor lists what an event would delete. A shell string is
// parsed; an argument vector is read as argv, first with the tool as the
// program and then, for generic tool names such as "shell", without it.
func removalTargetsFor(event ToolEvent, env removalEnv) []RemovalTarget {
	if strings.TrimSpace(event.Shell) != "" {
		dialect := dialectPOSIX
		if event.Context != nil {
			dialect = dialectFromName(event.Context["shell_dialect"])
		}
		return parseRemovals(event.Shell, dialect, env)
	}
	if len(event.Arguments) == 0 {
		return nil
	}
	if strings.TrimSpace(event.Tool) != "" {
		if targets := parseArgvRemovals(append([]string{event.Tool}, event.Arguments...), env); len(targets) > 0 {
			return targets
		}
	}
	return parseArgvRemovals(event.Arguments, env)
}

type removalVerdict struct {
	target RemovalTarget
	path   string
	action DecisionAction
	zone   string
	reason string
	source string
	// provider is the snapshot whose rule decided; empty for the floor.
	provider string
}

type resolvedRule struct {
	PathRule
	root string
}

type removalZones struct {
	ctx     RemovalContext
	home    string
	repo    string
	git     string
	config  string
	scratch string
	temps   []string
	rules   []resolvedRule
}

// EvaluateRemoval decides a set of removal targets. The strictest target
// decides the event; every target is reported as evidence.
func (ctx RemovalContext) EvaluateRemoval(targets []RemovalTarget) Decision {
	decision := Decision{ContractVersion: ContractVersion, Action: ActionAllow, Risk: RiskFilesystemRemoval, Reason: "the command deletes nothing", Evidence: []Evidence{}}
	if len(targets) == 0 {
		return decision
	}
	zones := ctx.zones()
	verdicts := make([]removalVerdict, 0, len(targets))
	for _, target := range targets {
		verdicts = append(verdicts, zones.classify(target))
	}
	worst := verdicts[0]
	for _, verdict := range verdicts[1:] {
		if actionRank(verdict.action) > actionRank(worst.action) {
			worst = verdict
		}
	}
	decision.Action, decision.Reason = worst.action, worst.reason
	if worst.provider != "" {
		decision.ProviderID, decision.UsedSnapshot = worst.provider, true
	}
	for _, verdict := range verdicts {
		facts := map[string]string{"verb": verdict.target.Verb, "raw": verdict.target.Raw, "zone": verdict.zone, "action": string(verdict.action), "recursive": fmt.Sprint(verdict.target.Recursive)}
		if verdict.path != "" {
			facts["path"] = verdict.path
		}
		decision.Evidence = append(decision.Evidence, Evidence{Code: "REMOVAL_" + strings.ToUpper(strings.ReplaceAll(verdict.zone, "-", "_")), Message: verdict.reason, Source: verdict.source, Facts: facts, Severity: actionSeverity(verdict.action)})
	}
	return decision
}

func (ctx RemovalContext) zones() *removalZones {
	z := &removalZones{ctx: ctx, home: resolveExisting(ctx.Home), repo: resolveExisting(ctx.RepoRoot)}
	if z.repo != "" {
		z.git = filepath.Join(z.repo, ".git")
		if dir := strings.TrimSpace(ctx.ProjectConfigDir); dir != "" {
			z.config = filepath.Join(z.repo, filepath.FromSlash(dir))
		}
		if dir := strings.TrimSpace(ctx.ScratchDir); dir != "" {
			z.scratch = filepath.Join(z.repo, filepath.FromSlash(dir))
		}
	}
	for _, root := range ctx.TempRoots {
		if resolved := resolveExisting(root); resolved != "" {
			z.temps = append(z.temps, resolved)
		}
	}
	for _, rule := range ctx.Rules {
		root := resolveExisting(rule.Root)
		if root == "" || !filepath.IsAbs(root) {
			continue
		}
		// A rule cannot open up what the floor protects; it can only add
		// protection there.
		if rule.Action != ActionDeny {
			if _, protected := z.hardFloor(root, false, true); protected {
				continue
			}
		}
		z.rules = append(z.rules, resolvedRule{PathRule: rule, root: root})
	}
	return z
}

func (z *removalZones) classify(target RemovalTarget) removalVerdict {
	verdict := removalVerdict{target: target, source: floorSource}
	if target.Unresolved != "" {
		verdict.action, verdict.zone = ActionAsk, "unresolved"
		verdict.reason = fmt.Sprintf("cannot tell what %q deletes: %s", target.Raw, target.Unresolved)
		return verdict
	}
	path := z.targetPath(target)
	verdict.path = path
	recursive := target.Recursive
	if recursive && !target.Glob && z.ctx.Stat != nil {
		// A recursive flag on a file still deletes only that file.
		if info, err := z.ctx.Stat(path); err == nil && !info.IsDir() {
			recursive = false
		}
	}
	subject := path
	if target.Glob {
		subject = "the contents of " + path
	}
	if floor, protected := z.hardFloor(path, target.Glob, recursive); protected {
		floor.target, floor.path = target, path
		floor.reason = subject + floor.reason
		return floor
	}
	if rule, ok := z.containingRule(path); ok {
		verdict.action, verdict.zone, verdict.source, verdict.provider = rule.Action, "declared", rule.Source, rule.Provider
		verdict.reason = subject + ruleClause(rule, "is inside")
	} else {
		verdict.action, verdict.zone, verdict.reason = z.floorDefault(path, recursive, target.Glob)
		verdict.reason = subject + verdict.reason
	}
	if recursive {
		z.stricterNested(&verdict, path, subject)
	}
	return verdict
}

// targetPath resolves symlinks in the parent only: deleting a link removes the
// link, not what it points to. A trailing separator or a glob follows it.
func (z *removalZones) targetPath(target RemovalTarget) string {
	raw := strings.TrimSpace(target.Raw)
	if target.Glob || strings.HasSuffix(raw, "/") || strings.HasSuffix(raw, `\`) {
		return resolveExisting(target.Path)
	}
	parent := filepath.Dir(target.Path)
	if parent == target.Path {
		return target.Path
	}
	return filepath.Join(resolveExisting(parent), filepath.Base(target.Path))
}

// hardFloor is the protection no declaration overrides. contents marks a
// glob, which deletes what the directory holds rather than the directory; a
// non-recursive glob at the home or repository root reaches only its top-level
// files and is left to the defaults.
func (z *removalZones) hardFloor(path string, contents, recursive bool) (removalVerdict, bool) {
	deny := func(zone, clause string) (removalVerdict, bool) {
		return removalVerdict{action: ActionDeny, zone: zone, reason: clause, source: floorSource}, true
	}
	parent := filepath.Dir(path)
	wholeTree := !contents || recursive
	switch {
	case parent == path:
		return deny("filesystem-root", " is a filesystem root")
	case !contents && filepath.Dir(parent) == parent:
		return deny("system-directory", " sits directly under the filesystem root")
	case wholeTree && z.home != "" && z.within(z.home, path):
		return deny("home-root", " is the home directory or contains it")
	case wholeTree && z.repo != "" && z.within(z.repo, path):
		return deny("repository-root", " is the repository root or contains it")
	case z.git != "" && z.within(path, z.git):
		return deny("repository-history", " is inside the repository's .git history")
	}
	return removalVerdict{}, false
}

func (z *removalZones) floorDefault(path string, recursive, contents bool) (DecisionAction, string, string) {
	if z.repo != "" && z.within(path, z.repo) {
		switch {
		case z.config != "" && z.within(path, z.config):
			return ActionAsk, "project-config", fmt.Sprintf(" is repository configuration under %s; deleting it needs the operator's confirmation", z.config)
		case z.scratch != "" && z.within(path, z.scratch) && (contents || path != z.scratch):
			return ActionAllow, "scratch", " is in the repository's scratch directory"
		case recursive:
			return ActionAsk, "repository-tree", " is a directory tree in the repository; recursive deletion needs the operator's confirmation"
		case contents:
			return ActionAsk, "repository-pattern", " in the repository are matched by a pattern rather than named; deleting them needs the operator's confirmation"
		default:
			return ActionAllow, "repository-path", " is a single path in the repository"
		}
	}
	for _, temp := range z.temps {
		if z.same(path, temp) {
			return ActionAsk, "temporary-root", " is a shared temporary directory; deleting it needs the operator's confirmation"
		}
		if z.within(path, temp) {
			return ActionAllow, "temporary", " is in a temporary directory"
		}
	}
	if z.home != "" && z.within(path, z.home) {
		return ActionDeny, "home", " is in the home directory outside every location declared safe to delete; ask the operator to delete it"
	}
	return ActionDeny, "outside", " is outside the repository, the temporary directories, and every declared location; ask the operator to delete it"
}

func (z *removalZones) containingRule(path string) (resolvedRule, bool) {
	var best resolvedRule
	found := false
	for _, rule := range z.rules {
		if z.within(path, rule.root) && (!found || len(rule.root) > len(best.root)) {
			best, found = rule, true
		}
	}
	return best, found
}

// stricterNested lets a declaration inside a recursive target win when it is
// stricter: deleting a tree deletes every location in it.
func (z *removalZones) stricterNested(verdict *removalVerdict, path, subject string) {
	for _, rule := range z.rules {
		if z.same(rule.root, path) || !z.within(rule.root, path) || actionRank(rule.Action) <= actionRank(verdict.action) {
			continue
		}
		verdict.action, verdict.zone, verdict.source, verdict.provider = rule.Action, "declared-nested", rule.Source, rule.Provider
		verdict.reason = fmt.Sprintf("%s contains %s", subject, strings.TrimPrefix(ruleClause(rule, ""), " "))
	}
}

// ruleClause explains a declaration in the words the agent will read.
func ruleClause(rule resolvedRule, relation string) string {
	what := map[DecisionAction]string{
		ActionAllow: "declares safe to delete",
		ActionAsk:   "says needs its owner's confirmation before deletion",
		ActionDeny:  "protects",
	}[rule.Action]
	clause := fmt.Sprintf(" %s %s, which %s %s", relation, rule.root, rule.Source, what)
	if relation == "" {
		clause = fmt.Sprintf(" %s, which %s %s", rule.root, rule.Source, what)
	}
	if reason := strings.TrimSpace(rule.Reason); reason != "" {
		clause += " (" + reason + ")"
	}
	return clause
}

func (z *removalZones) same(a, b string) bool {
	if z.ctx.CaseInsensitive {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// within reports whether path is root or lies under it.
func (z *removalZones) within(path, root string) bool {
	if root == "" || path == "" {
		return false
	}
	if z.ctx.CaseInsensitive {
		path, root = strings.ToLower(path), strings.ToLower(root)
	}
	if path == root {
		return true
	}
	if !strings.HasSuffix(root, string(filepath.Separator)) {
		root += string(filepath.Separator)
	}
	return strings.HasPrefix(path, root)
}

// resolveExisting resolves symlinks without requiring the path to exist, so a
// target that has not been created yet still resolves through its parents.
func resolveExisting(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	path = filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	parent := filepath.Dir(path)
	if parent == path {
		return path
	}
	return filepath.Join(resolveExisting(parent), filepath.Base(path))
}

func actionRank(action DecisionAction) int {
	switch action {
	case ActionDeny:
		return 3
	case ActionAsk:
		return 2
	case ActionAllow:
		return 1
	}
	return 0
}

func actionSeverity(action DecisionAction) string {
	switch action {
	case ActionDeny:
		return "error"
	case ActionAsk:
		return "warning"
	}
	return "info"
}
