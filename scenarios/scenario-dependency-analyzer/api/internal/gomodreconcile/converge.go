package gomodreconcile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/envkit-go"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

// ErrConcurrentEdit reports that a surface's go.mod or go.sum changed between
// computing a Convergence and writing it. Nothing is written; rerun.
var ErrConcurrentEdit = errors.New("go.mod or go.sum changed while reconciling; rerun the reconcile")

// RequirementChange is one third-party requirement that convergence adds,
// removes, or moves in a surface go.mod. In-repo modules and modules replaced
// by a local directory are excluded: their wiring is the replace plan.
type RequirementChange struct {
	Path       string
	From       string // "" when added
	To         string // "" when removed
	FromDirect bool
	Direct     bool
}

// Recorded reports whether the converged go.mod records a version for this
// requirement that it did not record before, or newly records it as a direct
// dependency. Those are the changes a dependency gate must examine; removals
// and direct-to-indirect moves never introduce anything.
func (c RequirementChange) Recorded() bool {
	if c.To == "" {
		return false
	}
	return c.From != c.To || (c.Direct && !c.FromDirect)
}

// String renders the change as "path from -> to".
func (c RequirementChange) String() string {
	switch {
	case c.From == "":
		return fmt.Sprintf("add %s %s%s", c.Path, c.To, indirectSuffix(c.Direct))
	case c.To == "":
		return fmt.Sprintf("drop %s %s", c.Path, c.From)
	case c.From != c.To:
		return fmt.Sprintf("move %s %s -> %s%s", c.Path, c.From, c.To, indirectSuffix(c.Direct))
	case c.Direct:
		return fmt.Sprintf("mark %s %s direct", c.Path, c.To)
	default:
		return fmt.Sprintf("mark %s %s indirect", c.Path, c.To)
	}
}

func indirectSuffix(direct bool) string {
	if direct {
		return ""
	}
	return " // indirect"
}

// Convergence is the complete go.mod/go.sum state a surface needs so that its
// requirements match its import closure: missing in-repo replaces, plus every
// third-party requirement and checksum that `go mod tidy` derives from the
// module graph. It is computed on a scratch copy; the surface is untouched
// until Write.
type Convergence struct {
	GoModPath string
	// Missing lists in-repo local replaces the converged go.mod adds or
	// repoints; AddRequire marks those whose requirement is new as well.
	Missing []MissingReplace
	// Dropped lists in-repo local replaces the converged go.mod removes
	// because no requirement uses them any longer.
	Dropped      []string
	Requirements []RequirementChange
	// Evidence lists directories (relative to the module) holding Go packages
	// under a docs/ tree without their own module boundary. Such packages
	// join this module, so Converge computes nothing and the surface must be
	// fenced first (docs-evidence convention).
	Evidence    []string
	GoModBefore string
	GoModAfter  string
	GoSumBefore string
	GoSumAfter  string
	// SumAdded counts checksum lines present after but not before. A missing
	// checksum fails the build just like a missing requirement.
	SumAdded int
	// vendor is the re-vendored tree for a module with a committed vendor/
	// directory, when it differs from the current tree.
	vendor *vendorStage
}

// VendorChanged reports whether the module's committed vendor/ tree would be
// re-vendored.
func (c *Convergence) VendorChanged() bool { return c != nil && c.vendor != nil }

// VendorModules returns vendor/modules.txt before and after re-vendoring.
func (c *Convergence) VendorModules() (before, after string) {
	if !c.VendorChanged() {
		return "", ""
	}
	return c.vendor.modulesTxtBefore, c.vendor.modulesTxtAfter
}

// VendorPreserved lists top-level vendor/ entries the go command does not own
// and that re-vendoring carries over unchanged.
func (c *Convergence) VendorPreserved() []string {
	if !c.VendorChanged() {
		return nil
	}
	return c.vendor.preserved
}

// Close releases the staged vendor tree, if any. Callers must Close every
// Convergence; Write consumes the staged tree but Close is still safe.
func (c *Convergence) Close() {
	if c != nil {
		c.vendor.close()
		c.vendor = nil
	}
}

// ModChanged reports whether go.mod would change.
func (c *Convergence) ModChanged() bool { return c != nil && c.GoModBefore != c.GoModAfter }

// SumChanged reports whether go.sum would change.
func (c *Convergence) SumChanged() bool { return c != nil && c.GoSumBefore != c.GoSumAfter }

// Changed reports whether the surface is not converged.
func (c *Convergence) Changed() bool { return c.ModChanged() || c.SumChanged() || c.VendorChanged() }

// Breaking reports whether the unconverged surface fails `go build`: a
// missing local replace, requirement, or checksum. The remaining drift
// (unused requirements, stale checksums, ordering, indirect markers) is
// housekeeping that builds tolerate.
func (c *Convergence) Breaking() bool {
	if c == nil {
		return false
	}
	if len(c.Missing) > 0 || c.SumAdded > 0 {
		return true
	}
	for _, change := range c.Requirements {
		if change.To != "" && change.From != change.To {
			return true
		}
	}
	return false
}

// Recorded returns the requirement changes a dependency gate must examine.
func (c *Convergence) Recorded() []RequirementChange {
	if c == nil {
		return nil
	}
	var out []RequirementChange
	for _, change := range c.Requirements {
		if change.Recorded() {
			out = append(out, change)
		}
	}
	return out
}

// Converge computes the converged go.mod and go.sum for one surface without
// modifying it. It copies both files to a scratch directory and runs the Go
// toolchain against the copy with -modfile, so relative replaces and source
// imports still resolve from the surface directory.
//
// Local replaces end up exactly matching what Go loads. Converge first seeds a
// replace for every in-repo module reachable through in-repo requirements, so
// tidy never tries to download an unpublished module, and tidies to a fixpoint
// (tidy can surface new in-repo requirements). It then drops every in-repo
// local replace whose module the tidied go.mod no longer requires: under graph
// pruning Go loads only modules the main go.mod lists, so those replaces are
// dead. A final tidy verifies the pruned state; if it fails, the seeded state
// is kept. go.sum is put in canonical order. The result is exactly what Write
// would install.
func Converge(ctx context.Context, goModPath string, topo Topology) (*Convergence, error) {
	moduleDir := filepath.Dir(goModPath)
	modBefore, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, err
	}
	sumBefore, err := readOptional(filepath.Join(moduleDir, "go.sum"))
	if err != nil {
		return nil, err
	}
	evidence, err := UnfencedEvidence(moduleDir, true)
	if err != nil {
		return nil, err
	}
	if len(evidence) > 0 {
		return &Convergence{GoModPath: goModPath, Evidence: evidence, GoModBefore: string(modBefore), GoModAfter: string(modBefore), GoSumBefore: string(sumBefore), GoSumAfter: string(sumBefore)}, nil
	}
	scratch, err := os.MkdirTemp("", "gomodreconcile-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(scratch)
	scratchMod := filepath.Join(scratch, "go.mod")
	scratchSum := filepath.Join(scratch, "go.sum")
	if err := os.WriteFile(scratchMod, modBefore, 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(scratchSum, sumBefore, 0o644); err != nil {
		return nil, err
	}
	tidy := func() error {
		return runGo(ctx, moduleDir, "mod", "tidy", "-modfile="+scratchMod)
	}

	const maxIters = 8
	for i := 0; i < maxIters; i++ {
		edits, err := seedEdits(ctx, scratchMod, moduleDir, topo)
		if err != nil {
			return nil, err
		}
		if len(edits) == 0 {
			break
		}
		if err := runGo(ctx, moduleDir, append(append([]string{"mod", "edit"}, edits...), scratchMod)...); err != nil {
			return nil, fmt.Errorf("go mod edit: %w", err)
		}
		if err := tidy(); err != nil {
			return nil, fmt.Errorf("go mod tidy after adding replaces: %w", err)
		}
	}
	// Always tidy once more: a surface with complete replaces can still lag its
	// in-repo modules' third-party requirements (the class where one new import
	// in a shared package breaks every consumer that lacks the requirement).
	if err := tidy(); err != nil {
		return nil, fmt.Errorf("go mod tidy: %w", err)
	}
	if err := pruneDeadReplaces(ctx, scratchMod, scratchSum, moduleDir, topo, tidy); err != nil {
		return nil, err
	}

	modAfter, err := os.ReadFile(scratchMod)
	if err != nil {
		return nil, err
	}
	sumAfter, err := readOptional(scratchSum)
	if err != nil {
		return nil, err
	}
	// tidy rewrites go.sum only when its checksum set changes, so a correct set
	// in a non-canonical order would stay untidy forever. Canonical order is a
	// pure function of the set.
	sumAfter = canonicalGoSum(sumAfter)
	if len(sumBefore) == 0 && len(sumAfter) == 0 {
		sumAfter = sumBefore
	}

	changes, err := requirementChanges(goModPath, modBefore, modAfter, topo)
	if err != nil {
		return nil, err
	}
	missing, dropped, err := replaceChanges(goModPath, modBefore, modAfter, topo)
	if err != nil {
		return nil, err
	}
	var vendor *vendorStage
	if hasVendorTree(moduleDir) {
		if err := os.WriteFile(scratchSum, sumAfter, 0o644); err != nil {
			return nil, err
		}
		if vendor, err = stageVendor(ctx, moduleDir, scratchMod); err != nil {
			return nil, err
		}
	}
	return &Convergence{
		vendor:       vendor,
		GoModPath:    goModPath,
		Missing:      missing,
		Dropped:      dropped,
		Requirements: changes,
		GoModBefore:  string(modBefore),
		GoModAfter:   string(modAfter),
		GoSumBefore:  string(sumBefore),
		GoSumAfter:   string(sumAfter),
		SumAdded:     len(sumLineDifference(sumAfter, sumBefore)),
	}, nil
}

// seedEdits returns `go mod edit` flags for the plan's missing requires and
// replaces, plus a replace for every in-repo module reachable through the
// in-repo requirements, so tidy resolves them from source.
func seedEdits(ctx context.Context, modFile, moduleDir string, topo Topology) ([]string, error) {
	missing, err := planModFile(ctx, modFile, moduleDir, topo)
	if err != nil {
		return nil, err
	}
	view, err := parseGoMod(ctx, modFile)
	if err != nil {
		return nil, err
	}
	var edits []string
	planned := map[string]struct{}{}
	reach := goModView{Module: view.Module}
	for _, m := range missing {
		if m.AddRequire {
			edits = append(edits, "-require="+m.Module+"@v0.0.0")
		}
		edits = append(edits, "-replace="+m.Module+"="+m.RelPath)
		planned[m.Module] = struct{}{}
		reach.Require = append(reach.Require, struct {
			Path     string
			Version  string
			Indirect bool
		}{Path: m.Module})
	}
	reach.Require = append(reach.Require, view.Require...)
	reachable, err := requiredInRepoModules(ctx, reach, topo)
	if err != nil {
		return nil, err
	}
	local := localReplaceTargets(view, moduleDir)
	for _, module := range reachable {
		if module == view.Module.Path {
			continue
		}
		if _, ok := planned[module]; ok {
			continue
		}
		if target, ok := local[module]; ok && target == filepath.Clean(topo[module]) {
			continue
		}
		rel, err := filepath.Rel(moduleDir, topo[module])
		if err != nil {
			continue
		}
		edits = append(edits, "-replace="+module+"="+localReplacePath(rel))
	}
	return edits, nil
}

// pruneDeadReplaces drops in-repo local replaces whose module the tidied
// go.mod does not require, then verifies with tidy. On failure it restores the
// seeded state, which is known to tidy.
func pruneDeadReplaces(ctx context.Context, modFile, sumFile, moduleDir string, topo Topology, tidy func() error) error {
	view, err := parseGoMod(ctx, modFile)
	if err != nil {
		return err
	}
	required := map[string]struct{}{}
	for _, req := range view.Require {
		required[req.Path] = struct{}{}
	}
	var drops []string
	for module := range localReplaceTargets(view, moduleDir) {
		if _, inRepo := topo[module]; !inRepo {
			continue
		}
		if _, ok := required[module]; ok {
			continue
		}
		drops = append(drops, "-dropreplace="+module)
	}
	if len(drops) == 0 {
		return nil
	}
	sort.Strings(drops)
	modSeeded, err := os.ReadFile(modFile)
	if err != nil {
		return err
	}
	sumSeeded, err := os.ReadFile(sumFile)
	if err != nil {
		return err
	}
	if err := runGo(ctx, moduleDir, append(append([]string{"mod", "edit"}, drops...), modFile)...); err != nil {
		return fmt.Errorf("go mod edit -dropreplace: %w", err)
	}
	restore := func() error {
		if err := os.WriteFile(modFile, modSeeded, 0o644); err != nil {
			return err
		}
		return os.WriteFile(sumFile, sumSeeded, 0o644)
	}
	if err := tidy(); err != nil {
		return restore()
	}
	// A dropped replace must have been dead: if tidy now records a requirement
	// it did not before (for example by resolving a published copy of an
	// in-repo module path), the replace was live, so keep the seeded state.
	pruned, err := parseGoMod(ctx, modFile)
	if err != nil {
		return err
	}
	for _, req := range pruned.Require {
		if _, ok := required[req.Path]; !ok {
			return restore()
		}
	}
	for _, req := range view.Require {
		if !requiresVersion(pruned, req.Path, req.Version) {
			return restore()
		}
	}
	return nil
}

func requiresVersion(view goModView, path, version string) bool {
	for _, req := range view.Require {
		if req.Path == path {
			return req.Version == version
		}
	}
	return false
}

// localReplaceTargets maps each locally replaced module path to its absolute
// target directory.
func localReplaceTargets(view goModView, moduleDir string) map[string]string {
	out := map[string]string{}
	for _, r := range view.Replace {
		if r.New.Version != "" {
			continue
		}
		target := r.New.Path
		if !filepath.IsAbs(target) {
			target = filepath.Join(moduleDir, target)
		}
		out[r.Old.Path] = filepath.Clean(target)
	}
	return out
}

// replaceChanges diffs the in-repo local replaces of two go.mod contents.
func replaceChanges(goModPath string, before, after []byte, topo Topology) (missing []MissingReplace, dropped []string, err error) {
	parse := func(data []byte) (map[string]string, map[string]struct{}, error) {
		file, err := modfile.Parse(goModPath, data, nil)
		if err != nil {
			return nil, nil, err
		}
		replaces := map[string]string{}
		for _, r := range file.Replace {
			if _, inRepo := topo[r.Old.Path]; inRepo && r.New.Version == "" {
				replaces[r.Old.Path] = r.New.Path
			}
		}
		requires := map[string]struct{}{}
		for _, req := range file.Require {
			requires[req.Mod.Path] = struct{}{}
		}
		return replaces, requires, nil
	}
	beforeReplaces, beforeRequires, err := parse(before)
	if err != nil {
		return nil, nil, fmt.Errorf("parse go.mod %s: %w", goModPath, err)
	}
	afterReplaces, _, err := parse(after)
	if err != nil {
		return nil, nil, fmt.Errorf("parse converged go.mod %s: %w", goModPath, err)
	}
	for module, target := range afterReplaces {
		if beforeReplaces[module] == target {
			continue
		}
		_, required := beforeRequires[module]
		missing = append(missing, MissingReplace{Module: module, RelPath: target, AddRequire: !required})
	}
	for module := range beforeReplaces {
		if _, ok := afterReplaces[module]; !ok {
			dropped = append(dropped, module)
		}
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i].Module < missing[j].Module })
	sort.Strings(dropped)
	return missing, dropped, nil
}

// Write installs the converged go.mod and go.sum. It refuses with
// ErrConcurrentEdit when either file changed since Converge read it, and
// replaces each file atomically.
func (c *Convergence) Write() error {
	if !c.Changed() {
		return nil
	}
	sumPath := filepath.Join(filepath.Dir(c.GoModPath), "go.sum")
	modNow, err := os.ReadFile(c.GoModPath)
	if err != nil {
		return err
	}
	sumNow, err := readOptional(sumPath)
	if err != nil {
		return err
	}
	if string(modNow) != c.GoModBefore || string(sumNow) != c.GoSumBefore {
		return ErrConcurrentEdit
	}
	// Swap the vendor tree first: it is the step that can refuse (a
	// concurrent edit), and go.mod/go.sum must never claim a state the
	// vendor tree does not hold.
	if c.VendorChanged() {
		if err := c.vendor.swapIn(filepath.Dir(c.GoModPath)); err != nil {
			return err
		}
	}
	if c.ModChanged() {
		if err := replaceFile(c.GoModPath, c.GoModAfter); err != nil {
			return err
		}
	}
	if c.SumChanged() {
		if err := replaceFile(sumPath, c.GoSumAfter); err != nil {
			return err
		}
	}
	return nil
}

func replaceFile(path, content string) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".reconcile-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func readOptional(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

type requirementState struct {
	version string
	direct  bool
}

// requirementChanges diffs the third-party requirements of two go.mod
// contents. Local-directory replacements identify in-repo or vendored-source
// modules whose version string carries no dependency decision.
func requirementChanges(goModPath string, before, after []byte, topo Topology) ([]RequirementChange, error) {
	beforeFile, err := modfile.Parse(goModPath, before, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.mod %s: %w", goModPath, err)
	}
	afterFile, err := modfile.Parse(goModPath, after, nil)
	if err != nil {
		return nil, fmt.Errorf("parse converged go.mod %s: %w", goModPath, err)
	}
	local := map[string]struct{}{}
	for _, file := range []*modfile.File{beforeFile, afterFile} {
		for _, replacement := range file.Replace {
			if replacement.New.Version == "" {
				local[replacement.Old.Path] = struct{}{}
			}
		}
	}
	collect := func(file *modfile.File) map[string]requirementState {
		out := map[string]requirementState{}
		for _, req := range file.Require {
			if _, inRepo := topo[req.Mod.Path]; inRepo {
				continue
			}
			if _, replaced := local[req.Mod.Path]; replaced {
				continue
			}
			out[req.Mod.Path] = requirementState{version: req.Mod.Version, direct: !req.Indirect}
		}
		return out
	}
	was, now := collect(beforeFile), collect(afterFile)
	paths := map[string]struct{}{}
	for path := range was {
		paths[path] = struct{}{}
	}
	for path := range now {
		paths[path] = struct{}{}
	}
	var changes []RequirementChange
	for path := range paths {
		b, inBefore := was[path]
		a, inAfter := now[path]
		if inBefore && inAfter && b == a {
			continue
		}
		changes = append(changes, RequirementChange{Path: path, From: b.version, To: a.version, FromDirect: inBefore && b.direct, Direct: inAfter && a.direct})
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	return changes, nil
}

// canonicalGoSum orders go.sum lines exactly as the go command writes them:
// modules sorted by path then semantic version (module.Sort), hashes sorted
// within each module version, duplicates and blank lines dropped.
func canonicalGoSum(data []byte) []byte {
	hashes := map[module.Version][]string{}
	var mods []module.Version
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			if strings.TrimSpace(line) != "" {
				// Not a go.sum line: leave the file as tidy produced it.
				return data
			}
			continue
		}
		mod := module.Version{Path: fields[0], Version: fields[1]}
		if _, seen := hashes[mod]; !seen {
			mods = append(mods, mod)
		}
		hashes[mod] = append(hashes[mod], fields[2])
	}
	module.Sort(mods)
	var b strings.Builder
	for _, mod := range mods {
		list := hashes[mod]
		sort.Strings(list)
		var previous string
		for _, hash := range list {
			if hash == previous {
				continue
			}
			previous = hash
			fmt.Fprintf(&b, "%s %s %s\n", mod.Path, mod.Version, hash)
		}
	}
	return []byte(b.String())
}

// sumLineDifference returns the non-blank lines in a that are not in b.
func sumLineDifference(a, b []byte) []string {
	inB := map[string]struct{}{}
	for _, line := range strings.Split(string(b), "\n") {
		inB[strings.TrimSpace(line)] = struct{}{}
	}
	var out []string
	for _, line := range strings.Split(string(a), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, ok := inB[line]; !ok {
			out = append(out, line)
		}
	}
	return out
}

func runGo(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = envkit.Toolchain(reconcileGoEnv(envkit.Env(os.Environ())), reconcileToolchain)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// reconcileToolchain appends -mod=mod through the toolchain overlay so the
// inherited GOFLAGS (and its width) survive; nothing assigns GOFLAGS.
var reconcileToolchain = envkit.ToolchainOptions{GoFlags: []string{"-mod=mod"}}

// reconcileGoEnv switches the workspace off; the spawn site composes the
// build width and -mod=mod through envkit.Toolchain.
func reconcileGoEnv(parent envkit.Env) envkit.Env {
	return envkit.WithOverlay(parent, envkit.SameScenario, envkit.Env{"GOWORK=off"})
}
