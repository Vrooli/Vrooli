package destinationreadiness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"data-backup-manager/internal/sysmounts"
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

// Remediator executes host volume remediation. It is a distinct seam from
// Preparer because remediation is not this scenario's to perform: the control
// plane owns host state, and the implementation of this interface is a client
// of it, not a private host-repair implementation.
type Remediator interface {
	Supported(action PreparationAction) (bool, string)
	Remediate(ctx context.Context, plan Plan, dryRun bool) (RemediationOutcome, error)
}

// Service owns readiness rules and preparation safety gates.
type Service struct {
	inspector       Inspector
	deviceInspector DeviceInspector
	preparer        Preparer
	remediator      Remediator
}

// NewService constructs a readiness service. Device-scoped inspection is wired
// when the inspector provides it, so a path-only fake stays a valid Inspector.
func NewService(inspector Inspector, preparer Preparer) *Service {
	s := &Service{inspector: inspector, preparer: preparer}
	if devices, ok := inspector.(DeviceInspector); ok {
		s.deviceInspector = devices
	}
	return s
}

// WithRemediator attaches the host remediation client.
func (s *Service) WithRemediator(remediator Remediator) *Service {
	s.remediator = remediator
	return s
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
	inspection, err := s.inspectForPlan(ctx, location, in)
	if err != nil {
		return Plan{}, err
	}
	if in.Action.IsRemediation() {
		if err := guardRemediationTarget(location, in, inspection); err != nil {
			return Plan{}, err
		}
		// A remediation sequence passes through an unmounted state by design, so
		// there is deliberately no "the destination directory must exist" gate
		// here: it would reject a resumed sequence for being exactly where the
		// previous step left it. What must hold is that a specific disk was
		// identified, which the two checks below require.
		if inspection.Identity.DevicePath == "" {
			return Plan{}, ErrPreparationRefused{Reason: "remediation needs a device path; this host did not report one for the destination"}
		}
		if !inspection.Identity.StrongIdentity() {
			return Plan{}, ErrPreparationRefused{Reason: "remediation needs a device UUID or serial to bind the plan to this disk; this host reported neither"}
		}
	} else if !inspection.LocationExists || !inspection.LocationIsDirectory {
		return Plan{}, ErrPreparationRefused{Reason: "destination path must already exist as a directory; refusing to prepare an unverified location"}
	}
	if !in.ExpectedDevice.Matches(DeviceIdentity{}) && !in.ExpectedDevice.MatchesDevice(inspection.Identity) {
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

	// Repair can discard inconsistent filesystem metadata, so it carries the
	// same acknowledgement gate as the data-destroying actions. Unmount, check
	// and mount change attachment or prove state; they do not.
	destructive := in.Action == ActionFormat || in.Action == ActionClearDirectory ||
		in.Action == ActionRelabel || in.Action == ActionRepairFilesystem
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

// inspectForPlan chooses the inspection lens for a plan. A remediation caller
// that names an expected device gets the device lens, because once the volume
// is unmounted its path no longer leads anywhere near it.
func (s *Service) inspectForPlan(ctx context.Context, location string, in PlanInput) (Inspection, error) {
	if in.Action.IsRemediation() && strings.TrimSpace(in.ExpectedDevice.DevicePath) != "" {
		if s.deviceInspector == nil {
			return Inspection{}, ErrPreparationRefused{Reason: "device-scoped inspection is unavailable on this host; cannot address an unmounted destination by device"}
		}
		inspection, err := s.deviceInspector.InspectDevice(ctx, in.ExpectedDevice)
		if err != nil {
			return Inspection{}, fmt.Errorf("inspect destination device: %w", err)
		}
		return inspection, nil
	}
	inspection, err := s.inspector.Inspect(ctx, location)
	if err != nil {
		return Inspection{}, fmt.Errorf("inspect destination: %w", err)
	}
	return inspection, nil
}

// guardRemediationTarget refuses a remediation plan that resolved to the wrong
// volume.
//
// A destination path is only a path. Once its volume is unmounted, the path
// becomes an ordinary directory on whatever filesystem owns its parent —
// usually the root filesystem — and "the volume that owns this path" silently
// becomes the host's system disk. Planning would then produce a real,
// confirmable plan naming /dev/<root>, which is precisely the mistake an
// operator is least able to catch. The destination volume being absent must
// read as "it is not here", never as "here is a different disk".
func guardRemediationTarget(location string, in PlanInput, inspection Inspection) error {
	mountpoint := cleanPath(inspection.Identity.Mountpoint)
	if !isSystemVolumePath(mountpoint) {
		return nil
	}
	if cleanPath(location) != mountpoint {
		return ErrPreparationRefused{Reason: fmt.Sprintf(
			"destination volume is not mounted at %s; the path now resolves to the host volume %s mounted at %s — plug the destination in, or name it with an expected device path",
			location, inspection.Identity.DevicePath, mountpoint)}
	}
	return ErrPreparationRefused{Reason: fmt.Sprintf(
		"refusing to remediate a system volume: %s is mounted at %s", inspection.Identity.DevicePath, mountpoint)}
}

// systemVolumePaths are mountpoints whose volume keeps the host running.
// Matching is by prefix so /boot/efi is covered by /boot.
var systemVolumePaths = []string{"/", "/boot", "/usr", "/etc", "/var", "/home", "/root", "/System", "/Applications", "/Library"}

func isSystemVolumePath(mountpoint string) bool {
	mountpoint = cleanPath(mountpoint)
	if mountpoint == "" || mountpoint == "." {
		return false
	}
	if mountpoint == "/" {
		return true
	}
	for _, sys := range systemVolumePaths {
		if sys == "/" {
			continue
		}
		if strings.EqualFold(mountpoint, sys) || strings.HasPrefix(strings.ToLower(mountpoint), strings.ToLower(sys)+"/") {
			return true
		}
	}
	return false
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
	inspection, err := s.reinspect(ctx, in.Plan)
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("inspect destination: %w", err)
	}
	if !s.identityStillMatches(in.Plan, inspection) {
		return ExecuteResult{}, ErrPreparationRefused{Reason: "device identity changed since plan creation"}
	}
	result := ExecuteResult{DryRun: in.DryRun, Action: in.Plan.Action, Location: in.Plan.TargetPath}

	if in.Plan.Action.IsRemediation() {
		if s.remediator == nil {
			return ExecuteResult{}, ErrPreparationRefused{Reason: "no host remediation client configured"}
		}
		outcome, remErr := s.remediator.Remediate(ctx, in.Plan, in.DryRun)
		result.Status, result.Changed = outcome.Status, outcome.Changed
		result.Backend, result.Command, result.Detail = outcome.Backend, outcome.Command, outcome.Detail
		result.OperatorCommand, result.RefusalReason = outcome.OperatorCommand, outcome.RefusalReason
		result.Consistent = outcome.Consistent
		if remErr != nil {
			return result, fmt.Errorf("remediate destination volume: %w", remErr)
		}
		return result, nil
	}

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

// reinspect re-observes the plan's target immediately before executing it.
// Remediation is inspected by device because the sequence deliberately unmounts
// the volume: a path-scoped observation would simply report the destination as
// gone and refuse the step that is supposed to follow.
func (s *Service) reinspect(ctx context.Context, plan Plan) (Inspection, error) {
	if !plan.Action.IsRemediation() {
		return s.inspector.Inspect(ctx, plan.Location)
	}
	if s.deviceInspector == nil {
		return Inspection{}, ErrPreparationRefused{Reason: "device-scoped inspection is unavailable on this host; remediation cannot re-verify the disk"}
	}
	return s.deviceInspector.InspectDevice(ctx, plan.Identity)
}

// identityStillMatches re-checks that the approved disk is the one present.
// Remediation compares by device, excluding the mountpoint it is authorized to
// change; every other action keeps the stricter path-bound comparison.
func (s *Service) identityStillMatches(plan Plan, inspection Inspection) bool {
	if plan.Action.IsRemediation() {
		return plan.Identity.MatchesDevice(inspection.Identity)
	}
	return plan.Identity.Matches(inspection.Identity)
}

func buildReport(location string, inspection Inspection, in AnalyzeInput) Report {
	subdir := strings.TrimSpace(in.ProposedSubdir)
	if subdir == "" {
		subdir = "vrooli-backups"
	}
	recLocation := filepath.Join(location, subdir)
	checks := []CheckResult{
		checkLocation(inspection),
		checkMountedReadWrite(inspection),
		checkFilesystemState(inspection),
		checkDirectoryEvidence(inspection),
		checkSeparateRoot(location, in.ProtectedPaths),
		checkExisting(location, recLocation, in.ExistingDestinations),
		checkFilesystem(inspection.Identity.Filesystem, inspection.MountDriver, in.CrossPlatformRequired),
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
		Platform:                       inspection.Platform,
		Confidence:                     identityConfidence(inspection),
		EvidenceSource:                 "mounted-volume-and-bounded-directory-read",
		ObservedAt:                     time.Now().UTC(),
		RepairSteps:                    repairSteps(inspection, checks),
		ReadOnlyCause:                  inspection.ReadOnlyCause,
		FilesystemState:                inspection.FilesystemState,
		DeviceWriteProtected:           inspection.DeviceWriteProtected,
		Mounted:                        inspection.Mounted,
	}
}

func identityConfidence(i Inspection) string {
	if i.Identity.UUID != "" || i.Identity.Serial != "" {
		return "high"
	}
	if i.Identity.DevicePath != "" && i.Identity.Mountpoint != "" {
		return "medium"
	}
	return "low"
}

// repairSteps turns the check results into an ordered remediation sequence.
// Steps are driven by the attributed cause rather than by the symptom, so an
// operator is never handed a list of alternatives to choose between.
func repairSteps(i Inspection, checks []CheckResult) []string {
	steps := []string{"preserve the diagnostic evidence and do not format or clear the destination"}
	seen := map[string]bool{}
	add := func(step string) {
		if step == "" || seen[step] {
			return
		}
		seen[step] = true
		steps = append(steps, step)
	}
	for _, c := range checks {
		switch c.Code {
		case "mounted_read_write", "destination_dirty":
			if c.Severity == SeverityFail {
				add(readOnlyRepairStep(i))
			}
		case "filesystem_suitability":
			if c.Severity == SeverityWarning || c.Severity == SeverityFail {
				// A driver veto and a format-limits warning need different
				// actions; the check already names the one that applies, so
				// prefer it over the generic platform-matrix advice.
				if strings.TrimSpace(c.NextAction) != "" {
					add(c.NextAction)
				} else {
					add("choose a filesystem whose limits match the declared platform support matrix")
				}
			}
		case "directory_inaccessible":
			if c.Severity == SeverityFail {
				add("restore directory access and re-run the read-only diagnosis")
			}
		}
	}
	steps = append(steps, "recheck device identity, repository readiness, and then run a verified backup")
	return steps
}

// readOnlyRepairStep names the single remediation that fits the attributed
// cause. An unattributed cause deliberately yields an inspect-first step: a
// blind repair on an unexplained read-only mount is how data gets lost.
func readOnlyRepairStep(i Inspection) string {
	switch {
	case i.DeviceWriteProtected || i.ReadOnlyCause == sysmounts.CauseDeviceWriteProtected:
		return "clear the block-device write protection; the filesystem is not the blocker and repairing it would change nothing"
	case i.ReadOnlyCause == sysmounts.CauseFilesystemDirty ||
		i.FilesystemState == sysmounts.FilesystemStateDirty ||
		i.FilesystemState == sysmounts.FilesystemStateNeedsCheck:
		return "check the filesystem, repair it under explicit confirmation, then remount read/write and re-verify the repository before trusting a backup"
	case i.ReadOnlyCause == sysmounts.CauseMountOption:
		return "read-only was requested for this mount; change the declared mount options rather than repairing a filesystem that is not damaged"
	default:
		return "attribute the read-only cause before acting; this host could not determine it, and an unexplained read-only mount must not be repaired blindly"
	}
}

func checkFilesystemState(i Inspection) CheckResult {
	evidence := ""
	if i.StateEvidence != "" {
		evidence = " (evidence: " + i.StateEvidence + ")"
	}
	switch i.FilesystemState {
	case sysmounts.FilesystemStateDirty:
		return CheckResult{Code: "destination_dirty", Severity: SeverityFail, Message: "filesystem reports a dirty state" + evidence, NextAction: "check and repair the filesystem, then remount and re-run readiness"}
	case sysmounts.FilesystemStateNeedsCheck:
		return CheckResult{Code: "destination_dirty", Severity: SeverityFail, Message: "filesystem reports that a check is required" + evidence, NextAction: "check and repair the filesystem, then remount and re-run readiness"}
	case sysmounts.FilesystemStateClean:
		return CheckResult{Code: "filesystem_state", Severity: SeverityPass, Message: "filesystem reports a clean state" + evidence}
	default:
		return CheckResult{Code: "filesystem_state", Severity: SeverityUnknown, Message: "filesystem dirty/needs-check state is not exposed on this host" + evidence}
	}
}

func checkDirectoryEvidence(i Inspection) CheckResult {
	if !i.LocationExists || !i.LocationIsDirectory {
		return CheckResult{Code: "directory_inaccessible", Severity: SeverityFail, Message: "destination directory does not exist or is not a directory"}
	}
	if i.ReadDirError != "" {
		return CheckResult{Code: "directory_inaccessible", Severity: SeverityFail, Message: "destination directory could not be inspected"}
	}
	return CheckResult{Code: "directory_inaccessible", Severity: SeverityPass, Message: "bounded directory inspection completed"}
}

func checkLocation(i Inspection) CheckResult {
	if !i.LocationExists {
		return CheckResult{Code: "destination_missing", Severity: SeverityFail, Message: "destination path does not exist", NextAction: "mount the intended volume or create the destination directory, then re-run readiness"}
	}
	if !i.LocationIsDirectory {
		return CheckResult{Code: "destination_inaccessible", Severity: SeverityFail, Message: "destination path is not a directory", NextAction: "choose a directory on the intended volume, then re-run readiness"}
	}
	return CheckResult{Code: "destination_present", Severity: SeverityPass, Message: "destination directory exists"}
}

func checkMountedReadWrite(i Inspection) CheckResult {
	if !i.LocationExists || !i.LocationIsDirectory {
		return CheckResult{Code: "mounted_read_write", Severity: SeverityUnknown, Message: "parent volume was inspected, but the destination path is not writable evidence"}
	}
	if !i.ReadOnly {
		return CheckResult{Code: "mounted_read_write", Severity: SeverityPass, Message: "mounted read/write"}
	}
	// A bare "mounted read-only" sends an operator hunting through three
	// unrelated remediations. Name the attributed cause so exactly one applies.
	check := CheckResult{Code: "mounted_read_write", Severity: SeverityFail}
	switch i.ReadOnlyCause {
	case sysmounts.CauseDeviceWriteProtected:
		check.Message = "mounted read-only because the block device is write-protected"
		check.NextAction = "clear the device write-protect (hardware switch or block-layer read-only flag); filesystem repair cannot restore writes"
	case sysmounts.CauseFilesystemDirty:
		check.Message = "mounted read-only because the filesystem carries a dirty or needs-check flag and the driver refused a read/write mount"
		check.NextAction = "check and repair the filesystem, then remount read/write and re-run readiness"
	case sysmounts.CauseMountOption:
		check.Message = "mounted read-only because read-only was explicitly requested for this mount"
		check.NextAction = "change the declared mount options if this destination is meant to be writable"
	default:
		check.Message = "mounted read-only for a cause this host could not attribute"
		check.NextAction = "inspect mount and device state before attempting any repair; an unattributed read-only mount must not be repaired blindly"
	}
	return check
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

// kernelFaultingDrivers are mount drivers whose write path is known to fault
// in kernel context rather than returning an error to the calling process. A
// backup repository served by one of these can take the whole host down, so
// they are refused as write destinations regardless of the on-disk format.
//
// `ntfs3` is here because it did exactly that on 2026-08-19: a routine backup
// write reached `BUG at fs/iomap/buffered-io.c:1061` through
// ntfs_file_write_iter and panicked the machine. The same volume served by
// ntfs-3g would have surfaced an ordinary I/O error to the backup process.
var kernelFaultingDrivers = map[string]string{
	"ntfs3": "the in-kernel ntfs3 driver faults in kernel context on write; a repository fault can panic the host rather than failing the backup",
}

// userspaceDrivers are mount drivers that run outside the kernel, so a fault
// is contained to the calling process. They are not endorsements — a userspace
// driver can still be slow or lossy — but they cannot take the host down.
var userspaceDrivers = map[string]struct{}{
	"fuseblk": {},
	"fuse":    {},
}

// checkFilesystem rates a destination's suitability from both the on-disk
// format and the driver currently serving it. The driver is checked first and
// can veto: format suitability is irrelevant if writing through the mount can
// panic the host.
func checkFilesystem(fs, driver string, crossPlatform bool) CheckResult {
	normalizedDriver := strings.ToLower(strings.TrimSpace(driver))
	if reason, faulting := kernelFaultingDrivers[normalizedDriver]; faulting {
		return CheckResult{
			Code:       "filesystem_suitability",
			Severity:   SeverityFail,
			Message:    "destination is mounted with the " + normalizedDriver + " driver: " + reason,
			NextAction: "reformat the drive to a filesystem with a first-class Linux driver, or remount it through a userspace driver such as ntfs-3g, before using it as a backup destination",
		}
	}

	normalizedFS := strings.ToLower(strings.TrimSpace(fs))
	_, userspace := userspaceDrivers[normalizedDriver]

	switch normalizedFS {
	case "ext4":
		if crossPlatform {
			return CheckResult{Code: "filesystem_suitability", Severity: SeverityWarning, Message: "ext4 is recommended for Linux-only backups but is not broadly cross-platform"}
		}
		return CheckResult{Code: "filesystem_suitability", Severity: SeverityPass, Message: "ext4 is recommended for Linux backup drives"}
	case "exfat":
		return CheckResult{Code: "filesystem_suitability", Severity: SeverityPass, Message: "exFAT is usable for cross-platform backup drives"}
	case "ntfs", "ntfs3":
		// NTFS is never a Pass. Even served by a userspace driver it carries
		// restore-fidelity caveats, and an unmounted volume gives no driver
		// evidence at all — the driver it *would* get on mount is unknown, and
		// on Linux the kernel driver is the automount default.
		if userspace {
			return CheckResult{
				Code:       "filesystem_suitability",
				Severity:   SeverityWarning,
				Message:    "NTFS served by a userspace driver contains faults to the backup process, but restore fidelity still depends on the driver; a native Linux filesystem is preferred",
				NextAction: "prefer ext4 for a Linux-only backup drive, or exFAT when the drive must stay cross-platform",
			}
		}
		return CheckResult{
			Code:       "filesystem_suitability",
			Severity:   SeverityWarning,
			Message:    "NTFS access and restore fidelity depend on which driver serves the mount, and this volume is not currently mounted through a known-safe one",
			NextAction: "confirm the mount driver before trusting this destination; on Linux the default in-kernel ntfs3 driver is refused for backup writes",
		}
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
	if action.IsRemediation() {
		if s.remediator == nil {
			return false, "host remediation client is not configured"
		}
		return s.remediator.Supported(action)
	}
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
	case ActionUnmount:
		return fmt.Sprintf("UNMOUNT %s SIZE %d", id.DevicePath, id.TotalBytes)
	case ActionCheckFilesystem:
		return fmt.Sprintf("CHECK %s FS %s SIZE %d", id.DevicePath, strings.TrimSpace(id.Filesystem), id.TotalBytes)
	case ActionRepairFilesystem:
		// The phrase names the disk, not just the action: an operator who has
		// two external drives attached must be able to see which one they are
		// authorising a metadata-modifying repair on.
		return fmt.Sprintf("REPAIR %s FS %s UUID %s SIZE %d", id.DevicePath, strings.TrimSpace(id.Filesystem), strings.TrimSpace(id.UUID), id.TotalBytes)
	case ActionMountReadWrite:
		return fmt.Sprintf("MOUNT %s READ-WRITE SIZE %d", id.DevicePath, id.TotalBytes)
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
