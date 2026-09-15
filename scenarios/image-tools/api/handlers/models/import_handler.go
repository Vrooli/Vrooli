package models

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	internalhfmeta "image-tools/internal/hfmeta"
	internaljobs "image-tools/internal/jobs"
	internalmodels "image-tools/internal/models"

	"connectrpc.com/connect"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// Guided model import handlers (plan capability D). InspectModelSource is the
// read-only "look before you leap" dry run; ImportModel composes inspect →
// operator-confirmed entry → add-only registration → durable install job.

// InspectModelSource inspects an import source and returns the proposed entry +
// inferred architecture without installing anything.
func (h *connectHandler) InspectModelSource(ctx context.Context, req *connect.Request[modelsv1.InspectModelSourceRequest]) (*connect.Response[modelsv1.InspectModelSourceResponse], error) {
	if h.deps.Hfmeta == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("model import unavailable"))
	}
	source := strings.TrimSpace(req.Msg.GetSource())
	if source == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("a source (HF repo id, URL, or local path) is required"))
	}
	meta, err := h.deps.Hfmeta.Inspect(ctx, source)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	prop := internalmodels.ProposeImport(meta)
	overlay, _ := h.overlay(ctx)
	installs, _ := h.installStates(ctx)

	offered := make([]string, 0, len(prop.EffectiveOps))
	for _, eo := range prop.EffectiveOps {
		if eo.Offerable() {
			offered = append(offered, eo.Op)
		}
	}
	return connect.NewResponse(&modelsv1.InspectModelSourceResponse{
		Source:   meta.Source,
		RepoId:   meta.RepoID,
		Revision: meta.Revision,
		Layout:   layoutToProto(meta.Layout),
		Architecture: &modelsv1.ArchitectureInference{
			Architecture: string(prop.Architecture),
			Confidence:   string(prop.Confidence),
			Evidence:     prop.Evidence,
		},
		License:           meta.License,
		Nsfw:              meta.NSFW,
		SizeBytes:         uint64(maxInt64(0, meta.TotalSize())),
		PipelineClass:     meta.PipelineClass,
		OfferedOperations: offered,
		Proposed:          domainToProto(prop.Entry, viewFor(prop.Entry, true, overlay, installs)),
	}), nil
}

// ImportModel registers the confirmed entry (add-only) and submits its install
// job (mirrors InstallModel). An already-installed entry returns no job.
func (h *connectHandler) ImportModel(ctx context.Context, req *connect.Request[modelsv1.ImportModelRequest]) (*connect.Response[modelsv1.ImportModelResponse], error) {
	if h.deps.Installer == nil || h.deps.Installer.Custom == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("custom models unavailable"))
	}
	if h.deps.Hfmeta == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("model import unavailable"))
	}
	source := strings.TrimSpace(req.Msg.GetSource())
	if source == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("a source is required"))
	}
	meta, err := h.deps.Hfmeta.Inspect(ctx, source)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	entry, err := internalmodels.BuildImportEntry(meta, internalmodels.ImportConfirm{
		ID:                     strings.TrimSpace(req.Msg.GetId()),
		Name:                   req.Msg.GetName(),
		Architecture:           internalmodels.Architecture(strings.TrimSpace(req.Msg.GetArchitecture())),
		Operations:             req.Msg.GetOperations(),
		AttestCommercialRights: req.Msg.GetAttestCommercialRights(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := h.deps.Installer.AddCustom(ctx, entry); err != nil {
		return nil, addCustomError(err)
	}

	overlay, _ := h.overlay(ctx)
	installs, _ := h.installStates(ctx)
	modelProto := domainToProto(entry, viewFor(entry, true, overlay, installs))

	if h.deps.Installer.Installed(ctx, entry.ID) {
		return connect.NewResponse(&modelsv1.ImportModelResponse{Model: modelProto, AlreadyInstalled: true}), nil
	}
	if h.deps.Jobs == nil {
		// Registered but no job runner wired: the entry exists; the operator can
		// install it via `models install`. Honest rather than a fake job id.
		return connect.NewResponse(&modelsv1.ImportModelResponse{Model: modelProto}), nil
	}

	payload, err := json.Marshal(internalmodels.InstallPayload{ModelID: entry.ID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("encode install request"))
	}
	eta := h.deps.EstimateInstallSeconds(entry.SizeMBApprox)
	job, err := h.deps.Jobs.Submit(ctx, internaljobs.Spec{
		Operation:        internalmodels.InstallJobOperation,
		Lane:             internaljobs.LaneCPU,
		Payload:          payload,
		EstimatedSeconds: eta,
	})
	if err != nil {
		h.deps.Logger.Printf("models.ImportModel submit: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("submit install job"))
	}
	return connect.NewResponse(&modelsv1.ImportModelResponse{
		Model:      modelProto,
		JobId:      job.ID,
		EtaSeconds: int32(eta),
	}), nil
}

func layoutToProto(l internalhfmeta.Layout) modelsv1.ModelLayout {
	switch l {
	case internalhfmeta.LayoutSingleFile:
		return modelsv1.ModelLayout_MODEL_LAYOUT_SINGLE_FILE
	case internalhfmeta.LayoutDiffusersRepo:
		return modelsv1.ModelLayout_MODEL_LAYOUT_DIFFUSERS_REPO
	default:
		return modelsv1.ModelLayout_MODEL_LAYOUT_UNSPECIFIED
	}
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
