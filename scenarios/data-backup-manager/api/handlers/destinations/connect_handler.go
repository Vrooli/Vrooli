package destinations

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/destinations"

	"connectrpc.com/connect"
	destinationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations"
	destinationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations/destinations_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Ensure connectHandler satisfies the generated handler interface.
var _ destinationsconnect.DestinationsServiceHandler = (*connectHandler)(nil)

// Deps wires the seams the Connect destinations handler needs.
type Deps struct {
	Service   destinations.Service
	Readiness *destinationreadiness.Service
	// RecoveryStore is durable journal storage. Production wiring uses DBM's
	// resolved state directory; tests provide an in-memory fake.
	RecoveryStore destinationreadiness.RecoveryStore
	DecisionFlow  *RecoveryDecisionFlow
	Logger        *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the destinations Connect-RPC handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) CreateDestination(ctx context.Context, req *connect.Request[destinationsv1.CreateDestinationRequest]) (*connect.Response[destinationsv1.CreateDestinationResponse], error) {
	d, err := h.deps.Service.CreateDestination(ctx, destinations.CreateInput{
		Name:      req.Msg.Name,
		Backend:   protoToBackend(req.Msg.BackendKind),
		Location:  req.Msg.Location,
		CapBytes:  req.Msg.CapBytes,
		CapPolicy: protoToCapPolicy(req.Msg.CapPolicy),
	})
	if err != nil {
		return nil, h.translate("CreateDestination", err)
	}
	return connect.NewResponse(&destinationsv1.CreateDestinationResponse{
		Destination: domainToProto(d),
	}), nil
}

func (h *connectHandler) GetDestination(ctx context.Context, req *connect.Request[destinationsv1.GetDestinationRequest]) (*connect.Response[destinationsv1.GetDestinationResponse], error) {
	d, err := h.deps.Service.GetDestination(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.translate("GetDestination", err)
	}
	return connect.NewResponse(&destinationsv1.GetDestinationResponse{
		Destination: domainToProto(d),
	}), nil
}

func (h *connectHandler) ListDestinations(ctx context.Context, req *connect.Request[destinationsv1.ListDestinationsRequest]) (*connect.Response[destinationsv1.ListDestinationsResponse], error) {
	list, err := h.deps.Service.ListDestinations(ctx, int(req.Msg.PageSize))
	if err != nil {
		return nil, h.translate("ListDestinations", err)
	}
	resp := &destinationsv1.ListDestinationsResponse{
		Destinations: make([]*destinationsv1.Destination, 0, len(list)),
	}
	for _, d := range list {
		resp.Destinations = append(resp.Destinations, domainToProto(d))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) UpdateDestination(ctx context.Context, req *connect.Request[destinationsv1.UpdateDestinationRequest]) (*connect.Response[destinationsv1.UpdateDestinationResponse], error) {
	d, err := h.deps.Service.UpdateDestination(ctx, destinations.UpdateInput{
		ID:        req.Msg.Id,
		CapBytes:  req.Msg.CapBytes,
		CapPolicy: protoToCapPolicy(req.Msg.CapPolicy),
	})
	if err != nil {
		return nil, h.translate("UpdateDestination", err)
	}
	return connect.NewResponse(&destinationsv1.UpdateDestinationResponse{
		Destination: domainToProto(d),
	}), nil
}

func (h *connectHandler) BindDestinationRecoveryIdentity(ctx context.Context, req *connect.Request[destinationsv1.BindDestinationRecoveryIdentityRequest]) (*connect.Response[destinationsv1.BindDestinationRecoveryIdentityResponse], error) {
	d, err := h.deps.Service.BindRecoveryIdentity(ctx, req.Msg.Id, protoToDeviceIdentity(req.Msg.Identity))
	if err != nil {
		return nil, h.translate("BindDestinationRecoveryIdentity", err)
	}
	return connect.NewResponse(&destinationsv1.BindDestinationRecoveryIdentityResponse{Destination: domainToProto(d)}), nil
}

func (h *connectHandler) DeleteDestination(ctx context.Context, req *connect.Request[destinationsv1.DeleteDestinationRequest]) (*connect.Response[destinationsv1.DeleteDestinationResponse], error) {
	removed, err := h.deps.Service.DeleteDestination(ctx, req.Msg.Id, req.Msg.DeleteRepository)
	if err != nil {
		return nil, h.translate("DeleteDestination", err)
	}
	return connect.NewResponse(&destinationsv1.DeleteDestinationResponse{Removed: removed}), nil
}

func (h *connectHandler) GetDestinationUsage(ctx context.Context, req *connect.Request[destinationsv1.GetDestinationUsageRequest]) (*connect.Response[destinationsv1.GetDestinationUsageResponse], error) {
	report, err := h.deps.Service.GetDestinationUsage(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.translate("GetDestinationUsage", err)
	}
	return connect.NewResponse(&destinationsv1.GetDestinationUsageResponse{
		UsageBytes: report.UsageBytes,
		CapBytes:   report.CapBytes,
		UsageState: usageStateToProto(report.UsageState),
		CapPolicy:  capPolicyToProto(report.CapPolicy),
	}), nil
}

func (h *connectHandler) AnalyzeDestination(ctx context.Context, req *connect.Request[destinationsv1.AnalyzeDestinationRequest]) (*connect.Response[destinationsv1.AnalyzeDestinationResponse], error) {
	if h.deps.Readiness == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("destination readiness service is not configured"))
	}
	registered, err := h.deps.Service.ListDestinations(ctx, 0)
	if err != nil {
		return nil, h.translate("ListDestinationsForReadiness", err)
	}
	existingLocations := make([]string, 0, len(registered))
	for _, destination := range registered {
		if destination.BackendKind == destinations.BackendFilesystem {
			existingLocations = append(existingLocations, destination.Location)
		}
	}
	report, err := h.deps.Readiness.Analyze(ctx, destinationreadiness.AnalyzeInput{
		Location:              req.Msg.Location,
		ProposedSubdir:        req.Msg.ProposedSubdir,
		SelectedTargetBytes:   req.Msg.SelectedTargetBytes,
		RetentionCopies:       int(req.Msg.RetentionCopies),
		ExistingDestinations:  existingLocations,
		CrossPlatformRequired: req.Msg.CrossPlatformRequired,
	})
	if err != nil {
		return nil, h.translateReadiness("AnalyzeDestination", err)
	}
	return connect.NewResponse(&destinationsv1.AnalyzeDestinationResponse{Report: readinessReportToProto(report)}), nil
}

func (h *connectHandler) PlanDestinationPreparation(ctx context.Context, req *connect.Request[destinationsv1.PlanDestinationPreparationRequest]) (*connect.Response[destinationsv1.PlanDestinationPreparationResponse], error) {
	if h.deps.Readiness == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("destination readiness service is not configured"))
	}
	plan, err := h.deps.Readiness.PlanPreparation(ctx, destinationreadiness.PlanInput{
		Location:       req.Msg.Location,
		Action:         protoToPreparationAction(req.Msg.Action),
		DesiredSubdir:  req.Msg.DesiredSubdir,
		DesiredLabel:   req.Msg.DesiredLabel,
		DesiredFS:      req.Msg.DesiredFilesystem,
		ExpectedDevice: protoToDeviceIdentity(req.Msg.ExpectedIdentity),
	})
	if err != nil {
		return nil, h.translateReadiness("PlanDestinationPreparation", err)
	}
	return connect.NewResponse(&destinationsv1.PlanDestinationPreparationResponse{Plan: preparationPlanToProto(plan)}), nil
}

func (h *connectHandler) ExecuteDestinationPreparation(ctx context.Context, req *connect.Request[destinationsv1.ExecuteDestinationPreparationRequest]) (*connect.Response[destinationsv1.ExecuteDestinationPreparationResponse], error) {
	if h.deps.Readiness == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("destination readiness service is not configured"))
	}
	plan := protoToPreparationPlan(req.Msg.Plan)
	dryRun := true
	if req.Msg.DryRun != nil {
		dryRun = req.Msg.GetDryRun()
	}
	if plan.Action == destinationreadiness.ActionRepairFilesystem && !dryRun {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("filesystem repair must run through a durable volume-recovery journal and verified Notification Hub decision"))
	}
	result, err := h.deps.Readiness.ExecutePreparation(ctx, destinationreadiness.ExecuteInput{
		Plan:                plan,
		Confirmation:        req.Msg.Confirmation,
		DryRun:              dryRun,
		AcknowledgeDataLoss: req.Msg.AcknowledgeDataLoss,
	})
	if err != nil {
		return nil, h.translateReadiness("ExecuteDestinationPreparation", err)
	}
	resp := &destinationsv1.ExecuteDestinationPreparationResponse{
		DryRun:          result.DryRun,
		Action:          preparationActionToProto(result.Action),
		Location:        result.Location,
		Status:          result.Status,
		Changed:         result.Changed,
		Backend:         result.Backend,
		Command:         append([]string(nil), result.Command...),
		Detail:          result.Detail,
		OperatorCommand: result.OperatorCommand,
		RefusalReason:   result.RefusalReason,
		Consistent:      result.Consistent,
	}
	if !result.DryRun {
		// A remediation that left the volume unmounted has no path to analyze.
		// Reporting the readiness of an absent path as a failure would bury the
		// step's own successful result, so the post-action report is attached
		// only when there is something to report on.
		analysisLocation := result.Location
		if analysisLocation == "" {
			analysisLocation = plan.Location
		}
		report, rerr := h.deps.Readiness.Analyze(ctx, destinationreadiness.AnalyzeInput{Location: analysisLocation})
		switch {
		case rerr == nil:
			resp.PostActionReport = readinessReportToProto(report)
		case plan.Action.IsRemediation():
			// Intentionally not fatal: see above.
		default:
			return nil, h.translateReadiness("ExecuteDestinationPreparationPostCheck", rerr)
		}
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) StartVolumeRecovery(ctx context.Context, req *connect.Request[destinationsv1.StartVolumeRecoveryRequest]) (*connect.Response[destinationsv1.StartVolumeRecoveryResponse], error) {
	if h.deps.Readiness == nil || h.deps.RecoveryStore == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("volume recovery is not configured"))
	}
	plans := make([]destinationreadiness.Plan, 0, len(req.Msg.Plans))
	for _, plan := range req.Msg.Plans {
		plans = append(plans, protoToPreparationPlan(plan))
	}
	journal, err := destinationreadiness.NewRecoveryJournal(
		req.Msg.Location,
		protoToDeviceIdentity(req.Msg.Identity),
		plans,
		time.Now().UTC(),
	)
	if err != nil {
		return nil, h.translateReadiness("StartVolumeRecovery", err)
	}
	if err := h.deps.RecoveryStore.Save(ctx, journal); err != nil {
		return nil, h.translateReadiness("StartVolumeRecovery", fmt.Errorf("persist recovery journal: %w", err))
	}
	return connect.NewResponse(&destinationsv1.StartVolumeRecoveryResponse{Journal: recoveryJournalToProto(journal)}), nil
}

func (h *connectHandler) GetVolumeRecovery(ctx context.Context, req *connect.Request[destinationsv1.GetVolumeRecoveryRequest]) (*connect.Response[destinationsv1.GetVolumeRecoveryResponse], error) {
	if h.deps.RecoveryStore == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("volume recovery is not configured"))
	}
	journal, err := h.deps.RecoveryStore.Load(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.translateReadiness("GetVolumeRecovery", err)
	}
	return connect.NewResponse(&destinationsv1.GetVolumeRecoveryResponse{Journal: recoveryJournalToProto(journal)}), nil
}

func (h *connectHandler) ResumeVolumeRecovery(ctx context.Context, req *connect.Request[destinationsv1.ResumeVolumeRecoveryRequest]) (*connect.Response[destinationsv1.ResumeVolumeRecoveryResponse], error) {
	if h.deps.Readiness == nil || h.deps.RecoveryStore == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("volume recovery is not configured"))
	}
	journal, err := h.deps.RecoveryStore.Load(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.translateReadiness("ResumeVolumeRecovery", err)
	}
	confirmations := recoveryConfirmations(req)
	dryRun := req.Msg.DryRun == nil || req.Msg.GetDryRun()
	acknowledgeDataLoss := req.Msg.AcknowledgeDataLoss
	if requiresNotificationApproval(journal, dryRun) {
		delete(confirmations, journal.Current)
		acknowledgeDataLoss = false
	}
	updated, resumeErr := h.deps.Readiness.ResumeRecovery(ctx, h.deps.RecoveryStore, journal, confirmations, acknowledgeDataLoss, dryRun)
	if resumeErr != nil {
		return h.handleRecoveryResumeError(ctx, updated, resumeErr, dryRun)
	}
	if updated.State == destinationreadiness.RecoveryCompleted && h.deps.DecisionFlow != nil {
		h.deps.DecisionFlow.NotifyRecoveryCompleted(ctx, updated)
	}
	return connect.NewResponse(&destinationsv1.ResumeVolumeRecoveryResponse{Journal: recoveryJournalToProto(updated)}), nil
}

func recoveryConfirmations(req *connect.Request[destinationsv1.ResumeVolumeRecoveryRequest]) map[int]string {
	confirmations := make(map[int]string, len(req.Msg.Confirmations))
	for index, confirmation := range req.Msg.Confirmations {
		if index >= 0 {
			confirmations[int(index)] = confirmation
		}
	}
	return confirmations
}

func requiresNotificationApproval(journal destinationreadiness.RecoveryJournal, dryRun bool) bool {
	return !dryRun && journal.Current >= 0 && journal.Current < len(journal.Steps) &&
		journal.Steps[journal.Current].Plan.Action == destinationreadiness.ActionRepairFilesystem
}

func (h *connectHandler) handleRecoveryResumeError(ctx context.Context, journal destinationreadiness.RecoveryJournal, resumeErr error, dryRun bool) (*connect.Response[destinationsv1.ResumeVolumeRecoveryResponse], error) {
	step, ok := currentRecoveryStep(journal)
	if !ok {
		return nil, h.translateReadiness("ResumeVolumeRecovery", resumeErr)
	}
	if step.Plan.Action == destinationreadiness.ActionMountReadWrite && h.deps.DecisionFlow != nil {
		h.deps.DecisionFlow.NotifyMountFailure(ctx, journal, resumeErr)
	}
	if dryRun || step.Plan.Action != destinationreadiness.ActionRepairFilesystem || !requiresRepairApproval(resumeErr) {
		return nil, h.translateReadiness("ResumeVolumeRecovery", resumeErr)
	}
	if h.deps.DecisionFlow == nil {
		journal.LastError = "filesystem repair is paused; Notification Hub decision delivery is not configured"
		_ = h.deps.RecoveryStore.Save(ctx, journal)
		return recoveryResponse(journal)
	}
	if err := h.deps.DecisionFlow.RaiseRepairApproval(ctx, journal); err != nil {
		h.deps.Logger.Printf("raise volume repair decision: %v", err)
		journal.LastError = "filesystem repair is paused; Notification Hub decision could not be raised: " + err.Error()
	} else {
		journal.LastError = "filesystem repair is paused pending a verified Notification Hub decision"
	}
	if err := h.deps.RecoveryStore.Save(ctx, journal); err != nil {
		return nil, h.translateReadiness("PersistVolumeRecoveryDecisionState", err)
	}
	return recoveryResponse(journal)
}

func currentRecoveryStep(journal destinationreadiness.RecoveryJournal) (destinationreadiness.RecoveryStep, bool) {
	if journal.Current < 0 || journal.Current >= len(journal.Steps) {
		return destinationreadiness.RecoveryStep{}, false
	}
	return journal.Steps[journal.Current], true
}

func requiresRepairApproval(err error) bool {
	return strings.Contains(err.Error(), "confirmation phrase did not match") || strings.Contains(err.Error(), "data loss acknowledgement required")
}

func recoveryResponse(journal destinationreadiness.RecoveryJournal) (*connect.Response[destinationsv1.ResumeVolumeRecoveryResponse], error) {
	return connect.NewResponse(&destinationsv1.ResumeVolumeRecoveryResponse{Journal: recoveryJournalToProto(journal)}), nil
}

// translate maps a domain error to a Connect error, logging only internal ones.
func (h *connectHandler) translate(op string, err error) error {
	connectErr := destinations.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("destinations.%s: %v", op, err)
	}
	return connectErr
}

func (h *connectHandler) translateReadiness(op string, err error) error {
	if errors.Is(err, destinationreadiness.ErrRecoveryNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var invalid destinationreadiness.ErrInvalidReadiness
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	var refused destinationreadiness.ErrPreparationRefused
	if errors.As(err, &refused) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	h.deps.Logger.Printf("destinations.%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, errors.New("internal destination readiness error"))
}

// domainToProto converts the internal Destination to its wire shape.
func domainToProto(d destinations.Destination) *destinationsv1.Destination {
	pd := &destinationsv1.Destination{
		Id:                  d.ID,
		Name:                d.Name,
		BackendKind:         backendToProto(d.BackendKind),
		Location:            d.Location,
		RelativePath:        d.RelativePath,
		RepositoryLocation:  d.RepositoryLocation,
		CapBytes:            d.CapBytes,
		CapPolicy:           capPolicyToProto(d.CapPolicy),
		EncryptionAlgorithm: d.EncryptionAlgorithm,
		SecretRef:           d.SecretRef,
	}
	if !d.CreatedAt.IsZero() {
		pd.CreatedAt = timestamppb.New(d.CreatedAt)
	}
	if !d.UpdatedAt.IsZero() {
		pd.UpdatedAt = timestamppb.New(d.UpdatedAt)
	}
	if d.DeviceIdentity != nil {
		pd.DeviceIdentity = deviceIdentityToProto(*d.DeviceIdentity)
	}
	return pd
}

func readinessReportToProto(r destinationreadiness.Report) *destinationsv1.DestinationReadinessReport {
	out := &destinationsv1.DestinationReadinessReport{
		Location:                       r.Location,
		OverallSeverity:                readinessSeverityToProto(r.OverallSeverity),
		Identity:                       deviceIdentityToProto(r.Identity),
		RecommendedDestinationLocation: r.RecommendedDestinationLocation,
		RecommendedAction:              r.RecommendedAction,
		Platform:                       r.Platform,
		Confidence:                     r.Confidence,
		EvidenceSource:                 r.EvidenceSource,
		RepairSteps:                    append([]string(nil), r.RepairSteps...),
		Checks:                         make([]*destinationsv1.DestinationReadinessCheck, 0, len(r.Checks)),
	}
	if !r.ObservedAt.IsZero() {
		out.ObservedAt = timestamppb.New(r.ObservedAt)
	}
	for _, c := range r.Checks {
		out.Checks = append(out.Checks, &destinationsv1.DestinationReadinessCheck{
			Code:       c.Code,
			Severity:   readinessSeverityToProto(c.Severity),
			Message:    c.Message,
			NextAction: c.NextAction,
		})
	}
	return out
}

func preparationPlanToProto(p destinationreadiness.Plan) *destinationsv1.DestinationPreparationPlan {
	return &destinationsv1.DestinationPreparationPlan{
		Id:                   p.ID,
		Action:               preparationActionToProto(p.Action),
		Location:             p.Location,
		TargetPath:           p.TargetPath,
		RelativePath:         p.RelativePath,
		Identity:             deviceIdentityToProto(p.Identity),
		DesiredLabel:         p.DesiredLabel,
		DesiredFilesystem:    p.DesiredFS,
		RequiresConfirmation: p.RequiresConfirm,
		Destructive:          p.Destructive,
		ConfirmationPhrase:   p.ConfirmationPhrase,
		Supported:            p.Supported,
		UnsupportedReason:    p.UnsupportedReason,
	}
}
