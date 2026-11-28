package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	internaladapters "image-tools/internal/adapters"
	internaljobs "image-tools/internal/jobs"
	internalmodels "image-tools/internal/models"

	"connectrpc.com/connect"

	adaptersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/adapters"
)

// Guided adapter import handlers (plan capability C). InspectAdapterSource is the
// read-only "look before you leap" dry run; ImportAdapter composes inspect →
// operator-confirmed entry → add-only registration → durable install job.

// InspectAdapterSource inspects an import source and returns the proposed entry +
// inferred kind/architecture without installing anything.
func (h *connectHandler) InspectAdapterSource(ctx context.Context, req *connect.Request[adaptersv1.InspectAdapterSourceRequest]) (*connect.Response[adaptersv1.InspectAdapterSourceResponse], error) {
	if h.deps.Inspector == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("adapter import unavailable"))
	}
	source := strings.TrimSpace(req.Msg.GetSource())
	if source == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("a source (HF repo id, URL, or local path) is required"))
	}
	meta, err := h.deps.Inspector.Inspect(ctx, source)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	prop := internaladapters.ProposeImport(meta)
	overlay, _ := h.overlay(ctx)
	installs, _ := h.installStates(ctx)

	return connect.NewResponse(&adaptersv1.InspectAdapterSourceResponse{
		Source:   meta.Source,
		RepoId:   meta.RepoID,
		Revision: meta.Revision,
		Kind: &adaptersv1.KindInference{
			Kind:     string(prop.Kind),
			Evidence: prop.KindEvidence,
		},
		Architecture: &adaptersv1.ArchitectureInference{
			Architecture: string(prop.Architecture),
			Confidence:   string(prop.Confidence),
			Evidence:     prop.ArchEvidence,
		},
		License:   meta.License,
		Nsfw:      meta.NSFW,
		SizeBytes: uint64(maxInt64(0, meta.TotalSize())),
		Proposed:  domainToProto(prop.Entry, viewFor(prop.Entry, true, overlay, installs)),
	}), nil
}

// ImportAdapter registers the confirmed entry (add-only) and submits its install
// job (mirrors InstallAdapter). An already-installed entry returns no job.
func (h *connectHandler) ImportAdapter(ctx context.Context, req *connect.Request[adaptersv1.ImportAdapterRequest]) (*connect.Response[adaptersv1.ImportAdapterResponse], error) {
	if h.deps.Installer == nil || h.deps.Installer.Custom == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("custom adapters unavailable"))
	}
	if h.deps.Inspector == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("adapter import unavailable"))
	}
	source := strings.TrimSpace(req.Msg.GetSource())
	if source == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("a source is required"))
	}
	meta, err := h.deps.Inspector.Inspect(ctx, source)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	entry, err := internaladapters.BuildImportEntry(meta, internaladapters.ImportConfirm{
		ID:                     strings.TrimSpace(req.Msg.GetId()),
		Name:                   req.Msg.GetName(),
		Kind:                   internaladapters.Kind(strings.TrimSpace(req.Msg.GetKind())),
		Architecture:           internalmodels.Architecture(strings.TrimSpace(req.Msg.GetArchitecture())),
		Preprocessor:           internaladapters.Preprocessor(strings.TrimSpace(req.Msg.GetPreprocessor())),
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
	adapterProto := domainToProto(entry, viewFor(entry, true, overlay, installs))

	if h.deps.Installer.Installed(ctx, entry.ID) {
		return connect.NewResponse(&adaptersv1.ImportAdapterResponse{Adapter: adapterProto, AlreadyInstalled: true}), nil
	}
	if h.deps.Jobs == nil {
		// Registered but no job runner wired: the entry exists; the operator can
		// install it via `adapters install`. Honest rather than a fake job id.
		return connect.NewResponse(&adaptersv1.ImportAdapterResponse{Adapter: adapterProto}), nil
	}

	payload, err := json.Marshal(internaladapters.InstallPayload{AdapterID: entry.ID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("encode install request"))
	}
	eta := h.deps.EstimateInstallSeconds(entry.SizeMBApprox)
	job, err := h.deps.Jobs.Submit(ctx, internaljobs.Spec{
		Operation:        internaladapters.InstallJobOperation,
		Lane:             internaljobs.LaneCPU,
		Payload:          payload,
		EstimatedSeconds: eta,
	})
	if err != nil {
		h.deps.Logger.Printf("adapters.ImportAdapter submit: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("submit install job"))
	}
	return connect.NewResponse(&adaptersv1.ImportAdapterResponse{
		Adapter:    adapterProto,
		JobId:      job.ID,
		EtaSeconds: int32(eta),
	}), nil
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
