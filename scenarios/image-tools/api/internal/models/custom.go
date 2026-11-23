package models

import (
	"net/url"
	"path"
	"strings"
)

// CompleteCustom fills what an operator-added model leaves out, so the shortest
// honest registration (an id, an operation, a backend and a weights source) is a
// complete entry. It records facts and never refuses on them: a missing licence is
// stored as "unverified" with a conditional commercial label, which keeps hosted,
// monetized and redistributed use refused while normal local use stays allowed
// (operator stance 2026-11-21). Anything the operator did state is kept.
func CompleteCustom(m Model) Model {
	if strings.TrimSpace(m.Name) == "" {
		m.Name = m.ID
	}
	if m.Tier == "" {
		m.Tier = TierNiceToHave
	}
	labels := &m.CapabilityLabels
	if strings.TrimSpace(labels.License) == "" {
		labels.License = "unverified"
	}
	if labels.CommercialUse == "" {
		labels.CommercialUse = CommercialUseConditional
	}
	if labels.CommercialUse != CommercialUseYes && strings.TrimSpace(labels.CommercialUseNotes) == "" {
		labels.CommercialUseNotes = unattestedImportNotes
	}
	if labels.Provenance == "" {
		labels.Provenance = ProvenanceUserImported
	}
	if !m.Hardware.CPUCapable && !m.Hardware.GPURequired {
		m.Hardware = importHardwareFor(m.Architecture)
	}
	if !m.Source.HasRepo() && len(m.Source.Assets) == 0 && strings.TrimSpace(m.Source.LocalPath) == "" {
		if asset, ok := assetFromURL(m.Source.DownloadURL); ok {
			m.Source.Assets = []Asset{asset}
		}
	}
	return m
}

// assetFromURL turns a direct download URL into the asset the installer fetches,
// naming it after the URL's file and typing it by extension.
func assetFromURL(raw string) (Asset, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Asset{}, false
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return Asset{}, false
	}
	filename := path.Base(u.Path)
	if filename == "" || filename == "." || filename == "/" {
		return Asset{}, false
	}
	kind := ArtifactGeneric
	switch strings.ToLower(path.Ext(filename)) {
	case ".onnx":
		kind = ArtifactONNX
	case ".safetensors":
		kind = ArtifactSafetensors
	case ".gguf":
		kind = ArtifactGGUF
	}
	return Asset{URL: raw, Filename: filename, Kind: kind}, true
}
