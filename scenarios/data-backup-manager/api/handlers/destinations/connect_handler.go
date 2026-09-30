package destinations

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/destinations"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	destinationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations"
	destinationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations/destinations_v1connect"
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
	report, err := h.deps.Readiness.Analyze(ctx, destinationreadiness.AnalyzeInput{
		Location:              req.Msg.Location,
		ProposedSubdir:        req.Msg.ProposedSubdir,
		SelectedTargetBytes:   req.Msg.SelectedTargetBytes,
		RetentionCopies:       int(req.Msg.RetentionCopies),
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
	confirmations := make(map[int]string, len(req.Msg.Confirmations))
	for index, confirmation := range req.Msg.Confirmations {
		if index >= 0 {
			confirmations[int(index)] = confirmation
		}
	}
	dryRun := true
	if req.Msg.DryRun != nil {
		dryRun = req.Msg.GetDryRun()
	}
	updated, resumeErr := h.deps.Readiness.ResumeRecovery(ctx, h.deps.RecoveryStore, journal, confirmations, req.Msg.AcknowledgeDataLoss, dryRun)
	if resumeErr != nil {
		return nil, h.translateReadiness("ResumeVolumeRecovery", resumeErr)
	}
	return connect.NewResponse(&destinationsv1.ResumeVolumeRecoveryResponse{Journal: recoveryJournalToProto(updated)}), nil
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

func recoveryJournalToProto(j destinationreadiness.RecoveryJournal) *destinationsv1.VolumeRecoveryJournal {
	out := &destinationsv1.VolumeRecoveryJournal{
		Version:      j.Version,
		Id:           j.ID,
		Location:     j.Location,
		RelativePath: j.RelativePath,
		Identity:     deviceIdentityToProto(j.Identity),
		Current:      int32(j.Current),
		State:        string(j.State),
		LastError:    j.LastError,
		Steps:        make([]*destinationsv1.VolumeRecoveryStep, 0, len(j.Steps)),
	}
	if !j.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(j.UpdatedAt)
	}
	for _, step := range j.Steps {
		out.Steps = append(out.Steps, &destinationsv1.VolumeRecoveryStep{
			Plan:     preparationPlanToProto(step.Plan),
			Status:   step.Status,
			Attempts: int32(step.Attempts),
			Detail:   step.Detail,
		})
	}
	return out
}

func protoToPreparationPlan(p *destinationsv1.DestinationPreparationPlan) destinationreadiness.Plan {
	if p == nil {
		return destinationreadiness.Plan{}
	}
	return destinationreadiness.Plan{
		ID:                 p.Id,
		Action:             protoToPreparationAction(p.Action),
		Location:           p.Location,
		TargetPath:         p.TargetPath,
		RelativePath:       p.RelativePath,
		Identity:           protoToDeviceIdentity(p.Identity),
		DesiredLabel:       p.DesiredLabel,
		DesiredFS:          p.DesiredFilesystem,
		RequiresConfirm:    p.RequiresConfirmation,
		Destructive:        p.Destructive,
		ConfirmationPhrase: p.ConfirmationPhrase,
		Supported:          p.Supported,
		UnsupportedReason:  p.UnsupportedReason,
	}
}

func deviceIdentityToProto(i destinationreadiness.DeviceIdentity) *destinationsv1.DestinationDeviceIdentity {
	return &destinationsv1.DestinationDeviceIdentity{
		DevicePath: i.DevicePath,
		Mountpoint: i.Mountpoint,
		Label:      i.Label,
		Filesystem: i.Filesystem,
		TotalBytes: i.TotalBytes,
		Model:      i.Model,
		Serial:     i.Serial,
		Uuid:       i.UUID,
	}
}

func protoToDeviceIdentity(i *destinationsv1.DestinationDeviceIdentity) destinationreadiness.DeviceIdentity {
	if i == nil {
		return destinationreadiness.DeviceIdentity{}
	}
	return destinationreadiness.DeviceIdentity{
		DevicePath: i.DevicePath,
		Mountpoint: i.Mountpoint,
		Label:      i.Label,
		Filesystem: i.Filesystem,
		TotalBytes: i.TotalBytes,
		Model:      i.Model,
		Serial:     i.Serial,
		UUID:       i.Uuid,
	}
}

// protoToBackend / backendToProto translate the proto BackendKind enum to the
// domain vocabulary so domain code never imports the generated enum.
func protoToBackend(k destinationsv1.BackendKind) destinations.BackendKind {
	switch k {
	case destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM:
		return destinations.BackendFilesystem
	case destinationsv1.BackendKind_BACKEND_KIND_S3:
		return destinations.BackendS3
	default:
		return ""
	}
}

func backendToProto(k destinations.BackendKind) destinationsv1.BackendKind {
	switch k {
	case destinations.BackendFilesystem:
		return destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM
	case destinations.BackendS3:
		return destinationsv1.BackendKind_BACKEND_KIND_S3
	default:
		return destinationsv1.BackendKind_BACKEND_KIND_UNSPECIFIED
	}
}

// protoToCapPolicy / capPolicyToProto translate the proto CapPolicy enum.
func protoToCapPolicy(p destinationsv1.CapPolicy) destinations.CapPolicy {
	switch p {
	case destinationsv1.CapPolicy_CAP_POLICY_ALERT_BLOCK:
		return destinations.CapPolicyAlertBlock
	case destinationsv1.CapPolicy_CAP_POLICY_ALERT_ONLY:
		return destinations.CapPolicyAlertOnly
	default:
		return ""
	}
}

func capPolicyToProto(p destinations.CapPolicy) destinationsv1.CapPolicy {
	switch p {
	case destinations.CapPolicyAlertBlock:
		return destinationsv1.CapPolicy_CAP_POLICY_ALERT_BLOCK
	case destinations.CapPolicyAlertOnly:
		return destinationsv1.CapPolicy_CAP_POLICY_ALERT_ONLY
	default:
		return destinationsv1.CapPolicy_CAP_POLICY_UNSPECIFIED
	}
}
