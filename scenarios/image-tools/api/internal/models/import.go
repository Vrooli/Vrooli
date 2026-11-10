package models

import (
	"fmt"
	"strings"

	"image-tools/internal/hfmeta"
)

// Guided model import (plan capability D). This file turns an inspected source
// (hfmeta.Metadata) into a concrete registry entry: it infers + proposes an
// architecture (InferArchitecture), maps the detected layout to a fetch strategy
// (single-file → Assets[] on the stable-diffusion.cpp backend; diffusers-repo →
// a pinned Repo snapshot on the diffusers backend — decision D3), assembles the
// capability labels (provenance=user-imported, license read-or-unverified), and
// defaults the entry to local tier. The operator confirms the architecture (D1)
// before BuildImportEntry produces the final add-only entry.
//
// Why entry assembly lives here (not the handler): it is catalog policy — which
// backend serves which layout, which ops a base architecture declares, how an
// unvetted license maps to the commercial-use posture. Keeping it in the models
// package makes it pure + unit-testable and keeps the handler a thin translator.

// ImportProposal is the dry-run result of inspecting an import source: the
// inferred architecture (+confidence/evidence) and the proposed registry entry
// the operator confirms before install. It executes + installs nothing.
type ImportProposal struct {
	Metadata     hfmeta.Metadata
	Architecture Architecture
	Confidence   Confidence
	Evidence     string
	// Entry is the registry entry that WOULD be added (carrying the inferred
	// architecture + default operations); the operator may override id/name/
	// architecture/operations before import.
	Entry Model
	// EffectiveOps is the op set Entry would offer (native + proven-derived), so
	// the wizard can show "you'll get text_to_image + derived img2img/inpaint".
	EffectiveOps []EffectiveOp
}

// ImportConfirm carries the operator-confirmed fields ImportModel applies over a
// proposal before installing.
type ImportConfirm struct {
	// ID is the new entry's id (must not collide with a seed model). Required.
	ID string
	// Name is the display name (defaults to the source's last path segment).
	Name string
	// Architecture overrides the inferred value; REQUIRED when inference returned
	// ConfidenceNone (never guess silently — decision D1).
	Architecture Architecture
	// Operations overrides the architecture's default base operations.
	Operations []string
	// AttestCommercialRights records the operator's attestation that they hold
	// commercial rights, lifting the public/BYOK serving block (decision D4).
	AttestCommercialRights bool
}

// ProposeImport inspects metadata into a proposal: it infers the architecture and
// assembles the default entry. The proposal's Entry carries a default id derived
// from the source; the operator confirms/overrides before ImportModel.
func ProposeImport(meta hfmeta.Metadata) ImportProposal {
	arch, conf, evidence := InferArchitecture(meta)
	entry := assembleEntry(meta, arch, meta.ImportID("imported-model"), meta.ImportName("Imported model"), nil)
	return ImportProposal{
		Metadata:     meta,
		Architecture: arch,
		Confidence:   conf,
		Evidence:     evidence,
		Entry:        entry,
		EffectiveOps: entry.EffectiveOps(),
	}
}

// BuildImportEntry assembles the final add-only entry from inspected metadata and
// the operator's confirmation. It fails closed when the architecture is still
// unresolved (inference none AND no confirmed architecture) or the layout cannot
// be mapped to a runnable backend — never a silently-guessed or un-runnable entry.
func BuildImportEntry(meta hfmeta.Metadata, confirm ImportConfirm) (Model, error) {
	if strings.TrimSpace(confirm.ID) == "" {
		return Model{}, fmt.Errorf("import: a confirmed id is required")
	}
	arch := confirm.Architecture
	if arch == "" || arch == ArchNone {
		// Fall back to inference only when the operator did not confirm one.
		if inferred, _, _ := InferArchitecture(meta); inferred != ArchNone {
			arch = inferred
		}
	}
	if arch == "" || arch == ArchNone {
		return Model{}, fmt.Errorf("import: architecture could not be inferred and was not confirmed — select one explicitly")
	}
	if !confirm.AttestCommercialRights {
		return Model{}, fmt.Errorf("import: confirm that you hold commercial rights before registering or downloading model weights")
	}
	if !arch.valid() {
		return Model{}, fmt.Errorf("import: unknown architecture %q", arch)
	}
	name := confirm.Name
	if strings.TrimSpace(name) == "" {
		name = meta.ImportName("Imported model")
	}
	entry := assembleEntry(meta, arch, confirm.ID, name, confirm.Operations)
	if entry.Backend == "" {
		return Model{}, fmt.Errorf("import: source layout %q has no runnable backend (provide a single-file checkpoint or a diffusers repo)", meta.Layout)
	}
	// The final import flow only accepts an explicit rights attestation. Dry-run
	// proposals remain conditional until the operator confirms in ImportConfirm.
	entry.CapabilityLabels.CommercialUse = CommercialUseYes
	entry.CapabilityLabels.CommercialUseNotes = "operator-attested commercial rights at import time"
	return entry, nil
}

// assembleEntry builds the registry entry for an inspected source + architecture.
// Operations default to the architecture's base ops unless overridden.
func assembleEntry(meta hfmeta.Metadata, arch Architecture, id, name string, ops []string) Model {
	if len(ops) == 0 {
		ops = baseOperationsFor(arch)
	}
	m := Model{
		ID:           id,
		Name:         name,
		Operations:   ops,
		Architecture: arch,
		Tier:         TierNiceToHave,
		SizeMBApprox: int(meta.TotalSize() >> 20),
		Hardware:     importHardwareFor(arch),
		CapabilityLabels: CapabilityLabels{
			NSFWCapable:      meta.NSFW,
			License:          meta.LicenseLabel(),
			BaseModelLineage: meta.BaseModel,
			Provenance:       ProvenanceUserImported,
		},
	}
	applyLayoutSource(&m, meta)
	return m
}

// importHardwareFor returns conservative, architecture-keyed hardware bounds for
// an imported model. It MUST set at least one of CPUCapable/GPURequired so the
// entry satisfies the registry's runnability invariant (a model with neither is
// un-runnable). SD1.5 is genuinely CPU-viable; the larger architectures are
// GPU-required (CPU generation is impractically slow) with a higher VRAM floor.
func importHardwareFor(arch Architecture) Hardware {
	switch arch {
	case ArchSDXL:
		return Hardware{GPURequired: true, MinVRAMGB: 8, MinRAMGB: 16}
	case ArchFlux:
		return Hardware{GPURequired: true, MinVRAMGB: 12, MinRAMGB: 32}
	case ArchSD15:
		return Hardware{CPUCapable: true, MinVRAMGB: 2, MinRAMGB: 8}
	default:
		// Unknown-but-valid architecture: assume CPU-viable so it is at least
		// offerable; the operator can refine the entry if needed.
		return Hardware{CPUCapable: true, MinVRAMGB: 4, MinRAMGB: 8}
	}
}

func applyLayoutSource(m *Model, meta hfmeta.Metadata) {
	switch meta.Layout {
	case hfmeta.LayoutSingleFile:
		m.Backend = "stable-diffusion.cpp"
		m.AltBackends = []string{BackendDiffusers}
	case hfmeta.LayoutDiffusersRepo:
		m.Backend = BackendDiffusers
	default:
		return
	}
	m.Source = meta.ImportSource(".safetensors", ".ckpt", ".gguf")
}

// baseOperationsFor is the native op set an imported base model declares for its
// architecture. Base txt2img checkpoints declare generate + transform and DERIVE
// inpaint/outpaint via the architecture table; instruction-edit architectures
// declare the single edit op.
func baseOperationsFor(arch Architecture) []string {
	switch arch {
	case ArchSD15, ArchSDXL, ArchFlux:
		return []string{"text_to_image", "image_to_image"}
	case ArchInstructPix2Pix, ArchQwenImageEdit, ArchLongCatImageEdit:
		return []string{"edit_instruct"}
	default:
		return nil
	}
}
