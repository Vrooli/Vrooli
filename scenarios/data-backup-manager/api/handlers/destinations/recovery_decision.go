package destinations

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"data-backup-manager/internal/destinationreadiness"

	"github.com/vrooli/api-core/operatordecision"
)

const (
	recoveryDecisionSource       = "data-backup-manager"
	recoveryDecisionKind         = "volume_repair"
	recoveryNotificationType     = "data_backup_manager.volume_recovery.notification.v1"
	recoveryDecisionWebhookPath  = "/api/v1/events/notification-hub"
	recoveryDecisionAskSubject   = "Backup drive filesystem repair needs approval"
	operatorActionsFileFullPath  = "/home/matthalloran8/.vrooli/research/ATTENTION.md"
	volumeRecoveryDecisionWindow = 24 * time.Hour
)

var actionableAttentionHeading = regexp.MustCompile(`(?m)^###\s+([^\s]+)\s+—\s+(.+?)\s*$`)

func currentOperatorActionsDigest() string {
	data, err := os.ReadFile(operatorActionsFileFullPath)
	if err != nil {
		return "unverified because the supervisor action file could not be read"
	}
	content := string(data)
	start := strings.Index(content, "## Actionable now")
	if start < 0 {
		return "unverified because the file has no Actionable now section"
	}
	section := content[start+len("## Actionable now"):]
	if next := strings.Index(section, "\n## "); next >= 0 {
		section = section[:next]
	}
	items := actionableAttentionHeading.FindAllStringSubmatch(section, -1)
	if len(items) == 0 {
		if strings.Contains(section, "No actionable items are recorded.") {
			return "none recorded"
		}
		return "unverified because actionable entries could not be parsed"
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, item[1]+" — "+item[2])
	}
	return strings.Join(parts, "; ")
}

// RecoveryDecisionFlow links a durable recovery journal to Notification Hub.
// It asks only for a destructive repair, and applies an answer only after the
// signed webhook has delivered a verified human response for the exact ask.
type RecoveryDecisionFlow struct {
	readiness       *destinationreadiness.Service
	recoveries      destinationreadiness.RecoveryStore
	asks            operatordecision.PendingStore
	publisher       operatordecision.Publisher
	raiser          operatordecision.Raiser
	now             func() time.Time
	logger          *log.Logger
	decisionEnabled bool
}

func newRecoveryDecisionFlow(readiness *destinationreadiness.Service, recoveries destinationreadiness.RecoveryStore, tasks operatordecision.PendingStore, publisher operatordecision.Publisher, decisionEnabled bool, logger *log.Logger) *RecoveryDecisionFlow {
	if logger == nil {
		logger = log.Default()
	}
	flow := &RecoveryDecisionFlow{readiness: readiness, recoveries: recoveries, asks: tasks, publisher: publisher, decisionEnabled: decisionEnabled, now: time.Now, logger: logger}
	flow.raiser = operatordecision.Raiser{
		Source: recoveryDecisionSource, Kind: recoveryDecisionKind, Store: tasks,
		Publisher: publisher, Clock: recoveryDecisionClock{flow: flow}, MaxOutstanding: 3,
		Window: volumeRecoveryDecisionWindow,
	}
	return flow
}

type recoveryDecisionClock struct{ flow *RecoveryDecisionFlow }

func (c recoveryDecisionClock) Now() time.Time {
	if c.flow == nil || c.flow.now == nil {
		return time.Now()
	}
	return c.flow.now()
}

// RaiseRepairApproval records and publishes an idempotent decision for one
// exact blocked repair step. The confirmation phrase and data-loss ack are not
// accepted from the API caller as authority; only a verified human callback
// supplies those gates.
func (f *RecoveryDecisionFlow) RaiseRepairApproval(ctx context.Context, journal destinationreadiness.RecoveryJournal) error {
	if !f.decisionDeliveryReady() {
		return fmt.Errorf("Notification Hub decision delivery is not configured")
	}
	step, err := pendingRepairStep(journal)
	if err != nil {
		return err
	}
	plan := step.Plan
	index := strconv.Itoa(journal.Current)
	askID := operatordecision.CorrelationID("dbm-repair", journal.ID, plan.ID, index)
	correlation := map[string]string{"recovery_id": journal.ID, "plan_id": plan.ID, "step_index": index}
	facts := operatordecision.Facts{
		Subject:  recoveryDecisionAskSubject,
		Question: "May Data Backup Manager run the exact filesystem repair below? It can discard filesystem metadata and may not resolve the repository-level I/O error. Approving authorizes one identity-bound attempt, followed by remount and verification; it does not authorize backup writes. Approve only if you accept that risk.",
		Options: []operatordecision.Option{
			{Key: "approve_repair", Label: "Approve this filesystem repair"},
			{Key: "decline_repair", Label: "Do not repair; leave the drive unchanged"},
		},
		// Recommend only the single attempt the operator asked to make
		// available as a risk-acceptance choice; state the uncertain benefit.
		Recommended:          "approve_repair",
		RecommendationReason: "This is the requested one-time, identity-bound risk-acceptance path. Prior clean-check and SMART evidence make the benefit uncertain, and the Kopia EIO may remain.",
		Reversible:           false,
		Urgency:              "high",
		Note:                 fmt.Sprintf("Recovery %s, step %d of %d. Destination: %s. Device: %s. Other currently actionable items: %s. Operator-required actions file: %s", journal.ID, journal.Current+1, len(journal.Steps), journal.Location, plan.Identity.StableString(), currentOperatorActionsDigest(), operatorActionsFileFullPath),
		BigPicture:           "This decision controls whether DBM may alter filesystem metadata on the selected backup drive. Approval authorizes one exact repair attempt; it does not guarantee repository recovery or enable backups. Declining leaves the drive unchanged and the recovery paused.",
		TechnicalDetails:     fmt.Sprintf("Action: %s. Plan: %s. Filesystem: %s. Stable identity: %s. Recovery pause detail: %s. The callback re-inspects identity and current host state, requires this exact plan to remain valid, and executes only through the control-plane volume owner. No automatic/default approval is configured. Other currently actionable items: %s. Operator actions file: %s", plan.Action, plan.ID, plan.Identity.Filesystem, plan.Identity.StableString(), journal.LastError, currentOperatorActionsDigest(), operatorActionsFileFullPath),
		Correlation:          correlation,
	}
	if err := facts.Validate(); err != nil {
		return fmt.Errorf("validate repair decision facts: %w", err)
	}
	task := operatordecision.PendingAsk{
		ID: askID, Subject: journal.ID, Extra: map[string]string{
			"recovery_id": journal.ID, "plan_id": plan.ID, "step_index": index,
		},
	}
	if _, err := f.raiser.Raise(ctx, task, facts); err != nil {
		return fmt.Errorf("raise filesystem repair decision: %w", err)
	}
	return nil
}

func (f *RecoveryDecisionFlow) decisionDeliveryReady() bool {
	return f != nil && f.decisionEnabled && f.publisher != nil && f.asks != nil
}

func pendingRepairStep(journal destinationreadiness.RecoveryJournal) (destinationreadiness.RecoveryStep, error) {
	if journal.Current < 0 || journal.Current >= len(journal.Steps) {
		return destinationreadiness.RecoveryStep{}, fmt.Errorf("recovery journal has no pending repair step")
	}
	step := journal.Steps[journal.Current]
	if step.Plan.Action != destinationreadiness.ActionRepairFilesystem || !step.Plan.Destructive || step.Status != "blocked" {
		return destinationreadiness.RecoveryStep{}, fmt.Errorf("recovery journal is not paused at a blocked destructive repair step")
	}
	if !journal.Identity.StrongIdentity() || !step.Plan.Identity.MatchesDevice(journal.Identity) {
		return destinationreadiness.RecoveryStep{}, fmt.Errorf("repair approval requires a stable identity matching the recovery journal")
	}
	return step, nil
}

// NotifyMountFailure publishes a regular Notification Hub alert when a
// recovery sequence cannot attach the selected volume. It is separate from a
// decision ask because the failure itself is informational; only a risky
// repair requires an answer.
func (f *RecoveryDecisionFlow) NotifyMountFailure(ctx context.Context, journal destinationreadiness.RecoveryJournal, cause error) {
	if f == nil || f.publisher == nil {
		return
	}
	detail := "The control-plane volume owner could not mount the selected backup drive."
	if cause != nil {
		detail += " " + cause.Error()
	}
	if err := f.publisher.PublishDomainEvent(ctx, recoveryNotification("Backup drive could not be mounted", detail, "volume-mount-failed", "high", journal.ID, "")); err != nil {
		f.logger.Printf("volume mount failure notification failed: %v", err)
	}
}

// NotifyRecoveryCompleted is used by the manual safe steps and by the verified
// repair callback so an operator sees the terminal outcome in the same feed.
func (f *RecoveryDecisionFlow) NotifyRecoveryCompleted(ctx context.Context, journal destinationreadiness.RecoveryJournal) {
	if f == nil || f.publisher == nil {
		return
	}
	if err := f.publisher.PublishDomainEvent(ctx, recoveryNotification(
		"Backup drive recovery completed",
		"The approved recovery sequence completed through the control-plane volume owner. DBM verified the selected volume is mounted read/write at its recorded mount root. Check destination readiness before allowing backup writes.",
		"volume-recovery-completed", "normal", journal.ID, "",
	)); err != nil {
		f.logger.Printf("volume recovery completion notification failed: %v", err)
	}
}

// Handles identifies the one decision producer this flow owns.
func (f *RecoveryDecisionFlow) Handles(sourceEventType string) bool {
	return sourceEventType == operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind)
}

// Apply enforces single-use, human-only, identity-bound callback semantics.
// Duplicate deliveries become benign no-ops; ambiguous in-flight filesystem
// work remains paused under RecoveryJournal's existing ambiguity protection.
func (f *RecoveryDecisionFlow) Apply(ctx context.Context, resolution operatordecision.DecisionResolution) (bool, error) {
	if !f.resolutionReady() {
		return false, nil
	}
	ref, ok := parseRecoveryDecisionRef(resolution)
	if !ok {
		return false, nil
	}
	task, err := f.asks.Get(ctx, ref.askID)
	if err != nil {
		return true, err
	}
	if !matchesRecoveryDecisionAsk(task, ref) {
		return true, nil
	}
	if handled, err := f.handlePriorResolution(ctx, task, resolution, ref); handled || err != nil {
		return true, err
	}
	if !isVerifiedRepairApproval(resolution) {
		return f.finishWithoutRepair(ctx, task, deniedRepairDecisionDetail(resolution), "volume-recovery-failed")
	}
	if resolution.Answer != "approve_repair" {
		return f.finishWithoutRepair(ctx, task, deniedRepairAnswerDetail(resolution), "volume-recovery-failed")
	}
	return f.applyApprovedRepair(ctx, task, resolution, ref)
}

func (f *RecoveryDecisionFlow) resolutionReady() bool {
	return f != nil && f.readiness != nil && f.recoveries != nil && f.asks != nil && f.publisher != nil
}

type recoveryDecisionRef struct {
	recoveryID string
	planID     string
	stepIndex  int
	askID      string
}

func parseRecoveryDecisionRef(resolution operatordecision.DecisionResolution) (recoveryDecisionRef, bool) {
	ref := recoveryDecisionRef{
		recoveryID: strings.TrimSpace(resolution.Correlation["recovery_id"]),
		planID:     strings.TrimSpace(resolution.Correlation["plan_id"]),
	}
	index, err := strconv.Atoi(strings.TrimSpace(resolution.Correlation["step_index"]))
	if ref.recoveryID == "" || ref.planID == "" || err != nil || index < 0 {
		return recoveryDecisionRef{}, false
	}
	ref.stepIndex = index
	ref.askID = operatordecision.CorrelationID("dbm-repair", ref.recoveryID, ref.planID, strconv.Itoa(index))
	return ref, true
}

func matchesRecoveryDecisionAsk(task *operatordecision.PendingAsk, ref recoveryDecisionRef) bool {
	return task != nil && task.Subject == ref.recoveryID && task.Extra["recovery_id"] == ref.recoveryID &&
		task.Extra["plan_id"] == ref.planID && task.Extra["step_index"] == strconv.Itoa(ref.stepIndex)
}

func (f *RecoveryDecisionFlow) handlePriorResolution(ctx context.Context, task *operatordecision.PendingAsk, resolution operatordecision.DecisionResolution, ref recoveryDecisionRef) (bool, error) {
	if task.State == operatordecision.PendingStatePending {
		return false, nil
	}
	if wasMisverifiedMountApproval(task, resolution) {
		return f.completeObservedMountedRecovery(ctx, task, resolution, ref.recoveryID, ref.planID, ref.stepIndex)
	}
	if wasMisclassifiedHumanApproval(task, resolution) {
		return false, f.reopenMisclassifiedHumanApproval(ctx, task)
	}
	f.logger.Printf("volume recovery decision replay ignored recovery=%s plan=%s local_state=%s title=%q", ref.recoveryID, ref.planID, task.State, task.Extra["result_title"])
	if task.Extra["result_notification_pending"] == "true" {
		return true, f.publishPendingResult(ctx, task)
	}
	return true, nil
}

func (f *RecoveryDecisionFlow) reopenMisclassifiedHumanApproval(ctx context.Context, task *operatordecision.PendingAsk) error {
	// Older DBM builds treated Notification Hub's canonical verified-human
	// identity as non-human. Only its exact signed approval replay reaches here.
	task.State = operatordecision.PendingStatePending
	task.ResolvedAt = nil
	task.ResolvedBy = ""
	for _, key := range []string{"result_title", "result_body", "result_category", "result_urgency", "result_notification_pending"} {
		delete(task.Extra, key)
	}
	if err := f.asks.Put(ctx, task); err != nil {
		return fmt.Errorf("reopen the misclassified verified approval for exact replay: %w", err)
	}
	return nil
}

func isVerifiedRepairApproval(resolution operatordecision.DecisionResolution) bool {
	return resolution.State == operatordecision.StateAnswered && !resolution.ByDefault &&
		isVerifiedHumanIdentity(resolution.AnsweredByIdentityKind) &&
		strings.TrimSpace(resolution.AnsweredBy) != "" && strings.TrimSpace(resolution.AnsweredByIdentitySource) != ""
}

func deniedRepairDecisionDetail(resolution operatordecision.DecisionResolution) string {
	if resolution.State == operatordecision.StateDismissed {
		return "The operator dismissed the repair decision without answering. The drive was left unchanged."
	}
	return "The repair decision did not include a verified human approval. The drive was left unchanged."
}

func deniedRepairAnswerDetail(resolution operatordecision.DecisionResolution) string {
	if resolution.Answer == "other" {
		return "The operator selected Other and supplied this direction: " + strings.TrimSpace(resolution.Note) + ". No filesystem repair was run; the drive was left unchanged."
	}
	return "Filesystem repair was not approved. The drive was left unchanged."
}

func (f *RecoveryDecisionFlow) applyApprovedRepair(ctx context.Context, task *operatordecision.PendingAsk, resolution operatordecision.DecisionResolution, ref recoveryDecisionRef) (bool, error) {
	journal, err := f.recoveries.Load(ctx, ref.recoveryID)
	if err != nil {
		return true, err
	}
	if completedRepairAlreadyAdvanced(journal, ref.stepIndex) {
		return true, f.completeOutcome(ctx, task, journal, resolution.AnsweredBy)
	}
	if ref.stepIndex >= len(journal.Steps) || journal.Current != ref.stepIndex {
		return f.finishWithoutRepair(ctx, task, "The recovery step changed while approval was pending. No repair was run; inspect and prepare a new decision if needed.", "volume-recovery-failed")
	}
	step := journal.Steps[ref.stepIndex]
	if !approvedRepairStepMatches(step, journal, ref.planID) {
		return f.finishWithoutRepair(ctx, task, "The approved recovery plan or selected device identity changed. No repair was run.", "volume-recovery-failed")
	}
	currentPlan, err := f.readiness.PlanPreparation(ctx, destinationreadiness.PlanInput{
		Location: journal.Location, Action: destinationreadiness.ActionRepairFilesystem,
		ExpectedDevice: journal.Identity,
	})
	if err != nil || !sameRepairPlan(currentPlan, step.Plan) {
		detail := "The selected device could not be revalidated. No repair was run."
		if err != nil {
			detail += " " + err.Error()
		}
		return f.finishWithoutRepair(ctx, task, detail, "volume-recovery-failed")
	}
	return f.executeApprovedRepair(ctx, task, resolution, journal, ref.stepIndex)
}

func completedRepairAlreadyAdvanced(journal destinationreadiness.RecoveryJournal, stepIndex int) bool {
	return stepIndex >= 0 && stepIndex < len(journal.Steps) && journal.Steps[stepIndex].Status == "completed" && journal.Current > stepIndex
}

func approvedRepairStepMatches(step destinationreadiness.RecoveryStep, journal destinationreadiness.RecoveryJournal, planID string) bool {
	return step.Plan.ID == planID && step.Plan.Action == destinationreadiness.ActionRepairFilesystem && step.Plan.Destructive &&
		step.Status == "blocked" && step.Plan.Identity.MatchesDevice(journal.Identity) && journal.Identity.StrongIdentity()
}

func sameRepairPlan(current, approved destinationreadiness.Plan) bool {
	return current.Action == approved.Action && current.TargetPath == approved.TargetPath && current.Identity.MatchesDevice(approved.Identity)
}

func (f *RecoveryDecisionFlow) executeApprovedRepair(ctx context.Context, task *operatordecision.PendingAsk, resolution operatordecision.DecisionResolution, journal destinationreadiness.RecoveryJournal, stepIndex int) (bool, error) {
	confirmations := make(map[int]string, len(journal.Steps)-stepIndex)
	for index := stepIndex; index < len(journal.Steps); index++ {
		if journal.Steps[index].Plan.RequiresConfirm {
			confirmations[index] = journal.Steps[index].Plan.ConfirmationPhrase
		}
	}
	updated, resumeErr := f.readiness.ResumeRecovery(ctx, f.recoveries, journal, confirmations, true, false)
	if resumeErr == nil && updated.State == destinationreadiness.RecoveryCompleted {
		return true, f.completeOutcome(ctx, task, updated, resolution.AnsweredBy)
	}
	return true, f.finishApprovedRepairFailure(ctx, task, resolution, updated, resumeErr)
}

func (f *RecoveryDecisionFlow) finishApprovedRepairFailure(ctx context.Context, task *operatordecision.PendingAsk, resolution operatordecision.DecisionResolution, updated destinationreadiness.RecoveryJournal, resumeErr error) error {
	detail := updated.LastError
	if resumeErr != nil {
		detail = resumeErr.Error()
	}
	if detail == "" {
		detail = "Recovery did not reach a verified completed state."
	}
	if updated.Current >= 0 && updated.Current < len(updated.Steps) && updated.Steps[updated.Current].Plan.Action == destinationreadiness.ActionMountReadWrite {
		f.NotifyMountFailure(ctx, updated, resumeErr)
	}
	return f.queueResult(ctx, task, operatordecision.PendingStateRejected, resolution.AnsweredBy,
		"Filesystem repair needs attention", detail, "volume-recovery-failed", "high")
}

func isVerifiedHumanIdentity(kind string) bool {
	return strings.EqualFold(strings.TrimSpace(kind), "verified_human")
}

func wasMisclassifiedHumanApproval(task *operatordecision.PendingAsk, resolution operatordecision.DecisionResolution) bool {
	return task != nil && task.State == operatordecision.PendingStateRejected && task.ResolvedBy == "" &&
		task.Extra["result_title"] == "Backup drive recovery needs attention" &&
		task.Extra["result_body"] == "The repair decision did not include a verified human approval. The drive was left unchanged." &&
		verifiedApprovalReplay(resolution, "")
}

func wasMisverifiedMountApproval(task *operatordecision.PendingAsk, resolution operatordecision.DecisionResolution) bool {
	return task != nil && task.State == operatordecision.PendingStateRejected && strings.TrimSpace(task.ResolvedBy) != "" &&
		task.Extra["result_title"] == "Filesystem repair needs attention" &&
		task.Extra["result_body"] == "preparation refused: mount action completed without a verified read/write mount" &&
		verifiedApprovalReplay(resolution, task.ResolvedBy)
}

func verifiedApprovalReplay(resolution operatordecision.DecisionResolution, expectedBy string) bool {
	if resolution.State != operatordecision.StateAnswered || resolution.ByDefault || resolution.Answer != "approve_repair" ||
		!isVerifiedHumanIdentity(resolution.AnsweredByIdentityKind) || strings.TrimSpace(resolution.AnsweredByIdentitySource) == "" {
		return false
	}
	answeredBy := strings.TrimSpace(resolution.AnsweredBy)
	return answeredBy != "" && (expectedBy == "" || answeredBy == strings.TrimSpace(expectedBy))
}

func (f *RecoveryDecisionFlow) completeObservedMountedRecovery(ctx context.Context, task *operatordecision.PendingAsk, resolution operatordecision.DecisionResolution, recoveryID, planID string, stepIndex int) (bool, error) {
	journal, err := f.recoveries.Load(ctx, recoveryID)
	if err != nil {
		return true, err
	}
	mountStepIndex := stepIndex + 1
	if !matchesObservedMountedRecovery(journal, stepIndex, mountStepIndex, planID) {
		f.logger.Printf("volume recovery mounted-state replay did not match journal recovery=%s plan=%s current=%d steps=%d", recoveryID, planID, journal.Current, len(journal.Steps))
		return true, nil
	}
	if _, err := f.verifyMountedRecoveryLocation(ctx, journal.Location, journal.Identity); err != nil {
		f.logger.Printf("volume recovery mounted-state replay verification failed recovery=%s: %v", recoveryID, err)
		return true, nil
	}
	journal.Steps[mountStepIndex].Status = "completed"
	journal.Current = len(journal.Steps)
	journal.State = destinationreadiness.RecoveryCompleted
	journal.LastError = ""
	journal.UpdatedAt = f.now().UTC()
	if err := f.recoveries.Save(ctx, journal); err != nil {
		return true, fmt.Errorf("persist verified mounted recovery: %w", err)
	}
	f.logger.Printf("volume recovery mounted-state replay completed recovery=%s plan=%s", recoveryID, planID)
	return true, f.completeOutcome(ctx, task, journal, resolution.AnsweredBy)
}

func matchesObservedMountedRecovery(journal destinationreadiness.RecoveryJournal, repairStepIndex, mountStepIndex int, planID string) bool {
	return repairStepIndex >= 0 && mountStepIndex < len(journal.Steps) && journal.Current == mountStepIndex &&
		journal.Steps[repairStepIndex].Plan.ID == planID &&
		journal.Steps[repairStepIndex].Plan.Action == destinationreadiness.ActionRepairFilesystem &&
		journal.Steps[repairStepIndex].Status == "completed" &&
		journal.Steps[mountStepIndex].Plan.Action == destinationreadiness.ActionMountReadWrite &&
		journal.Steps[mountStepIndex].Status == "blocked" && journal.Identity.StrongIdentity()
}

func (f *RecoveryDecisionFlow) completeOutcome(ctx context.Context, task *operatordecision.PendingAsk, journal destinationreadiness.RecoveryJournal, by string) error {
	_, err := f.verifyMountedRecoveryLocation(ctx, journal.Location, journal.Identity)
	if err != nil {
		return f.queueResult(ctx, task, operatordecision.PendingStateApproved, by,
			"Filesystem repair completed but the backup drive needs attention", "The repair journal completed, but DBM could not verify the selected volume is mounted and writable: "+err.Error(), "volume-recovery-failed", "high")
	}
	return f.queueResult(ctx, task, operatordecision.PendingStateApproved, by,
		"Backup drive recovery completed", "The approved filesystem repair and recovery steps completed through the control-plane owner, and DBM verified the selected volume is mounted read/write.", "volume-recovery-completed", "normal")
}

// verifyMountedRecoveryLocation uses the same location-based observer as
// destination readiness. Some platforms can resolve a device by stable UUID
// while omitting its current mountpoint, so the device-only lens is not enough
// to prove a completed mount. The location report is then bound back to the
// journal's stable identity and exact mount root before it can complete a run.
func (f *RecoveryDecisionFlow) verifyMountedRecoveryLocation(ctx context.Context, location string, expected destinationreadiness.DeviceIdentity) (destinationreadiness.Report, error) {
	report, err := f.readiness.Analyze(ctx, destinationreadiness.AnalyzeInput{Location: location})
	if err != nil {
		return destinationreadiness.Report{}, fmt.Errorf("inspect recovery mount location: %w", err)
	}
	if !expected.MatchesDevice(report.Identity) {
		return destinationreadiness.Report{}, fmt.Errorf("mounted location resolved to a different device identity")
	}
	if filepath.Clean(report.Identity.Mountpoint) != filepath.Clean(expected.Mountpoint) {
		return destinationreadiness.Report{}, fmt.Errorf("mounted location resolved to mount root %q, expected %q", report.Identity.Mountpoint, expected.Mountpoint)
	}
	if !report.Mounted || report.DeviceWriteProtected || report.ReadOnlyCause != "" {
		return destinationreadiness.Report{}, fmt.Errorf("selected volume is not verified mounted read/write")
	}
	for _, check := range report.Checks {
		if check.Code == "mounted_read_write" && check.Severity == destinationreadiness.SeverityPass {
			return report, nil
		}
	}
	return destinationreadiness.Report{}, fmt.Errorf("readiness did not verify the selected volume is mounted read/write")
}
