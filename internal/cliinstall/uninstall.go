package cliinstall

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/trustposture"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

const (
	installRecordVersion = 2
	planVersion          = 1
	breakGlassAudience   = trustposture.BreakGlassUninstallAudience
	breakGlassScope      = trustposture.BreakGlassUninstallScope
)

// UninstallBreakGlassAudience and UninstallBreakGlassScope are the stable
// claims used by the project-level destructive command. The verifier lives in
// cmd/vrooli so this package remains testable without key material.
const (
	UninstallBreakGlassAudience = breakGlassAudience
	UninstallBreakGlassScope    = breakGlassScope
)

type InstallScope string

const (
	ScopeAgent   InstallScope = "agent"
	ScopeRuntime InstallScope = "runtime"
	ScopeAll     InstallScope = "all"
)

type InstallEntryKind string

const (
	EntryFile      InstallEntryKind = "file"
	EntryDirectory InstallEntryKind = "directory"
	EntryBinary    InstallEntryKind = "binary"
	EntryService   InstallEntryKind = "service"
	EntryPackage   InstallEntryKind = "package"
	EntryImage     InstallEntryKind = "image"
	EntryContainer InstallEntryKind = "container"
	EntryVolume    InstallEntryKind = "volume"
	EntryNetwork   InstallEntryKind = "network"
)

// ObservedBefore is the package or artifact state measured immediately before
// Vrooli attempted to install it. Unknown is deliberately conservative: it
// can never authorize removal.
type ObservedBefore string

const (
	ObservedPresent ObservedBefore = "present"
	ObservedAbsent  ObservedBefore = "absent"
	ObservedUnknown ObservedBefore = "unknown"
)

// InstallAction records what the control plane did after the before-state
// probe. Only installed-on-absent is eligible for automatic removal.
type InstallAction string

const (
	ActionInstalled InstallAction = "installed"
	ActionAdopted   InstallAction = "adopted"
	ActionUpgraded  InstallAction = "upgraded"
	ActionNoOp      InstallAction = "no_op"
)

// InstallProvenance is the ownership evidence for host packages and
// container artifacts. It is nested so the artifact identity and the
// ownership decision cannot be confused with one another.
type InstallProvenance struct {
	PackageManager string         `json:"package_manager,omitempty"`
	PackageName    string         `json:"package_name,omitempty"`
	ObservedBefore ObservedBefore `json:"observed_before"`
	Action         InstallAction  `json:"action"`
	VersionBefore  string         `json:"version_before,omitempty"`
	VersionAfter   string         `json:"version_after,omitempty"`
	OwningNode     string         `json:"owning_node,omitempty"`
	Shared         bool           `json:"shared,omitempty"`
	Attributable   bool           `json:"attributable"`
}

// InstallEntry is an explicit artifact created by Vrooli. Prefix is recorded
// per entry because the runtime checkout, operator runtime home, and native
// service-manager directories are different safe roots on some platforms.
type InstallEntry struct {
	Scope          InstallScope      `json:"scope"`
	Kind           InstallEntryKind  `json:"kind"`
	Path           string            `json:"path"`
	Prefix         string            `json:"prefix"`
	Resource       string            `json:"resource,omitempty"`
	ServiceManager string            `json:"service_manager,omitempty"`
	ServiceName    string            `json:"service_name,omitempty"`
	ServiceDomain  string            `json:"service_domain,omitempty"`
	Provenance     InstallProvenance `json:"provenance,omitempty"`
}

type InstallRecord struct {
	Version          int                         `json:"version"`
	Prefix           string                      `json:"prefix"`
	UpdatedAt        string                      `json:"updated_at"`
	Entries          []InstallEntry              `json:"entries"`
	RuntimeProviders []RuntimeProviderProvenance `json:"runtime_providers,omitempty"`
}

// RuntimeProviderProvenance is durable capability metadata rather than a
// removal candidate. It records which container-runtime provider satisfied
// setup and where its daemon was reached, while leaving provider cleanup to
// its own explicit lifecycle (never to uninstall-time inference).
type RuntimeProviderProvenance struct {
	Capability     string         `json:"capability"`
	Provider       string         `json:"provider"`
	Endpoint       string         `json:"endpoint"`
	ObservedBefore ObservedBefore `json:"observed_before"`
	Action         InstallAction  `json:"action"`
	OwningNode     string         `json:"owning_node,omitempty"`
	RecordedAt     string         `json:"recorded_at"`
	Attributable   bool           `json:"attributable"`
}

type UninstallMode string

const (
	UninstallPlanMode  UninstallMode = "plan"
	UninstallApplyMode UninstallMode = "apply"
)

type UninstallRequest struct {
	Mode          UninstallMode
	PlanID        string
	Scope         InstallScope
	ConfirmTarget string
	BreakGlass    string
}

type DiskSnapshot struct {
	Path        string `json:"path"`
	Exists      bool   `json:"exists"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type UninstallPlan struct {
	Version         int                 `json:"version"`
	ID              string              `json:"plan_id"`
	CreatedAt       string              `json:"created_at"`
	Target          string              `json:"target"`
	Scope           InstallScope        `json:"scope"`
	RecordDigest    string              `json:"record_digest"`
	Remove          []InstallEntry      `json:"remove"`
	Keep            []UninstallDecision `json:"keep"`
	CannotAttribute []UninstallDecision `json:"cannot_attribute"`
	// Entries is retained as the apply set and compatibility view. It is
	// always equal to Remove for newly written plans.
	Entries []InstallEntry `json:"entries"`
	Disk    []DiskSnapshot `json:"disk"`
}

type UninstallDecision struct {
	InstallEntry
	Reason string `json:"reason"`
}

func (p UninstallPlan) RemoveOrEntries() []InstallEntry {
	if p.Remove != nil {
		return p.Remove
	}
	return p.Entries
}

type RemovalReceiptEntry struct {
	Scope InstallScope     `json:"scope"`
	Kind  InstallEntryKind `json:"kind"`
	Path  string           `json:"path"`
}

type RemovalReceipt struct {
	PlanID    string                `json:"plan_id"`
	Target    string                `json:"target"`
	Scope     InstallScope          `json:"scope"`
	RemovedAt string                `json:"removed_at"`
	Removed   []RemovalReceiptEntry `json:"removed"`
}

// Uninstaller is the command-facing seam. Production passes a real remover
// from cmd/vrooli; tests pass a recorder. There is no package-level deletion
// hook and no discovery operation on Apply.
type Uninstaller interface {
	Plan(UninstallRequest) (UninstallPlan, error)
	Apply(UninstallRequest) (RemovalReceipt, error)
}

// Remover has exactly one mutation operation. The entry carries enough typed
// context for the production implementation to stop a native service before
// removing its recorded unit file.
type Remover interface {
	Remove(InstallEntry) error
}

// BreakGlassVerifier is injected by the process entry point. It must verify
// the signed credential, audience, lifetime, and destructive scope.
type BreakGlassVerifier func(token string, now time.Time) error

// NewFileRemover is intentionally the only production constructor for the
// unexported real remover. cmd/vrooli is the only caller; tests inject a
// recording Remover into NewUninstallService instead.
func NewFileRemover(home string) Remover { return fileRemover{home: filepath.Clean(home)} }

type fileRemover struct {
	home string
}

func (r fileRemover) Remove(entry InstallEntry) error {
	if err := validateEntry(entry, r.home); err != nil {
		return err
	}
	if isContainerEntry(entry.Kind) {
		return removeContainerArtifact(entry)
	}
	if entry.Kind == EntryPackage {
		return removeHostPackage(entry)
	}
	if entry.Kind == EntryService {
		if err := stopRecordedService(entry); err != nil {
			return err
		}
	}
	if err := os.RemoveAll(entry.Path); err != nil {
		return fmt.Errorf("remove %s: %w", entry.Path, err)
	}
	return nil
}

func removeHostPackage(entry InstallEntry) error {
	manager := strings.ToLower(strings.TrimSpace(entry.Provenance.PackageManager))
	packageName := strings.TrimSpace(entry.Provenance.PackageName)
	if packageName == "" {
		packageName = strings.TrimSpace(entry.Path)
	}
	if manager == "" || packageName == "" {
		return fmt.Errorf("remove package: manager and package name are required")
	}
	var command string
	var args []string
	switch manager {
	case "brew", "homebrew":
		command, args = "brew", []string{"uninstall", "--", packageName}
	case "apt", "apt-get":
		command, args = manager, []string{"remove", "-y", "--", packageName}
	case "dnf", "yum":
		command, args = manager, []string{"remove", "-y", packageName}
	case "pacman":
		command, args = manager, []string{"-Rns", "--noconfirm", packageName}
	case "winget":
		command, args = manager, []string{"uninstall", "--id", packageName, "--exact", "--silent"}
	default:
		return fmt.Errorf("remove package %q: unsupported package manager %q", packageName, manager)
	}
	return runNativeRemovalCommand(command, args...)
}

func removeContainerArtifact(entry InstallEntry) error {
	artifact := strings.TrimSpace(entry.Path)
	if artifact == "" {
		return fmt.Errorf("remove container artifact: name is required")
	}
	var args []string
	switch entry.Kind {
	case EntryNetwork:
		args = []string{"network", "rm", artifact}
	case EntryContainer:
		args = []string{"rm", "-f", artifact}
	case EntryVolume:
		args = []string{"volume", "rm", artifact}
	case EntryImage:
		args = []string{"rmi", artifact}
	default:
		return fmt.Errorf("remove container artifact: unsupported kind %q", entry.Kind)
	}
	return runNativeRemovalCommand("docker", args...)
}

func runNativeRemovalCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

type uninstallService struct {
	root     string
	home     string
	remover  Remover
	verify   BreakGlassVerifier
	hostname func() (string, error)
	now      func() time.Time
}

// NewUninstallService builds the safe orchestration layer around an injected
// remover. It does not construct or retain any production deletion primitive.
func NewUninstallService(root, home string, remover Remover, verify BreakGlassVerifier) (Uninstaller, error) {
	if strings.TrimSpace(home) == "" {
		return nil, errors.New("uninstall: home is required")
	}
	if remover == nil {
		return nil, errors.New("uninstall: remover is required")
	}
	return &uninstallService{
		root:     filepath.Clean(root),
		home:     filepath.Clean(home),
		remover:  remover,
		verify:   verify,
		hostname: os.Hostname,
		now:      time.Now,
	}, nil
}

func InstallRecordPath(home string) (string, error) {
	state, err := repocontract.RuntimeHomeEntryPath(filepath.Clean(home), repocontract.HomeKeyState)
	if err != nil {
		return "", fmt.Errorf("resolve uninstall record directory: %w", err)
	}
	return filepath.Join(state, "install-record.json"), nil
}

func uninstallPlanPath(home, id string) (string, error) {
	state, err := repocontract.RuntimeHomeEntryPath(filepath.Clean(home), repocontract.HomeKeyState)
	if err != nil {
		return "", fmt.Errorf("resolve uninstall plan directory: %w", err)
	}
	if !isPlanID(id) {
		return "", fmt.Errorf("invalid uninstall plan id %q", id)
	}
	return filepath.Join(state, "uninstall-plans", id+".json"), nil
}

func LoadInstallRecord(home string) (InstallRecord, error) {
	path, err := InstallRecordPath(home)
	if err != nil {
		return InstallRecord{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return emptyInstallRecord(home), nil
	}
	if err != nil {
		return InstallRecord{}, fmt.Errorf("read install record %s: %w", path, err)
	}
	var record InstallRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return InstallRecord{}, fmt.Errorf("decode install record %s: %w", path, err)
	}
	// Normalize before validation so records written by the previous ledger
	// version can reach the migration path below. Validation first would make
	// the version upgrade unreachable and strand every CLI reinstall behind a
	// stale install record.
	record = normalizeRecord(record)
	if err := validateRecord(record, home); err != nil {
		return InstallRecord{}, err
	}
	return record, nil
}

func WriteInstallRecord(home string, record InstallRecord) error {
	record = normalizeRecord(record)
	if err := validateRecord(record, home); err != nil {
		return err
	}
	path, err := InstallRecordPath(home)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("encode install record: %w", err)
	}
	return config.WriteOwnedFile(path, append(data, '\n'), 0o600)
}

// RecordInstallEntries merges explicit artifacts into the durable record. It
// never scans the host for possible Vrooli paths; callers must name what they
// created. That property is what makes an unrecorded install produce an empty
// uninstall plan instead of an unsafe guess.
func RecordInstallEntries(home string, entries ...InstallEntry) error {
	record, err := LoadInstallRecord(home)
	if err != nil {
		return err
	}
	record.Entries = append(record.Entries, entries...)
	record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return WriteInstallRecord(home, record)
}

// RecordToolArtifacts is the single recording seam used by fetched and
// language-specific tool installers. Keeping the write here prevents each
// handler from growing its own ledger semantics while still allowing the
// handler to name the exact paths it created.
func RecordToolArtifacts(home string, entries ...InstallEntry) error {
	if len(entries) == 0 {
		return errors.New("record tool artifacts: at least one artifact is required")
	}
	return RecordInstallEntries(home, entries...)
}

// RecordContainerRuntime records a selected container-runtime provider in
// the durable ledger. It is intentionally metadata rather than an install
// entry, so uninstall cannot mistake a provider label or endpoint for a path
// or a command to remove.
func RecordContainerRuntime(home, provider, endpoint, node string, before ObservedBefore, action InstallAction) error {
	provider = strings.TrimSpace(provider)
	endpoint = strings.TrimSpace(endpoint)
	if provider == "" || endpoint == "" {
		return errors.New("record container runtime: provider and endpoint are required")
	}
	if before != ObservedPresent && before != ObservedAbsent && before != ObservedUnknown {
		return fmt.Errorf("record container runtime: invalid observed-before state %q", before)
	}
	if action != ActionInstalled && action != ActionAdopted && action != ActionUpgraded && action != ActionNoOp {
		return fmt.Errorf("record container runtime: invalid action %q", action)
	}
	record, err := LoadInstallRecord(home)
	if err != nil {
		return err
	}
	entry := RuntimeProviderProvenance{
		Capability: "container-runtime", Provider: provider, Endpoint: endpoint,
		ObservedBefore: before, Action: action, OwningNode: strings.TrimSpace(node),
		RecordedAt: time.Now().UTC().Format(time.RFC3339), Attributable: before != ObservedUnknown,
	}
	for _, existing := range record.RuntimeProviders {
		if existing.Capability == entry.Capability && existing.Provider == entry.Provider && existing.Endpoint == entry.Endpoint {
			return nil
		}
	}
	record.RuntimeProviders = append(record.RuntimeProviders, entry)
	record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return WriteInstallRecord(home, record)
}

// RecordPackageInstall is the shared recording helper for every host package
// install path. The caller supplies the state probe captured immediately before
// the command and the observed version after it returns; no uninstall-time
// inference is permitted.
func RecordPackageInstall(home, manager, packageName string, before ObservedBefore, action InstallAction, versionBefore, versionAfter, node string, shared bool) error {
	manager = strings.TrimSpace(manager)
	packageName = strings.TrimSpace(packageName)
	if manager == "" || packageName == "" {
		return errors.New("record package install: package manager and package name are required")
	}
	return RecordInstallEntries(home, InstallEntry{
		Scope: ScopeRuntime,
		Kind:  EntryPackage,
		Path:  packageName,
		Provenance: InstallProvenance{
			PackageManager: manager,
			PackageName:    packageName,
			ObservedBefore: before,
			Action:         action,
			VersionBefore:  strings.TrimSpace(versionBefore),
			VersionAfter:   strings.TrimSpace(versionAfter),
			OwningNode:     strings.TrimSpace(node),
			Shared:         shared,
			Attributable:   before != ObservedUnknown && action != "",
		},
	})
}

// RecordContainerArtifact records a named Docker artifact without introducing
// a filesystem discovery path. Container artifacts are removed in the
// network, container, volume, image order by the frozen plan.
func RecordContainerArtifact(home string, scope InstallScope, kind InstallEntryKind, name, resource, node string, shared bool) error {
	return RecordContainerArtifactWithProvenance(home, scope, kind, name, resource, node, shared, ObservedAbsent, ActionInstalled)
}

// RecordContainerArtifactWithProvenance records the observation made before a
// container operation.  A pre-existing artifact is recorded as adopted so it
// remains visible in the uninstall plan without becoming removable ownership.
// Unknown observations are deliberately unattributable.
func RecordContainerArtifactWithProvenance(home string, scope InstallScope, kind InstallEntryKind, name, resource, node string, shared bool, before ObservedBefore, action InstallAction) error {
	if !isContainerEntry(kind) || strings.TrimSpace(name) == "" {
		return errors.New("record container artifact: kind and name are required")
	}
	if before != ObservedPresent && before != ObservedAbsent && before != ObservedUnknown {
		return fmt.Errorf("record container artifact: invalid observed-before state %q", before)
	}
	if action != ActionInstalled && action != ActionAdopted && action != ActionUpgraded && action != ActionNoOp {
		return fmt.Errorf("record container artifact: invalid action %q", action)
	}
	return RecordInstallEntries(home, InstallEntry{
		Scope:    scope,
		Kind:     kind,
		Path:     strings.TrimSpace(name),
		Resource: strings.TrimSpace(resource),
		Provenance: InstallProvenance{
			ObservedBefore: before,
			Action:         action,
			OwningNode:     strings.TrimSpace(node),
			Shared:         shared,
			Attributable:   before != ObservedUnknown,
		},
	})
}

// RecordProjectSetup closes a setup transaction by recording the project
// checkout that setup owns and ensuring the durable record exists. Individual
// CLI installers record their exact binary and sidecar at the point of
// creation; this function deliberately does not scan an existing install
// directory and misclassify pre-existing tools as Vrooli-owned.
func RecordProjectSetup(root, home string) error {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || root == "" || !filepath.IsAbs(root) {
		return errors.New("record project setup: absolute project root is required")
	}
	if info, err := os.Stat(root); err != nil {
		return fmt.Errorf("record project setup root %s: %w", root, err)
	} else if !info.IsDir() {
		return fmt.Errorf("record project setup root %s is not a directory", root)
	}
	return RecordInstallEntries(home, InstallEntry{
		Scope:  ScopeRuntime,
		Kind:   EntryDirectory,
		Path:   root,
		Prefix: filepath.Dir(root),
	})
}

// RecordServiceInstall is shared by setup-owned service installers. The path
// remains explicit and the service manager metadata lets the platform remover
// unload it without a second discovery pass.
func RecordServiceInstall(home string, scope InstallScope, path, manager, name, domain string) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || path == "" {
		return errors.New("record service install: path is required")
	}
	return RecordInstallEntries(home, InstallEntry{
		Scope: scope, Kind: EntryService, Path: path, Prefix: filepath.Dir(path),
		ServiceManager: strings.TrimSpace(manager), ServiceName: strings.TrimSpace(name), ServiceDomain: strings.TrimSpace(domain),
	})
}

func (s *uninstallService) Plan(request UninstallRequest) (UninstallPlan, error) {
	if err := validatePlanRequest(request); err != nil {
		return UninstallPlan{}, err
	}
	target, err := s.confirmTarget(request.ConfirmTarget)
	if err != nil {
		return UninstallPlan{}, err
	}
	record, err := LoadInstallRecord(s.home)
	if err != nil {
		return UninstallPlan{}, err
	}
	entries := filterEntries(record.Entries, request.Scope)
	remove, keep, cannotAttribute := classifyEntries(entries)
	for _, entry := range remove {
		if err := validateEntry(entry, s.home); err != nil {
			return UninstallPlan{}, err
		}
	}
	sortEntries(remove)
	disk := make([]DiskSnapshot, 0, len(remove))
	existing := remove[:0]
	for _, entry := range remove {
		if entry.Kind == EntryPackage || isContainerEntry(entry.Kind) {
			existing = append(existing, entry)
			continue
		}
		fingerprint, exists, snapshotErr := snapshotPath(entry.Path)
		if snapshotErr != nil {
			return UninstallPlan{}, snapshotErr
		}
		if !exists {
			continue
		}
		existing = append(existing, entry)
		disk = append(disk, DiskSnapshot{Path: entry.Path, Exists: true, Fingerprint: fingerprint})
	}
	remove = existing
	id, err := newPlanID()
	if err != nil {
		return UninstallPlan{}, err
	}
	now := s.now().UTC()
	plan := UninstallPlan{
		Version:         planVersion,
		ID:              id,
		CreatedAt:       now.Format(time.RFC3339),
		Target:          target,
		Scope:           request.Scope,
		RecordDigest:    digestRecord(record),
		Remove:          remove,
		Keep:            keep,
		CannotAttribute: cannotAttribute,
		Entries:         remove,
		Disk:            disk,
	}
	path, err := uninstallPlanPath(s.home, id)
	if err != nil {
		return UninstallPlan{}, err
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return UninstallPlan{}, fmt.Errorf("encode uninstall plan: %w", err)
	}
	if err := config.WriteOwnedFile(path, append(data, '\n'), 0o600); err != nil {
		return UninstallPlan{}, fmt.Errorf("write uninstall plan: %w", err)
	}
	return plan, nil
}

func (s *uninstallService) Apply(request UninstallRequest) (RemovalReceipt, error) {
	if request.Mode != UninstallApplyMode || !isPlanID(request.PlanID) {
		return RemovalReceipt{}, &SafetyError{Code: "plan_required", Detail: "--apply requires a valid plan id"}
	}
	target, err := s.confirmTarget(request.ConfirmTarget)
	if err != nil {
		return RemovalReceipt{}, err
	}
	if s.verify == nil || strings.TrimSpace(request.BreakGlass) == "" {
		return RemovalReceipt{}, &SafetyError{Code: "break_glass_required", Detail: "a valid uninstall break-glass token is required"}
	}
	if err := s.verify(request.BreakGlass, s.now()); err != nil {
		return RemovalReceipt{}, &SafetyError{Code: "break_glass_required", Detail: err.Error()}
	}
	path, err := uninstallPlanPath(s.home, request.PlanID)
	if err != nil {
		return RemovalReceipt{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return RemovalReceipt{}, &SafetyError{Code: "plan_unavailable", Path: path, Detail: err.Error()}
	}
	var plan UninstallPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return RemovalReceipt{}, &SafetyError{Code: "plan_invalid", Path: path, Detail: err.Error()}
	}
	if plan.ID != request.PlanID || plan.Version != planVersion || plan.Target != target {
		return RemovalReceipt{}, &SafetyError{Code: "plan_stale", Path: path, Detail: "plan identity or target no longer matches"}
	}
	if request.Scope != "" && request.Scope != plan.Scope {
		return RemovalReceipt{}, &SafetyError{Code: "plan_stale", Path: path, Detail: "requested scope differs from frozen plan"}
	}
	record, err := LoadInstallRecord(s.home)
	if err != nil {
		return RemovalReceipt{}, err
	}
	if digestRecord(record) != plan.RecordDigest {
		return RemovalReceipt{}, &SafetyError{Code: "plan_stale", Path: mustInstallRecordPath(s.home), Detail: "install record changed since planning"}
	}
	entries := plan.Remove
	if entries == nil {
		entries = plan.Entries
	}
	for _, entry := range entries {
		if err := validateEntry(entry, s.home); err != nil {
			return RemovalReceipt{}, err
		}
	}
	if err := verifyFrozenDisk(plan); err != nil {
		return RemovalReceipt{}, err
	}
	receipt := RemovalReceipt{PlanID: plan.ID, Target: target, Scope: plan.Scope, RemovedAt: s.now().UTC().Format(time.RFC3339), Removed: make([]RemovalReceiptEntry, 0, len(entries))}
	for _, entry := range entries {
		if err := s.remover.Remove(entry); err != nil {
			return RemovalReceipt{}, fmt.Errorf("remove frozen entry %s: %w", entry.Path, err)
		}
		receipt.Removed = append(receipt.Removed, RemovalReceiptEntry{Scope: entry.Scope, Kind: entry.Kind, Path: entry.Path})
	}
	return receipt, nil
}

type SafetyError struct {
	Code   string
	Path   string
	Detail string
}

func (e *SafetyError) Error() string {
	parts := []string{e.Code}
	if e.Path != "" {
		parts = append(parts, e.Path)
	}
	if e.Detail != "" {
		parts = append(parts, e.Detail)
	}
	return strings.Join(parts, ": ")
}

func (s *uninstallService) confirmTarget(confirm string) (string, error) {
	hostname, err := s.hostname()
	if err != nil {
		return "", &SafetyError{Code: "target_unavailable", Detail: err.Error()}
	}
	target := strings.TrimSpace(confirm)
	if target == "" || !strings.EqualFold(target, strings.TrimSpace(hostname)) {
		return "", &SafetyError{Code: "target_mismatch", Detail: fmt.Sprintf("confirmed %q but live hostname is %q", target, hostname)}
	}
	return strings.TrimSpace(hostname), nil
}

func validatePlanRequest(request UninstallRequest) error {
	if request.Mode != UninstallPlanMode {
		return &SafetyError{Code: "plan_required", Detail: "--plan is required"}
	}
	if !validScope(request.Scope) {
		return &SafetyError{Code: "invalid_scope", Detail: fmt.Sprintf("scope %q must be agent, runtime, or all", request.Scope)}
	}
	if strings.TrimSpace(request.ConfirmTarget) == "" {
		return &SafetyError{Code: "target_required", Detail: "--confirm-target is required"}
	}
	return nil
}

func validScope(scope InstallScope) bool {
	return scope == ScopeAgent || scope == ScopeRuntime || scope == ScopeAll
}

func filterEntries(entries []InstallEntry, scope InstallScope) []InstallEntry {
	out := make([]InstallEntry, 0, len(entries))
	for _, entry := range entries {
		if scope == ScopeAll || entry.Scope == scope {
			out = append(out, entry)
		}
	}
	return out
}

func isContainerEntry(kind InstallEntryKind) bool {
	switch kind {
	case EntryImage, EntryContainer, EntryVolume, EntryNetwork:
		return true
	default:
		return false
	}
}

// classifyEntries is the only ownership decision used by planning. It does
// not inspect the host and therefore cannot discover a new removal candidate.
// Explicit filesystem entries remain removable because their path itself is
// the ownership record. Packages require complete provenance evidence.
func classifyEntries(entries []InstallEntry) ([]InstallEntry, []UninstallDecision, []UninstallDecision) {
	remove := make([]InstallEntry, 0, len(entries))
	keep := make([]UninstallDecision, 0)
	cannotAttribute := make([]UninstallDecision, 0)
	for _, entry := range entries {
		if isContainerEntry(entry.Kind) {
			provenance := entry.Provenance
			if !provenance.Attributable || provenance.ObservedBefore == "" || provenance.ObservedBefore == ObservedUnknown || provenance.Action == "" {
				cannotAttribute = append(cannotAttribute, UninstallDecision{InstallEntry: entry, Reason: "container artifact provenance is incomplete"})
				continue
			}
			if provenance.Shared {
				keep = append(keep, UninstallDecision{InstallEntry: entry, Reason: "container artifact is marked shared"})
				continue
			}
			if provenance.Action != ActionInstalled || provenance.ObservedBefore != ObservedAbsent {
				keep = append(keep, UninstallDecision{InstallEntry: entry, Reason: "container artifact was not installed onto an absent host state"})
				continue
			}
			remove = append(remove, entry)
			continue
		}
		if entry.Kind != EntryPackage {
			remove = append(remove, entry)
			continue
		}
		provenance := entry.Provenance
		if !provenance.Attributable || provenance.ObservedBefore == ObservedUnknown || strings.TrimSpace(string(provenance.Action)) == "" {
			cannotAttribute = append(cannotAttribute, UninstallDecision{InstallEntry: entry, Reason: "package provenance is incomplete or predates the current ledger version"})
			continue
		}
		if provenance.Action != ActionInstalled {
			keep = append(keep, UninstallDecision{InstallEntry: entry, Reason: fmt.Sprintf("action was %s, not installed", provenance.Action)})
			continue
		}
		if provenance.ObservedBefore != ObservedAbsent {
			keep = append(keep, UninstallDecision{InstallEntry: entry, Reason: fmt.Sprintf("package was %s before installation", provenance.ObservedBefore)})
			continue
		}
		if provenance.Shared {
			keep = append(keep, UninstallDecision{InstallEntry: entry, Reason: "package is marked shared"})
			continue
		}
		remove = append(remove, entry)
	}
	sort.Slice(keep, func(i, j int) bool { return decisionLess(keep[i], keep[j]) })
	sort.Slice(cannotAttribute, func(i, j int) bool { return decisionLess(cannotAttribute[i], cannotAttribute[j]) })
	return remove, keep, cannotAttribute
}

func decisionLess(a, b UninstallDecision) bool {
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	return a.Reason < b.Reason
}

func sortEntries(entries []InstallEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if containerRemovalRank(entries[i].Kind) != containerRemovalRank(entries[j].Kind) {
			return containerRemovalRank(entries[i].Kind) < containerRemovalRank(entries[j].Kind)
		}
		if entries[i].Scope != entries[j].Scope {
			return entries[i].Scope < entries[j].Scope
		}
		if entries[i].Path != entries[j].Path {
			return entries[i].Path < entries[j].Path
		}
		return entries[i].Kind < entries[j].Kind
	})
}

func containerRemovalRank(kind InstallEntryKind) int {
	switch kind {
	case EntryNetwork:
		return 0
	case EntryContainer:
		return 1
	case EntryVolume:
		return 2
	case EntryImage:
		return 3
	default:
		return 4
	}
}

func normalizeRecord(record InstallRecord) InstallRecord {
	if record.Version == 0 {
		record.Version = installRecordVersion
	}
	if record.Version < installRecordVersion {
		for i := range record.Entries {
			if record.Entries[i].Kind == EntryPackage && record.Entries[i].Provenance.ObservedBefore == "" {
				record.Entries[i].Provenance = InstallProvenance{
					ObservedBefore: ObservedUnknown,
					Attributable:   false,
				}
			}
		}
		record.Version = installRecordVersion
	}
	if record.Entries == nil {
		record.Entries = []InstallEntry{}
	}
	if record.RuntimeProviders == nil {
		record.RuntimeProviders = []RuntimeProviderProvenance{}
	}
	seen := make(map[string]struct{}, len(record.Entries))
	entries := make([]InstallEntry, 0, len(record.Entries))
	for _, entry := range record.Entries {
		entry.Path = filepath.Clean(strings.TrimSpace(entry.Path))
		entry.Prefix = filepath.Clean(strings.TrimSpace(entry.Prefix))
		key := string(entry.Scope) + "\x00" + string(entry.Kind) + "\x00" + entry.Path
		if entry.Path == "." || entry.Path == "" {
			continue
		}
		if entry.Kind != EntryPackage && !isContainerEntry(entry.Kind) && (entry.Prefix == "." || entry.Prefix == "") {
			continue
		}
		if entry.Kind == EntryPackage && entry.Provenance.ObservedBefore == "" {
			entry.Provenance.ObservedBefore = ObservedUnknown
		}
		if isContainerEntry(entry.Kind) && entry.Provenance.ObservedBefore == "" {
			entry.Provenance.ObservedBefore = ObservedUnknown
			entry.Provenance.Attributable = false
		}
		if entry.Kind == EntryPackage && entry.Provenance.PackageName == "" {
			entry.Provenance.PackageName = entry.Path
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		entries = append(entries, entry)
	}
	sortEntries(entries)
	record.Entries = entries
	return record
}

func validateRecord(record InstallRecord, home string) error {
	if record.Version != installRecordVersion {
		return fmt.Errorf("install record: unsupported version %d", record.Version)
	}
	for _, entry := range record.Entries {
		if !validScope(entry.Scope) {
			return fmt.Errorf("install record: invalid scope %q", entry.Scope)
		}
		if entry.Kind != EntryFile && entry.Kind != EntryDirectory && entry.Kind != EntryBinary && entry.Kind != EntryService && entry.Kind != EntryPackage && !isContainerEntry(entry.Kind) {
			return fmt.Errorf("install record: invalid kind %q", entry.Kind)
		}
		if entry.Kind == EntryPackage {
			if strings.TrimSpace(entry.Provenance.PackageName) == "" {
				return fmt.Errorf("install record: package entry %q has no package name", entry.Path)
			}
			if entry.Provenance.ObservedBefore != ObservedPresent && entry.Provenance.ObservedBefore != ObservedAbsent && entry.Provenance.ObservedBefore != ObservedUnknown {
				return fmt.Errorf("install record: invalid observed-before state %q", entry.Provenance.ObservedBefore)
			}
			if entry.Provenance.Action != "" && entry.Provenance.Action != ActionInstalled && entry.Provenance.Action != ActionAdopted && entry.Provenance.Action != ActionUpgraded && entry.Provenance.Action != ActionNoOp {
				return fmt.Errorf("install record: invalid action %q", entry.Provenance.Action)
			}
			continue
		}
		if isContainerEntry(entry.Kind) {
			if strings.TrimSpace(entry.Path) == "" {
				return fmt.Errorf("install record: container artifact name is required")
			}
			if entry.Provenance.ObservedBefore != ObservedPresent && entry.Provenance.ObservedBefore != ObservedAbsent && entry.Provenance.ObservedBefore != ObservedUnknown {
				return fmt.Errorf("install record: invalid container observed-before state %q", entry.Provenance.ObservedBefore)
			}
			if entry.Provenance.Action != "" && entry.Provenance.Action != ActionInstalled && entry.Provenance.Action != ActionAdopted && entry.Provenance.Action != ActionUpgraded && entry.Provenance.Action != ActionNoOp {
				return fmt.Errorf("install record: invalid container action %q", entry.Provenance.Action)
			}
			continue
		}
		if err := validateEntry(entry, home); err != nil {
			return err
		}
	}
	for _, provider := range record.RuntimeProviders {
		if strings.TrimSpace(provider.Capability) == "" || strings.TrimSpace(provider.Provider) == "" || strings.TrimSpace(provider.Endpoint) == "" {
			return fmt.Errorf("install record: runtime provider requires capability, provider, and endpoint")
		}
		if provider.ObservedBefore != ObservedPresent && provider.ObservedBefore != ObservedAbsent && provider.ObservedBefore != ObservedUnknown {
			return fmt.Errorf("install record: invalid runtime observed-before state %q", provider.ObservedBefore)
		}
		if provider.Action != ActionInstalled && provider.Action != ActionAdopted && provider.Action != ActionUpgraded && provider.Action != ActionNoOp {
			return fmt.Errorf("install record: invalid runtime action %q", provider.Action)
		}
	}
	return nil
}

func emptyInstallRecord(home string) InstallRecord {
	home = filepath.Clean(home)
	return InstallRecord{Version: installRecordVersion, Prefix: filepath.Join(home, ".vrooli"), Entries: []InstallEntry{}, RuntimeProviders: []RuntimeProviderProvenance{}}
}

func validateEntry(entry InstallEntry, home string) error {
	if entry.Kind == EntryPackage {
		if strings.TrimSpace(entry.Path) == "" || strings.TrimSpace(entry.Provenance.PackageName) == "" {
			return &SafetyError{Code: "package_identity_missing", Path: entry.Path, Detail: "package entries require a package name"}
		}
		return nil
	}
	if isContainerEntry(entry.Kind) {
		if strings.TrimSpace(entry.Path) == "" {
			return &SafetyError{Code: "container_identity_missing", Detail: "container artifact entries require an artifact name"}
		}
		return nil
	}
	path, prefix := filepath.Clean(entry.Path), filepath.Clean(entry.Prefix)
	if !filepath.IsAbs(path) || !filepath.IsAbs(prefix) {
		return &SafetyError{Code: "path_outside_prefix", Path: path, Detail: "recorded paths and prefixes must be absolute"}
	}
	home = filepath.Clean(home)
	if path == home || path == string(filepath.Separator) {
		return &SafetyError{Code: "path_forbidden", Path: path, Detail: "$HOME and filesystem root are never removable"}
	}
	rel, err := filepath.Rel(prefix, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return &SafetyError{Code: "path_outside_prefix", Path: path, Detail: fmt.Sprintf("not beneath recorded prefix %s", prefix)}
	}
	if err := validateResolvedPath(path, prefix); err != nil {
		return err
	}
	return nil
}

func validateResolvedPath(path, prefix string) error {
	resolvedPrefix, prefixErr := filepath.EvalSymlinks(prefix)
	if prefixErr != nil && !errors.Is(prefixErr, os.ErrNotExist) {
		return &SafetyError{Code: "path_unreadable", Path: prefix, Detail: prefixErr.Error()}
	}
	probe := path
	for {
		resolved, err := filepath.EvalSymlinks(probe)
		if err == nil {
			comparisonPrefix := prefix
			if prefixErr == nil {
				comparisonPrefix = resolvedPrefix
			}
			rel, relErr := filepath.Rel(comparisonPrefix, resolved)
			if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return &SafetyError{Code: "symlink_outside_prefix", Path: path, Detail: fmt.Sprintf("resolves to %s", resolved)}
			}
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return &SafetyError{Code: "path_unreadable", Path: probe, Detail: err.Error()}
		}
		if filepath.Clean(probe) == filepath.Clean(prefix) || filepath.Dir(probe) == probe {
			// The recorded path is absent and no existing symlink component was
			// found. The lexical prefix check above remains the authority.
			return nil
		}
		probe = filepath.Dir(probe)
	}
}

func verifyFrozenDisk(plan UninstallPlan) error {
	for _, expected := range plan.Disk {
		fingerprint, exists, err := snapshotPath(expected.Path)
		if err != nil {
			return err
		}
		if !exists || !expected.Exists || fingerprint != expected.Fingerprint {
			return &SafetyError{Code: "plan_stale", Path: expected.Path, Detail: "disk no longer matches frozen inventory"}
		}
	}
	return nil
}

func snapshotPath(path string) (string, bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("snapshot %s: %w", path, err)
	}
	hash := sha256.New()
	writeInfo := func(relative string, fileInfo os.FileInfo) error {
		_, _ = io.WriteString(hash, relative+"\x00"+fileInfo.Mode().String()+"\x00")
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(filepath.Join(path, relative))
			if err != nil {
				return err
			}
			_, _ = io.WriteString(hash, target)
			return nil
		}
		_, _ = io.WriteString(hash, fmt.Sprintf("%d\x00", fileInfo.Size()))
		if fileInfo.Mode().IsRegular() {
			file, err := os.Open(filepath.Join(path, relative))
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(hash, file)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			return closeErr
		}
		return nil
	}
	if !info.IsDir() {
		if err := writeInfo(".", info); err != nil {
			return "", false, fmt.Errorf("snapshot %s: %w", path, err)
		}
	} else {
		if err := filepath.Walk(path, func(current string, fileInfo os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relative, err := filepath.Rel(path, current)
			if err != nil {
				return err
			}
			return writeInfo(relative, fileInfo)
		}); err != nil {
			return "", false, fmt.Errorf("snapshot %s: %w", path, err)
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), true, nil
}

func digestRecord(record InstallRecord) string {
	data, _ := json.Marshal(normalizeRecord(record))
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func newPlanID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate uninstall plan id: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

func isPlanID(id string) bool {
	if len(id) != 32 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func mustInstallRecordPath(home string) string {
	path, err := InstallRecordPath(home)
	if err != nil {
		return filepath.Join(filepath.Clean(home), ".vrooli", "state", "install-record.json")
	}
	return path
}

func stopRecordedService(entry InstallEntry) error {
	manager := strings.ToLower(strings.TrimSpace(entry.ServiceManager))
	name := strings.TrimSpace(entry.ServiceName)
	switch {
	case manager == "systemd" || (manager == "" && runtime.GOOS == "linux"):
		if name == "" {
			name = filepath.Base(entry.Path)
		}
		args := []string{"disable", "--now", name}
		if strings.EqualFold(entry.ServiceDomain, "system") {
			return runNativeServiceCommand("systemctl", args...)
		}
		return runNativeServiceCommand("systemctl", append([]string{"--user"}, args...)...)
	case manager == "launchd" || (manager == "" && runtime.GOOS == "darwin"):
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(entry.Path), filepath.Ext(entry.Path))
		}
		domain := strings.TrimSpace(entry.ServiceDomain)
		if domain == "" {
			domain = "gui/" + currentUserID()
		}
		return runNativeServiceCommand("launchctl", "bootout", domain+"/"+name)
	default:
		return nil
	}
}

func currentUserID() string {
	if uid := strings.TrimSpace(os.Getenv("UID")); uid != "" {
		return uid
	}
	if current, err := user.Current(); err == nil && strings.TrimSpace(current.Uid) != "" {
		return strings.TrimSpace(current.Uid)
	}
	return "0"
}

func runNativeServiceCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		// A missing/already stopped service is safe and idempotent. Other failures
		// must stop uninstall before the unit file is removed.
		if strings.Contains(strings.ToLower(string(output)), "not found") || strings.Contains(strings.ToLower(string(output)), "no such process") {
			return nil
		}
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}
