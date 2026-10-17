package signals

import (
	"context"
	"sync"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

// GraphContext bundles the inputs every Signal needs plus the per-run
// shared caches (community detection, glossary lookups). The aggregator
// builds the context once per scoring batch; signals receive a value
// copy with shared pointers to immutable caches.
//
// GraphContext is created via NewGraphContext so callers can't construct
// half-populated contexts. The embedded *Caches is goroutine-safe so the
// same context can be passed to a worker pool batch-scoring many chunks
// concurrently.
type GraphContext struct {
	Scenario string
	Snapshot graph.GraphSnapshot
	// DomainMap is the derived domain map for the scenario (replaces the
	// deleted per-scenario architecture manifest). Signals read declared
	// domains, owned paths, and glossary vocabulary from it.
	DomainMap domains.DerivedDomainMap
	Caches    *Caches
}

// Caches is the shared cache surface for one scoring batch. Access is
// goroutine-safe: ScoreBatch shares one *Caches across workers.
type Caches struct {
	mu          sync.RWMutex
	gitCoEditMu sync.Mutex
	// community is the per-package Louvain community id used by the
	// import-cluster signal. Written once (idempotently) and read by every
	// subsequent score.
	community map[string]int
	// domainPackages maps package_id -> domain_name for the derived domain
	// map. Several signals need this same index, so it is computed once per
	// scoring batch and cached here.
	domainPackages map[string]string
	// filePackages maps file_id -> package_id for package lookup hot paths.
	filePackages map[string]string
	// packageIDs is the set of package ids in the snapshot.
	packageIDs map[string]struct{}
	// exportedSymbolsByFile maps file_id -> exported symbol names.
	exportedSymbolsByFile map[string][]string
	// packageImporters maps package_id -> importing package ids.
	packageImporters map[string][]string
	// packageImportingTests maps package_id -> importing test files.
	packageImportingTests map[string][]graph.FileNode
	// gitCoEdit maps file path -> parsed git commits touching that path.
	gitCoEdit any
}

// CommunitySnapshot returns the current community map, or nil if the
// cache has not been populated yet. Returns a defensive copy so callers
// can iterate without holding the lock.
func (c *Caches) CommunitySnapshot() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.community) == 0 {
		return nil
	}
	out := make(map[string]int, len(c.community))
	for k, v := range c.community {
		out[k] = v
	}
	return out
}

// SetCommunity replaces the community cache atomically. Subsequent
// calls are no-ops if the cache is already populated (the value is
// idempotent — same graph in, same clusters out).
func (c *Caches) SetCommunity(in map[string]int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.community) > 0 {
		return
	}
	c.community = make(map[string]int, len(in))
	for k, v := range in {
		c.community[k] = v
	}
}

// DomainPackagesSnapshot returns the current package-to-domain index, or nil
// if the cache has not been populated yet. Returns a defensive copy.
func (c *Caches) DomainPackagesSnapshot() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.domainPackages) == 0 {
		return nil
	}
	out := make(map[string]string, len(c.domainPackages))
	for k, v := range c.domainPackages {
		out[k] = v
	}
	return out
}

// SetDomainPackages replaces the package-to-domain cache atomically.
// Subsequent calls are no-ops if the cache is already populated.
func (c *Caches) SetDomainPackages(in map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.domainPackages) > 0 {
		return
	}
	c.domainPackages = make(map[string]string, len(in))
	for k, v := range in {
		c.domainPackages[k] = v
	}
}

// FilePackagesSnapshot returns the cached file->package and package-id indexes.
func (c *Caches) FilePackagesSnapshot() (map[string]string, map[string]struct{}) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.filePackages) == 0 && len(c.packageIDs) == 0 {
		return nil, nil
	}
	files := make(map[string]string, len(c.filePackages))
	for k, v := range c.filePackages {
		files[k] = v
	}
	pkgs := make(map[string]struct{}, len(c.packageIDs))
	for k := range c.packageIDs {
		pkgs[k] = struct{}{}
	}
	return files, pkgs
}

// SetFilePackages stores the file->package and package-id indexes.
func (c *Caches) SetFilePackages(files map[string]string, pkgs map[string]struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.filePackages) > 0 || len(c.packageIDs) > 0 {
		return
	}
	c.filePackages = make(map[string]string, len(files))
	for k, v := range files {
		c.filePackages[k] = v
	}
	c.packageIDs = make(map[string]struct{}, len(pkgs))
	for k := range pkgs {
		c.packageIDs[k] = struct{}{}
	}
}

// ExportedSymbolsByFileSnapshot returns exported symbols grouped by file id.
func (c *Caches) ExportedSymbolsByFileSnapshot() map[string][]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.exportedSymbolsByFile) == 0 {
		return nil
	}
	out := make(map[string][]string, len(c.exportedSymbolsByFile))
	for k, v := range c.exportedSymbolsByFile {
		out[k] = append([]string(nil), v...)
	}
	return out
}

// SetExportedSymbolsByFile stores exported symbols grouped by file id.
func (c *Caches) SetExportedSymbolsByFile(in map[string][]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.exportedSymbolsByFile) > 0 {
		return
	}
	c.exportedSymbolsByFile = make(map[string][]string, len(in))
	for k, v := range in {
		c.exportedSymbolsByFile[k] = append([]string(nil), v...)
	}
}

// PackageImportersSnapshot returns package importer ids grouped by package id.
func (c *Caches) PackageImportersSnapshot() map[string][]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.packageImporters) == 0 {
		return nil
	}
	out := make(map[string][]string, len(c.packageImporters))
	for k, v := range c.packageImporters {
		out[k] = append([]string(nil), v...)
	}
	return out
}

// SetPackageImporters stores package importer ids grouped by package id.
func (c *Caches) SetPackageImporters(in map[string][]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.packageImporters) > 0 {
		return
	}
	c.packageImporters = make(map[string][]string, len(in))
	for k, v := range in {
		c.packageImporters[k] = append([]string(nil), v...)
	}
}

// PackageImportingTestsSnapshot returns importing test files grouped by package id.
func (c *Caches) PackageImportingTestsSnapshot() map[string][]graph.FileNode {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.packageImportingTests) == 0 {
		return nil
	}
	out := make(map[string][]graph.FileNode, len(c.packageImportingTests))
	for k, v := range c.packageImportingTests {
		out[k] = append([]graph.FileNode(nil), v...)
	}
	return out
}

// SetPackageImportingTests stores importing test files grouped by package id.
func (c *Caches) SetPackageImportingTests(in map[string][]graph.FileNode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.packageImportingTests) > 0 {
		return
	}
	c.packageImportingTests = make(map[string][]graph.FileNode, len(in))
	for k, v := range in {
		c.packageImportingTests[k] = append([]graph.FileNode(nil), v...)
	}
}

// GitCoEditSnapshot returns a signal-owned git co-edit cache value.
func (c *Caches) GitCoEditSnapshot() any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.gitCoEdit
}

// SetGitCoEdit stores a signal-owned git co-edit cache value once.
func (c *Caches) SetGitCoEdit(in any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.gitCoEdit != nil {
		return
	}
	c.gitCoEdit = in
}

// GitCoEditGetOrCompute serializes the first expensive git history read for a
// scoring context. It stores exactly one value and returns it to all callers.
func (c *Caches) GitCoEditGetOrCompute(ctx context.Context, compute func(context.Context) (any, error)) (any, error) {
	if cached := c.GitCoEditSnapshot(); cached != nil {
		return cached, nil
	}
	c.gitCoEditMu.Lock()
	defer c.gitCoEditMu.Unlock()
	if cached := c.GitCoEditSnapshot(); cached != nil {
		return cached, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	value, err := compute(ctx)
	if err != nil {
		return nil, err
	}
	c.SetGitCoEdit(value)
	return value, nil
}

// NewGraphContext constructs a fresh context with an empty Caches.
func NewGraphContext(scenario string, snap graph.GraphSnapshot, m domains.DerivedDomainMap) GraphContext {
	return GraphContext{
		Scenario:  scenario,
		Snapshot:  snap,
		DomainMap: m,
		Caches:    &Caches{},
	}
}
