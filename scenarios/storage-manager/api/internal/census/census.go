// Package census measures the storage surface without mutating it.
package census

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
	corestorage "github.com/vrooli/api-core/storage"
)

//go:embed schema.sql
var censusSchemaSQL string

type Entry struct {
	Owner       string      `json:"owner"`
	Kind        string      `json:"kind,omitempty"`
	Name        string      `json:"name"`
	Path        string      `json:"path"`
	Bytes       int64       `json:"bytes"`
	Declared    bool        `json:"declared"`
	Disposition Disposition `json:"disposition"`
}

type Finding struct {
	Code        string      `json:"code"`
	Severity    string      `json:"severity"`
	Owner       string      `json:"owner,omitempty"`
	Kind        string      `json:"kind,omitempty"`
	Path        string      `json:"path,omitempty"`
	Message     string      `json:"message"`
	Disposition Disposition `json:"disposition"`
}

// Disposition is the accounting state of bytes or an observation. Unknown
// and inaccessible data is deliberately non-reclaimable until an owner proves
// otherwise.
type Disposition string

const (
	DispositionOwnedManaged      Disposition = "owned_managed"
	DispositionOwnedProtected    Disposition = "owned_protected"
	DispositionOwnedUnreadable   Disposition = "owned_unreadable"
	DispositionDeclaredExternal  Disposition = "declared_external"
	DispositionUnownedReadable   Disposition = "unowned_readable"
	DispositionUnreadableUnknown Disposition = "unreadable_unknown"
	DispositionAggregateDrift    Disposition = "aggregate_drift"
)

type Report struct {
	SnapshotID               string                `json:"snapshot_id,omitempty"`
	ObservedAt               time.Time             `json:"observed_at,omitempty"`
	Root                     string                `json:"root"`
	MeasuredBytes            int64                 `json:"measured_bytes"`
	AttributedBytes          int64                 `json:"attributed_bytes"`
	DriftBytes               int64                 `json:"drift_bytes"`
	UnattributedBytes        int64                 `json:"unattributed_bytes"`
	UnattributedKnown        bool                  `json:"-"`
	UnattributedRoots        []RootTotal           `json:"unattributed_roots,omitempty"`
	AccountingResidualBytes  int64                 `json:"accounting_residual_bytes,omitempty"`
	AccountingToleranceBytes int64                 `json:"accounting_tolerance_bytes"`
	UnreadableBytes          int64                 `json:"unreadable_bytes,omitempty"`
	Closed                   bool                  `json:"closed"`
	AccountingIdentity       bool                  `json:"accounting_identity"`
	Confidence               string                `json:"confidence"`
	ScanCoverage             ScanCoverage          `json:"scan_coverage"`
	GrowthSlopeBytesPerHour  *float64              `json:"growth_slope_bytes_per_hour,omitempty"`
	OwnerCounts              map[string]int        `json:"owner_counts,omitempty"`
	UnreadablePaths          []string              `json:"unreadable_paths,omitempty"`
	Findings                 []Finding             `json:"findings,omitempty"`
	Entries                  []Entry               `json:"entries"`
	DispositionBytes         map[Disposition]int64 `json:"disposition_bytes,omitempty"`
	FrameworkRoots           []string              `json:"framework_roots,omitempty"`
	ScanPolicy               ScanPolicy            `json:"scan_policy"`
	SnapshotAgeSeconds       *float64              `json:"snapshot_age_seconds,omitempty"`
	StalenessVerdict         string                `json:"staleness_verdict,omitempty"`
}

// RootTotal is one independently actionable part of the unattributed
// remainder. The roots are a partition: their totals never overlap.
type RootTotal struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// ScanPolicy is deliberately data-backed. Operators can change the scope,
// reporting floor, or exclusion rationale without rebuilding storage-manager.
type ScanPolicy struct {
	Roots      []PolicyRoot      `json:"roots,omitempty"`
	FloorBytes int64             `json:"floor_bytes"`
	Exclusions []PolicyExclusion `json:"exclusions"`
}

type PolicyRoot struct {
	Path   string `json:"path"`
	Reason string `json:"reason,omitempty"`
	// Optional roots are skipped silently when absent. The default class
	// roots are optional because a host that never used a class has no
	// directory for it, and that is not a finding.
	Optional bool `json:"optional,omitempty"`
}

// PolicyExclusion removes a subtree from the walk. Path excludes one absolute
// location; Name excludes every directory with that base name wherever it
// appears (for example node_modules). Both need a reason. Excluded bytes are
// still measured by statfs and stay in the unscanned remainder at the device
// root; the report counts which name exclusions were hit so the remainder is
// explainable.
type PolicyExclusion struct {
	Path   string `json:"path,omitempty"`
	Name   string `json:"name,omitempty"`
	Reason string `json:"reason"`
}

const defaultCensusFloorBytes = 1 << 20

type ScanCoverage struct {
	MeasuredBytes    int64  `json:"measured_bytes"`
	ScannedBytes     int64  `json:"scanned_bytes,omitempty"`
	DeviceUsedBytes  int64  `json:"device_used_bytes,omitempty"`
	DeviceTotalBytes int64  `json:"device_total_bytes,omitempty"`
	Complete         bool   `json:"complete"`
	MeasuredByDevice bool   `json:"measured_by_device"`
	PrivilegeLevel   string `json:"privilege_level,omitempty"`
	DegradedReason   string `json:"degraded_reason,omitempty"`
	// ScannedRoots are the trees the walk actually descended into. Bytes on
	// the device outside them are measured by statfs only and reported as the
	// remainder at the device root.
	ScannedRoots []string `json:"scanned_roots,omitempty"`
	// PrunedDirectories counts subtrees skipped by name exclusions, keyed by
	// the excluded name, so an operator can explain the unscanned remainder.
	PrunedDirectories map[string]int `json:"pruned_directories,omitempty"`
	// FilesScanned is the number of regular files whose bytes were counted.
	FilesScanned int64 `json:"files_scanned,omitempty"`
}

func (r Report) MarshalJSON() ([]byte, error) {
	type alias Report
	data, err := json.Marshal(alias(r))
	if err != nil {
		return nil, err
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, err
	}
	if !r.UnattributedKnown {
		object["unattributed_bytes"] = nil
	}
	return json.Marshal(object)
}

// UnmarshalJSON restores the internal knowledge bit that MarshalJSON keeps
// out of the public contract. Without this, a persisted report with a real
// unattributed total would be re-encoded as null by status/history handlers
// after a restart, even though the durable payload contained the value.
func (r *Report) UnmarshalJSON(data []byte) error {
	type alias Report
	if err := json.Unmarshal(data, (*alias)(r)); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	r.UnattributedKnown = false
	if raw, ok := fields["unattributed_bytes"]; ok && string(raw) != "null" {
		r.UnattributedKnown = true
	}
	return nil
}

type Declaration struct {
	Name     string
	Path     string
	Budgeted bool
}

// Scan walks root and attributes each file to at most one declaration. It
// never creates, removes, renames, or opens a database for writing.
func Scan(root string, manifests map[string][]Declaration) (Report, error) {
	declarations := make([]resolvedDeclaration, 0)
	owners := make([]string, 0, len(manifests))
	for owner := range manifests {
		owners = append(owners, owner)
	}
	sort.Strings(owners)
	for _, owner := range owners {
		for _, declaration := range manifests[owner] {
			declarations = append(declarations, resolvedDeclaration{owner: owner, name: declaration.Name, path: declaration.Path, class: corestorage.ClassCache})
		}
	}
	return scan(root, declarations, nil)
}

// ScanInventory measures all declarations loaded from scenarios, resources,
// tools, and safeguards. Relative scenario paths are rooted at the scenario;
// other relative paths are rooted beside their native manifest.
func ScanInventory(root string, inventory corestorage.OwnerInventory) (Report, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Report{}, fmt.Errorf("census: resolve root: %w", err)
	}
	policy, err := LoadPolicy(root)
	if err != nil {
		return Report{}, err
	}
	return scanInventory(root, inventory, policy, true, hostFileSystem{}, NewDeviceProbe())
}

// ScanInventoryWithPolicy is the deterministic seam used by acceptance tests
// and by operators who need a one-off policy. Production callers should use
// ScanInventory so the checked-in policy and device root are authoritative.
func ScanInventoryWithPolicy(root string, inventory corestorage.OwnerInventory, policy ScanPolicy) (Report, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Report{}, fmt.Errorf("census: resolve root: %w", err)
	}
	return scanInventory(root, inventory, policy, false, hostFileSystem{}, NewDeviceProbe())
}

func scanInventory(root string, inventory corestorage.OwnerInventory, policy ScanPolicy, deviceScoped bool, filesystem FileSystem, probe DeviceProbe) (Report, error) {
	declarations := make([]resolvedDeclaration, 0)
	ownerRoots := make([]string, 0)
	counts := map[string]int{}
	findings := make([]Finding, 0, len(inventory.Findings))
	for _, finding := range inventory.Findings {
		findings = append(findings, Finding{Code: finding.Code, Severity: finding.Severity, Owner: finding.OwnerID, Kind: string(finding.OwnerKind), Path: finding.ManifestPath, Message: finding.Message})
	}
	for _, owner := range inventory.Owners {
		kind := string(owner.Kind)
		counts[kind]++
		for _, declaration := range owner.StorageEntries {
			path, resolveErr := corestorage.ResolveOwnerStoragePath(root, owner, declaration, corestorage.Platform(runtime.GOOS), corestorage.PlatformSeams{})
			if resolveErr != nil {
				findings = append(findings, Finding{Code: "unresolvable_storage_path", Severity: "error", Owner: owner.ID, Kind: kind, Path: owner.ManifestPath, Message: declaration.Name + ": " + resolveErr.Error()})
				continue
			}
			declarations = append(declarations, resolvedDeclaration{owner: owner.ID, kind: kind, name: declaration.Name, path: path, class: declaration.Class})
		}
		ownerRoots = append(ownerRoots, ownerStorageRoots(root, owner)...)
	}
	orphans := orphanScenarioRoots(root, inventory, filesystem)
	report, err := scanWithPolicyUsing(scanRequest{
		displayRoot:  root,
		policy:       policy,
		declarations: declarations,
		ownerRoots:   ownerRoots,
		orphanRoots:  orphans,
		findings:     findings,
		deviceScoped: deviceScoped,
	}, filesystem, probe)
	if err != nil {
		return Report{}, err
	}
	report.OwnerCounts = counts
	report.FrameworkRoots = existingRoots(ownerRoots)
	return report, nil
}

// orphanScenarioRoots lists scenarios/<name> directories that no owner claims
// and that carry no canonical manifest. Their bytes are summed by the main
// census walk (they are inside the repository root) rather than by a second
// recursive walk of each directory, which used to re-read every orphaned
// node_modules tree on every census.
func orphanScenarioRoots(repoRoot string, inventory corestorage.OwnerInventory, filesystem FileSystem) []string {
	base := filepath.Join(repoRoot, "scenarios")
	owned := make(map[string]bool, len(inventory.Owners))
	for _, owner := range inventory.Owners {
		if owner.Kind == corestorage.OwnerScenario {
			owned[owner.ID] = true
		}
	}
	var out []string
	_ = filesystem.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == base || !entry.IsDir() {
			return nil
		}
		relative, relErr := filepath.Rel(base, path)
		if relErr != nil || filepath.Dir(relative) != "." {
			return fs.SkipDir
		}
		if owned[filepath.Base(path)] {
			return fs.SkipDir
		}
		if _, statErr := filesystem.Stat(filepath.Join(path, ".vrooli", "service.json")); statErr == nil {
			return fs.SkipDir
		}
		out = append(out, filepath.Clean(path))
		return fs.SkipDir
	})
	sort.Strings(out)
	return out
}

type resolvedDeclaration struct {
	owner string
	kind  string
	name  string
	path  string
	class corestorage.Class
}

func scan(root string, declarations []resolvedDeclaration, initialFindings []Finding) (Report, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Report{}, err
	}
	return scanWithPolicyUsing(scanRequest{displayRoot: root, policy: ScanPolicy{Roots: []PolicyRoot{{Path: root}}, FloorBytes: defaultCensusFloorBytes}, declarations: declarations, findings: initialFindings}, hostFileSystem{}, NewDeviceProbe())
}

// ScanWithFileSystem is the deterministic census seam used by platform and
// filesystem tests. It has the same accounting behavior as Scan, but cannot
// touch the host filesystem unless the caller supplies the host implementation.
func ScanWithFileSystem(root string, manifests map[string][]Declaration, fs FileSystem, probe DeviceProbe) (Report, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Report{}, err
	}
	declarations := make([]resolvedDeclaration, 0)
	for owner, entries := range manifests {
		for _, entry := range entries {
			path := entry.Path
			if !filepath.IsAbs(path) {
				path = filepath.Join(absRoot, filepath.FromSlash(strings.TrimPrefix(filepath.ToSlash(path), filepath.ToSlash(root)+"/")))
			}
			declarations = append(declarations, resolvedDeclaration{owner: owner, name: entry.Name, path: path, class: corestorage.ClassCache})
		}
	}
	return scanWithPolicyUsing(scanRequest{displayRoot: absRoot, policy: ScanPolicy{Roots: []PolicyRoot{{Path: absRoot}}, FloorBytes: defaultCensusFloorBytes}, declarations: declarations}, fs, probe)
}

// scanRequest carries one census's inputs. Declarations are attributed,
// ownerRoots turn undeclared bytes into drift, orphanRoots are summed for the
// STORAGE_PATH_ORPHANED finding, and deviceScoped switches the denominator to
// statfs with the mount point as accounting root.
type scanRequest struct {
	displayRoot  string
	policy       ScanPolicy
	declarations []resolvedDeclaration
	ownerRoots   []string
	orphanRoots  []string
	findings     []Finding
	deviceScoped bool
}

func scanWithPolicyUsing(request scanRequest, filesystem FileSystem, probe DeviceProbe) (Report, error) {
	displayRoot, err := filepath.Abs(request.displayRoot)
	if err != nil {
		return Report{}, err
	}
	declarations, ownerRoots, initialFindings, deviceScoped := request.declarations, request.ownerRoots, request.findings, request.deviceScoped
	unreadable := make([]string, 0)
	resolved, err := resolvePolicy(displayRoot, request.policy, deviceScoped, filesystem)
	if err != nil {
		return Report{}, err
	}
	policy, scanRoots, exclusions := resolved.policy, resolved.roots, resolved.exclusions
	optionalRoots := make(map[string]bool, len(policy.Roots))
	for _, root := range policy.Roots {
		if root.Optional {
			optionalRoots[filepath.Clean(root.Path)] = true
		}
	}
	// A declaration outside every policy root must still be measured or its
	// owner would read as empty. Add such paths as extra roots rather than
	// widening the policy to the device.
	if deviceScoped {
		for _, declaration := range declarations {
			path := filepath.Clean(declaration.path)
			if underAny(path, scanRoots) || containsRoot(scanRoots, path) || excluded(path, exclusions) {
				continue
			}
			if _, statErr := filesystem.Stat(path); statErr != nil {
				continue
			}
			scanRoots = append(scanRoots, path)
		}
		scanRoots = outermostRoots(scanRoots)
	}
	accountingRoot := displayRoot
	if deviceScoped {
		accountingRoot = resolved.deviceRoot
	}
	orphanRoots := make([]string, 0, len(request.orphanRoots))
	orphanBytes := make(map[string]int64, len(request.orphanRoots))
	for _, root := range request.orphanRoots {
		orphanRoots = append(orphanRoots, filepath.Clean(root))
	}
	prunedNames := make(map[string]int)
	var filesScanned int64
	seenRoots := make(map[string]struct{}, len(scanRoots))
	// Partitioning by device keeps the hot inode key to one uint64 while still
	// representing the required (device,inode) pair exactly.
	seenInodes := make(map[uint64]map[uint64]struct{})
	findings := append([]Finding(nil), initialFindings...)
	var unreadableBytes int64
	sort.Slice(declarations, func(i, j int) bool {
		if declarations[i].path != declarations[j].path {
			return declarations[i].path < declarations[j].path
		}
		if declarations[i].owner != declarations[j].owner {
			return declarations[i].owner < declarations[j].owner
		}
		return declarations[i].name < declarations[j].name
	})
	declarationByPath := make(map[string][]int, len(declarations))
	for index, declaration := range declarations {
		declarationByPath[filepath.Clean(declaration.path)] = append(declarationByPath[filepath.Clean(declaration.path)], index)
	}
	entries := make([]Entry, len(declarations))
	for i, declaration := range declarations {
		entries[i] = Entry{Owner: declaration.owner, Kind: declaration.kind, Name: declaration.name, Path: declaration.path, Declared: true, Disposition: dispositionForClass(declaration.class)}
	}
	var scanned, attributed, drift int64
	walked := make([]string, 0, len(scanRoots))
	unattributedTree := &unattributedNode{path: accountingRoot, buckets: map[string]int64{}}
	for _, rawRoot := range scanRoots {
		root, absErr := filepath.Abs(rawRoot)
		if absErr != nil {
			return Report{}, absErr
		}
		root = filepath.Clean(root)
		if _, exists := seenRoots[root]; exists {
			continue
		}
		seenRoots[root] = struct{}{}
		if _, statErr := filesystem.Stat(root); statErr != nil {
			if errors.Is(statErr, fs.ErrNotExist) {
				if !optionalRoots[root] {
					findings = append(findings, Finding{Code: "missing_scan_root", Severity: "warning", Path: root, Message: "declared or candidate storage root does not exist"})
				}
				continue
			}
			findings = append(findings, Finding{Code: "unreadable_path", Severity: "error", Path: root, Message: statErr.Error()})
			unreadable = append(unreadable, root)
			continue
		}
		rootMetadata, rootMetaErr := inspectPathWith(filesystem, root)
		if rootMetaErr != nil {
			findings = append(findings, Finding{Code: "unreadable_path", Severity: "error", Path: root, Message: rootMetaErr.Error()})
			unreadable = append(unreadable, root)
			continue
		}
		walked = append(walked, root)
		walkErr := filesystem.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				unreadable = append(unreadable, path)
				if d != nil && !d.IsDir() {
					if info, infoErr := d.Info(); infoErr == nil {
						unreadableBytes += info.Size()
					}
				}
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if excluded(path, exclusions) {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() && path != root {
				if name := d.Name(); policy.prunesName(name) {
					prunedNames[name]++
					return fs.SkipDir
				}
			}
			if d.Type()&fs.ModeSymlink != 0 {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			// DirEntry carries the name and type only; size, allocation,
			// device, and inode need one lstat per entry, which is the
			// minimum for a byte census. d.Info() would issue the same call.
			metadata, metaErr := inspectPathWith(filesystem, path)
			if metaErr != nil {
				unreadable = append(unreadable, path)
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if rootMetadata.device != 0 && metadata.device != 0 && rootMetadata.device != metadata.device {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			// Only a multiply-linked inode can be reached twice. Remembering
			// every inode made the seen set proportional to the file count of
			// the device (gigabytes of resident memory on a 13M-inode host).
			if metadata.identity.valid && metadata.links > 1 {
				inodes := seenInodes[metadata.identity.device]
				if inodes == nil {
					inodes = make(map[uint64]struct{})
					seenInodes[metadata.identity.device] = inodes
				}
				if _, exists := inodes[metadata.identity.inode]; exists {
					return nil
				}
				inodes[metadata.identity.inode] = struct{}{}
			}
			bytes := metadata.bytes
			if deviceScoped {
				bytes = metadata.allocated
			}
			matches := declarationMatches(path, declarationByPath)
			scanned += bytes
			filesScanned++
			if len(matches) > 0 {
				chosen := matches[0]
				entries[chosen].Bytes += bytes
				attributed += bytes
				if len(matches) > 1 {
					for _, index := range matches[1:] {
						findings = append(findings, Finding{Code: "overlap", Severity: "warning", Owner: entries[index].Owner, Kind: entries[index].Kind, Path: path, Message: fmt.Sprintf("file also matched declaration %s/%s", entries[chosen].Owner, entries[chosen].Name)})
					}
				}
				return nil
			}
			if underAny(path, ownerRoots) {
				drift += bytes
				return nil
			}
			for _, orphan := range orphanRoots {
				if isWithin(path, orphan) {
					orphanBytes[orphan] += bytes
					break
				}
			}
			recordUnattributed(unattributedTree, root, filepath.Dir(path), bytes)
			return nil
		})
		if walkErr != nil {
			return Report{}, fmt.Errorf("scan %s: %w", root, walkErr)
		}
	}
	for _, orphan := range orphanRoots {
		if bytes := orphanBytes[orphan]; bytes > 0 {
			findings = append(findings, Finding{Code: "STORAGE_PATH_ORPHANED", Severity: "warning", Kind: string(corestorage.OwnerScenario), Path: orphan, Message: fmt.Sprintf("scenario directory has no canonical .vrooli/service.json; %d bytes are outside the owner inventory", bytes)})
		}
	}
	for _, path := range unreadable {
		findings = append(findings, Finding{Code: "unreadable_path", Severity: "error", Path: path, Message: "census could not read this path"})
	}
	for i := range findings {
		if findings[i].Disposition == "" {
			findings[i].Disposition = dispositionForFinding(findings[i].Code)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Owner != entries[j].Owner {
			return entries[i].Owner < entries[j].Owner
		}
		return entries[i].Name < entries[j].Name
	})
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		if findings[i].Owner != findings[j].Owner {
			return findings[i].Owner < findings[j].Owner
		}
		return findings[i].Path < findings[j].Path
	})
	coverage := deviceCoverageWith(probe, displayRoot, scanned, len(unreadable) == 0)
	coverage.Complete = coverage.Complete && len(unreadable) == 0
	measured := scanned
	if deviceScoped && coverage.MeasuredByDevice {
		measured = coverage.DeviceUsedBytes
	}
	coverage.ScannedBytes = scanned
	coverage.MeasuredBytes = measured
	coverage.ScannedRoots = sortedStrings(walked)
	coverage.FilesScanned = filesScanned
	if len(prunedNames) > 0 {
		coverage.PrunedDirectories = prunedNames
	}
	unattributed := measured - attributed - drift
	deviceResidual := measured - scanned
	// statfs gives a device-wide denominator even when an individual subtree is
	// unreadable. The unreadable paths remain visible, while their bytes stay in
	// the device residual instead of making the accounting answer disappear.
	known := !deviceScoped || coverage.MeasuredByDevice
	tolerance := int64(0)
	if deviceScoped {
		tolerance = 4 << 20
	}
	rootTotals := unattributedRoots(unattributedTree, policy.FloorBytes)
	if deviceResidual > 0 {
		rootTotals = append(rootTotals, RootTotal{Path: accountingRoot, Bytes: deviceResidual})
	}
	rootTotals = mergeRootTotals(rootTotals)
	var reportedRoots int64
	for _, rootTotal := range rootTotals {
		reportedRoots += rootTotal.Bytes
	}
	identityResidual := measured - attributed - drift - reportedRoots
	if identityResidual != 0 {
		findings = append(findings, Finding{Code: "STORAGE_PATH_UNACCOUNTED", Severity: "warning", Path: accountingRoot, Message: fmt.Sprintf("%d measured bytes remain outside the reported accounting roots", identityResidual)})
	}
	identity := known && absInt64(identityResidual) <= tolerance
	confidence := "degraded"
	if identity {
		confidence = "full"
	}
	dispositions := make(map[Disposition]int64)
	for _, entry := range entries {
		dispositions[entry.Disposition] += entry.Bytes
	}
	if drift > 0 {
		dispositions[DispositionAggregateDrift] += drift
	}
	if unattributed > 0 && known {
		dispositions[DispositionUnownedReadable] += unattributed
	}
	if unreadableBytes > 0 {
		dispositions[DispositionUnreadableUnknown] += unreadableBytes
	}
	return Report{Root: accountingRoot, MeasuredBytes: measured, AttributedBytes: attributed, DriftBytes: drift, UnattributedBytes: unattributed, UnattributedKnown: known, UnattributedRoots: rootTotals, AccountingResidualBytes: identityResidual, AccountingToleranceBytes: tolerance, UnreadableBytes: unreadableBytes, Closed: identity, AccountingIdentity: identity, Confidence: confidence, ScanCoverage: coverage, UnreadablePaths: sortedStrings(unreadable), Findings: findings, Entries: entries, DispositionBytes: dispositions, ScanPolicy: policy}, nil
}

func dispositionForClass(class corestorage.Class) Disposition {
	switch class {
	case corestorage.ClassData, corestorage.ClassConfig, corestorage.ClassState:
		return DispositionOwnedProtected
	case corestorage.ClassCache, corestorage.ClassLogs, corestorage.ClassTestRuns:
		return DispositionOwnedManaged
	default:
		return DispositionOwnedManaged
	}
}

func dispositionForFinding(code string) Disposition {
	switch code {
	case "unreadable_path", "unresolvable_storage_path":
		return DispositionUnreadableUnknown
	case "STORAGE_PATH_ORPHANED":
		return DispositionUnownedReadable
	case "STORAGE_PATH_UNACCOUNTED", "overlap":
		return DispositionAggregateDrift
	default:
		return DispositionUnownedReadable
	}
}

func mergeRootTotals(input []RootTotal) []RootTotal {
	byPath := make(map[string]int64, len(input))
	for _, root := range input {
		byPath[root.Path] += root.Bytes
	}
	result := make([]RootTotal, 0, len(byPath))
	for path, bytes := range byPath {
		if bytes > 0 {
			result = append(result, RootTotal{Path: path, Bytes: bytes})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Bytes != result[j].Bytes {
			return result[i].Bytes > result[j].Bytes
		}
		return result[i].Path < result[j].Path
	})
	return result
}

type unattributedNode struct {
	path    string
	buckets map[string]int64
}

const unattributedReportingDepth = 4

// recordUnattributed keeps a bounded reporting partition while the walk is in
// progress. Retaining every directory in a multi-million-file device tree
// would make the reporting floor an allocation trap; a fixed path depth keeps
// the operator-facing roots useful while making memory independent of file
// count below that depth.
func recordUnattributed(tree *unattributedNode, root, parent string, amount int64) {
	if tree == nil || amount <= 0 {
		return
	}
	bucket := filepath.Clean(parent)
	relative, err := filepath.Rel(filepath.Clean(root), bucket)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		bucket = filepath.Clean(root)
	} else if relative != "." {
		parts := strings.Split(relative, string(filepath.Separator))
		if len(parts) > unattributedReportingDepth {
			parts = parts[:unattributedReportingDepth]
		}
		bucket = filepath.Join(append([]string{filepath.Clean(root)}, parts...)...)
	}
	tree.buckets[bucket] += amount
}

func unattributedRoots(tree *unattributedNode, floor int64) []RootTotal {
	if floor <= 0 {
		floor = defaultCensusFloorBytes
	}
	if tree == nil {
		return nil
	}
	// Fold small buckets upward, deepest first, so every emitted root is at
	// least the reporting floor unless it is the device root itself.
	buckets := make(map[string]int64, len(tree.buckets))
	paths := make([]string, 0, len(tree.buckets))
	for path, bytes := range tree.buckets {
		buckets[path] += bytes
		paths = append(paths, path)
	}
	sort.Slice(paths, func(i, j int) bool {
		depth := func(path string) int { return len(strings.Split(filepath.Clean(path), string(filepath.Separator))) }
		left, right := depth(paths[i]), depth(paths[j])
		if left != right {
			return left > right
		}
		return paths[i] > paths[j]
	})
	for _, path := range paths {
		if buckets[path] >= floor || filepath.Clean(path) == filepath.Clean(tree.path) {
			continue
		}
		parent := filepath.Dir(path)
		buckets[parent] += buckets[path]
		delete(buckets, path)
	}
	result := make([]RootTotal, 0, len(buckets))
	for path, bytes := range buckets {
		if bytes > 0 {
			result = append(result, RootTotal{Path: path, Bytes: bytes})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Bytes != result[j].Bytes {
			return result[i].Bytes > result[j].Bytes
		}
		return result[i].Path < result[j].Path
	})
	return result
}

func existingRoots(roots []string) []string {
	seen := make(map[string]struct{}, len(roots))
	result := make([]string, 0, len(roots))
	for _, raw := range roots {
		path, err := filepath.Abs(raw)
		if err != nil {
			continue
		}
		path = filepath.Clean(path)
		if _, err := (hostFileSystem{}).Stat(path); err != nil {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

func scenarioCandidateRoots(repoRoot string, owner corestorage.OwnerManifest) []string {
	scenarioRoot := filepath.Dir(filepath.Dir(owner.ManifestPath))
	if !filepath.IsAbs(scenarioRoot) {
		scenarioRoot = filepath.Join(repoRoot, scenarioRoot)
	}
	relative := []string{
		"data", "logs", "storage", "state", "uploads", "models", "cache", "runtime",
		filepath.Join("api", "data"), filepath.Join("api", "storage"),
	}
	result := make([]string, 0, len(relative))
	for _, name := range relative {
		path := filepath.Join(scenarioRoot, name)
		if _, err := (hostFileSystem{}).Stat(path); err == nil {
			result = append(result, path)
		}
	}
	return result
}

func sortedStrings(values []string) []string { sort.Strings(values); return values }

func ownerStorageRoot(owner corestorage.OwnerManifest) string {
	path := filepath.Dir(owner.ManifestPath)
	if owner.Kind == corestorage.OwnerScenario {
		path = filepath.Dir(path)
	}
	return filepath.Clean(path)
}

// ownerStorageRoots includes the resolver-owned per-class namespaces in
// addition to the legacy manifest-adjacent root. The latter remains during the
// migration so repository-local scenario artifacts are still attributed; the
// class roots are what make ~/.local/share/vrooli/<kind>/<owner> visible even
// when an owner omitted that class from its declaration.
func ownerStorageRoots(repoRoot string, owner corestorage.OwnerManifest) []string {
	// Keep the manifest-adjacent root in the census migration boundary so
	// repository-local metadata remains attributed; validation deliberately
	// uses only class roots and bounded candidate directories.
	roots := []string{ownerStorageRoot(owner)}
	if owner.Kind == corestorage.OwnerScenario {
		// Keep repository-local runtime artifact directories attributable during
		// the migration, but never treat the whole source tree as an owner's
		// storage namespace.
		roots = append(roots, scenarioCandidateRoots(repoRoot, owner)...)
	}
	for _, class := range []corestorage.Class{corestorage.ClassConfig, corestorage.ClassData, corestorage.ClassCache, corestorage.ClassLogs, corestorage.ClassState} {
		path, err := corestorage.ResolveOwnerStoragePath(repoRoot, owner, corestorage.StorageEntry{Rung: corestorage.RungOwned, Kind: "dir", Class: class}, corestorage.HostPlatform(), corestorage.PlatformSeams{})
		if err == nil {
			roots = append(roots, path)
		}
	}
	seen := make(map[string]struct{}, len(roots))
	result := make([]string, 0, len(roots))
	for _, path := range roots {
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}
	return result
}

func containsRoot(roots []string, path string) bool {
	for _, root := range roots {
		if filepath.Clean(root) == filepath.Clean(path) {
			return true
		}
	}
	return false
}

// prunesName reports whether a policy name exclusion matches a directory
// base name.
func (p ScanPolicy) prunesName(name string) bool {
	for _, exclusion := range p.Exclusions {
		if exclusion.Name != "" && exclusion.Name == name {
			return true
		}
	}
	return false
}

func underAny(path string, roots []string) bool {
	for _, root := range roots {
		if isWithin(path, root) {
			return true
		}
	}
	return false
}

func isWithin(path, root string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ""
}

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

// declarationMatches finds declarations that contain a file by walking the
// file's ancestor chain. The old implementation compared every file with
// every declaration, which made a legitimate whole-host declaration turn a
// census into O(files*declarations) work. Ancestor lookup keeps the same
// overlap semantics while making the hot path proportional to path depth.
func declarationMatches(path string, byPath map[string][]int) []int {
	matches := make([]int, 0, 1)
	for candidate := filepath.Clean(path); ; candidate = filepath.Dir(candidate) {
		if indexes := byPath[candidate]; len(indexes) > 0 {
			matches = append(matches, indexes...)
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			break
		}
	}
	return matches
}

// Snapshot persistence is intentionally a small per-domain seam. The JSON
// payload keeps history forward-compatible while the indexed columns make
// the operator's common trend query cheap.
type SnapshotStore struct {
	db *database.RoutedDB
	// freshness is the age at which Latest turns a snapshot from "current"
	// to "stale". It follows the scheduled interval so a longer schedule does
	// not report every snapshot as stale.
	freshness time.Duration
}

func NewSnapshotStore(db *database.RoutedDB) *SnapshotStore {
	if db == nil {
		return nil
	}
	return &SnapshotStore{db: db, freshness: DefaultInterval}
}

// WithFreshness sets the age beyond which Latest reports a snapshot stale.
func (s *SnapshotStore) WithFreshness(window time.Duration) *SnapshotStore {
	if s != nil && window > 0 {
		s.freshness = window
	}
	return s
}

func (s *SnapshotStore) Save(ctx context.Context, report Report) (Report, error) {
	if s == nil {
		return report, nil
	}
	report.ObservedAt = time.Now().UTC()
	report.SnapshotID = snapshotID(report)
	devicePoints, err := s.devicePoints(ctx, report.Root, 24*time.Hour)
	if err != nil {
		return report, err
	}
	devicePoints = append(devicePoints, growthPoint{At: report.ObservedAt, Bytes: report.MeasuredBytes})
	if slope, ok := fitSlope(devicePoints); ok {
		report.GrowthSlopeBytesPerHour = &slope
	}
	payload, err := json.Marshal(report)
	if err != nil {
		return Report{}, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO census_snapshots (id, observed_at, root, measured_bytes, attributed_bytes, unattributed_bytes, confidence, report_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, report.SnapshotID, report.ObservedAt.Format(time.RFC3339Nano), report.Root, report.MeasuredBytes, report.AttributedBytes, report.UnattributedBytes, report.Confidence, string(payload))
	if err != nil {
		return Report{}, fmt.Errorf("save census snapshot: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO census_snapshot_metrics (snapshot_id, drift_bytes, growth_slope_bytes_per_hour) VALUES (?, ?, ?)`, report.SnapshotID, report.DriftBytes, report.GrowthSlopeBytesPerHour); err != nil {
		return Report{}, fmt.Errorf("save census snapshot metrics: %w", err)
	}
	for _, entry := range report.Entries {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO census_entry_samples (snapshot_id, observed_at, root, owner_kind, owner_id, entry_name, bytes) VALUES (?, ?, ?, ?, ?, ?, ?)`, report.SnapshotID, report.ObservedAt.Format(time.RFC3339Nano), report.Root, entry.Kind, entry.Owner, entry.Name, entry.Bytes); err != nil {
			return Report{}, fmt.Errorf("save census entry sample %s/%s/%s: %w", entry.Kind, entry.Owner, entry.Name, err)
		}
	}
	if err := s.pruneEntrySamples(ctx, report.ObservedAt.Add(-30*24*time.Hour)); err != nil {
		return Report{}, err
	}
	if err := s.pruneSnapshots(ctx, report.ObservedAt); err != nil {
		return Report{}, err
	}
	return report, nil
}

// Prune applies the census retention policy without requiring a new filesystem
// scan. This is used by the low-frequency maintenance scheduler so legacy
// report blobs cannot remain indefinitely when census collection is disabled.
func (s *SnapshotStore) Prune(ctx context.Context, now time.Time) error {
	if s == nil {
		return nil
	}
	if err := s.pruneEntrySamples(ctx, now.UTC().Add(-30*24*time.Hour)); err != nil {
		return err
	}
	return s.pruneSnapshots(ctx, now.UTC())
}

// pruneEntrySamples keeps the projected read model bounded independently of
// the immutable snapshot retention policy. The report blobs remain available
// for forensic callers; growth queries only need the recent sample window.
func (s *SnapshotStore) pruneEntrySamples(ctx context.Context, before time.Time) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM census_entry_samples WHERE observed_at < ?`, before.UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("prune census entry samples: %w", err)
	}
	return nil
}

// pruneSnapshots keeps recent snapshots densely for growth queries and one
// daily snapshot for the older forensic window. Snapshot reports are derived
// evidence; pruning them cannot affect the source filesystem or its budgets.
func (s *SnapshotStore) pruneSnapshots(ctx context.Context, now time.Time) error {
	const denseSnapshotLimit = 100
	cutoff := now.UTC().Add(-365 * 24 * time.Hour).Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `DELETE FROM census_snapshot_metrics WHERE snapshot_id IN (SELECT id FROM census_snapshots WHERE observed_at < ?)`, cutoff); err != nil {
		return fmt.Errorf("prune expired census metrics: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM census_snapshots WHERE observed_at < ?`, cutoff); err != nil {
		return fmt.Errorf("prune expired census snapshots: %w", err)
	}
	// SQLite's date() groups RFC3339 timestamps by UTC day. Keep the newest
	// observed row for each root/day beyond the dense 30-day window. Do not
	// infer recency from the opaque snapshot ID: callers may use any stable ID.
	denseCutoff := now.UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `DELETE FROM census_snapshot_metrics WHERE snapshot_id IN (
		SELECT s.id FROM census_snapshots s
		WHERE s.observed_at < ? AND EXISTS (
			SELECT 1 FROM census_snapshots newer
			WHERE newer.root = s.root AND date(newer.observed_at) = date(s.observed_at)
			  AND newer.observed_at < ?
			  AND (newer.observed_at > s.observed_at OR (newer.observed_at = s.observed_at AND newer.id > s.id))
		)
	)`, denseCutoff, denseCutoff); err != nil {
		return fmt.Errorf("prune sparse census metrics: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM census_snapshots
		WHERE observed_at < ? AND EXISTS (
			SELECT 1 FROM census_snapshots newer
			WHERE newer.root = census_snapshots.root AND date(newer.observed_at) = date(census_snapshots.observed_at)
			  AND newer.observed_at < ?
			  AND (newer.observed_at > census_snapshots.observed_at OR (newer.observed_at = census_snapshots.observed_at AND newer.id > census_snapshots.id))
		)`, denseCutoff, denseCutoff); err != nil {
		return fmt.Errorf("prune sparse census snapshots: %w", err)
	}
	// Keep the dense window bounded as well. A frequent census must not turn
	// its opaque report blobs into an unbounded database: one report can be
	// megabytes even though the indexed sample rows are small. The newest rows
	// remain available for history and growth; older rows are already
	// represented by the one-per-day forensic retention above.
	denseCutoff = now.UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `DELETE FROM census_snapshot_metrics WHERE snapshot_id IN (
		SELECT old.id FROM census_snapshots old
		WHERE old.observed_at >= ? AND (
			SELECT COUNT(*) FROM census_snapshots newer
			WHERE newer.root = old.root AND newer.observed_at >= ?
			  AND (newer.observed_at > old.observed_at OR (newer.observed_at = old.observed_at AND newer.id >= old.id))
		) > ?
	)`, denseCutoff, denseCutoff, denseSnapshotLimit); err != nil {
		return fmt.Errorf("prune dense census metrics: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM census_snapshots
		WHERE observed_at >= ? AND id IN (
			SELECT old.id FROM census_snapshots old
			WHERE old.observed_at >= ? AND (
				SELECT COUNT(*) FROM census_snapshots newer
				WHERE newer.root = old.root AND newer.observed_at >= ?
				  AND (newer.observed_at > old.observed_at OR (newer.observed_at = old.observed_at AND newer.id >= old.id))
			) > ?
		)`, denseCutoff, denseCutoff, denseCutoff, denseSnapshotLimit); err != nil {
		return fmt.Errorf("prune dense census snapshots: %w", err)
	}
	return nil
}

type growthPoint struct {
	At    time.Time
	Bytes int64
}

func (s *SnapshotStore) devicePoints(ctx context.Context, root string, window time.Duration) ([]growthPoint, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT observed_at, measured_bytes FROM census_snapshots WHERE root = ? AND observed_at >= ? ORDER BY observed_at ASC`, root, time.Now().UTC().Add(-window).Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("load census device samples: %w", err)
	}
	defer func() { _ = rows.Close() }()
	points := make([]growthPoint, 0)
	for rows.Next() {
		var raw string
		var bytes int64
		if err := rows.Scan(&raw, &bytes); err != nil {
			return nil, err
		}
		at, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, err
		}
		points = append(points, growthPoint{At: at, Bytes: bytes})
	}
	return points, rows.Err()
}

func fitSlope(points []growthPoint) (float64, bool) {
	// Device capacity projections are deliberately conservative: require six
	// snapshots so a short-lived write burst cannot become a days-to-full
	// signal. Owner growth projections have their own three-sample minimum.
	if len(points) < 6 {
		return 0, false
	}
	meanX, meanY := 0.0, 0.0
	for _, point := range points {
		meanX += point.At.Sub(points[0].At).Hours()
		meanY += float64(point.Bytes)
	}
	meanX /= float64(len(points))
	meanY /= float64(len(points))
	var numerator, denominator float64
	for _, point := range points {
		x := point.At.Sub(points[0].At).Hours() - meanX
		numerator += x * (float64(point.Bytes) - meanY)
		denominator += x * x
	}
	if denominator == 0 {
		return 0, false
	}
	return numerator / denominator, true
}

func (s *SnapshotStore) latest(ctx context.Context, root string) (*Report, error) {
	var payload string
	err := s.db.QueryRowContext(ctx, `SELECT report_json FROM census_snapshots WHERE root = ? ORDER BY observed_at DESC LIMIT 1`, root).Scan(&payload)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load census snapshot: %w", err)
	}
	var report Report
	if err := json.Unmarshal([]byte(payload), &report); err != nil {
		return nil, fmt.Errorf("decode census snapshot: %w", err)
	}
	return &report, nil
}

// Latest returns the most recent persisted report without walking the
// filesystem. Callers can use SnapshotAgeSeconds and StalenessVerdict to make
// freshness explicit to operators.
func (s *SnapshotStore) Latest(ctx context.Context, root string) (*Report, error) {
	if s == nil {
		return nil, nil
	}
	report, err := s.latest(ctx, root)
	if err != nil || report == nil {
		return report, err
	}
	age := time.Since(report.ObservedAt).Seconds()
	if age < 0 {
		age = 0
	}
	report.SnapshotAgeSeconds = &age
	freshness := s.freshness
	if freshness <= 0 {
		freshness = DefaultInterval
	}
	if age <= freshness.Seconds() {
		report.StalenessVerdict = "current"
	} else {
		report.StalenessVerdict = "stale"
	}
	return report, nil
}

func (s *SnapshotStore) History(ctx context.Context, root string, limit int) ([]Report, error) {
	if s == nil {
		return []Report{}, nil
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT report_json FROM census_snapshots WHERE root = ? ORDER BY observed_at DESC LIMIT ?`, root, limit)
	if err != nil {
		return nil, fmt.Errorf("load census history: %w", err)
	}
	defer func() { _ = rows.Close() }()
	result := make([]Report, 0, limit)
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("scan census history: %w", err)
		}
		var report Report
		if err := json.Unmarshal([]byte(payload), &report); err != nil {
			return nil, fmt.Errorf("decode census history: %w", err)
		}
		result = append(result, report)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read census history: %w", err)
	}
	return result, nil
}

type SnapshotSummary struct {
	SnapshotID        string    `json:"snapshot_id"`
	ObservedAt        time.Time `json:"observed_at"`
	Root              string    `json:"root"`
	MeasuredBytes     int64     `json:"measured_bytes"`
	AttributedBytes   int64     `json:"attributed_bytes"`
	DriftBytes        int64     `json:"drift_bytes"`
	UnattributedBytes int64     `json:"unattributed_bytes"`
	Confidence        string    `json:"confidence"`
	GrowthSlope       *float64  `json:"growth_slope_bytes_per_hour,omitempty"`
}

func (s *SnapshotStore) Count(ctx context.Context, root string) (int, error) {
	if s == nil {
		return 0, nil
	}
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM census_snapshots WHERE root = ?`, root).Scan(&count); err != nil {
		return 0, fmt.Errorf("count census snapshots: %w", err)
	}
	return count, nil
}

// BackfillEntrySamples upgrades snapshots written before the indexed sample
// table existed. It is idempotent and only inserts missing rows, so startup
// can safely resume after an interrupted backfill.
func (s *SnapshotStore) BackfillEntrySamples(ctx context.Context, root string, limit int) (int, error) {
	history, err := s.History(ctx, root, limit)
	if err != nil {
		return 0, err
	}
	inserted := 0
	for _, report := range history {
		if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO census_snapshot_metrics (snapshot_id, drift_bytes, growth_slope_bytes_per_hour) VALUES (?, ?, ?)`, report.SnapshotID, report.DriftBytes, report.GrowthSlopeBytesPerHour); err != nil {
			return inserted, err
		}
		for _, entry := range report.Entries {
			result, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO census_entry_samples (snapshot_id, observed_at, root, owner_kind, owner_id, entry_name, bytes) VALUES (?, ?, ?, ?, ?, ?, ?)`, report.SnapshotID, report.ObservedAt.Format(time.RFC3339Nano), report.Root, entry.Kind, entry.Owner, entry.Name, entry.Bytes)
			if err != nil {
				return inserted, err
			}
			if count, _ := result.RowsAffected(); count > 0 {
				inserted += int(count)
			}
		}
	}
	return inserted, nil
}

// Summaries reads only indexed columns. It is the default operator-facing
// history shape for large hosts; full report blobs remain available through
// History for forensic callers.
func (s *SnapshotStore) Summaries(ctx context.Context, root string, limit int) ([]SnapshotSummary, error) {
	if s == nil {
		return []SnapshotSummary{}, nil
	}
	if limit < 1 || limit > 1000 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT s.id, s.observed_at, s.root, s.measured_bytes, s.attributed_bytes, COALESCE(m.drift_bytes, 0), s.unattributed_bytes, s.confidence, m.growth_slope_bytes_per_hour FROM census_snapshots s LEFT JOIN census_snapshot_metrics m ON m.snapshot_id = s.id WHERE s.root = ? ORDER BY s.observed_at DESC LIMIT ?`, root, limit)
	if err != nil {
		return nil, fmt.Errorf("load census summaries: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]SnapshotSummary, 0, limit)
	for rows.Next() {
		var summary SnapshotSummary
		var raw string
		var slope sql.NullFloat64
		if err := rows.Scan(&summary.SnapshotID, &raw, &summary.Root, &summary.MeasuredBytes, &summary.AttributedBytes, &summary.DriftBytes, &summary.UnattributedBytes, &summary.Confidence, &slope); err != nil {
			return nil, err
		}
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, err
		}
		summary.ObservedAt = parsed
		if slope.Valid {
			value := slope.Float64
			summary.GrowthSlope = &value
		}
		out = append(out, summary)
	}
	return out, rows.Err()
}

func snapshotID(report Report) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s/%d/%d/%d", report.Root, report.MeasuredBytes, report.AttributedBytes, report.ObservedAt.UnixNano())))
	return "census-" + hex.EncodeToString(hash[:8])
}

// Schema is embedded so the census domain participates in the same
// idempotent per-domain bootstrap as cleanup and fleet inventories.
func Schema() string { return censusSchemaSQL }
