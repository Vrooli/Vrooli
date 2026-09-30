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
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/config"
)

const (
	uninstallPlanParameterA = 16
	uninstallPlanParameterB = 2
	uninstallPlanParameterC = 3
	uninstallPlanParameterD = 4
)

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
		if entry.Volatile || entry.Kind == EntryPackage || isContainerEntry(entry.Kind) {
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
	if strings.TrimSpace(request.PlanID) != "" {
		if !isPlanID(request.PlanID) {
			return UninstallPlan{}, &SafetyError{Code: "invalid_plan_id", Detail: "plan id contains unsafe characters"}
		}
		id = strings.TrimSpace(request.PlanID)
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
	plan.PlanHash = ComputePlanHash(plan)
	path, err := s.planPath(id)
	if err != nil {
		return UninstallPlan{}, err
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return UninstallPlan{}, fmt.Errorf("encode uninstall plan: %w", err)
	}
	if err := config.WriteOwnedFile(path, append(data, '\n'), tuning.PermSecret); err != nil {
		return UninstallPlan{}, fmt.Errorf("write uninstall plan: %w", err)
	}
	return plan, nil
}

//nolint:gocyclo // uninstall applies ordered artifact ownership and rollback decisions across install records.
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
	path, err := s.planPath(request.PlanID)
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
	if s.boundVerify != nil {
		if err := s.boundVerify(request.BreakGlass, request, plan, s.now()); err != nil {
			return RemovalReceipt{}, &SafetyError{Code: "break_glass_required", Detail: err.Error()}
		}
	}
	if plan.ID != request.PlanID || plan.Version != planVersion || plan.Target != target {
		return RemovalReceipt{}, &SafetyError{Code: "plan_stale", Path: path, Detail: "plan identity or target no longer matches"}
	}
	if plan.PlanHash == "" {
		plan.PlanHash = ComputePlanHash(plan)
	}
	if plan.PlanHash != ComputePlanHash(plan) {
		return RemovalReceipt{}, &SafetyError{Code: "plan_stale", Path: path, Detail: "resolved artifact list hash changed"}
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
	attempt := RemovalAttempt{StartedAt: s.now().UTC().Format(time.RFC3339), Applied: make([]RemovalReceiptEntry, 0)}
	for _, entry := range entries {
		if receiptEntryApplied(plan.Applied, entry) {
			continue
		}
		if err := s.remover.Remove(entry); err != nil {
			attempt.FinishedAt = s.now().UTC().Format(time.RFC3339)
			attempt.Error = fmt.Sprintf("remove frozen entry %s: %v", entry.Path, err)
			plan.Attempts = append(plan.Attempts, attempt)
			_ = s.writeUninstallPlan(plan)
			return removalReceipt(plan, target, request.AuthorizingUser), fmt.Errorf("remove frozen entry %s: %w", entry.Path, err)
		}
		removed := RemovalReceiptEntry{Scope: entry.Scope, Kind: entry.Kind, Path: entry.Path}
		plan.Applied = append(plan.Applied, removed)
		attempt.Applied = append(attempt.Applied, removed)

		if err := s.writeUninstallPlan(plan); err != nil {
			attempt.FinishedAt = s.now().UTC().Format(time.RFC3339)
			attempt.Error = "persist apply progress: " + err.Error()
			plan.Attempts = append(plan.Attempts, attempt)
			_ = s.writeUninstallPlan(plan)
			return removalReceipt(plan, target, request.AuthorizingUser), err
		}
	}
	attempt.FinishedAt = s.now().UTC().Format(time.RFC3339)
	plan.Attempts = append(plan.Attempts, attempt)
	if err := s.writeUninstallPlan(plan); err != nil {
		return removalReceipt(plan, target, request.AuthorizingUser), err
	}
	return removalReceipt(plan, target, request.AuthorizingUser), nil
}

// Verify reads a frozen plan and observes the artifact classes that can be
// checked safely without granting any mutation capability. It never invokes
// the remover and never re-runs discovery.
func (s *uninstallService) Verify(request UninstallRequest) (UninstallVerification, error) {
	if request.Mode != UninstallVerifyMode || !isPlanID(request.PlanID) {
		return UninstallVerification{}, &SafetyError{Code: "plan_required", Detail: "--verify requires a valid plan id"}
	}
	target, err := s.confirmTarget(request.ConfirmTarget)
	if err != nil {
		return UninstallVerification{}, err
	}
	path, err := s.planPath(request.PlanID)
	if err != nil {
		return UninstallVerification{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return UninstallVerification{}, &SafetyError{Code: "plan_unavailable", Path: path, Detail: err.Error()}
	}
	var plan UninstallPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return UninstallVerification{}, &SafetyError{Code: "plan_invalid", Path: path, Detail: err.Error()}
	}
	if plan.ID != request.PlanID || plan.Version != planVersion || plan.Target != target || (request.Scope != "" && request.Scope != plan.Scope) {
		return UninstallVerification{}, &SafetyError{Code: "plan_stale", Path: path, Detail: "plan identity, target, or scope no longer matches"}
	}
	if plan.PlanHash == "" || plan.PlanHash != ComputePlanHash(plan) {
		return UninstallVerification{}, &SafetyError{Code: "plan_stale", Path: path, Detail: "resolved artifact list hash changed"}
	}
	verification := UninstallVerification{
		PlanID: plan.ID, PlanHash: plan.PlanHash, Target: target, Scope: plan.Scope,
		VerifiedAt: s.now().UTC().Format(time.RFC3339), Complete: true,
		Removed:   append([]RemovalReceiptEntry(nil), plan.Applied...),
		Preserved: append([]UninstallDecision(nil), plan.Keep...), CannotAttribute: append([]UninstallDecision(nil), plan.CannotAttribute...),
	}
	for _, entry := range plan.RemoveOrEntries() {
		observed := RemovalReceiptEntry{Scope: entry.Scope, Kind: entry.Kind, Path: entry.Path}
		if isContainerEntry(entry.Kind) || entry.Kind == EntryPackage {
			verification.NotChecked = append(verification.NotChecked, observed)
			verification.Complete = false
			continue
		}
		if _, err := os.Lstat(entry.Path); err == nil {
			verification.Remaining = append(verification.Remaining, observed)
			verification.Complete = false
		} else if !errors.Is(err, os.ErrNotExist) {
			verification.NotChecked = append(verification.NotChecked, observed)
			verification.Complete = false
		}
	}
	return verification, nil
}

func (s *uninstallService) writeUninstallPlan(plan UninstallPlan) error {
	path, err := s.planPath(plan.ID)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("encode uninstall plan progress: %w", err)
	}
	return config.WriteOwnedFile(path, append(data, '\n'), tuning.PermSecret)
}

func receiptEntryApplied(applied []RemovalReceiptEntry, entry InstallEntry) bool {
	for _, prior := range applied {
		if prior.Scope == entry.Scope && prior.Kind == entry.Kind && prior.Path == entry.Path {
			return true
		}
	}
	return false
}

func removalReceipt(plan UninstallPlan, target, authorizingUser string) RemovalReceipt {
	return RemovalReceipt{
		PlanID: plan.ID, PlanHash: plan.PlanHash, Target: target, Scope: plan.Scope,
		RemovedAt: time.Now().UTC().Format(time.RFC3339), AuthorizingUser: strings.TrimSpace(authorizingUser),
		Removed: append([]RemovalReceiptEntry(nil), plan.Applied...), Preserved: append([]UninstallDecision(nil), plan.Keep...),
		CannotAttribute: append([]UninstallDecision(nil), plan.CannotAttribute...), Attempts: append([]RemovalAttempt(nil), plan.Attempts...),
	}
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
	if strings.TrimSpace(request.PlanID) != "" && !isPlanID(request.PlanID) {
		return &SafetyError{Code: "invalid_plan_id", Detail: "plan id contains unsafe characters"}
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
		return uninstallPlanParameterB
	case EntryImage:
		return uninstallPlanParameterC
	default:
		return uninstallPlanParameterD
	}
}

//nolint:gocyclo // record normalization handles legacy field aliases and mutually exclusive ownership forms.
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
		if err := validateRecordEntry(entry, home); err != nil {
			return err
		}
	}
	return validateRuntimeProviders(record.RuntimeProviders)
}

func validateRecordEntry(entry InstallEntry, home string) error {
	if !validScope(entry.Scope) {
		return fmt.Errorf("install record: invalid scope %q", entry.Scope)
	}
	if !validEntryKind(entry.Kind) {
		return fmt.Errorf("install record: invalid kind %q", entry.Kind)
	}
	if entry.Kind == EntryPackage {
		return validatePackageProvenance(entry)
	}
	if isContainerEntry(entry.Kind) {
		return validateContainerProvenance(entry)
	}
	return validateEntry(entry, home)
}

func validEntryKind(kind InstallEntryKind) bool {
	return kind == EntryFile || kind == EntryDirectory || kind == EntryBinary || kind == EntryService || kind == EntryPackage || isContainerEntry(kind)
}

func validatePackageProvenance(entry InstallEntry) error {
	if strings.TrimSpace(entry.Provenance.PackageName) == "" {
		return fmt.Errorf("install record: package entry %q has no package name", entry.Path)
	}
	if err := validateObservedBefore(entry.Provenance.ObservedBefore, ""); err != nil {
		return err
	}
	return validateInstallAction(entry.Provenance.Action, "")
}

func validateContainerProvenance(entry InstallEntry) error {
	if strings.TrimSpace(entry.Path) == "" {
		return fmt.Errorf("install record: container artifact name is required")
	}
	if err := validateObservedBefore(entry.Provenance.ObservedBefore, ""); err != nil {
		return err
	}
	return validateInstallAction(entry.Provenance.Action, "")
}

func validateObservedBefore(state ObservedBefore, subject string) error {
	if state != ObservedPresent && state != ObservedAbsent && state != ObservedUnknown {
		if subject == "" {
			return fmt.Errorf("install record: invalid observed-before state %q", state)
		}
		return fmt.Errorf("install record: invalid %s observed-before state %q", subject, state)
	}
	return nil
}

func validateInstallAction(action InstallAction, subject string) error {
	if action != "" && action != ActionInstalled && action != ActionAdopted && action != ActionUpgraded && action != ActionNoOp {
		if subject == "" {
			return fmt.Errorf("install record: invalid action %q", action)
		}
		return fmt.Errorf("install record: invalid %s action %q", subject, action)
	}
	return nil
}

func validateRuntimeProviders(providers []RuntimeProviderProvenance) error {
	for _, provider := range providers {
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
	return InstallRecord{Version: installRecordVersion, Prefix: filepath.Join(home, repocontractmeta.ProjectConfigDir), Entries: []InstallEntry{}, RuntimeProviders: []RuntimeProviderProvenance{}}
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
			return nil
		}
		probe = filepath.Dir(probe)
	}
}

func verifyFrozenDisk(plan UninstallPlan) error {
	for _, expected := range plan.Disk {
		if receiptEntryAppliedPath(plan.Applied, expected.Path) {
			continue
		}
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

func receiptEntryAppliedPath(applied []RemovalReceiptEntry, path string) bool {
	for _, entry := range applied {
		if filepath.Clean(entry.Path) == filepath.Clean(path) {
			return true
		}
	}
	return false
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
	raw := make([]byte, uninstallPlanParameterA)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate uninstall plan id: %w", err)
	}
	return hex.EncodeToString(raw), nil
}
