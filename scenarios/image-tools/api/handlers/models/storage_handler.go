package models

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"

	internalmodels "image-tools/internal/models"
	"image-tools/internal/modelstorage"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// ModelStorage is the model-weight storage lifecycle owner the storage RPCs
// read and drive (satisfied by *modelstorage.Service).
type ModelStorage interface {
	Config() modelstorage.Config
	Plan(ctx context.Context, trigger string, needBytes int64, requested []string) (modelstorage.Report, error)
	Offload(ctx context.Context, trigger string, needBytes int64, requested []string, dryRun bool) (modelstorage.Report, []modelstorage.Receipt, error)
	Receipts(ctx context.Context, limit int) ([]modelstorage.Receipt, error)
	SetKeep(ctx context.Context, id string, keep bool, reason string) error
}

func (h *connectHandler) GetModelStorage(ctx context.Context, req *connect.Request[modelsv1.GetModelStorageRequest]) (*connect.Response[modelsv1.GetModelStorageResponse], error) {
	if h.deps.Storage == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("model storage management unavailable"))
	}
	report, err := h.deps.Storage.Plan(ctx, "status", 0, nil)
	if err != nil {
		h.deps.Logger.Printf("models.GetModelStorage: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("evaluate model storage"))
	}
	limit := int(req.Msg.GetReceiptLimit())
	if limit <= 0 {
		limit = 20
	}
	receipts, err := h.deps.Storage.Receipts(ctx, limit)
	if err != nil {
		h.deps.Logger.Printf("models.GetModelStorage receipts: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("read offload receipts"))
	}
	return connect.NewResponse(&modelsv1.GetModelStorageResponse{Report: storageReportToProto(report, h.deps.Storage.Config()), Receipts: receiptsToProto(receipts)}), nil
}

func (h *connectHandler) OffloadModels(ctx context.Context, req *connect.Request[modelsv1.OffloadModelsRequest]) (*connect.Response[modelsv1.OffloadModelsResponse], error) {
	if h.deps.Storage == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("model storage management unavailable"))
	}
	requested := make([]string, 0, len(req.Msg.GetModelIds()))
	for _, id := range req.Msg.GetModelIds() {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, _, ok := h.resolve(ctx, id); !ok {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("model not found: "+id))
		}
		requested = append(requested, id)
	}
	need := req.Msg.GetNeedBytes()
	if need > 1<<50 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("need_bytes is out of range"))
	}
	report, receipts, err := h.deps.Storage.Offload(ctx, modelstorage.TriggerRequested, int64(need), requested, !req.Msg.GetApply())
	if err != nil && len(receipts) == 0 {
		h.deps.Logger.Printf("models.OffloadModels: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("evaluate model offload"))
	}
	if err != nil {
		h.deps.Logger.Printf("models.OffloadModels (partial): %v", err)
	}
	return connect.NewResponse(&modelsv1.OffloadModelsResponse{
		Report:   storageReportToProto(report, h.deps.Storage.Config()),
		Receipts: receiptsToProto(receipts),
		Applied:  req.Msg.GetApply(),
	}), nil
}

func (h *connectHandler) SetModelKeep(ctx context.Context, req *connect.Request[modelsv1.SetModelKeepRequest]) (*connect.Response[modelsv1.SetModelKeepResponse], error) {
	if h.deps.Storage == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("model storage management unavailable"))
	}
	id := strings.TrimSpace(req.Msg.GetId())
	if err := h.deps.Storage.SetKeep(ctx, id, req.Msg.GetKeep(), req.Msg.GetReason()); err != nil {
		if errors.Is(err, internalmodels.ErrModelNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("model not found"))
		}
		h.deps.Logger.Printf("models.SetModelKeep: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("set model keep pin"))
	}
	return connect.NewResponse(&modelsv1.SetModelKeepResponse{Id: id, Kept: req.Msg.GetKeep()}), nil
}

func storageReportToProto(r modelstorage.Report, cfg modelstorage.Config) *modelsv1.ModelStorageReport {
	p := cfg.Policy
	out := &modelsv1.ModelStorageReport{
		ObservedAt: formatTime(r.ObservedAt),
		Trigger:    r.Trigger,
		Policy: &modelsv1.ModelStoragePolicy{
			FloorBytes:                nonNegative(p.FloorBytes),
			ReserveBytes:              nonNegative(p.ReserveBytes),
			AmpleBytes:                nonNegative(p.AmpleBytes),
			MinIdleSeconds:            int64(p.MinIdle / time.Second),
			MaxIdleSeconds:            int64(p.MaxIdle / time.Second),
			RecentInstallGraceSeconds: int64(p.RecentInstall / time.Second),
			AutoAcquire:               cfg.AutoAcquire,
			AutoOffload:               cfg.AutoOffload,
			IntervalSeconds:           int64(cfg.Interval / time.Second),
			StagingGraceSeconds:       r.Staging.GraceSeconds,
		},
		FreeBytes:              nonNegative(r.Pressure.FreeBytes),
		HeadroomBytes:          r.Pressure.HeadroomBytes,
		Mode:                   r.Pressure.Mode,
		IdleThresholdSeconds:   -1,
		NeedBytes:              nonNegative(r.NeedBytes),
		TargetHeadroomBytes:    r.TargetHeadroomBytes,
		SelectedBytes:          nonNegative(r.SelectedBytes),
		ProjectedHeadroomBytes: r.ProjectedHeadroomBytes,
		ShortfallBytes:         nonNegative(r.ShortfallBytes),
		InstalledBytes:         nonNegative(r.InstalledBytes),
		StagingBytes:           nonNegative(r.Staging.TotalBytes),
		AbandonedStagingBytes:  nonNegative(r.Staging.AbandonedBytes),
		AbandonedStagingCount:  int32(r.Staging.AbandonedCount),
	}
	if !r.Pressure.Unlimited {
		out.IdleThresholdSeconds = int64(r.Pressure.IdleThreshold / time.Second)
	}
	for _, c := range r.Candidates {
		out.Candidates = append(out.Candidates, &modelsv1.ModelStorageCandidate{
			ModelId:           c.ModelID,
			Bytes:             nonNegative(c.Bytes),
			BytesBasis:        c.BytesBasis,
			InstalledAt:       formatTime(c.InstalledAt),
			LastSuccessfulUse: formatTime(c.LastSuccessfulUse),
			LastUse:           formatTime(c.LastUse),
			LastUseBasis:      c.LastUseBasis,
			IdleSeconds:       int64(c.Idle / time.Second),
			ReacquireSeconds:  c.ReacquireSeconds,
			Priority:          c.Priority,
			Protected:         append([]string(nil), c.Protected...),
			Selected:          c.Selected,
			SelectionReason:   c.SelectionReason,
		})
	}
	return out
}

func receiptsToProto(receipts []modelstorage.Receipt) []*modelsv1.ModelOffloadReceipt {
	out := make([]*modelsv1.ModelOffloadReceipt, 0, len(receipts))
	for _, r := range receipts {
		out = append(out, &modelsv1.ModelOffloadReceipt{
			Id: r.ID, ModelId: r.ModelID, Trigger: r.Trigger, Reason: r.Reason, Outcome: r.Outcome, Error: r.Error,
			Bytes: nonNegative(r.Bytes), Checksum: r.Checksum, LastUseAt: formatTime(r.LastUseAt), LastUseBasis: r.LastUseBasis,
			IdleSeconds: r.IdleSeconds, IdleThresholdSeconds: r.IdleThresholdSeconds, Mode: r.Mode,
			FreeBeforeBytes: nonNegative(r.FreeBeforeBytes), FreeAfterBytes: nonNegative(r.FreeAfterBytes), CreatedAt: formatTime(r.CreatedAt),
		})
	}
	return out
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func nonNegative(n int64) uint64 {
	if n < 0 {
		return 0
	}
	return uint64(n)
}
