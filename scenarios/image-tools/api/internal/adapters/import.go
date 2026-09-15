package adapters

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"image-tools/internal/fetch"
	"image-tools/internal/hfmeta"
	"image-tools/internal/models"
	"image-tools/internal/safety"
)

// Guided adapter import (plan capability C, Phase 2 item 2). It reuses the
// Phase-1 inspect machinery (hfmeta) but is kind-aware: an adapter source is
// classified as LoRA / ControlNet / IP-Adapter and its compatible base
// architecture is inferred (then user-confirmed), producing an add-only custom
// adapter entry. Like the model import flow this is pure + unit-testable; the
// handler is a thin translator.

// ImportProposal is the dry-run result of inspecting an adapter source: the
// inferred kind + compatible architecture (+confidence/evidence) and the proposed
// catalog entry the operator confirms before install. It installs nothing.
type ImportProposal struct {
	Metadata     hfmeta.Metadata
	Kind         Kind
	KindEvidence string
	Architecture models.Architecture
	Confidence   models.Confidence
	ArchEvidence string
	// Entry is the catalog entry that WOULD be added; the operator may override
	// id/name/kind/architecture/preprocessor before import.
	Entry Adapter
}

// ImportConfirm carries the operator-confirmed fields ImportAdapter applies over a
// proposal before installing.
type ImportConfirm struct {
	// ID is the new entry's id (must not collide with a seed adapter). Required.
	ID string
	// Name is the display name (defaults to the source's last path segment).
	Name string
	// Kind overrides the inferred kind; REQUIRED when inference is ambiguous.
	Kind Kind
	// Architecture overrides the inferred compatible architecture; REQUIRED when
	// inference returned none (never guess silently).
	Architecture models.Architecture
	// Preprocessor sets the ControlNet preprocessor (ignored for other kinds).
	Preprocessor Preprocessor
	// AttestCommercialRights records the operator's attestation, lifting the
	// public/BYOK serving block (decision D4, mirrors the model import flow).
	AttestCommercialRights bool
}

var idSanitizer = regexp.MustCompile(`[^a-z0-9._-]+`)

// ProposeImport inspects metadata into a proposal: it infers the kind +
// architecture and assembles the default entry. The proposal's Entry carries a
// default id derived from the source; the operator confirms/overrides before
// ImportAdapter.
func ProposeImport(meta hfmeta.Metadata) ImportProposal {
	kind, kindEv := InferKind(meta)
	arch, conf, archEv := models.InferArchitecture(meta)
	entry := assembleEntry(meta, kind, arch, "", defaultID(meta), defaultName(meta))
	return ImportProposal{
		Metadata:     meta,
		Kind:         kind,
		KindEvidence: kindEv,
		Architecture: arch,
		Confidence:   conf,
		ArchEvidence: archEv,
		Entry:        entry,
	}
}

// BuildImportEntry assembles the final add-only entry from inspected metadata and
// the operator's confirmation. It fails closed when the kind or architecture is
// still unresolved, or the layout cannot be mapped to a fetch strategy — never a
// silently-guessed or un-installable entry.
func BuildImportEntry(meta hfmeta.Metadata, confirm ImportConfirm) (Adapter, error) {
	if strings.TrimSpace(confirm.ID) == "" {
		return Adapter{}, fmt.Errorf("import: a confirmed id is required")
	}
	kind := confirm.Kind
	if kind == "" {
		if inferred, _ := InferKind(meta); inferred != "" {
			kind = inferred
		}
	}
	if !kind.valid() {
		return Adapter{}, fmt.Errorf("import: adapter kind could not be inferred and was not confirmed — select lora, controlnet, or ip-adapter")
	}
	arch := confirm.Architecture
	if arch == "" || arch == models.ArchNone {
		if inferred, _, _ := models.InferArchitecture(meta); inferred != models.ArchNone {
			arch = inferred
		}
	}
	if arch == "" || arch == models.ArchNone {
		return Adapter{}, fmt.Errorf("import: compatible base architecture could not be inferred and was not confirmed — select one explicitly")
	}
	if !arch.Valid() {
		return Adapter{}, fmt.Errorf("import: unknown architecture %q", arch)
	}
	pre := confirm.Preprocessor
	if kind == KindControlNet && pre == "" {
		pre = PreprocessorNone
	}
	if kind != KindControlNet {
		pre = ""
	}
	name := confirm.Name
	if strings.TrimSpace(name) == "" {
		name = defaultName(meta)
	}
	entry := assembleEntry(meta, kind, arch, pre, confirm.ID, name)
	if !entry.Source.HasFetchStrategy() {
		return Adapter{}, fmt.Errorf("import: source layout %q has no resolvable adapter weights", meta.Layout)
	}
	// Commercial posture mirrors the model import flow (decision D4).
	if confirm.AttestCommercialRights {
		entry.CapabilityLabels.CommercialUse = models.CommercialUseYes
		entry.CapabilityLabels.CommercialUseNotes = "operator-attested commercial rights at import time"
	} else {
		entry.CapabilityLabels.CommercialUse = models.CommercialUseConditional
		entry.CapabilityLabels.CommercialUseNotes = "user-imported; commercial rights unverified — public/BYOK serving blocked until attested"
	}
	return entry, nil
}

// assembleEntry builds the catalog entry for an inspected source + kind + arch.
func assembleEntry(meta hfmeta.Metadata, kind Kind, arch models.Architecture, pre Preprocessor, id, name string) Adapter {
	a := Adapter{
		ID:           id,
		Name:         name,
		Kind:         kind,
		Architecture: arch,
		Weight:       defaultWeight(kind),
		Preprocessor: pre,
		ScaleRange:   defaultScaleRange(kind),
		SizeMBApprox: int(meta.TotalSize() >> 20),
		CapabilityLabels: models.CapabilityLabels{
			NSFWCapable:      meta.NSFW,
			License:          licenseLabel(meta.License),
			BaseModelLineage: meta.BaseModel,
			Provenance:       models.ProvenanceUserImported,
		},
		// Imported adapters are never Ready: execution is proven per kind×arch by an
		// attended GPU run, not by import (no vaporware).
		Ready:   false,
		Pending: "user-imported adapter; run the attended GPU e2e for this kind to prove it before flipping Ready",
		Enabled: false,
	}
	applyLayoutSource(&a, meta)
	return a
}

// applyLayoutSource sets the fetch strategy from the detected layout: a single-
// file adapter installs as a direct Asset; a diffusers repo installs as a pinned
// snapshot. A local source records its path.
func applyLayoutSource(a *Adapter, meta hfmeta.Metadata) {
	if isLocalSource(meta) {
		a.Source.LocalPath = meta.Source
		return
	}
	switch meta.Layout {
	case hfmeta.LayoutSingleFile:
		filename, url := singleFileAsset(meta)
		if url != "" {
			a.Source.Assets = []fetch.Asset{{
				URL:      url,
				Filename: filename,
				Kind:     checkpointKind(filename),
			}}
		}
	case hfmeta.LayoutDiffusersRepo:
		if meta.RepoID != "" {
			a.Source.Repo = fetch.RepoSpec{RepoID: meta.RepoID, Revision: meta.Revision}
		}
	}
}

// InferKind classifies an adapter source by its model-card tags, then filename
// hints. It returns an empty Kind when nothing is recognizable (the wizard then
// requires an explicit pick — never a silent guess).
func InferKind(meta hfmeta.Metadata) (Kind, string) {
	for _, t := range meta.Tags {
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "lora":
			return KindLoRA, fmt.Sprintf("model-card tag %q", t)
		case "controlnet":
			return KindControlNet, fmt.Sprintf("model-card tag %q", t)
		case "ip-adapter", "ip_adapter", "ipadapter":
			return KindIPAdapter, fmt.Sprintf("model-card tag %q", t)
		}
	}
	for _, f := range meta.Files {
		name := strings.ToLower(path.Base(f.Path))
		switch {
		case strings.Contains(name, "ip-adapter") || strings.Contains(name, "ip_adapter"):
			return KindIPAdapter, fmt.Sprintf("filename %q", path.Base(f.Path))
		case strings.Contains(name, "controlnet") || name == "control_net.safetensors":
			return KindControlNet, fmt.Sprintf("filename %q", path.Base(f.Path))
		case strings.Contains(name, "lora"):
			return KindLoRA, fmt.Sprintf("filename %q", path.Base(f.Path))
		}
	}
	return "", "no recognizable adapter kind tag or filename"
}

// defaultWeight is the consent weight an imported adapter of a kind carries. An
// IP-Adapter transfers identity/style ⇒ high; LoRA/ControlNet default to none
// (the operator can raise an identity-class LoRA at confirm time later).
func defaultWeight(kind Kind) safety.Weight {
	if kind == KindIPAdapter {
		return safety.WeightHigh
	}
	return safety.WeightNone
}

// defaultScaleRange is the conditioning-strength range an imported adapter of a
// kind defaults to (LoRA/ControlNet 0..2 @1.0; IP-Adapter 0..1 @0.6).
func defaultScaleRange(kind Kind) ScaleRange {
	if kind == KindIPAdapter {
		return ScaleRange{Min: 0, Max: 1, Default: 0.6}
	}
	return ScaleRange{Min: 0, Max: 2, Default: 1.0}
}

// licenseLabel normalizes a declared license to the catalog label (empty ⇒ the
// explicit "unverified" marker the safety tier gate keys on).
func licenseLabel(declared string) string {
	if strings.TrimSpace(declared) == "" {
		return "unverified"
	}
	return declared
}

func defaultID(meta hfmeta.Metadata) string {
	base := meta.RepoID
	if base == "" {
		base = meta.Source
	}
	base = strings.ToLower(path.Base(strings.TrimRight(base, "/")))
	base = strings.TrimSuffix(base, path.Ext(base))
	base = idSanitizer.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-._")
	if base == "" {
		base = "imported-adapter"
	}
	return "imported-" + base
}

func defaultName(meta hfmeta.Metadata) string {
	base := meta.RepoID
	if base == "" {
		base = path.Base(strings.TrimRight(meta.Source, "/"))
	}
	if strings.TrimSpace(base) == "" {
		return "Imported adapter"
	}
	return base
}

// singleFileAsset returns the on-disk filename + resolvable URL for a single-file
// source (an HF repo's top-level weight, or a direct URL).
func singleFileAsset(meta hfmeta.Metadata) (filename, url string) {
	if isURLSource(meta) {
		name := path.Base(strings.SplitN(meta.Source, "?", 2)[0])
		return name, meta.Source
	}
	if meta.RepoID != "" {
		name := topLevelWeight(meta)
		if name == "" {
			return "", ""
		}
		rev := meta.Revision
		if rev == "" {
			rev = "main"
		}
		return name, fmt.Sprintf("https://huggingface.co/%s/resolve/%s/%s", meta.RepoID, rev, name)
	}
	return "", ""
}

// topLevelWeight returns the first top-level weight filename in the listing.
func topLevelWeight(meta hfmeta.Metadata) string {
	for _, f := range meta.Files {
		if strings.Contains(strings.Trim(f.Path, "/"), "/") {
			continue
		}
		name := path.Base(f.Path)
		switch strings.ToLower(path.Ext(name)) {
		case ".safetensors", ".ckpt", ".bin", ".pth":
			return name
		}
	}
	return ""
}

func checkpointKind(filename string) fetch.ArtifactKind {
	switch strings.ToLower(path.Ext(filename)) {
	case ".safetensors":
		return fetch.ArtifactSafetensors
	default:
		return fetch.ArtifactGeneric
	}
}

func isURLSource(meta hfmeta.Metadata) bool {
	return strings.HasPrefix(meta.Source, "http://") || strings.HasPrefix(meta.Source, "https://")
}

func isLocalSource(meta hfmeta.Metadata) bool {
	return strings.HasPrefix(meta.Source, "/") || strings.HasPrefix(meta.Source, ".")
}
