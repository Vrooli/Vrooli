// Package hfmeta is the scenario-local seam that gathers the facts needed to
// import a model WITHOUT installing it: where the weights live, what shape they
// are (a single checkpoint file vs a diffusers multi-file repo), which pipeline
// class / architecture they declare, and the governance-relevant metadata
// (license, NSFW flag, approximate size).
//
// Why this boundary exists: model import (plan capability D) has a "look before
// you leap" step — inspect a HuggingFace repo id (or a direct URL / local path),
// infer the architecture, and ask the user to confirm — that must run cheaply and
// offline-testably, separate from the heavy snapshot download (internal/fetch).
// hfmeta is that inspection seam: an interface (Fetcher) tests fake, plus one
// production implementation (HFClient) that probes the HuggingFace Hub via the
// same governed huggingface_hub dependency the snapshot fetcher uses. It returns
// raw, source-of-truth Metadata; it does NOT decide the architecture — that is
// the model catalog's SSOT (models.InferArchitecture consumes Metadata), so the
// architecture enum stays in one place.
//
// Scenario-local by decision (plan §8, requester-confirmed): kept inside
// image-tools until a second scenario actually needs HF import (YAGNI). Promote
// to a shared package only when that consumer appears.
package hfmeta

import "context"

// Layout classifies a model source's on-disk shape, which determines how it
// installs: a single-file checkpoint routes to a direct Asset download, a
// diffusers repo routes to a pinned snapshot fetch (no user decision — plan D3).
type Layout string

const (
	// LayoutUnknown means the shape could not be determined (e.g. an opaque URL
	// with no inspectable listing). The import wizard must ask the user.
	LayoutUnknown Layout = ""
	// LayoutSingleFile is one self-contained checkpoint (.safetensors/.ckpt) —
	// the "Case A" of ChatGPT's framing; installs via fetch.Asset.
	LayoutSingleFile Layout = "single-file"
	// LayoutDiffusersRepo is a model_index.json + transformer/vae/text_encoder/…
	// subdir tree — "Case B"; installs via a pinned fetch.RepoSpec snapshot.
	LayoutDiffusersRepo Layout = "diffusers-repo"
)

// FileInfo is one file in a source listing, used for layout detection + size.
type FileInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// Metadata is the inspected, install-relevant facts about a model source. It is
// raw source-of-truth: it records what the source DECLARES (pipeline class, tags,
// license), never a derived decision. models.InferArchitecture turns it into an
// architecture proposal; the import flow turns it into a registry entry.
type Metadata struct {
	// Source is the repo id / URL / path exactly as the user supplied it.
	Source string `json:"source"`
	// RepoID is the HuggingFace repo id when the source is (or resolved to) one.
	RepoID string `json:"repo_id"`
	// Revision is the resolved IMMUTABLE commit SHA (HF), so the later snapshot
	// install pins reproducibly. Empty for non-HF sources.
	Revision string `json:"revision"`
	// Layout is the detected on-disk shape (single-file vs diffusers-repo).
	Layout Layout `json:"layout"`
	// PipelineClass is the diffusers pipeline class declared in model_index.json
	// (_class_name), e.g. "StableDiffusionXLPipeline". Empty for single-file
	// checkpoints that ship no model_index.json.
	PipelineClass string `json:"pipeline_class"`
	// Tags are the model-card tags (e.g. "stable-diffusion-xl", "lora", "text-to-image").
	Tags []string `json:"tags"`
	// BaseModel is the model card's base_model field when present (lineage hint).
	BaseModel string `json:"base_model"`
	// Files is the source file listing (drives layout detection + size sum).
	Files []FileInfo `json:"files"`
	// License is the declared SPDX-ish license id (e.g. "openrail++", "apache-2.0")
	// or "" when the source declares none (⇒ treated as unverified on import).
	License string `json:"license"`
	// NSFW reports the HuggingFace "not-for-all-audiences" flag / tag.
	NSFW bool `json:"nsfw"`
	// SizeBytes is the approximate total size of the weight files.
	SizeBytes int64 `json:"size_bytes"`
}

// TotalSize sums the file listing (a convenience when SizeBytes was not provided
// directly by the source).
func (m Metadata) TotalSize() int64 {
	if m.SizeBytes > 0 {
		return m.SizeBytes
	}
	var total int64
	for _, f := range m.Files {
		total += f.Size
	}
	return total
}

// Fetcher resolves Metadata for a model source (a HF repo id, a direct weight
// URL, or a local path). It is the seam: tests inject a fake; production wires
// HFClient. It installs nothing and downloads no weights — only the small
// metadata needed to propose an import.
type Fetcher interface {
	Inspect(ctx context.Context, source string) (Metadata, error)
}
