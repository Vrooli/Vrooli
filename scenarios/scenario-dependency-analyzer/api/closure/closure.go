// Package closure owns source-distribution closure analysis for consumers
// such as scenario-to-repository. It does not assemble archives or publish;
// it produces the authoritative inventory and explicit unresolved obligations.
package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	resourcemanifest "github.com/vrooli/vrooli/internal/resources/manifest"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

type File struct {
	SourcePath string   `json:"sourcePath"`
	ExportPath string   `json:"exportPath"`
	SHA256     string   `json:"sha256"`
	Mode       uint32   `json:"mode"`
	SizeBytes  int64    `json:"sizeBytes"`
	Reasons    []string `json:"reasonRefs"`
}

type Node struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

type Rewrite struct {
	Kind        string `json:"kind"`
	ProposalRef string `json:"proposalRef"`
}

// Target identifies the platform for which a shipment is being resolved.
type Target struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
}

type InclusionReason struct {
	Capability string `json:"capability"`
	Parent     string `json:"parent,omitempty"`
	Relation   string `json:"relation"`
}

type Item struct {
	ID             string            `json:"id"`
	Kind           string            `json:"kind"`
	Name           string            `json:"name"`
	Version        string            `json:"version,omitempty"`
	Constraints    []string          `json:"constraints,omitempty"`
	Required       bool              `json:"required"`
	Reasons        []InclusionReason `json:"reasons"`
	PlatformStatus string            `json:"platformStatus,omitempty"`
}

type Footprint struct {
	Bytes              int64    `json:"bytes"`
	RequiredTools      []string `json:"requiredTools,omitempty"`
	RequiredSafeguards []string `json:"requiredSafeguards,omitempty"`
	Privileges         []string `json:"privileges,omitempty"`
}

type SourceClosure struct {
	SourceDigest  string    `json:"sourceDigest"`
	Scenario      string    `json:"scenario"`
	Selections    []string  `json:"selections,omitempty"`
	Target        Target    `json:"target,omitempty"`
	Nodes         []Node    `json:"nodes"`
	Items         []Item    `json:"items,omitempty"`
	Files         []File    `json:"files"`
	Rewrites      []Rewrite `json:"rewrites"`
	Unresolved    []string  `json:"unresolved"`
	Complete      bool      `json:"complete"`
	Footprint     Footprint `json:"footprint,omitempty"`
	ClosureDigest string    `json:"closureDigest"`
}

type Resolver struct{}

func NewResolver() Resolver { return Resolver{} }

// Resolve retains the original source-root behavior. It is used for a single
// scenario source tree and intentionally does not infer a repository-wide
// dependency closure.
func (Resolver) Resolve(root, scenario, sourceDigest string) (SourceClosure, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return SourceClosure{}, fmt.Errorf("resolve source root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return SourceClosure{}, fmt.Errorf("source root: %w", err)
	}
	if !info.IsDir() {
		return SourceClosure{}, fmt.Errorf("source root is not a directory")
	}
	result := SourceClosure{SourceDigest: sourceDigest, Scenario: scenario, Nodes: []Node{{ID: "scenario:" + scenario, Kind: "scenario_source"}}}
	if err := appendTree(&result, root, "", "edge:root"); err != nil {
		return SourceClosure{}, err
	}
	result.Nodes = append(result.Nodes, Node{ID: "runtime:declared", Kind: "runtime_requirement"})
	result.Complete = true
	if err := finalize(&result); err != nil {
		return SourceClosure{}, err
	}
	return result, nil
}

// ResolveSelection computes the canonical union for selected scenarios. The
// resolver reads the canonical service and resource manifest models, walks
// dependencies transitively, and emits one item per compatible artifact with
// every inclusion reason retained. Input order is deliberately not part of
// the result, so callers can compare closure digests safely.
func (Resolver) ResolveSelection(root string, selections []string, target Target) (SourceClosure, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return SourceClosure{}, fmt.Errorf("resolve repository root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return SourceClosure{}, fmt.Errorf("repository root: %w", err)
	}
	if !info.IsDir() {
		return SourceClosure{}, fmt.Errorf("repository root is not a directory")
	}
	target.OS = normalizeOS(target.OS)
	target.Architecture = strings.ToLower(strings.TrimSpace(target.Architecture))
	if target.OS == "" {
		return SourceClosure{}, fmt.Errorf("closure target OS is required")
	}
	selected := uniqueSorted(selections)
	if len(selected) == 0 {
		return SourceClosure{}, fmt.Errorf("at least one scenario selection is required")
	}
	result := SourceClosure{Scenario: selected[0], Selections: selected, Target: target}
	state := selectionState{root: root, target: target, result: &result, items: map[string]*Item{}, visiting: map[string]bool{}, visited: map[string]bool{}, resourceVisiting: map[string]bool{}, resourceVisited: map[string]bool{}, files: map[string]struct{}{}}
	for _, name := range selected {
		if err := state.visitScenario(name, name, "selected"); err != nil {
			return SourceClosure{}, err
		}
	}
	result.Items = sortedItems(state.items)
	result.Nodes = make([]Node, 0, len(result.Items)+1)
	for _, item := range result.Items {
		result.Nodes = append(result.Nodes, Node{ID: item.ID, Kind: item.Kind})
	}
	result.Nodes = append(result.Nodes, Node{ID: "runtime:declared", Kind: "runtime_requirement"})
	result.Complete = true
	if err := finalize(&result); err != nil {
		return SourceClosure{}, err
	}
	return result, nil
}

type selectionState struct {
	root             string
	target           Target
	result           *SourceClosure
	items            map[string]*Item
	visiting         map[string]bool
	visited          map[string]bool
	resourceVisiting map[string]bool
	resourceVisited  map[string]bool
	resourceStack    []string
	stack            []string
	files            map[string]struct{}
}

func (s *selectionState) visitScenario(name, capability, relation string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("scenario dependency declared by %q has an empty name", capability)
	}
	if s.visiting[name] {
		start := 0
		for i, value := range s.stack {
			if value == name {
				start = i
				break
			}
		}
		cycle := append(append([]string(nil), s.stack[start:]...), name)
		return fmt.Errorf("scenario dependency cycle detected: %s", strings.Join(cycle, " -> "))
	}
	path := filepath.Join(s.root, "scenarios", name, ".vrooli", "service.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("missing required scenario declaration %q (chain %s): %w", name, strings.Join(append(append([]string(nil), s.stack...), name), " -> "), err)
	}
	var manifest scenariomodel.ServiceManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return fmt.Errorf("decode scenario declaration %q: %w", name, err)
	}
	if s.visited[name] {
		s.addItem("scenario", name, manifest.Version, "", capability, parentOf(s.stack), relation, true, "")
		return nil
	}
	s.addItem("scenario", name, manifest.Version, "", capability, parentOf(s.stack), relation, true, "")
	if err := s.appendFiles(filepath.Join(s.root, "scenarios", name), filepath.Join("scenarios", name), "scenario:"+name); err != nil {
		return err
	}
	if s.visited[name] {
		return nil
	}
	s.visiting[name] = true
	s.stack = append(s.stack, name)
	defer func() {
		delete(s.visiting, name)
		s.visited[name] = true
		s.stack = s.stack[:len(s.stack)-1]
	}()

	for _, depName := range sortedDependencyMap(manifest.Dependencies.Scenarios) {
		dep := manifest.Dependencies.Scenarios[depName]
		if !dep.Required && !dep.Enabled {
			continue
		}
		if err := s.visitScenario(depName, capability, "scenario_dependency"); err != nil {
			return err
		}
		s.addReason("scenario:"+depName, capability, name, "scenario_dependency", dep.Required)
	}
	for _, depName := range sortedDependencyMap(manifest.Dependencies.Resources) {
		dep := manifest.Dependencies.Resources[depName]
		if !dep.Required && !dep.Enabled {
			continue
		}
		if err := s.visitResource(depName, dep.VersionRange, dep.Required, capability, name); err != nil {
			return err
		}
	}
	for _, declaration := range manifest.HostTools {
		if err := s.addHostRequirement("host_tool", declaration.Name, declaration.Required, capability, name, filepath.Join("internal", "tools", declaration.Name, "tool.json"), string(declaration.DerivePrivilege(s.target.OS))); err != nil {
			return err
		}
	}
	for _, declaration := range manifest.HostSafeguards {
		if err := s.addHostRequirement("host_safeguard", declaration.Name, declaration.Required, capability, name, filepath.Join("internal", "safeguards", declaration.Name, "safeguard.json"), string(declaration.DerivePrivilege(s.target.OS))); err != nil {
			return err
		}
	}
	for _, credential := range manifest.Credentials.All() {
		// The closure contains only the opaque address. Values never enter the
		// manifest, digest, or shipment inventory.
		id := strings.TrimSpace(credential.LogicalID) + ":" + credential.ResolvedField()
		s.addItem("credential_ref", id, "", "", capability, name, "credential_reference", credential.Required, "")
	}
	return nil
}

func (s *selectionState) visitResource(name, constraint string, required bool, capability, parent string) error {
	name = strings.TrimSpace(name)
	path := filepath.Join(s.root, "resources", name, "resource.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("missing required resource declaration %q (chain %s -> resource:%s): %w", name, strings.Join(s.stack, " -> "), name, err)
	}
	var manifest resourcemanifest.ResourceManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return fmt.Errorf("decode resource declaration %q: %w", name, err)
	}
	status, compatible := resourcePlatformStatus(raw, s.target)
	if !compatible {
		return fmt.Errorf("resource %q is incompatible with %s/%s (chain %s -> resource:%s): %s", name, s.target.OS, s.target.Architecture, strings.Join(s.stack, " -> "), name, status)
	}
	if s.resourceVisiting[name] {
		start := 0
		for i, value := range s.resourceStack {
			if value == name {
				start = i
				break
			}
		}
		cycle := append(append([]string(nil), s.resourceStack[start:]...), name)
		return fmt.Errorf("resource dependency cycle detected: %s", strings.Join(cycle, " -> "))
	}
	if _, ok := s.items["resource:"+name]; !ok {
		s.addItem("resource", name, "", constraint, capability, parent, "resource_dependency", required, status)
	} else if err := s.mergeConstraint("resource:"+name, constraint, capability, parent); err != nil {
		return err
	} else {
		s.addItem("resource", name, "", constraint, capability, parent, "resource_dependency", required, status)
	}
	if s.resourceVisited[name] {
		return nil
	}
	if err := s.addFile(path, filepath.Join("resources", name, "resource.json"), "resource:"+name); err != nil {
		return err
	}
	s.resourceVisiting[name] = true
	s.resourceStack = append(s.resourceStack, name)
	defer func() {
		delete(s.resourceVisiting, name)
		s.resourceVisited[name] = true
		s.resourceStack = s.resourceStack[:len(s.resourceStack)-1]
	}()
	for _, child := range uniqueSorted(manifest.Dependencies) {
		if err := s.visitResource(child, "", true, capability, name); err != nil {
			return err
		}
	}
	for _, declaration := range manifest.HostTools {
		if err := s.addHostRequirement("host_tool", declaration.Name, declaration.Required, capability, name, filepath.Join("internal", "tools", declaration.Name, "tool.json"), string(declaration.DerivePrivilege(s.target.OS))); err != nil {
			return err
		}
	}
	for _, declaration := range manifest.HostSafeguards {
		if err := s.addHostRequirement("host_safeguard", declaration.Name, declaration.Required, capability, name, filepath.Join("internal", "safeguards", declaration.Name, "safeguard.json"), string(declaration.DerivePrivilege(s.target.OS))); err != nil {
			return err
		}
	}
	return nil
}

func (s *selectionState) addHostRequirement(kind, name string, required bool, capability, parent, catalog, privilege string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("%s declared by %q has no name", kind, parent)
	}
	if required {
		if _, err := os.Stat(filepath.Join(s.root, catalog)); err != nil {
			return fmt.Errorf("missing required %s declaration %q (chain %s -> %s:%s): %w", kind, name, strings.Join(s.stack, " -> "), kind, name, err)
		}
	}
	s.addItem(kind, name, "", "", capability, parent, "host_requirement", required, "")
	if _, err := os.Stat(filepath.Join(s.root, catalog)); err == nil {
		if err := s.addFile(filepath.Join(s.root, catalog), catalog, kind+":"+name); err != nil {
			return err
		}
	}
	if kind == "host_tool" && required {
		s.result.Footprint.RequiredTools = appendUnique(s.result.Footprint.RequiredTools, name)
	}
	if kind == "host_safeguard" && required {
		s.result.Footprint.RequiredSafeguards = appendUnique(s.result.Footprint.RequiredSafeguards, name)
	}
	if required && strings.TrimSpace(privilege) != "" {
		s.result.Footprint.Privileges = appendUnique(s.result.Footprint.Privileges, privilege)
	}
	return nil
}

func (s *selectionState) addItem(kind, name, version, constraint, capability, parent, relation string, required bool, platformStatus string) {
	id := kind + ":" + name
	item, ok := s.items[id]
	if !ok {
		item = &Item{ID: id, Kind: kind, Name: name, Version: version, PlatformStatus: platformStatus}
		s.items[id] = item
	}
	item.Required = item.Required || required
	if item.Version == "" {
		item.Version = version
	}
	if item.PlatformStatus == "" {
		item.PlatformStatus = platformStatus
	}
	if constraint != "" && !contains(item.Constraints, constraint) {
		item.Constraints = append(item.Constraints, constraint)
		sort.Strings(item.Constraints)
	}
	s.addReason(id, capability, parent, relation, required)
}

func (s *selectionState) addReason(id, capability, parent, relation string, required bool) {
	item, ok := s.items[id]
	if !ok {
		return
	}
	reason := InclusionReason{Capability: capability, Parent: parent, Relation: relation}
	for _, existing := range item.Reasons {
		if existing == reason {
			return
		}
	}
	item.Reasons = append(item.Reasons, reason)
	sort.Slice(item.Reasons, func(i, j int) bool {
		if item.Reasons[i].Capability != item.Reasons[j].Capability {
			return item.Reasons[i].Capability < item.Reasons[j].Capability
		}
		if item.Reasons[i].Parent != item.Reasons[j].Parent {
			return item.Reasons[i].Parent < item.Reasons[j].Parent
		}
		return item.Reasons[i].Relation < item.Reasons[j].Relation
	})
	item.Required = item.Required || required
}

func (s *selectionState) mergeConstraint(id, constraint, capability, parent string) error {
	if strings.TrimSpace(constraint) == "" {
		return nil
	}
	item := s.items[id]
	if item == nil {
		return nil
	}
	for _, existing := range item.Constraints {
		if !constraintsIntersect(existing, constraint) {
			return fmt.Errorf("incompatible version constraints for %s: %q from %s and %q from %s", id, existing, reasonParent(item), constraint, parentOrCapability(capability, parent))
		}
	}
	return nil
}

func (s *selectionState) appendFiles(root, exportRoot, reason string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if entry.IsDir() {
			if rel == ".git" || rel == "node_modules" || rel == "dist" || rel == "build" {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			inside, err := filepath.Rel(root, target)
			if err != nil || inside == ".." || strings.HasPrefix(filepath.ToSlash(inside), "../") {
				return fmt.Errorf("symlink escapes source root: %s", filepath.ToSlash(filepath.Join(exportRoot, rel)))
			}
			return nil
		}
		return s.addFile(path, filepath.ToSlash(filepath.Join(exportRoot, rel)), reason)
	})
}

func (s *selectionState) addFile(path, exportPath, reason string) error {
	if privatePath(exportPath) {
		return nil
	}
	if _, exists := s.files[exportPath]; exists {
		for i := range s.result.Files {
			if s.result.Files[i].ExportPath == exportPath {
				s.result.Files[i].Reasons = appendUnique(s.result.Files[i].Reasons, reason)
				return nil
			}
		}
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("unsupported source entry: %s", exportPath)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(data)
	s.result.Files = append(s.result.Files, File{SourcePath: filepath.ToSlash(exportPath), ExportPath: filepath.ToSlash(exportPath), SHA256: hex.EncodeToString(hash[:]), Mode: uint32(info.Mode().Perm()), SizeBytes: int64(len(data)), Reasons: []string{reason}})
	s.files[exportPath] = struct{}{}
	s.result.Footprint.Bytes += int64(len(data))
	return nil
}

func appendTree(result *SourceClosure, root, exportRoot, reason string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
			if part == ".git" || part == "node_modules" || part == "dist" || part == "build" {
				if entry.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}
		if entry.Type()&os.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			inside, err := filepath.Rel(root, target)
			if err != nil || inside == ".." || strings.HasPrefix(filepath.ToSlash(inside), "../") {
				return fmt.Errorf("symlink escapes source root: %s", rel)
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported source entry: %s", rel)
		}
		if privatePath(rel) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(data)
		result.Files = append(result.Files, File{SourcePath: filepath.ToSlash(rel), ExportPath: filepath.ToSlash(filepath.Join(exportRoot, rel)), SHA256: hex.EncodeToString(hash[:]), Mode: uint32(entry.Type().Perm()), SizeBytes: int64(len(data)), Reasons: []string{reason}})
		return nil
	})
}

func finalize(result *SourceClosure) error {
	sort.Slice(result.Files, func(i, j int) bool { return result.Files[i].ExportPath < result.Files[j].ExportPath })
	for i := range result.Files {
		result.Files[i].Reasons = uniqueSorted(result.Files[i].Reasons)
	}
	result.Footprint.RequiredTools = uniqueSorted(result.Footprint.RequiredTools)
	result.Footprint.RequiredSafeguards = uniqueSorted(result.Footprint.RequiredSafeguards)
	result.Footprint.Privileges = uniqueSorted(result.Footprint.Privileges)
	canonical, err := json.Marshal(struct {
		Selections []string  `json:"selections"`
		Target     Target    `json:"target"`
		Items      []Item    `json:"items"`
		Files      []File    `json:"files"`
		Footprint  Footprint `json:"footprint"`
		Unresolved []string  `json:"unresolved"`
	}{result.Selections, result.Target, result.Items, result.Files, result.Footprint, result.Unresolved})
	if err != nil {
		return err
	}
	digest := sha256.Sum256(canonical)
	result.ClosureDigest = "sha256:" + hex.EncodeToString(digest[:])
	return nil
}

func sortedItems(items map[string]*Item) []Item {
	result := make([]Item, 0, len(items))
	for _, item := range items {
		item.Constraints = uniqueSorted(item.Constraints)
		sort.Slice(item.Reasons, func(i, j int) bool {
			if item.Reasons[i].Capability != item.Reasons[j].Capability {
				return item.Reasons[i].Capability < item.Reasons[j].Capability
			}
			if item.Reasons[i].Parent != item.Reasons[j].Parent {
				return item.Reasons[i].Parent < item.Reasons[j].Parent
			}
			return item.Reasons[i].Relation < item.Reasons[j].Relation
		})
		result = append(result, *item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func resourcePlatformStatus(raw []byte, target Target) (string, bool) {
	var wrapper struct {
		Platforms map[string]string `json:"platforms"`
	}
	if json.Unmarshal(raw, &wrapper) != nil || len(wrapper.Platforms) == 0 {
		return "no platform restriction", true
	}
	keys := []string{target.OS}
	if target.Architecture != "" {
		keys = append([]string{target.OS + "-" + target.Architecture, target.OS + "+" + target.Architecture}, keys...)
	}
	for _, key := range keys {
		if status, ok := wrapper.Platforms[key]; ok {
			status = strings.ToLower(strings.TrimSpace(status))
			// A partial declaration is an unresolved shipment obligation, not a
			// compatible artifact. Treating it as complete would let activation
			// claim a verified footprint while silently relying on an undeclared
			// fallback or host-side acquisition.
			return status, status == "supported" || status == "build-verified"
		}
	}
	return "manifest has no supported entry for target", false
}

func normalizeOS(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "darwin", "mac", "macos":
		return "macos"
	case "win", "win32", "windows":
		return "windows"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func sortedDependencyMap[T any](values map[string]T) []string {
	result := make([]string, 0, len(values))
	for name := range values {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func uniqueSorted(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func appendUnique(values []string, value string) []string { return uniqueSorted(append(values, value)) }

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func parentOf(stack []string) string {
	if len(stack) == 0 {
		return ""
	}
	return stack[len(stack)-1]
}

func parentOrCapability(capability, parent string) string {
	if parent != "" {
		return parent
	}
	return capability
}

func reasonParent(item *Item) string {
	for _, reason := range item.Reasons {
		if reason.Parent != "" {
			return reason.Parent
		}
	}
	return "prior declaration"
}

type (
	semver       struct{ major, minor, patch int }
	versionRange struct {
		lower      *semver
		upper      *semver
		lowerEqual bool
		upperEqual bool
	}
)

func constraintsIntersect(left, right string) bool {
	a, ok := parseRange(left)
	if !ok {
		return false
	}
	b, ok := parseRange(right)
	if !ok {
		return false
	}
	if a.lower != nil && b.upper != nil && compare(*a.lower, *b.upper) > 0 {
		return false
	}
	if b.lower != nil && a.upper != nil && compare(*b.lower, *a.upper) > 0 {
		return false
	}
	if a.lower != nil && b.upper != nil && compare(*a.lower, *b.upper) == 0 && (!a.lowerEqual || !b.upperEqual) {
		return false
	}
	if b.lower != nil && a.upper != nil && compare(*b.lower, *a.upper) == 0 && (!b.lowerEqual || !a.upperEqual) {
		return false
	}
	return true
}

func parseRange(value string) (versionRange, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "*" {
		return versionRange{}, true
	}
	result := versionRange{}
	for _, token := range strings.Fields(strings.ReplaceAll(value, ",", " ")) {
		operator := "="
		version := token
		for _, candidate := range []string{">=", "<=", ">", "<", "=", "^", "~"} {
			if strings.HasPrefix(token, candidate) {
				operator, version = candidate, strings.TrimSpace(strings.TrimPrefix(token, candidate))
				break
			}
		}
		parsed, ok := parseVersion(version)
		if !ok {
			return versionRange{}, false
		}
		switch operator {
		case "=":
			result.lower, result.upper, result.lowerEqual, result.upperEqual = &parsed, &parsed, true, true
		case ">=":
			result.lower, result.lowerEqual = &parsed, true
		case ">":
			result.lower, result.lowerEqual = &parsed, false
		case "<=":
			result.upper, result.upperEqual = &parsed, true
		case "<":
			result.upper, result.upperEqual = &parsed, false
		case "^":
			upper := semver{major: parsed.major + 1}
			result.lower, result.upper, result.lowerEqual, result.upperEqual = &parsed, &upper, true, false
		case "~":
			upper := semver{major: parsed.major, minor: parsed.minor + 1}
			result.lower, result.upper, result.lowerEqual, result.upperEqual = &parsed, &upper, true, false
		}
	}
	return result, true
}

func parseVersion(value string) (semver, bool) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	parts := strings.Split(value, ".")
	if len(parts) > 3 {
		return semver{}, false
	}
	values := [3]int{}
	for i, part := range parts {
		if part == "x" || part == "X" || part == "*" {
			return semver{}, false
		}
		number, err := strconv.Atoi(part)
		if err != nil {
			return semver{}, false
		}
		values[i] = number
	}
	return semver{major: values[0], minor: values[1], patch: values[2]}, true
}

func compare(a, b semver) int {
	if a.major != b.major {
		if a.major < b.major {
			return -1
		}
		return 1
	}
	if a.minor != b.minor {
		if a.minor < b.minor {
			return -1
		}
		return 1
	}
	if a.patch < b.patch {
		return -1
	}
	if a.patch > b.patch {
		return 1
	}
	return 0
}

func privatePath(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.HasPrefix(base, ".env") || strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".key") || strings.HasSuffix(base, ".p12") || strings.Contains(filepath.ToSlash(path), ".vrooli/plan-artifacts/")
}
