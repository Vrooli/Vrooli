package adapters

import (
	"time"

	internaladapters "image-tools/internal/adapters"
	internalmodels "image-tools/internal/models"

	adaptersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/adapters"
)

// adapterView bundles a seed/custom Adapter with the runtime facts the wire shape
// carries on top of the static catalog entry (effective enabled, custom origin,
// on-disk install state).
type adapterView struct {
	enabled bool
	custom  bool
	install *internaladapters.InstallRecord // nil when no install record exists
}

// domainToProto converts a catalog Adapter to its wire shape. enabled is the
// EFFECTIVE enabled state (seed default overlaid with the SQLite override); the
// install state reflects on-disk weight presence.
func domainToProto(a internaladapters.Adapter, v adapterView) *adaptersv1.Adapter {
	return &adaptersv1.Adapter{
		Id:               a.ID,
		Name:             a.Name,
		Kind:             string(a.Kind),
		Architecture:     string(a.Architecture),
		Weight:           string(a.Weight),
		Preprocessor:     string(a.Preprocessor),
		ScaleRange:       scaleRangeToProto(a.ScaleRange),
		SizeMbApprox:     int32(a.SizeMBApprox),
		CapabilityLabels: labelsToProto(a.CapabilityLabels),
		Ready:            a.Ready,
		Pending:          a.Pending,
		Enabled:          v.enabled,
		Install:          installToProto(v.install),
		Custom:           v.custom,
	}
}

func scaleRangeToProto(r internaladapters.ScaleRange) *adaptersv1.ScaleRange {
	return &adaptersv1.ScaleRange{
		Min:     r.Min,
		Max:     r.Max,
		Default: r.Default,
	}
}

func installToProto(r *internaladapters.InstallRecord) *adaptersv1.InstallState {
	if r == nil {
		return &adaptersv1.InstallState{Installed: false}
	}
	at := ""
	if !r.InstalledAt.IsZero() {
		at = r.InstalledAt.UTC().Format(time.RFC3339)
	}
	return &adaptersv1.InstallState{
		Installed:   r.Installed,
		Path:        r.Path,
		Checksum:    r.Checksum,
		SizeBytes:   uint64(r.SizeBytes), //nolint:gosec // on-disk sizes are non-negative
		InstalledAt: at,
	}
}

func labelsToProto(l internalmodels.CapabilityLabels) *adaptersv1.CapabilityLabels {
	return &adaptersv1.CapabilityLabels{
		NsfwCapable:        l.NSFWCapable,
		License:            l.License,
		CommercialUse:      commercialUseToProto(l.CommercialUse),
		CommercialUseNotes: l.CommercialUseNotes,
		BaseModelLineage:   l.BaseModelLineage,
		KnownRisks:         l.KnownRisks,
		Provenance:         l.Provenance,
	}
}

func commercialUseToProto(c internalmodels.CommercialUse) adaptersv1.CommercialUse {
	switch c {
	case internalmodels.CommercialUseYes:
		return adaptersv1.CommercialUse_COMMERCIAL_USE_YES
	case internalmodels.CommercialUseNo:
		return adaptersv1.CommercialUse_COMMERCIAL_USE_NO
	case internalmodels.CommercialUseConditional:
		return adaptersv1.CommercialUse_COMMERCIAL_USE_CONDITIONAL
	default:
		return adaptersv1.CommercialUse_COMMERCIAL_USE_UNSPECIFIED
	}
}

func doctorReportToProto(r internaladapters.CatalogDoctorReport) *adaptersv1.DoctorCatalogResponse {
	out := &adaptersv1.DoctorCatalogResponse{
		Ok:       r.OK,
		Findings: make([]*adaptersv1.CatalogFinding, 0, len(r.Findings)),
	}
	for _, f := range r.Findings {
		out.Findings = append(out.Findings, &adaptersv1.CatalogFinding{
			Severity:  findingSeverityToProto(f.Severity),
			Code:      f.Code,
			AdapterId: f.AdapterID,
			Message:   f.Message,
		})
	}
	return out
}

func findingSeverityToProto(s internaladapters.FindingSeverity) adaptersv1.CatalogFindingSeverity {
	switch s {
	case internaladapters.FindingWarning:
		return adaptersv1.CatalogFindingSeverity_CATALOG_FINDING_SEVERITY_WARNING
	case internaladapters.FindingError:
		return adaptersv1.CatalogFindingSeverity_CATALOG_FINDING_SEVERITY_ERROR
	default:
		return adaptersv1.CatalogFindingSeverity_CATALOG_FINDING_SEVERITY_UNSPECIFIED
	}
}
