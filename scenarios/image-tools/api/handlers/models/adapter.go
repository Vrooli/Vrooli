package models

import (
	"time"

	internalmodels "image-tools/internal/models"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// modelView bundles a registry/custom Model with the runtime facts the wire
// shape carries on top of the static catalog entry.
type modelView struct {
	enabled bool
	custom  bool
	install *internalmodels.InstallRecord // nil when no install record exists
}

// domainToProto converts a registry Model to its wire shape. enabled is the
// EFFECTIVE enabled state (seed default overlaid with the SQLite override), so
// the wire never disagrees with what the selector would actually consider; the
// install state reflects on-disk weight presence.
func domainToProto(m internalmodels.Model, v modelView) *modelsv1.Model {
	return &modelsv1.Model{
		Id:               m.ID,
		Name:             m.Name,
		Operations:       append([]string(nil), m.Operations...),
		DefaultFor:       append([]string(nil), m.DefaultFor...),
		Tier:             string(m.Tier),
		Backend:          m.Backend,
		AltBackends:      append([]string(nil), m.AltBackends...),
		RequiresComfyui:  m.RequiresComfyUI,
		SizeMbApprox:     int32(m.SizeMBApprox),
		QuantVariants:    append([]string(nil), m.QuantVariants...),
		Hardware:         hardwareToProto(m.Hardware),
		CapabilityLabels: labelsToProto(m.CapabilityLabels),
		Enabled:          v.enabled,
		Custom:           v.custom,
		Install:          installToProto(v.install),
	}
}

func installToProto(r *internalmodels.InstallRecord) *modelsv1.InstallState {
	if r == nil {
		return &modelsv1.InstallState{Installed: false}
	}
	at := ""
	if !r.InstalledAt.IsZero() {
		at = r.InstalledAt.UTC().Format(time.RFC3339)
	}
	return &modelsv1.InstallState{
		Installed:   r.Installed,
		Path:        r.Path,
		Checksum:    r.Checksum,
		SizeBytes:   uint64(r.SizeBytes), //nolint:gosec // on-disk sizes are non-negative
		InstalledAt: at,
	}
}

func hardwareToProto(h internalmodels.Hardware) *modelsv1.Hardware {
	return &modelsv1.Hardware{
		CpuCapable:  h.CPUCapable,
		GpuRequired: h.GPURequired,
		MinVramGb:   int32(h.MinVRAMGB),
		MinRamGb:    int32(h.MinRAMGB),
		OsArch:      append([]string(nil), h.OSArch...),
		SpeedNote:   h.SpeedNote,
	}
}

func labelsToProto(l internalmodels.CapabilityLabels) *modelsv1.CapabilityLabels {
	return &modelsv1.CapabilityLabels{
		NsfwCapable:        l.NSFWCapable,
		License:            l.License,
		CommercialUse:      commercialUseToProto(l.CommercialUse),
		CommercialUseNotes: l.CommercialUseNotes,
		BaseModelLineage:   l.BaseModelLineage,
		KnownRisks:         l.KnownRisks,
	}
}

func commercialUseToProto(c internalmodels.CommercialUse) modelsv1.CommercialUse {
	switch c {
	case internalmodels.CommercialUseYes:
		return modelsv1.CommercialUse_COMMERCIAL_USE_YES
	case internalmodels.CommercialUseNo:
		return modelsv1.CommercialUse_COMMERCIAL_USE_NO
	case internalmodels.CommercialUseConditional:
		return modelsv1.CommercialUse_COMMERCIAL_USE_CONDITIONAL
	default:
		return modelsv1.CommercialUse_COMMERCIAL_USE_UNSPECIFIED
	}
}

// protoToDomain converts a wire Model into a domain Model for custom-entry
// registration. Runtime-only fields (install state) are ignored; a custom entry
// is enabled by default so it is immediately selectable once its weights exist.
func protoToDomain(p *modelsv1.Model) internalmodels.Model {
	m := internalmodels.Model{
		ID:              p.GetId(),
		Name:            p.GetName(),
		Operations:      append([]string(nil), p.GetOperations()...),
		DefaultFor:      append([]string(nil), p.GetDefaultFor()...),
		Tier:            internalmodels.Tier(p.GetTier()),
		Backend:         p.GetBackend(),
		AltBackends:     append([]string(nil), p.GetAltBackends()...),
		RequiresComfyUI: p.GetRequiresComfyui(),
		SizeMBApprox:    int(p.GetSizeMbApprox()),
		QuantVariants:   append([]string(nil), p.GetQuantVariants()...),
		Enabled:         true,
	}
	if h := p.GetHardware(); h != nil {
		m.Hardware = internalmodels.Hardware{
			CPUCapable:  h.GetCpuCapable(),
			GPURequired: h.GetGpuRequired(),
			MinVRAMGB:   int(h.GetMinVramGb()),
			MinRAMGB:    int(h.GetMinRamGb()),
			OSArch:      append([]string(nil), h.GetOsArch()...),
			SpeedNote:   h.GetSpeedNote(),
		}
	}
	if l := p.GetCapabilityLabels(); l != nil {
		m.CapabilityLabels = internalmodels.CapabilityLabels{
			NSFWCapable:        l.GetNsfwCapable(),
			License:            l.GetLicense(),
			CommercialUse:      commercialUseFromProto(l.GetCommercialUse()),
			CommercialUseNotes: l.GetCommercialUseNotes(),
			BaseModelLineage:   l.GetBaseModelLineage(),
			KnownRisks:         l.GetKnownRisks(),
		}
	}
	return m
}

func commercialUseFromProto(c modelsv1.CommercialUse) internalmodels.CommercialUse {
	switch c {
	case modelsv1.CommercialUse_COMMERCIAL_USE_YES:
		return internalmodels.CommercialUseYes
	case modelsv1.CommercialUse_COMMERCIAL_USE_NO:
		return internalmodels.CommercialUseNo
	case modelsv1.CommercialUse_COMMERCIAL_USE_CONDITIONAL:
		return internalmodels.CommercialUseConditional
	default:
		return ""
	}
}

func blocklistToProto(b internalmodels.BlocklistEntry) *modelsv1.BlocklistEntry {
	return &modelsv1.BlocklistEntry{
		Id:                              b.ID,
		Operations:                      append([]string(nil), b.Operations...),
		License:                         b.License,
		Reason:                          b.Reason,
		ExportingOnnxRemovesRestriction: b.ExportingONNXRemovesRestriction,
	}
}

func doctorReportToProto(r internalmodels.CatalogDoctorReport) *modelsv1.DoctorCatalogResponse {
	out := &modelsv1.DoctorCatalogResponse{
		Ok:       r.OK,
		Findings: make([]*modelsv1.CatalogFinding, 0, len(r.Findings)),
	}
	for _, f := range r.Findings {
		out.Findings = append(out.Findings, &modelsv1.CatalogFinding{
			Severity:  findingSeverityToProto(f.Severity),
			Code:      f.Code,
			ModelId:   f.ModelID,
			Operation: f.Operation,
			Message:   f.Message,
		})
	}
	return out
}

func findingSeverityToProto(s internalmodels.FindingSeverity) modelsv1.CatalogFindingSeverity {
	switch s {
	case internalmodels.FindingWarning:
		return modelsv1.CatalogFindingSeverity_CATALOG_FINDING_SEVERITY_WARNING
	case internalmodels.FindingError:
		return modelsv1.CatalogFindingSeverity_CATALOG_FINDING_SEVERITY_ERROR
	default:
		return modelsv1.CatalogFindingSeverity_CATALOG_FINDING_SEVERITY_UNSPECIFIED
	}
}
