package destinationreadiness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// Inspector is the read-only seam for platform/device inspection.
type Inspector interface {
	Inspect(ctx context.Context, location string) (Inspection, error)
}

// Preparer is the only seam allowed to perform preparation effects.
type Preparer interface {
	Supported(action PreparationAction) (bool, string)
	Execute(ctx context.Context, plan Plan) error
}

// Service owns readiness rules and preparation safety gates.
type Service struct {
	inspector Inspector
	preparer  Preparer
}

// NewService constructs a readiness service.
func NewService(inspector Inspector, preparer Preparer) *Service {
	return &Service{inspector: inspector, preparer: preparer}
}

// Analyze evaluates a destination candidate without writing to it.
func (s *Service) Analyze(ctx context.Context, in AnalyzeInput) (Report, error) {
	location := cleanPath(in.Location)
	if location == "" {
		return Report{}, ErrInvalidReadiness{Field: "location", Reason: "required"}
	}
	if s.inspector == nil {
		return Report{}, ErrInvalidReadiness{Field: "inspector", Reason: "required"}
	}
	inspection, err := s.inspector.Inspect(ctx, location)
	if err != nil {
		return Report{}, fmt.Errorf("inspect destination: %w", err)
	}
	return buildReport(location, inspection, in), nil
}

// PlanPreparation creates a non-mutating preparation plan.
func (s *Service) PlanPreparation(ctx context.Context, in PlanInput) (Plan, error) {
	location := cleanPath(in.Location)
	if location == "" {
		return Plan{}, ErrInvalidReadiness{Field: "location", Reason: "required"}
	}
	if in.Action == "" {
		return Plan{}, ErrInvalidReadiness{Field: "action", Reason: "required"}
	}
	if s.inspector == nil {
		return Plan{}, ErrInvalidReadiness{Field: "inspector", Reason: "required"}
	}
	inspection, err := s.inspector.Inspect(ctx, location)
	if err != nil {
		return Plan{}, fmt.Errorf("inspect destination: %w", err)
	}
	if !in.ExpectedDevice.Matches(DeviceIdentity{}) && !in.ExpectedDevice.Matches(inspection.Identity) {
		return Plan{}, ErrPreparationRefused{Reason: "observed device identity does not match expected identity"}
	}

	targetPath := location
	if in.Action == ActionCreateSubdir {
		subdir := strings.TrimSpace(in.DesiredSubdir)
		if subdir == "" {
			subdir = "vrooli-backups"
		}
		if strings.ContainsAny(subdir, `/\`) || subdir == "." || subdir == ".." {
			return Plan{}, ErrInvalidReadiness{Field: "desired_subdir", Reason: "must be a single directory name"}
		}
		targetPath = filepath.Join(location, subdir)
	}
	if overlapsAny(targetPath, normalizePaths(in.ProtectedPaths)) {
		return Plan{}, ErrPreparationRefused{Reason: "target path overlaps protected data"}
	}

	destructive := in.Action == ActionFormat || in.Action == ActionClearDirectory || in.Action == ActionRelabel
	supported, unsupportedReason := s.supported(in.Action)
	plan := Plan{
		ID:                 planID(in.Action, inspection.Identity, targetPath, in.DesiredFS, in.DesiredLabel),
		Action:             in.Action,
		Location:           location,
		TargetPath:         targetPath,
		Identity:           inspection.Identity,
		DesiredLabel:       strings.TrimSpace(in.DesiredLabel),
		DesiredFS:          strings.TrimSpace(in.DesiredFS),
		RequiresConfirm:    true,
		Destructive:        destructive,
		ConfirmationPhrase: confirmationPhrase(in.Action, inspection.Identity, in.DesiredFS, in.DesiredLabel, targetPath),
		Supported:          supported,
		UnsupportedReason:  unsupportedReason,
	}
	return plan, nil
}

// ExecutePreparation executes a previously generated preparation plan. It
// defaults to safe behavior: dry-run performs all validation but never calls the
// preparer.
func (s *Service) ExecutePreparation(ctx context.Context, in ExecuteInput) (ExecuteResult, error) {
	if in.Plan.ID == "" {
		return ExecuteResult{}, ErrInvalidReadiness{Field: "plan", Reason: "required"}
	}
	if !in.Plan.Supported {
		return ExecuteResult{}, ErrPreparationRefused{Reason: "unsupported action: " + in.Plan.UnsupportedReason}
	}
	if in.Plan.RequiresConfirm && strings.TrimSpace(in.Confirmation) != in.Plan.ConfirmationPhrase {
		return ExecuteResult{}, ErrPreparationRefused{Reason: "confirmation phrase did not match"}
	}
	if in.Plan.Destructive && !in.AcknowledgeDataLoss {
		return ExecuteResult{}, ErrPreparationRefused{Reason: "data loss acknowledgement required"}
	}
	if s.inspector == nil {
		return ExecuteResult{}, ErrInvalidReadiness{Field: "inspector", Reason: "required"}
	}
	inspection, err := s.inspector.Inspect(ctx, in.Plan.Location)
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("inspect destination: %w", err)
	}
	if !in.Plan.Identity.Matches(inspection.Identity) {
		return ExecuteResult{}, ErrPreparationRefused{Reason: "device identity changed since plan creation"}
	}
	result := ExecuteResult{DryRun: in.DryRun, Action: in.Plan.Action, Location: in.Plan.TargetPath}
	if in.DryRun {
		return result, nil
	}
	if s.preparer == nil {
		return ExecuteResult{}, ErrPreparationRefused{Reason: "no preparer configured"}
	}
	if err := s.preparer.Execute(ctx, in.Plan); err != nil {
		return ExecuteResult{}, fmt.Errorf("execute preparation: %w", err)
	}
	return result, nil
}

func buildReport(location string, inspection Inspection, in AnalyzeInput) Report {
	subdir := strings.TrimSpace(in.ProposedSubdir)
	if subdir == "" {
		subdir = "vrooli-backups"
	}
	recLocation := filepath.Join(location, subdir)
	checks := []CheckResult{
		checkMountedReadWrite(inspection),
		checkSeparateRoot(location, in.ProtectedPaths),
		checkExisting(location, recLocation, in.ExistingDestinations),
		checkFilesystem(inspection.Identity.Filesystem, in.CrossPlatformRequired),
		checkCapacity(inspection.FreeBytes, in.SelectedTargetBytes, in.RetentionCopies),
		checkRootNonEmpty(inspection),
		checkInstallerMedia(inspection),
		checkSubdir(location, recLocation),
	}
	overall := aggregate(checks)
	action := "use_subdirectory"
	if overall == SeverityFail {
		action = "pick_different_drive_or_remediate"
	}
	return Report{
		Location:                       location,
		OverallSeverity:                overall,
		Identity:                       inspection.Identity,
		Checks:                         checks,
		RecommendedDestinationLocation: recLocation,
		RecommendedAction:              action,
	}
}

func checkMountedReadWrite(i Inspection) CheckResult {
	if i.ReadOnly {
		return CheckResult{Code: "mounted_read_write", Severity: SeverityFail, Message: "mounted read-only"}
	}
	return CheckResult{Code: "mounted_read_write", Severity: SeverityPass, Message: "mounted read/write"}
}

func checkSeparateRoot(location string, protected []string) CheckResult {
	if overlapsAny(location, normalizePaths(protected)) {
		return CheckResult{Code: "separate_root", Severity: SeverityFail, Message: "location overlaps protected data"}
	}
	return CheckResult{Code: "separate_root", Severity: SeverityPass, Message: "location is separate from protected data"}
}

func checkExisting(location, recLocation string, existing []string) CheckResult {
	for _, e := range normalizePaths(existing) {
		if e == location || e == recLocation {
			return CheckResult{Code: "destination_not_registered", Severity: SeverityWarning, Message: "destination is already configured"}
		}
	}
	return CheckResult{Code: "destination_not_registered", Severity: SeverityPass, Message: "destination is not already configured"}
}

func checkFilesystem(fs string, crossPlatform bool) CheckResult {
	switch strings.ToLower(strings.TrimSpace(fs)) {
	case "ext4":
		if crossPlatform {
			return CheckResult{Code: "filesystem_suitability", Severity: SeverityWarning, Message: "ext4 is recommended for Linux-only backups but is not broadly cross-platform"}
		}
		return CheckResult{Code: "filesystem_suitability", Severity: SeverityPass, Message: "ext4 is recommended for Linux backup drives"}
	case "exfat":
		return CheckResult{Code: "filesystem_suitability", Severity: SeverityPass, Message: "exFAT is usable for cross-platform backup drives"}
	case "ntfs", "ntfs3":
		return CheckResult{Code: "filesystem_suitability", Severity: SeverityPass, Message: "NTFS is usable for filesystem backup repositories on this mounted drive"}
	case "vfat", "fat32", "msdos":
		return CheckResult{Code: "filesystem_suitability", Severity: SeverityWarning, Message: "FAT32 has a 4 GiB per-file limit and is not recommended for serious backup repositories"}
	case "":
		return CheckResult{Code: "filesystem_suitability", Severity: SeverityUnknown, Message: "filesystem type is unknown"}
	default:
		return CheckResult{Code: "filesystem_suitability", Severity: SeverityWarning, Message: "filesystem has not been validated for backup repository use"}
	}
}

func checkCapacity(freeBytes, targetBytes int64, copies int) CheckResult {
	if targetBytes <= 0 {
		return CheckResult{Code: "capacity", Severity: SeverityUnknown, Message: "target size not selected; capacity fit is unknown"}
	}
	if copies <= 0 {
		copies = 1
	}
	required := targetBytes * int64(copies)
	if freeBytes < required {
		return CheckResult{Code: "capacity", Severity: SeverityFail, Message: "free space is below selected target estimate"}
	}
	if freeBytes < required+required/10 {
		return CheckResult{Code: "capacity", Severity: SeverityWarning, Message: "free space is close to selected target estimate"}
	}
	return CheckResult{Code: "capacity", Severity: SeverityPass, Message: "free space fits selected target estimate"}
}

func checkRootNonEmpty(i Inspection) CheckResult {
	if i.NonEmptyRoot {
		return CheckResult{Code: "root_non_empty", Severity: SeverityWarning, Message: "mount root is not empty; use a dedicated backup subdirectory"}
	}
	return CheckResult{Code: "root_non_empty", Severity: SeverityPass, Message: "mount root is empty"}
}

func checkInstallerMedia(i Inspection) CheckResult {
	if i.InstallerMedia {
		return CheckResult{Code: "installer_media_detected", Severity: SeverityWarning, Message: "installer-media files detected; review before using this drive"}
	}
	return CheckResult{Code: "installer_media_detected", Severity: SeverityPass, Message: "installer-media signatures not detected"}
}

func checkSubdir(location, recLocation string) CheckResult {
	if cleanPath(location) == cleanPath(recLocation) {
		return CheckResult{Code: "subdir_recommended", Severity: SeverityWarning, Message: "mount root selected; a dedicated backup subdirectory is recommended"}
	}
	return CheckResult{Code: "subdir_recommended", Severity: SeverityPass, Message: "dedicated backup subdirectory is recommended"}
}

func aggregate(checks []CheckResult) CheckSeverity {
	overall := SeverityPass
	for _, c := range checks {
		switch c.Severity {
		case SeverityFail:
			return SeverityFail
		case SeverityWarning:
			if overall != SeverityUnknown {
				overall = SeverityWarning
			}
		case SeverityUnknown:
			if overall == SeverityPass {
				overall = SeverityUnknown
			}
		}
	}
	return overall
}

func (s *Service) supported(action PreparationAction) (bool, string) {
	if s.preparer != nil {
		return s.preparer.Supported(action)
	}
	if runtime.GOOS != "linux" {
		return false, "preparation execution is not implemented on " + runtime.GOOS
	}
	return false, "no preparer configured"
}

func confirmationPhrase(action PreparationAction, id DeviceIdentity, fs, label, targetPath string) string {
	switch action {
	case ActionFormat:
		return fmt.Sprintf("FORMAT %s AS %s LABEL %s SIZE %d", id.DevicePath, strings.TrimSpace(fs), strings.TrimSpace(label), id.TotalBytes)
	case ActionClearDirectory:
		return fmt.Sprintf("CLEAR %s ON %s SIZE %d", targetPath, id.DevicePath, id.TotalBytes)
	case ActionRelabel:
		return fmt.Sprintf("RELABEL %s TO %s SIZE %d", id.DevicePath, strings.TrimSpace(label), id.TotalBytes)
	default:
		return fmt.Sprintf("PREPARE %s ON %s SIZE %d", targetPath, id.DevicePath, id.TotalBytes)
	}
}

func planID(action PreparationAction, id DeviceIdentity, targetPath, fs, label string) string {
	sum := sha256.Sum256([]byte(string(action) + "|" + id.StableString() + "|" + targetPath + "|" + fs + "|" + label))
	return hex.EncodeToString(sum[:])[:16]
}

func cleanPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	return filepath.Clean(p)
}

func normalizePaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p = cleanPath(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func overlapsAny(loc string, protected []string) bool {
	loc = cleanPath(loc)
	for _, p := range protected {
		if pathsOverlap(loc, p) {
			return true
		}
	}
	return false
}

func pathsOverlap(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return a == b || within(a, b) || within(b, a)
}

func within(child, parent string) bool {
	if child == parent {
		return true
	}
	sep := string(filepath.Separator)
	if parent == sep {
		return strings.HasPrefix(child, sep)
	}
	return strings.HasPrefix(child, parent+sep)
}
