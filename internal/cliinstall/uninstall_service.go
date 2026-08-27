package cliinstall

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/artifactledger"

	"github.com/vrooli/api-core/cleanupplan"
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
	Scope  InstallScope     `json:"scope"`
	Kind   InstallEntryKind `json:"kind"`
	Path   string           `json:"path"`
	Prefix string           `json:"prefix"`
	// Volatile marks an explicitly owned runtime surface whose contents are
	// expected to change while its service is alive (for example an agent
	// heartbeat/state directory). It remains an exact removal target; only the
	// content-hash TOCTOU check is skipped for this ledger-declared surface.
	Volatile       bool              `json:"volatile,omitempty"`
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
	UninstallPlanMode   UninstallMode = "plan"
	UninstallApplyMode  UninstallMode = "apply"
	UninstallVerifyMode UninstallMode = "verify"
)

type UninstallRequest struct {
	Mode            UninstallMode
	PlanID          string
	Scope           InstallScope
	ConfirmTarget   string
	BreakGlass      string
	AuthorizingUser string
	MachineID       string
	NodeID          string
	OperationID     string
	PlanHash        string
}

type DiskSnapshot struct {
	Path        string `json:"path"`
	Exists      bool   `json:"exists"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type UninstallPlan struct {
	Version      int          `json:"version"`
	ID           string       `json:"plan_id"`
	CreatedAt    string       `json:"created_at"`
	Target       string       `json:"target"`
	Scope        InstallScope `json:"scope"`
	RecordDigest string       `json:"record_digest"`
	// PlanHash covers the resolved artifact lists, not the request, id, or
	// timestamps. Bridge binds its cleanup capability to this stable value.
	PlanHash        string              `json:"plan_hash"`
	Remove          []InstallEntry      `json:"remove"`
	Keep            []UninstallDecision `json:"keep"`
	CannotAttribute []UninstallDecision `json:"cannot_attribute"`
	// Entries is retained as the apply set and compatibility view. It is
	// always equal to Remove for newly written plans.
	Entries  []InstallEntry        `json:"entries"`
	Disk     []DiskSnapshot        `json:"disk"`
	Applied  []RemovalReceiptEntry `json:"applied,omitempty"`
	Attempts []RemovalAttempt      `json:"attempts,omitempty"`
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

// ComputePlanHash returns the stable digest of the resolved artifact
// classification. It intentionally excludes the caller's request, plan id,
// record digest, and timestamps so retries and transport changes cannot alter
// the authorization subject.
func ComputePlanHash(plan UninstallPlan) string {
	return cleanupplan.HashResolvedArtifacts(plan.RemoveOrEntries(), plan.Keep, plan.CannotAttribute)
}

type RemovalReceiptEntry struct {
	Scope InstallScope     `json:"scope"`
	Kind  InstallEntryKind `json:"kind"`
	Path  string           `json:"path"`
}

type RemovalReceipt struct {
	PlanID          string                `json:"plan_id"`
	PlanHash        string                `json:"plan_hash"`
	Target          string                `json:"target"`
	Scope           InstallScope          `json:"scope"`
	RemovedAt       string                `json:"removed_at"`
	AuthorizingUser string                `json:"authorizing_user,omitempty"`
	Removed         []RemovalReceiptEntry `json:"removed"`
	Preserved       []UninstallDecision   `json:"preserved"`
	CannotAttribute []UninstallDecision   `json:"cannot_attribute"`
	Attempts        []RemovalAttempt      `json:"attempts"`
}

// UninstallVerification is a read-only post-apply observation. Entries whose
// state cannot be checked without invoking a platform-specific package or
// container client are reported as not_checked rather than being claimed
// absent.
type UninstallVerification struct {
	PlanID          string                `json:"plan_id"`
	PlanHash        string                `json:"plan_hash"`
	Target          string                `json:"target"`
	Scope           InstallScope          `json:"scope"`
	VerifiedAt      string                `json:"verified_at"`
	Complete        bool                  `json:"complete"`
	Removed         []RemovalReceiptEntry `json:"removed"`
	Remaining       []RemovalReceiptEntry `json:"remaining"`
	NotChecked      []RemovalReceiptEntry `json:"not_checked"`
	Preserved       []UninstallDecision   `json:"preserved"`
	CannotAttribute []UninstallDecision   `json:"cannot_attribute"`
}

// RemovalAttempt is durable progress evidence for a resumable apply. A failed
// attempt is not erased when a later retry succeeds.
type RemovalAttempt struct {
	StartedAt  string                `json:"started_at"`
	FinishedAt string                `json:"finished_at"`
	Applied    []RemovalReceiptEntry `json:"applied"`
	Error      string                `json:"error,omitempty"`
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

// BoundBreakGlassVerifier adds the frozen operation context to the existing
// signature/lifetime check. It is optional so older non-Bridge callers retain
// their compatibility path while cleanup can require complete binding.
type BoundBreakGlassVerifier func(token string, request UninstallRequest, plan UninstallPlan, now time.Time) error

type UninstallOption func(*uninstallService)

func WithBoundBreakGlassVerifier(verify BoundBreakGlassVerifier) UninstallOption {
	return func(s *uninstallService) { s.boundVerify = verify }
}

// NewFileRemover is intentionally the only production constructor for the
// unexported real remover. cmd/vrooli is the only caller; tests inject a
// recording Remover into NewUninstallService instead. The optional service
// names are deferred only for the paired helper's self-cleanup path; all other
// service entries are stopped before their unit files are removed.
func NewFileRemover(home string, deferredServiceNames ...string) Remover {
	deferred := make(map[string]struct{}, len(deferredServiceNames))
	for _, name := range deferredServiceNames {
		if name = normalizeServiceName(name); name != "" {
			deferred[name] = struct{}{}
		}
	}
	home = filepath.Clean(home)
	// A ledger failure here is not fatal to constructing the remover: it is
	// surfaced at removal time, where refusing to delete is meaningful. A nil
	// ledger makes Remove refuse rather than delete unrecorded.
	ledger, _ := artifactledger.New(home)
	return fileRemover{home: home, deferredServiceNames: deferred, ledger: ledger}
}

type fileRemover struct {
	home                 string
	deferredServiceNames map[string]struct{}
	ledger               *artifactledger.Ledger
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
	if entry.Kind == EntryService && !r.defersServiceStop(entry) {
		if err := stopRecordedService(entry); err != nil {
			return err
		}
	}
	if r.ledger == nil {
		// Refusing is the safe direction: an uninstall nothing recorded is the
		// fault class this ledger exists to end.
		return fmt.Errorf("remove %s: no removal ledger is configured", entry.Path)
	}
	if err := r.ledger.Guard(artifactledger.Removal{
		Path:      entry.Path,
		Kind:      string(entry.Kind),
		Component: "cliinstall.fileRemover",
		Predicate: uninstallPlanPredicate,
	}, func() error { return os.RemoveAll(entry.Path) }); err != nil {
		return fmt.Errorf("remove %s: %w", entry.Path, err)
	}
	return nil
}

// uninstallPlanPredicate is the rule the uninstall remover enforces. Unlike a
// reclamation, this path never decides for itself what to delete: it applies an
// operator-authorized plan, and the receipt says so.
const uninstallPlanPredicate = "entry listed in an authorized uninstall plan"

// defersServiceStop is used by the paired privileged helper while it is
// applying its own frozen cleanup plan. Stopping either Bridge service before
// the terminal receipt is reported would tear down the reporting path (and, on
// some service managers, kill the helper that is doing the removal). The unit
// file is still removed in this call; the owning process is shut down by the
// transport after the receipt has crossed the control-plane boundary.
func (r fileRemover) defersServiceStop(entry InstallEntry) bool {
	if entry.Kind != EntryService || strings.TrimSpace(entry.ServiceName) == "" {
		return false
	}
	_, ok := r.deferredServiceNames[normalizeServiceName(entry.ServiceName)]
	return ok
}

// normalizeServiceName lets the cleanup guard use one logical service name
// across native managers: systemd records commonly carry a .service suffix,
// while launchd records carry the Bridge reverse-DNS prefix.
func normalizeServiceName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".service")
	name = strings.TrimPrefix(name, "com.vrooli.bridge.")
	return name
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
	root        string
	home        string
	planDir     string
	remover     Remover
	verify      BreakGlassVerifier
	boundVerify BoundBreakGlassVerifier
	hostname    func() (string, error)
	now         func() time.Time
}

// NewUninstallService builds the safe orchestration layer around an injected
// remover. It does not construct or retain any production deletion primitive.
func NewUninstallService(root, home string, remover Remover, verify BreakGlassVerifier, options ...UninstallOption) (Uninstaller, error) {
	if strings.TrimSpace(home) == "" {
		return nil, errors.New("uninstall: home is required")
	}
	if remover == nil {
		return nil, errors.New("uninstall: remover is required")
	}
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || root == "." {
		return nil, errors.New("uninstall: repository root is required")
	}
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return nil, fmt.Errorf("uninstall: load repository contract: %w", err)
	}
	state, err := contract.RuntimeHomeEntry(home, repocontract.HomeKeyState)
	if err != nil {
		return nil, fmt.Errorf("uninstall: resolve state directory: %w", err)
	}
	service := &uninstallService{
		root:     filepath.Clean(root),
		home:     filepath.Clean(home),
		planDir:  filepath.Join(state.AbsPath, "uninstall-plans"),
		remover:  remover,
		verify:   verify,
		hostname: os.Hostname,
		now:      time.Now,
	}
	for _, option := range options {
		option(service)
	}
	return service, nil
}

func InstallRecordPath(home string) (string, error) {
	state, err := repocontract.RuntimeHomeEntryPath(filepath.Clean(home), repocontract.HomeKeyState)
	if err != nil {
		return "", fmt.Errorf("resolve uninstall record directory: %w", err)
	}
	return filepath.Join(state, "install-record.json"), nil
}

func uninstallPlanPathIn(planDir, id string) (string, error) {
	planDir = filepath.Clean(strings.TrimSpace(planDir))
	if planDir == "" || planDir == "." {
		return "", errors.New("resolve uninstall plan directory: plan directory is required")
	}
	if !isPlanID(id) {
		return "", fmt.Errorf("invalid uninstall plan id %q", id)
	}
	return filepath.Join(planDir, id+".json"), nil
}

func (s *uninstallService) planPath(id string) (string, error) {
	return uninstallPlanPathIn(s.planDir, id)
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
	return config.WriteOwnedFile(path, append(data, '\n'), tuning.PermSecret)
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
