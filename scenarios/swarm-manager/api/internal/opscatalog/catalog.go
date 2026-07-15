// Package opscatalog loads and indexes the authored data documents of the
// declarative agent-operations layer — versioned operation contracts, system
// binding defaults, transition policies, and the target-capability registry —
// from the on-disk catalog beside the scenario (operation-contracts/, bindings/,
// policy/). It is the read side that the operation runner consumes.
//
// The loader fails CLOSED at startup: a document that does not parse, does not
// pass its agentops schema + semantic validator, collides on identity, or
// conflicts on version aborts the whole load with an actionable error naming the
// offending file. A half-valid catalog can never be served, because a runner
// resolving a binding against a partially-loaded catalog would silently pick a
// stale or wrong mode. Every loaded document carries a pinned content revision
// (canonical digest) so the runner can record exactly which authored bytes
// decided a run.
package opscatalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"swarm-manager/internal/agentops"
)

// Standard catalog sub-directory names. The scenario ships these beside modes/.
const (
	DirOperationContracts = "operation-contracts"
	DirBindings           = "bindings"
	DirPolicy             = "policy"
)

// ErrCatalogEmpty is returned by Load when the catalog root exists but declares
// no operation contracts at all. An empty catalog is treated as a fail-closed
// configuration error rather than a silently runnable no-op.
var ErrCatalogEmpty = errors.New("operation catalog declares no operation contracts")

// ContractKey identifies one exact operation-contract revision.
type ContractKey struct {
	ID      agentops.OperationID
	Version string
}

func (k ContractKey) String() string { return string(k.ID) + "@" + k.Version }

// LoadedContract is an authored operation contract plus its pinned revision.
type LoadedContract struct {
	Contract agentops.OperationContract
	Revision string // canonical sha256 digest of the authored document
	Source   string // absolute path the document was loaded from
}

// LoadedBinding is an authored system-default binding plus provenance.
type LoadedBinding struct {
	Binding  agentops.OperationBinding
	Revision string
	Source   string
}

// LoadedPolicy is an authored transition policy plus its pinned revision. The
// revision is what an execution pins as policy_revision.
type LoadedPolicy struct {
	Policy   agentops.TransitionPolicy
	Revision string
	Source   string
}

// Catalog is the immutable, validated index of the authored agent-operations
// data. Construct it with Load; all lookups are read-only and concurrency-safe.
type Catalog struct {
	contracts  map[ContractKey]LoadedContract
	latest     map[agentops.OperationID]string // id -> highest authored version
	bindings   []LoadedBinding
	policies   map[string]LoadedPolicy // policy id -> policy
	targetCaps map[agentops.TargetKind]agentops.TargetCapabilityDescriptor
}

// Load reads and validates the whole catalog rooted at dir. Missing sub-
// directories are tolerated only for bindings/ and policy/ (a fresh scenario may
// ship contracts before any override or policy); a missing or empty
// operation-contracts/ is a fail-closed error. The target-capability registry is
// the agentops Go SSOT, not a loaded document, so it is always fully populated.
func Load(dir string) (*Catalog, error) {
	c := &Catalog{
		contracts:  map[ContractKey]LoadedContract{},
		latest:     map[agentops.OperationID]string{},
		policies:   map[string]LoadedPolicy{},
		targetCaps: map[agentops.TargetKind]agentops.TargetCapabilityDescriptor{},
	}
	for _, d := range agentops.TargetCapabilities() {
		c.targetCaps[d.TargetKind] = d
	}
	if err := c.loadContracts(filepath.Join(dir, DirOperationContracts)); err != nil {
		return nil, err
	}
	if len(c.contracts) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrCatalogEmpty, filepath.Join(dir, DirOperationContracts))
	}
	if err := c.loadBindings(filepath.Join(dir, DirBindings)); err != nil {
		return nil, err
	}
	if err := c.loadPolicies(filepath.Join(dir, DirPolicy)); err != nil {
		return nil, err
	}
	return c, nil
}

// jsonFiles returns the sorted set of *.json files directly or recursively under
// dir. A missing dir yields (nil, nil) so optional sub-directories are tolerated
// by callers that choose to.
func jsonFiles(dir string) ([]string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("catalog path %s is not a directory", dir)
	}
	var out []string
	walkErr := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".json") && !strings.HasPrefix(d.Name(), ".") {
			out = append(out, path)
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	sort.Strings(out)
	return out, nil
}

func (c *Catalog) loadContracts(dir string) error {
	files, err := jsonFiles(dir)
	if err != nil {
		return fmt.Errorf("scan operation contracts: %w", err)
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read operation contract %s: %w", f, err)
		}
		if err := agentops.ValidateOperationContract(raw); err != nil {
			return fmt.Errorf("operation contract %s is invalid: %w", f, err)
		}
		oc, err := agentops.DecodeOperationContract(raw)
		if err != nil {
			return fmt.Errorf("decode operation contract %s: %w", f, err)
		}
		rev, err := agentops.CanonicalDigest(raw)
		if err != nil {
			return fmt.Errorf("digest operation contract %s: %w", f, err)
		}
		key := ContractKey{ID: oc.ID, Version: oc.Version}
		if existing, dup := c.contracts[key]; dup {
			// Same id+version from two files: a hard conflict unless the bytes are
			// byte-identical (harmless re-declaration), which we still reject to
			// keep exactly one source of truth per revision.
			return fmt.Errorf("operation contract %s conflicts: %s and %s both declare %s", key, existing.Source, f, key)
		}
		c.contracts[key] = LoadedContract{Contract: oc, Revision: rev, Source: f}
		if cur, ok := c.latest[oc.ID]; !ok || semverLess(cur, oc.Version) {
			c.latest[oc.ID] = oc.Version
		}
	}
	return nil
}

func (c *Catalog) loadBindings(dir string) error {
	files, err := jsonFiles(dir)
	if err != nil {
		return fmt.Errorf("scan bindings: %w", err)
	}
	// A system binding document may declare only system-default layer bindings;
	// override bindings (item/initiative/invocation) live in domain storage, not
	// the shipped catalog. Reject an override in the catalog dir fail-closed.
	seen := map[string]string{} // operation+version -> source, for duplicate detection
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read binding %s: %w", f, err)
		}
		if err := agentops.ValidateBinding(raw); err != nil {
			return fmt.Errorf("binding %s is invalid: %w", f, err)
		}
		b, err := agentops.DecodeBinding(raw)
		if err != nil {
			return fmt.Errorf("decode binding %s: %w", f, err)
		}
		if b.Layer != agentops.LayerSystemDefault {
			return fmt.Errorf("binding %s declares %s layer: only system-default bindings belong in the shipped catalog (overrides live in domain storage)", f, b.Layer)
		}
		if _, ok := c.contracts[latestOrExact(c, b.Operation, b.OperationVersion)]; !ok {
			return fmt.Errorf("binding %s references operation %s that the catalog does not declare", f, b.Operation)
		}
		dupKey := string(b.Operation) + "@" + b.OperationVersion
		if prev, ok := seen[dupKey]; ok {
			return fmt.Errorf("two system-default bindings for operation %q: %s and %s", dupKey, prev, f)
		}
		seen[dupKey] = f
		rev, err := agentops.CanonicalDigest(raw)
		if err != nil {
			return fmt.Errorf("digest binding %s: %w", f, err)
		}
		c.bindings = append(c.bindings, LoadedBinding{Binding: b, Revision: rev, Source: f})
	}
	return nil
}

func (c *Catalog) loadPolicies(dir string) error {
	files, err := jsonFiles(dir)
	if err != nil {
		return fmt.Errorf("scan policies: %w", err)
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read policy %s: %w", f, err)
		}
		if err := agentops.ValidateTransitionPolicy(raw); err != nil {
			return fmt.Errorf("transition policy %s is invalid: %w", f, err)
		}
		p, err := agentops.DecodeTransitionPolicy(raw)
		if err != nil {
			return fmt.Errorf("decode transition policy %s: %w", f, err)
		}
		if existing, dup := c.policies[p.ID]; dup {
			return fmt.Errorf("transition policy id %q conflicts: %s and %s", p.ID, existing.Source, f)
		}
		rev, err := agentops.CanonicalDigest(raw)
		if err != nil {
			return fmt.Errorf("digest policy %s: %w", f, err)
		}
		c.policies[p.ID] = LoadedPolicy{Policy: p, Revision: rev, Source: f}
	}
	return nil
}

func latestOrExact(c *Catalog, id agentops.OperationID, version string) ContractKey {
	if version != "" {
		return ContractKey{ID: id, Version: version}
	}
	return ContractKey{ID: id, Version: c.latest[id]}
}

// Contract returns the authored contract for id at an exact version, or the
// highest authored version when version is empty. ok=false when absent.
func (c *Catalog) Contract(id agentops.OperationID, version string) (LoadedContract, bool) {
	lc, ok := c.contracts[latestOrExact(c, id, version)]
	return lc, ok
}

// Contracts returns every loaded contract, ordered by id then version.
func (c *Catalog) Contracts() []LoadedContract {
	out := make([]LoadedContract, 0, len(c.contracts))
	for _, lc := range c.contracts {
		out = append(out, lc)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Contract.ID != out[j].Contract.ID {
			return out[i].Contract.ID < out[j].Contract.ID
		}
		return semverLess(out[i].Contract.Version, out[j].Contract.Version)
	})
	return out
}

// SystemBindings returns the authored system-default bindings.
func (c *Catalog) SystemBindings() []LoadedBinding {
	return append([]LoadedBinding(nil), c.bindings...)
}

// SystemBindingFor returns the system-default binding for an operation, if one
// is authored. Bindings pinned to an exact operation_version win over version-
// agnostic ones for that operation.
func (c *Catalog) SystemBindingFor(op agentops.OperationID, version string) (LoadedBinding, bool) {
	var agnostic *LoadedBinding
	for i := range c.bindings {
		b := c.bindings[i]
		if b.Binding.Operation != op {
			continue
		}
		if b.Binding.OperationVersion == version && version != "" {
			return b, true
		}
		if b.Binding.OperationVersion == "" {
			agnostic = &c.bindings[i]
		}
	}
	if agnostic != nil {
		return *agnostic, true
	}
	return LoadedBinding{}, false
}

// Policy returns the transition policy with the given id.
func (c *Catalog) Policy(id string) (LoadedPolicy, bool) {
	p, ok := c.policies[id]
	return p, ok
}

// Policies returns every loaded policy, ordered by id.
func (c *Catalog) Policies() []LoadedPolicy {
	out := make([]LoadedPolicy, 0, len(c.policies))
	for _, p := range c.policies {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Policy.ID < out[j].Policy.ID })
	return out
}

// PolicyForDomain returns the single transition policy governing a domain kind,
// or ok=false when none (or ambiguously more than one) is authored. Resolution
// is fail-closed: two policies claiming the same domain is an authoring error
// the runner must not paper over by picking arbitrarily.
func (c *Catalog) PolicyForDomain(domainKind string) (LoadedPolicy, bool) {
	var found *LoadedPolicy
	for id := range c.policies {
		p := c.policies[id]
		if p.Policy.DomainKind != domainKind {
			continue
		}
		if found != nil {
			return LoadedPolicy{}, false
		}
		lp := p
		found = &lp
	}
	if found == nil {
		return LoadedPolicy{}, false
	}
	return *found, true
}

// TargetCapability returns the capability descriptor for a target kind.
func (c *Catalog) TargetCapability(kind agentops.TargetKind) (agentops.TargetCapabilityDescriptor, bool) {
	d, ok := c.targetCaps[kind]
	return d, ok
}

// semverLess compares two dotted numeric versions; malformed segments sort as 0.
func semverLess(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var ai, bi int
		if i < len(as) {
			ai = atoiSafe(as[i])
		}
		if i < len(bs) {
			bi = atoiSafe(bs[i])
		}
		if ai != bi {
			return ai < bi
		}
	}
	return false
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return n
		}
		n = n*10 + int(r-'0')
	}
	return n
}
