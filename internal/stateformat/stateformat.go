// Package stateformat defines the persistent-state compatibility model for the
// root control plane.
//
// A store describes one persisted format family: the version this executable
// writes, the versions it can read, the oldest version whose data the current
// write version remains readable as, and the canonical schema digest of every
// version the executable knows.
//
// Compatibility is directional. The rollback image must be able to read what
// the candidate writes. Identity equality is the degenerate case and is not
// the rule.
//
// This package is deliberately self-contained. It does not scan source, hash
// packages, or depend on the legacy statecontract inventory. Descriptors are
// generated from compiled types and SQL DDL by the owning package.
package stateformat

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
)

// ManifestFormat is the wire format of Manifest. A reader that does not know a
// manifest format cannot prove compatibility and must treat it as breaking.
const ManifestFormat = 1

// Version is one store format version. Versions are positive and increase
// monotonically within a store.
type Version int

// Store is one persisted format family as compiled into an executable.
type Store struct {
	// Name is the stable store identity. It is not a package path.
	Name string `json:"name"`
	// Write is the format this executable writes.
	Write Version `json:"write"`
	// Reads is every format this executable decodes into the same shapes it
	// writes. Every read version has a schema digest. It must include Write.
	Reads []Version `json:"reads"`
	// MigratesFrom is the oldest format this executable upgrades in place. It
	// is separate from Reads because upgrading changes data: a v1 database is
	// migrated to Write, not decoded as Write. Zero means no claim.
	MigratesFrom Version `json:"migrates_from,omitempty"`
	// AdditiveFrom is the oldest version such that every transition from it
	// through Write is additive: a reader of AdditiveFrom can read Write data
	// without losing the fields it consumes. Zero means no additive guarantee.
	// The claim is earned by downlevel fixtures, not asserted.
	AdditiveFrom Version `json:"additive_from,omitempty"`
	// Schemas maps every format this executable knows to its canonical schema
	// digest. A version known to both images must have the same digest.
	Schemas map[Version]string `json:"schemas"`
	// StatePaths lists the files, relative to the runtime home, that hold this
	// store's data. A forced transition snapshots exactly these so a rollback
	// restores the bytes as well as the image. A store with no declared paths
	// cannot be recovered by a forced transition, only by an operator.
	StatePaths []string `json:"state_paths,omitempty"`
}

// Manifest is the complete set of stores compiled into one executable.
type Manifest struct {
	Format int     `json:"format"`
	Stores []Store `json:"stores"`
	// Digest covers Format and Stores. It is a convenience identity, never the
	// compatibility decision.
	Digest string `json:"digest"`
}

// Verdict is the result of comparing two manifests.
type Verdict string

const (
	// Compatible means the prior image can read everything the candidate
	// writes and the candidate can read what already exists.
	Compatible Verdict = "compatible"
	// Breaking means automatic rollback cannot be relied on. Either the prior
	// cannot read the candidate's writes or the candidate cannot read the
	// serving data.
	Breaking Verdict = "breaking"
	// Unknown means the comparison could not be made. Callers must treat it as
	// breaking.
	Unknown Verdict = "unknown"
)

// Change names one difference that matters to the verdict.
type Change struct {
	Store  string `json:"store"`
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

// Change kinds. They are stable identifiers for receipts and diagnostics.
const (
	ChangeFormat              = "format"
	ChangeUnqualified         = "unqualified"
	ChangeAdded               = "added-store"
	ChangeRemoved             = "removed-store"
	ChangeSchema              = "schema-mismatch"
	ChangePriorUnreadable     = "prior-cannot-read-candidate"
	ChangeCandidateUnreadable = "candidate-cannot-read-serving"
)

// Assessment is the full result, including informational changes.
type Assessment struct {
	Verdict Verdict  `json:"verdict"`
	Changes []Change `json:"changes,omitempty"`
}

// AllowsAutomaticRollback reports whether ordinary activation may proceed.
func (a Assessment) AllowsAutomaticRollback() bool { return a.Verdict == Compatible }

// Seal validates the manifest and fills Digest.
func (m *Manifest) Seal() error {
	m.Digest = ""
	if err := m.validate(false); err != nil {
		return err
	}
	digest, err := canonicalDigest(struct {
		Format int     `json:"format"`
		Stores []Store `json:"stores"`
	}{m.Format, m.Stores})
	if err != nil {
		return err
	}
	m.Digest = digest
	return nil
}

// Validate checks structure and the self-digest.
func (m Manifest) Validate() error { return m.validate(true) }

func (m Manifest) validate(requireDigest bool) error {
	if m.Format != ManifestFormat {
		return fmt.Errorf("state-format manifest v%d is not v%d", m.Format, ManifestFormat)
	}
	if len(m.Stores) == 0 {
		return fmt.Errorf("state-format manifest declares no stores")
	}
	previous := ""
	for i, store := range m.Stores {
		if i > 0 && previous >= store.Name {
			return fmt.Errorf("state-format stores are not canonical at %q", store.Name)
		}
		previous = store.Name
		if err := store.validate(); err != nil {
			return fmt.Errorf("store %q: %w", store.Name, err)
		}
	}
	if requireDigest {
		expected := m
		if err := expected.Seal(); err != nil {
			return err
		}
		if m.Digest != expected.Digest {
			return fmt.Errorf("state-format manifest does not match its digest")
		}
	}
	return nil
}

func (s Store) validate() error {
	if s.Name == "" {
		return fmt.Errorf("name is empty")
	}
	if s.Write <= 0 {
		return fmt.Errorf("write version must be positive")
	}
	if len(s.Reads) == 0 {
		return fmt.Errorf("reads is empty")
	}
	seen := map[Version]bool{}
	sorted := true
	for i, v := range s.Reads {
		if v <= 0 {
			return fmt.Errorf("read version must be positive")
		}
		if seen[v] {
			return fmt.Errorf("read version %d is duplicated", v)
		}
		seen[v] = true
		if i > 0 && s.Reads[i-1] > v {
			sorted = false
		}
	}
	if !sorted {
		return fmt.Errorf("reads is not sorted")
	}
	if !seen[s.Write] {
		return fmt.Errorf("reads does not include the write version %d", s.Write)
	}
	if s.AdditiveFrom < 0 || s.AdditiveFrom > s.Write {
		return fmt.Errorf("additive_from %d is outside 0..%d", s.AdditiveFrom, s.Write)
	}
	if s.MigratesFrom < 0 || s.MigratesFrom > s.Write {
		return fmt.Errorf("migrates_from %d is outside 0..%d", s.MigratesFrom, s.Write)
	}
	if len(s.Schemas) == 0 {
		return fmt.Errorf("schemas is empty")
	}
	for v, digest := range s.Schemas {
		if v <= 0 {
			return fmt.Errorf("schema version must be positive")
		}
		if !isDigest(digest) {
			return fmt.Errorf("schema digest for v%d is not a hex SHA-256", v)
		}
		if !seen[v] {
			return fmt.Errorf("schemas declares v%d but reads does not; a known schema must be a read version", v)
		}
	}
	if _, ok := s.Schemas[s.Write]; !ok {
		return fmt.Errorf("schemas has no digest for the write version %d", s.Write)
	}
	seenPaths := map[string]bool{}
	for _, p := range s.StatePaths {
		if p == "" || path.IsAbs(p) || path.Clean(p) != p || strings.HasPrefix(p, "..") {
			return fmt.Errorf("state path %q must be a clean relative path inside the runtime home", p)
		}
		if seenPaths[p] {
			return fmt.Errorf("state path %q is duplicated", p)
		}
		seenPaths[p] = true
	}
	return nil
}

// Check compares a serving manifest with a candidate manifest.
//
// The prior is the rollback image. The candidate is the image that would
// activate. Compatible means automatic rollback is valid.
func Check(prior, candidate Manifest) Assessment {
	if err := prior.Validate(); err != nil {
		return Assessment{Verdict: Unknown, Changes: []Change{{Kind: ChangeUnqualified, Detail: "serving manifest: " + err.Error()}}}
	}
	if err := candidate.Validate(); err != nil {
		return Assessment{Verdict: Unknown, Changes: []Change{{Kind: ChangeUnqualified, Detail: "candidate manifest: " + err.Error()}}}
	}
	priorStores := storeIndex(prior.Stores)
	candidateStores := storeIndex(candidate.Stores)
	changes := make([]Change, 0)
	breaking := false

	add := func(store, kind, detail string, breaks bool) {
		changes = append(changes, Change{Store: store, Kind: kind, Detail: detail})
		if breaks {
			breaking = true
		}
	}

	for _, name := range sortedStoreNames(candidateStores) {
		c := candidateStores[name]
		p, exists := priorStores[name]
		if !exists {
			// A store the serving image never reads cannot make its rollback
			// unsafe. New tables, files, and namespaces are inert to it.
			add(name, ChangeAdded, fmt.Sprintf("new store at v%d", c.Write), false)
			continue
		}
		// A candidate must be able to read or upgrade the data it inherits, or
		// it cannot start on this host at all.
		if !containsVersion(c.Reads, p.Write) && !(c.MigratesFrom > 0 && p.Write >= c.MigratesFrom) {
			add(name, ChangeCandidateUnreadable, fmt.Sprintf("candidate reads %v and migrates from v%d, so it cannot open the serving v%d", c.Reads, c.MigratesFrom, p.Write), true)
		}
		// Shared versions must mean the same thing on both sides. A changed
		// digest without a version bump is retroactive reinterpretation.
		for _, v := range sharedVersions(p.Schemas, c.Schemas) {
			if p.Schemas[v] != c.Schemas[v] {
				add(name, ChangeSchema, fmt.Sprintf("v%d schema digest changed without a version bump", v), true)
			}
		}
		if containsVersion(p.Reads, c.Write) {
			continue
		}
		if c.AdditiveFrom > 0 && p.Write >= c.AdditiveFrom && c.Write >= p.Write {
			continue
		}
		add(name, ChangePriorUnreadable, fmt.Sprintf("serving reads %v; candidate writes v%d with additive_from v%d", p.Reads, c.Write, c.AdditiveFrom), true)
	}

	for _, name := range sortedStoreNames(priorStores) {
		if _, exists := candidateStores[name]; exists {
			continue
		}
		add(name, ChangeRemoved, "store is absent from the candidate; a rollback may still read it", true)
	}

	verdict := Compatible
	if breaking {
		verdict = Breaking
	}
	return Assessment{Verdict: verdict, Changes: changes}
}

// Opaque models an unmigrated owner as a single-version identity store. Any
// change to its digest is breaking, which preserves the old safety envelope
// while stores are migrated one at a time. Delete opaque entries at cutover.
func Opaque(name, digest string) (Store, error) {
	if isDigest(digest) {
		return Store{Name: name, Write: 1, Reads: []Version{1}, Schemas: map[Version]string{1: digest}}, nil
	}
	// Accept arbitrary canonical JSON such as a legacy owner digest by
	// hashing it, so callers do not need to know the digest form.
	sum := sha256.Sum256([]byte(digest))
	return Store{Name: name, Write: 1, Reads: []Version{1}, Schemas: map[Version]string{1: hex.EncodeToString(sum[:])}}, nil
}

// DigestSchema returns the canonical digest of a schema value. Maps are
// encoded with sorted keys, so the result is stable across machines.
func DigestSchema(value any) (string, error) { return canonicalDigest(value) }

// SortStores orders stores by name after validating them.
func SortStores(stores []Store) error {
	seen := map[string]bool{}
	for _, store := range stores {
		if seen[store.Name] {
			return fmt.Errorf("store %q is duplicated", store.Name)
		}
		seen[store.Name] = true
		if err := store.validate(); err != nil {
			return fmt.Errorf("store %q: %w", store.Name, err)
		}
	}
	sort.Slice(stores, func(i, j int) bool { return stores[i].Name < stores[j].Name })
	return nil
}

func canonicalDigest(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func storeIndex(stores []Store) map[string]Store {
	index := make(map[string]Store, len(stores))
	for _, store := range stores {
		index[store.Name] = store
	}
	return index
}

func sortedStoreNames(index map[string]Store) []string {
	names := make([]string, 0, len(index))
	for name := range index {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sharedVersions(a, b map[Version]string) []Version {
	versions := make([]Version, 0)
	for v := range a {
		if _, ok := b[v]; ok {
			versions = append(versions, v)
		}
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })
	return versions
}

func containsVersion(versions []Version, want Version) bool {
	for _, v := range versions {
		if v == want {
			return true
		}
	}
	return false
}

func isDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
