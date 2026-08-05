package destinations

import (
	"context"
	"errors"
	"log"

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
	Logger    *log.Logger
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
		DryRun:   result.DryRun,
		Action:   preparationActionToProto(result.Action),
		Location: result.Location,
	}
	if !result.DryRun {
		report, rerr := h.deps.Readiness.Analyze(ctx, destinationreadiness.AnalyzeInput{Location: plan.Location})
		if rerr != nil {
			return nil, h.translateReadiness("ExecuteDestinationPreparationPostCheck", rerr)
		}
		resp.PostActionReport = readinessReportToProto(report)
	}
	return connect.NewResponse(resp), nil
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

func protoToPreparationPlan(p *destinationsv1.DestinationPreparationPlan) destinationreadiness.Plan {
	if p == nil {
		return destinationreadiness.Plan{}
	}
	return destinationreadiness.Plan{
		ID:                 p.Id,
		Action:             protoToPreparationAction(p.Action),
		Location:           p.Location,
		TargetPath:         p.TargetPath,
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

// usageStateToProto translates the domain UsageState to the proto enum.
func usageStateToProto(s destinations.UsageState) destinationsv1.UsageState {
	switch s {
	case destinations.UsageStateWithin:
		return destinationsv1.UsageState_USAGE_STATE_WITHIN
	case destinations.UsageStateNear:
		return destinationsv1.UsageState_USAGE_STATE_NEAR
	case destinations.UsageStateOver:
		return destinationsv1.UsageState_USAGE_STATE_OVER
	default:
		return destinationsv1.UsageState_USAGE_STATE_UNSPECIFIED
	}
}

func readinessSeverityToProto(s destinationreadiness.CheckSeverity) destinationsv1.ReadinessSeverity {
	switch s {
	case destinationreadiness.SeverityPass:
		return destinationsv1.ReadinessSeverity_READINESS_SEVERITY_PASS
	case destinationreadiness.SeverityWarning:
		return destinationsv1.ReadinessSeverity_READINESS_SEVERITY_WARNING
	case destinationreadiness.SeverityFail:
		return destinationsv1.ReadinessSeverity_READINESS_SEVERITY_FAIL
	case destinationreadiness.SeverityUnknown:
		return destinationsv1.ReadinessSeverity_READINESS_SEVERITY_UNKNOWN
	default:
		return destinationsv1.ReadinessSeverity_READINESS_SEVERITY_UNSPECIFIED
	}
}

func preparationActionToProto(a destinationreadiness.PreparationAction) destinationsv1.PreparationAction {
	switch a {
	case destinationreadiness.ActionCreateSubdir:
		return destinationsv1.PreparationAction_PREPARATION_ACTION_CREATE_SUBDIR
	case destinationreadiness.ActionRelabel:
		return destinationsv1.PreparationAction_PREPARATION_ACTION_RELABEL
	case destinationreadiness.ActionClearDirectory:
		return destinationsv1.PreparationAction_PREPARATION_ACTION_CLEAR_DIRECTORY
	case destinationreadiness.ActionFormat:
		return destinationsv1.PreparationAction_PREPARATION_ACTION_FORMAT
	default:
		return destinationsv1.PreparationAction_PREPARATION_ACTION_UNSPECIFIED
	}
}

func protoToPreparationAction(a destinationsv1.PreparationAction) destinationreadiness.PreparationAction {
	switch a {
	case destinationsv1.PreparationAction_PREPARATION_ACTION_CREATE_SUBDIR:
		return destinationreadiness.ActionCreateSubdir
	case destinationsv1.PreparationAction_PREPARATION_ACTION_RELABEL:
		return destinationreadiness.ActionRelabel
	case destinationsv1.PreparationAction_PREPARATION_ACTION_CLEAR_DIRECTORY:
		return destinationreadiness.ActionClearDirectory
	case destinationsv1.PreparationAction_PREPARATION_ACTION_FORMAT:
		return destinationreadiness.ActionFormat
	default:
		return ""
	}
}
