package modelpromotion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"audio-tools/internal/modelcatalog"
	trustfloor "audio-tools/internal/qualification"
)

// QualificationEvidenceStore is a product-facing reader over the lab-owned
// evidence repository. Its types belong to qualification, not experiment.
type QualificationEvidenceStore interface {
	ListQualificationEvidence(context.Context, trustfloor.QualificationEvidenceFilter) ([]trustfloor.QualificationEvidence, error)
}

type ExperimentResolver struct {
	Store   QualificationEvidenceStore
	Catalog *modelcatalog.Catalog
}

func (r ExperimentResolver) ResolvePromotionReceipt(ctx context.Context, facet, reference string, model modelcatalog.Model) (Receipt, error) {
	if facet != FacetCorpusQuality && facet != FacetLatency && facet != FacetMemory {
		return Receipt{}, fmt.Errorf("experiment store does not own facet %q", facet)
	}
	if r.Store == nil || r.Catalog == nil {
		return Receipt{}, errors.New("experiment receipt store or model catalog is unavailable")
	}
	items, err := r.Store.ListQualificationEvidence(ctx, trustfloor.QualificationEvidenceFilter{ModelID: model.ID})
	if err != nil {
		return Receipt{}, err
	}
	for _, item := range items {
		if item.ID != reference {
			continue
		}
		canonical, ok := r.Catalog.ByID(item.ModelID)
		if !ok || canonical.ID != model.ID || canonical.Capability != model.Capability {
			return Receipt{}, errors.New("receipt model capability does not match the canonical model catalog")
		}
		if facet == FacetCorpusQuality && item.Kind != "corpus_quality" {
			return Receipt{}, fmt.Errorf("receipt kind %q is not corpus_quality", item.Kind)
		}
		if facet == FacetMemory && item.Kind != trustfloor.QualificationModelMemory {
			return Receipt{}, fmt.Errorf("receipt kind %q is not model_memory", item.Kind)
		}
		if facet == FacetLatency {
			if err := validateLatencyReceipt(item, model); err != nil {
				return Receipt{}, err
			}
		}
		if !item.Passed {
			return Receipt{}, errors.New("owner receipt outcome is failed")
		}
		if !modelcatalogHasEngine(model, item.EngineID) {
			return Receipt{}, fmt.Errorf("receipt engine %q is not declared for model %q", item.EngineID, model.ID)
		}
		if facet == FacetCorpusQuality {
			if err := validateQualityMachineJSON(item.MachineJSON, item, model); err != nil {
				return Receipt{}, err
			}
		}
		if facet == FacetMemory {
			if err := validateMemoryReceipt(item, model, canonical); err != nil {
				return Receipt{}, err
			}
		}
		owner, _ := ownerForFacet(facet)
		return Receipt{
			Reference: reference, Owner: owner, Facet: facet,
			ModelID: item.ModelID, Capability: canonical.Capability, Passed: true,
		}, nil
	}
	return Receipt{}, fmt.Errorf("owner receipt id %q not found for model %q", reference, model.ID)
}

func validateMemoryReceipt(item trustfloor.QualificationEvidence, requested, canonical modelcatalog.Model) error {
	if item.Kind != trustfloor.QualificationModelMemory {
		return fmt.Errorf("receipt kind %q is not model_memory", item.Kind)
	}
	if !item.Passed {
		return errors.New("owner receipt outcome is failed")
	}
	if requested.ID != canonical.ID || requested.Capability != canonical.Capability {
		return errors.New("requested model does not match the canonical model catalog")
	}
	requestedDigest, requestedErr := declarationDigest(requested)
	canonicalDigest, canonicalErr := declarationDigest(canonical)
	if requestedErr != nil || canonicalErr != nil || requestedDigest != canonicalDigest {
		return errors.New("requested model declaration does not match the canonical model catalog")
	}
	if !modelcatalogHasEngine(canonical, item.EngineID) {
		return fmt.Errorf("receipt engine %q is not declared for model %q", item.EngineID, canonical.ID)
	}
	receipt, err := trustfloor.ParseModelMemoryReceipt(item.MachineJSON, item.Passed)
	if err != nil {
		return err
	}
	cell := receipt.ProviderCell
	if cell.EngineID != item.EngineID || cell.ModelID != canonical.ID || cell.Capability != string(canonical.Capability) || cell.Strategy != item.Strategy {
		return errors.New("model-memory provider_cell identity does not match stored receipt and canonical model")
	}
	if canonical.Runtime == "" || canonical.Artifact == nil {
		return errors.New("canonical model lacks the runtime model or pinned artifact identity required for memory qualification")
	}
	model := receipt.Model
	expectedRuntimeModelID := canonical.RuntimeModelID
	if expectedRuntimeModelID == "" {
		expectedRuntimeModelID = canonical.ID
	}
	if model.Runtime != canonical.Runtime || model.RuntimeModelID != expectedRuntimeModelID ||
		model.ArtifactResource != canonical.Artifact.Resource || model.ArtifactName != canonical.Artifact.Name {
		return errors.New("model-memory runtime or artifact identity does not match the canonical model catalog")
	}
	if canonical.Hardware.MinRAMGB <= 0 || int64(canonical.Hardware.MinRAMGB) > int64(^uint64(0)>>1)/(1024*1024*1024) {
		return errors.New("canonical model has no valid declared min_ram_gb budget")
	}
	expectedBudget := int64(canonical.Hardware.MinRAMGB) * 1024 * 1024 * 1024
	if receipt.Execution.MemoryBudgetBytes != expectedBudget {
		return fmt.Errorf("model-memory budget %d does not equal canonical min_ram_gb budget %d", receipt.Execution.MemoryBudgetBytes, expectedBudget)
	}
	return nil
}

// validateLatencyReceipt accepts only the persisted browser product-path
// receipt whose owner measured the interactive-latency gate. A replay report
// with timing numbers alone does not qualify: the soak owner requires at
// least 30 realtime attempts and passing latency assertions before writing
// this claim.
func validateLatencyReceipt(item trustfloor.QualificationEvidence, model modelcatalog.Model) error {
	if !item.Passed {
		return errors.New("owner receipt outcome is failed")
	}
	if item.Kind != "browser_product_path" {
		return fmt.Errorf("receipt kind %q is not browser_product_path", item.Kind)
	}
	var payload struct {
		Lane       string `json:"lane"`
		RunID      string `json:"run_id"`
		Browser    bool   `json:"browser_product_path"`
		Acceptance map[string]struct {
			InBand                 bool `json:"in_band"`
			NegativeControlsPassed bool `json:"negative_controls_passed"`
			Attempts               int  `json:"attempts"`
		} `json:"acceptance"`
	}
	if err := json.Unmarshal(item.MachineJSON, &payload); err != nil {
		return fmt.Errorf("parse owner latency receipt: %w", err)
	}
	claim := payload.Acceptance["interactive-latency"]
	if !item.Passed || payload.Lane != "realtime" || payload.RunID == "" || !payload.Browser || item.ArtifactRef == "" ||
		item.ModelID != model.ID || item.EngineID == "" || item.Strategy == "" || item.PolicyProfile == "" ||
		!modelcatalogHasEngine(model, item.EngineID) || !claim.InBand || !claim.NegativeControlsPassed || claim.Attempts < 30 {
		return errors.New("owner latency receipt lacks a passing, identity-matched realtime interactive-latency claim")
	}
	return nil
}

type qualityMachineReceipt struct {
	ProviderCell struct {
		EngineID   string `json:"engine_id"`
		ModelID    string `json:"model_id"`
		Capability string `json:"capability"`
		Strategy   string `json:"strategy"`
	} `json:"provider_cell"`
	Acceptance map[string]struct {
		InBand                 bool `json:"in_band"`
		NegativeControlsPassed bool `json:"negative_controls_passed"`
	} `json:"acceptance"`
	QualityMeasurement struct {
		Passed       bool `json:"passed"`
		ProviderCell struct {
			EngineID   string `json:"engine_id"`
			ModelID    string `json:"model_id"`
			Capability string `json:"capability"`
			Strategy   string `json:"strategy"`
		} `json:"provider_cell"`
	} `json:"quality_measurement"`
}

func validateQualityMachineJSON(raw []byte, item trustfloor.QualificationEvidence, model modelcatalog.Model) error {
	var machine qualityMachineReceipt
	if err := json.Unmarshal(raw, &machine); err != nil {
		return fmt.Errorf("parse owner quality receipt: %w", err)
	}
	claim := machine.Acceptance["corpus-quality"]
	for _, cell := range []struct {
		name, engine, model, capability, strategy string
	}{
		{"receipt provider cell", machine.ProviderCell.EngineID, machine.ProviderCell.ModelID, machine.ProviderCell.Capability, machine.ProviderCell.Strategy},
		{"quality measurement provider cell", machine.QualityMeasurement.ProviderCell.EngineID, machine.QualityMeasurement.ProviderCell.ModelID, machine.QualityMeasurement.ProviderCell.Capability, machine.QualityMeasurement.ProviderCell.Strategy},
	} {
		if cell.engine != item.EngineID || cell.model != model.ID || cell.capability != string(model.Capability) || cell.strategy != item.Strategy {
			return fmt.Errorf("%s identity does not match model %q capability %q and stored receipt", cell.name, model.ID, model.Capability)
		}
	}
	if !machine.QualityMeasurement.Passed || !claim.InBand || !claim.NegativeControlsPassed {
		return errors.New("owner quality receipt lacks a passing measured quality outcome")
	}
	return nil
}

func modelcatalogHasEngine(model modelcatalog.Model, engineID string) bool {
	for _, id := range model.EngineIDs {
		if id == engineID {
			return true
		}
	}
	return false
}
