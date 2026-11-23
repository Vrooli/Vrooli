package models

import (
	"fmt"
	"path"
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
	// Family is the diffusers adapter the import resolved from the repo's pipeline
	// class ("" when the source is not a diffusers repo or has no adapter yet).
	Family string
	// Needs lists, in plain words, what still stands between this proposal and a
	// model that runs. Empty means the entry is selectable as soon as it installs.
	Needs []string
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
	// It is optional: an unattested import runs locally (normal use) and carries a
	// conditional commercial label that keeps hosted, monetized and redistributed
	// use refused.
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
		Family:       entry.Runtime.Family,
		Needs:        importNeeds(meta, arch, conf, entry),
	}
}

// importNeeds says in plain words what an import still lacks to be runnable, so a
// source that cannot run says which piece is missing and where it goes instead of
// registering a model that fails at its first job.
func importNeeds(meta hfmeta.Metadata, arch Architecture, conf Confidence, entry Model) []string {
	var needs []string
	if arch == ArchNone || conf == ConfidenceNone {
		needs = append(needs, "the architecture could not be inferred: pass --architecture (one of "+architectureNames()+")")
	}
	if entry.Backend == "" {
		needs = append(needs, fmt.Sprintf("the source layout %q has no runnable backend: provide a single-file checkpoint or a diffusers repo", meta.Layout))
	}
	if gap := adapterGap(meta, arch, entry); gap != "" {
		needs = append(needs, gap)
	}
	if fam, ok := DiffusersFamilyByName(entry.Runtime.Family); ok && !fam.Ready {
		needs = append(needs, fmt.Sprintf("the %q adapter is not proven runnable yet: %s", fam.Name, fam.Pending))
	}
	return needs
}

// adapterGap names the missing adapter when a diffusers repo needs one and none
// is registered for its pipeline class ("" otherwise).
func adapterGap(meta hfmeta.Metadata, arch Architecture, entry Model) string {
	if meta.Layout != hfmeta.LayoutDiffusersRepo || entry.Runtime.Family != "" || !wantsFamilyAdapter(arch, meta.PipelineClass) {
		return ""
	}
	return fmt.Sprintf("no diffusers adapter is registered for pipeline class %q: add a DiffusersFamily row in api/internal/models/families.go and an Adapter in api/internal/sidecar/py/image_tools_sidecar/_diffusers.py (docs/reference/model-registry.md, \"Add a diffusers family\")", meta.PipelineClass)
}

// wantsFamilyAdapter reports whether a diffusers repo of this architecture runs
// through a family adapter. Base SD/SDXL repos run through the derived
// techniques (img2img, inpaint) and need none; edit-specialised and unified
// architectures do.
func wantsFamilyAdapter(arch Architecture, pipelineClass string) bool {
	if strings.TrimSpace(pipelineClass) == "" {
		return false
	}
	switch arch {
	case ArchSD15, ArchSDXL, ArchNone:
		return false
	}
	return true
}

func architectureNames() string {
	names := make([]string, 0, len(architectures))
	for _, a := range Architectures() {
		names = append(names, string(a))
	}
	return strings.Join(names, "|")
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
	// Never register a model whose first job is certain to fail: say what is
	// missing instead.
	if gap := adapterGap(meta, arch, entry); gap != "" {
		return Model{}, fmt.Errorf("import: %s", gap)
	}
	// Licensing never blocks normal local use (operator stance 2026-11-21): an
	// unattested import runs locally and keeps hosted, monetized and redistributed
	// use refused by its conditional label; an attestation lifts that label.
	if confirm.AttestCommercialRights {
		entry.CapabilityLabels.CommercialUse = CommercialUseYes
		entry.CapabilityLabels.CommercialUseNotes = "operator-attested commercial rights at import time"
	}
	return entry, nil
}

// assembleEntry builds the registry entry for an inspected source + architecture.
// Operations default to the architecture's base ops unless overridden.
func assembleEntry(meta hfmeta.Metadata, arch Architecture, id, name string, ops []string) Model {
	defaultOps := len(ops) == 0
	if defaultOps {
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
			NSFWCapable:        meta.NSFW,
			License:            meta.LicenseLabel(),
			CommercialUse:      CommercialUseConditional,
			CommercialUseNotes: unattestedImportNotes,
			BaseModelLineage:   meta.BaseModel,
			Provenance:         ProvenanceUserImported,
		},
	}
	applyLayoutSource(&m, meta)
	applyFamily(&m, meta, defaultOps)
	// An import is selectable the moment its weights install, graded "untested"
	// until evidence arrives; only an adapter that is not yet proven keeps it off.
	m.Enabled = m.Backend != "" && !familyPending(m)
	if !m.Enabled && m.Backend != "" {
		m.Disabled = &DisabledReason{Kind: DisabledRuntime, Reason: pendingFamilyReason(m)}
	}
	return m
}

// pendingFamilyReason is the reason an import of a registered-but-unproven
// adapter family stays off.
func pendingFamilyReason(m Model) string {
	fam, _ := DiffusersFamilyByName(m.Runtime.Family)
	if fam.Pending != "" {
		return fam.Pending
	}
	return "its diffusers adapter family (" + m.Runtime.Family + ") is registered but not yet proven to run"
}

// unattestedImportNotes explains the conditional label of an import whose
// licence the operator has not vouched for. It is a recorded fact, not a gate on
// local use.
const unattestedImportNotes = "licence not verified at import: normal local use is allowed; hosted, monetized or redistributed use needs the licence checked and --attest-commercial-rights"

// applyFamily binds a diffusers repo to the adapter its pipeline class names, so
// the import runs with no operator-supplied runtime fields. When the operator did
// not list operations, the family's operations replace the architecture's base
// ones (a unified family also edits; an edit family only edits).
func applyFamily(m *Model, meta hfmeta.Metadata, defaultOps bool) {
	if m.Backend != BackendDiffusers {
		return
	}
	fam, ok := DiffusersFamilyForPipelineClass(meta.PipelineClass)
	if !ok {
		return
	}
	m.Runtime = Runtime{Family: fam.Name, PipelineClass: fam.PipelineClass, MinRuntime: fam.MinRuntime, Variant: m.Runtime.Variant}
	if defaultOps && len(fam.Operations) > 0 {
		m.Operations = append([]string(nil), fam.Operations...)
	}
}

func familyPending(m Model) bool {
	fam, ok := DiffusersFamilyByName(m.Runtime.Family)
	return ok && !fam.Ready
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
	if m.Source.HasRepo() {
		plan := planRepoFetch(meta)
		m.Source.Repo.AllowPatterns = plan.Files
		m.Runtime.Variant = plan.Variant
		if plan.Bytes > 0 {
			m.SizeMBApprox = int(plan.Bytes >> 20)
		}
	}
}

// repoFetchPlan is what an imported diffusers repo actually downloads.
type repoFetchPlan struct {
	// Files are the exact repo paths to fetch (nil fetches the whole repo, used
	// when the listing is unknown or has no component folders to prefer).
	Files []string
	// Variant is the diffusers weight variant the files are, "" for full precision.
	Variant string
	// Bytes is the size of Files (0 when the listing carries no sizes).
	Bytes int64
}

// planRepoFetch chooses the files a diffusers repo needs so an import cannot end
// up downloading three copies of the weights. A repo lists, beside its component
// folders, a duplicate root checkpoint, alternate formats (.bin, .ckpt, .onnx,
// .msgpack) and often both fp16 and full-precision copies of every component.
// The plan keeps the component folders only, safetensors only, and one precision:
// fp16 when every weight folder has an fp16 copy (loaded with variant="fp16"),
// otherwise full precision.
func planRepoFetch(meta hfmeta.Metadata) repoFetchPlan {
	type entry struct {
		path    string
		size    int64
		dir     string
		weight  bool
		variant bool
	}
	var all []entry
	hasSafetensors, components := false, false
	for _, f := range meta.Files {
		p := strings.Trim(f.Path, "/")
		if p == "" {
			continue
		}
		e := entry{path: p, size: f.Size, dir: path.Dir(p)}
		if e.dir != "." {
			components = true
		}
		base := path.Base(p)
		switch ext := strings.ToLower(path.Ext(base)); {
		case ext == ".safetensors":
			e.weight, hasSafetensors = true, true
			e.variant = strings.Contains(base, ".fp16")
		case strings.HasSuffix(base, ".safetensors.index.json"):
			e.variant = strings.Contains(base, ".fp16")
			e.weight = true
		case altWeightExt[ext]:
			e.weight = true // an alternate weight format, dropped below when safetensors exist
		}
		all = append(all, e)
	}
	if !components {
		return repoFetchPlan{}
	}

	// Precision: fp16 only when every weight folder has it.
	fp16Dirs, weightDirs := map[string]bool{}, map[string]bool{}
	for _, e := range all {
		if e.weight && e.dir != "." && strings.HasSuffix(strings.ToLower(e.path), ".safetensors") {
			weightDirs[e.dir] = true
			if e.variant {
				fp16Dirs[e.dir] = true
			}
		}
	}
	useFP16 := len(fp16Dirs) > 0 && len(fp16Dirs) == len(weightDirs)

	var plan repoFetchPlan
	if useFP16 {
		plan.Variant = "fp16"
	}
	for _, e := range all {
		if e.weight {
			if e.dir == "." {
				continue // the duplicate root checkpoint
			}
			if hasSafetensors && (!strings.Contains(e.path, ".safetensors")) {
				continue // alternate format beside safetensors
			}
			if e.variant != useFP16 {
				continue // the other precision
			}
		}
		plan.Files = append(plan.Files, e.path)
		plan.Bytes += e.size
	}
	return plan
}

// altWeightExt are weight formats a diffusers repo ships beside safetensors.
var altWeightExt = map[string]bool{".bin": true, ".ckpt": true, ".pt": true, ".pth": true, ".onnx": true, ".onnx_data": true, ".msgpack": true, ".h5": true, ".ot": true, ".gguf": true}

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
