package models

import (
	internalmodels "image-tools/internal/models"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// domainToProto converts a registry Model to its wire shape. enabled is the
// EFFECTIVE enabled state (seed default overlaid with the SQLite override), so
// the wire never disagrees with what the selector would actually consider.
func domainToProto(m internalmodels.Model, enabled bool) *modelsv1.Model {
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
		Enabled:          enabled,
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

func blocklistToProto(b internalmodels.BlocklistEntry) *modelsv1.BlocklistEntry {
	return &modelsv1.BlocklistEntry{
		Id:                              b.ID,
		Operations:                      append([]string(nil), b.Operations...),
		License:                         b.License,
		Reason:                          b.Reason,
		ExportingOnnxRemovesRestriction: b.ExportingONNXRemovesRestriction,
	}
}
