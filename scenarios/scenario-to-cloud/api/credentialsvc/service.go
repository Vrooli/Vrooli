// Package credentialsvc is the Connect implementation of
// vrooli.scenario_to_cloud.v1.credentials.CredentialsService and the shared
// request/response shapes behind the REST credential routes. The lifecycle
// itself lives in package credentials; this package adapts transports and
// maps every failure onto the typed apierrors.Error, carrying the durable
// operation alongside a non-terminal refusal so a client always sees the
// truthful standing.
package credentialsvc

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/credentials"
	"scenario-to-cloud/deploymentsvc"
	"scenario-to-cloud/domain"

	credentialsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/credentials"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/credentials/credentialsv1connect"
)

// SchemaVersion is the response schema version clients negotiate on.
const SchemaVersion = "1"

// RotateRequest is the transport-neutral rotation request. Value is inbound
// only and never persisted or echoed.
type RotateRequest struct {
	DeploymentID string `json:"deployment_id"`
	BindingID    string `json:"binding_id"`
	Value        string `json:"value,omitempty"`
	RequestKey   string `json:"request_key,omitempty"`
}

// RevokeRequest revokes the active version of a binding.
type RevokeRequest struct {
	DeploymentID string `json:"deployment_id"`
	BindingID    string `json:"binding_id"`
	RequestKey   string `json:"request_key,omitempty"`
}

// RecoverRequest re-provisions a deployment onto a replacement host.
type RecoverRequest struct {
	DeploymentID string `json:"deployment_id"`
	BundleRef    string `json:"bundle_ref"`
	Passphrase   string `json:"passphrase,omitempty"`
	RequestKey   string `json:"request_key,omitempty"`
}

// ResumeRequest continues a non-terminal operation.
type ResumeRequest struct {
	DeploymentID      string `json:"deployment_id"`
	RotationID        string `json:"rotation_id"`
	OperatorConfirmed bool   `json:"operator_confirmed"`
}

// BreakGlassRequest issues an emergency version.
type BreakGlassRequest struct {
	DeploymentID  string `json:"deployment_id"`
	BindingID     string `json:"binding_id"`
	Scope         string `json:"scope"`
	WindowSeconds int64  `json:"window_seconds"`
	Confirmation  string `json:"confirmation"`
	Operator      string `json:"operator"`
	RequestKey    string `json:"request_key,omitempty"`
}

// Lifecycle is what the server provides: it resolves the deployment's target
// and runs the credential lifecycle against it. An error may be accompanied
// by the durable operation it refers to.
type Lifecycle interface {
	ListBindings(ctx context.Context, deploymentID string) ([]credentials.BindingView, error)
	Rotate(ctx context.Context, req RotateRequest) (*domain.CredentialRotation, error)
	Revoke(ctx context.Context, req RevokeRequest) (*domain.CredentialRotation, error)
	Recover(ctx context.Context, req RecoverRequest) (*domain.CredentialRotation, error)
	GetRotation(ctx context.Context, deploymentID, rotationID string) (*domain.CredentialRotation, error)
	Resume(ctx context.Context, req ResumeRequest) (*domain.CredentialRotation, error)
	BreakGlass(ctx context.Context, req BreakGlassRequest) (*domain.CredentialRotation, error)
}

// Service implements credentialsv1connect.CredentialsServiceHandler.
type Service struct {
	lifecycle Lifecycle
}

// New builds the service.
func New(lifecycle Lifecycle) *Service { return &Service{lifecycle: lifecycle} }

// Handler returns the Connect mount path and handler.
func (s *Service) Handler(opts ...connect.HandlerOption) (string, http.Handler) {
	return credentialsv1connect.NewCredentialsServiceHandler(s, opts...)
}

// OperationError attaches the operation to a typed error so a non-terminal
// standing (revocation_incomplete, pending_operator_input) travels with its
// refusal. Untyped errors become internal errors.
func OperationError(err error, rotation *domain.CredentialRotation) *apierrors.Error {
	if err == nil {
		return nil
	}
	typed := apierrors.As(err)
	if typed == nil {
		typed = apierrors.Internal("credential lifecycle failed", err)
	}
	if typed.HTTPStatus == 0 {
		if status, ok := credentials.HTTPStatusFor(typed.Code); ok {
			typed.HTTPStatus = status
		}
	}
	if rotation != nil {
		typed = typed.WithDetail("operation", rotation)
	}
	return typed
}

func (s *Service) ListBindings(ctx context.Context, req *connect.Request[credentialsv1.ListBindingsRequest]) (*connect.Response[credentialsv1.ListBindingsResponse], error) {
	if strings.TrimSpace(req.Msg.GetDeploymentId()) == "" {
		return nil, deploymentsvc.ConnectError(apierrors.New(apierrors.CodeInvalidRequest, "deployment_id is required"))
	}
	views, err := s.lifecycle.ListBindings(ctx, req.Msg.GetDeploymentId())
	if err != nil {
		return nil, deploymentsvc.ConnectError(OperationError(err, nil))
	}
	out := &credentialsv1.ListBindingsResponse{SchemaVersion: SchemaVersion}
	for _, view := range views {
		out.Bindings = append(out.Bindings, BindingViewProto(view))
	}
	return connect.NewResponse(out), nil
}

func (s *Service) RotateCredential(ctx context.Context, req *connect.Request[credentialsv1.RotateCredentialRequest]) (*connect.Response[credentialsv1.CredentialOperationResponse], error) {
	rotation, err := s.lifecycle.Rotate(ctx, RotateRequest{DeploymentID: req.Msg.GetDeploymentId(), BindingID: req.Msg.GetBindingId(), Value: req.Msg.GetValue(), RequestKey: req.Msg.GetRequestKey()})
	return operationResponse(rotation, err)
}

func (s *Service) RevokeCredential(ctx context.Context, req *connect.Request[credentialsv1.RevokeCredentialRequest]) (*connect.Response[credentialsv1.CredentialOperationResponse], error) {
	rotation, err := s.lifecycle.Revoke(ctx, RevokeRequest{DeploymentID: req.Msg.GetDeploymentId(), BindingID: req.Msg.GetBindingId(), RequestKey: req.Msg.GetRequestKey()})
	return operationResponse(rotation, err)
}

func (s *Service) RecoverCredentials(ctx context.Context, req *connect.Request[credentialsv1.RecoverCredentialsRequest]) (*connect.Response[credentialsv1.CredentialOperationResponse], error) {
	rotation, err := s.lifecycle.Recover(ctx, RecoverRequest{DeploymentID: req.Msg.GetDeploymentId(), BundleRef: req.Msg.GetBundleRef(), Passphrase: req.Msg.GetPassphrase(), RequestKey: req.Msg.GetRequestKey()})
	return operationResponse(rotation, err)
}

func (s *Service) GetRotation(ctx context.Context, req *connect.Request[credentialsv1.GetRotationRequest]) (*connect.Response[credentialsv1.CredentialOperationResponse], error) {
	rotation, err := s.lifecycle.GetRotation(ctx, req.Msg.GetDeploymentId(), req.Msg.GetRotationId())
	return operationResponse(rotation, err)
}

func (s *Service) ResumeRotation(ctx context.Context, req *connect.Request[credentialsv1.ResumeRotationRequest]) (*connect.Response[credentialsv1.CredentialOperationResponse], error) {
	rotation, err := s.lifecycle.Resume(ctx, ResumeRequest{DeploymentID: req.Msg.GetDeploymentId(), RotationID: req.Msg.GetRotationId(), OperatorConfirmed: req.Msg.GetOperatorConfirmed()})
	return operationResponse(rotation, err)
}

func (s *Service) BreakGlass(ctx context.Context, req *connect.Request[credentialsv1.BreakGlassRequest]) (*connect.Response[credentialsv1.CredentialOperationResponse], error) {
	rotation, err := s.lifecycle.BreakGlass(ctx, BreakGlassRequest{DeploymentID: req.Msg.GetDeploymentId(), BindingID: req.Msg.GetBindingId(), Scope: req.Msg.GetScope(), WindowSeconds: req.Msg.GetWindowSeconds(), Confirmation: req.Msg.GetConfirmation(), Operator: req.Msg.GetOperator(), RequestKey: req.Msg.GetRequestKey()})
	return operationResponse(rotation, err)
}

func operationResponse(rotation *domain.CredentialRotation, err error) (*connect.Response[credentialsv1.CredentialOperationResponse], error) {
	if err != nil {
		return nil, deploymentsvc.ConnectError(OperationError(err, rotation))
	}
	return connect.NewResponse(&credentialsv1.CredentialOperationResponse{SchemaVersion: SchemaVersion, Operation: OperationProto(rotation)}), nil
}

// BindingViewProto maps a binding and its acks.
func BindingViewProto(view credentials.BindingView) *credentialsv1.BindingView {
	out := &credentialsv1.BindingView{Binding: BindingProto(view.Binding), LifecycleState: view.LifecycleState, LifecycleDetail: view.LifecycleDetail, NextAction: view.NextAction}
	for _, ack := range view.Acks {
		out.Acks = append(out.Acks, &credentialsv1.CredentialAck{BindingId: ack.BindingID, Consumer: ack.Consumer, Version: ack.Version, VerifiedAt: timestamppb.New(ack.VerifiedAt)})
	}
	return out
}

// BindingProto maps a binding. It carries references only.
func BindingProto(b domain.CredentialBinding) *credentialsv1.CredentialBinding {
	out := &credentialsv1.CredentialBinding{
		Id: b.ID, DeploymentId: b.DeploymentID,
		Descriptor_:  &credentialsv1.CredentialDescriptor{LogicalId: b.Descriptor.LogicalID, Field: b.Descriptor.Field},
		Class:        string(b.Class),
		SourceClass:  b.SourceClass,
		TargetType:   b.Target.Type,
		TargetName:   b.Target.Name,
		Version:      versionProto(b.Version),
		ConsumerRefs: append([]string(nil), b.ConsumerRefs...),
		GrantRef:     b.GrantRef, RecoveryKeyRef: b.RecoveryKeyRef,
		State:     string(b.State),
		CreatedAt: timestamppb.New(b.CreatedAt), UpdatedAt: timestamppb.New(b.UpdatedAt),
	}
	if b.PreviousVersion != nil {
		out.PreviousVersion = versionProto(*b.PreviousVersion)
	}
	return out
}

func versionProto(v domain.CredentialVersion) *credentialsv1.CredentialVersion {
	out := &credentialsv1.CredentialVersion{Number: v.Number, ContentRef: v.ContentRef}
	if !v.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(v.CreatedAt)
	}
	if v.ExpiresAt != nil {
		out.ExpiresAt = timestamppb.New(*v.ExpiresAt)
	}
	return out
}

// OperationProto maps a lifecycle operation.
func OperationProto(r *domain.CredentialRotation) *credentialsv1.CredentialOperation {
	if r == nil {
		return nil
	}
	out := &credentialsv1.CredentialOperation{
		Id: r.ID, DeploymentId: r.DeploymentID, BindingId: r.BindingID, Kind: string(r.Kind),
		FromVersion: r.FromVersion, ToVersion: r.ToVersion, State: string(r.State),
		Unreached: append([]string(nil), r.Unreached...),
		CreatedAt: timestamppb.New(r.CreatedAt), UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
	for _, c := range r.Consumers {
		out.Consumers = append(out.Consumers, &credentialsv1.ConsumerProgress{Consumer: c.Consumer, State: string(c.State), Version: c.Version, Reason: c.Reason, UpdatedAt: timestamppb.New(c.UpdatedAt)})
	}
	if r.PendingOperatorInput != nil {
		h := r.PendingOperatorInput
		out.PendingOperatorInput = &credentialsv1.OperatorHandoff{Reference: h.Reference, Provider: h.Provider, Instruction: h.Instruction, ResumeWith: h.ResumeWith, RequestedAt: timestamppb.New(h.RequestedAt)}
	}
	if r.ResumeAfter != nil {
		out.ResumeAfter = timestamppb.New(*r.ResumeAfter)
	}
	if r.BreakGlass != nil {
		w := r.BreakGlass
		out.BreakGlass = &credentialsv1.BreakGlassWindow{Scope: w.Scope, Operator: w.Operator, IssuedAt: timestamppb.New(w.IssuedAt), ExpiresAt: timestamppb.New(w.ExpiresAt), PredecessorRef: w.PredecessorRef, ConfirmationRef: w.ConfirmationRef}
		if w.AutoRevokedAt != nil {
			out.BreakGlass.AutoRevokedAt = timestamppb.New(*w.AutoRevokedAt)
		}
	}
	for _, receipt := range r.Receipts {
		out.Receipts = append(out.Receipts, receiptProto(receipt))
	}
	if r.Error != nil {
		out.Error = &credentialsv1.OperationError{Code: r.Error.Code, Message: r.Error.Message}
	}
	if r.CompletedAt != nil {
		out.CompletedAt = timestamppb.New(*r.CompletedAt)
	}
	return out
}

func receiptProto(receipt domain.CredentialReceipt) *credentialsv1.Receipt {
	out := &credentialsv1.Receipt{Step: receipt.Step, State: receipt.State, Outcome: receipt.Outcome, At: timestamppb.New(receipt.At), TargetReceiptRef: receipt.TargetReceiptRef, Limitations: append([]string(nil), receipt.Limitations...)}
	if len(receipt.Details) > 0 {
		if details, err := structpb.NewStruct(jsonSafe(receipt.Details)); err == nil {
			out.Details = details
		}
	}
	return out
}

// jsonSafe converts detail values structpb cannot hold directly (time.Time,
// []string, nested typed values) into plain JSON-compatible values.
func jsonSafe(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = jsonSafeValue(v)
	}
	return out
}

func jsonSafeValue(v any) any {
	switch t := v.(type) {
	case nil, bool, string, float64, float32, int, int32, int64, uint, uint32, uint64:
		return normaliseNumber(t)
	case time.Time:
		return t.UTC().Format(time.RFC3339Nano)
	case *time.Time:
		if t == nil {
			return nil
		}
		return t.UTC().Format(time.RFC3339Nano)
	case []string:
		out := make([]any, 0, len(t))
		for _, s := range t {
			out = append(out, s)
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, jsonSafeValue(item))
		}
		return out
	case map[string]any:
		return jsonSafe(t)
	case map[string]string:
		out := make(map[string]any, len(t))
		for k, s := range t {
			out[k] = s
		}
		return out
	default:
		// Anything else goes through its JSON encoding so nothing is lost.
		raw, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		var generic any
		if err := json.Unmarshal(raw, &generic); err != nil {
			return string(raw)
		}
		return generic
	}
}

func normaliseNumber(v any) any {
	switch t := v.(type) {
	case int:
		return float64(t)
	case int32:
		return float64(t)
	case int64:
		return float64(t)
	case uint:
		return float64(t)
	case uint32:
		return float64(t)
	case uint64:
		return float64(t)
	case float32:
		return float64(t)
	}
	return v
}
