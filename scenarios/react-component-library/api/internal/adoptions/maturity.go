package adoptions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
)

// CatalogMaturityReader projects the catalog's declared asset and persisted
// gate evidence into the adoption seam. It deliberately does not rerun
// browser or build gates during an adoption request; evidence freshness and
// gate execution belong to catalog coverage.
type CatalogMaturityReader struct {
	root     string
	evidence *catalogcoverage.EvidenceStore
}

func NewCatalogMaturityReader(root string, evidence *catalogcoverage.EvidenceStore) *CatalogMaturityReader {
	return &CatalogMaturityReader{root: root, evidence: evidence}
}

// CatalogGateReader projects the latest exact-version observation for a
// contract gate into adoption preflight. Missing or stale observations remain
// explicitly unmeasured; they never become an accidental pass.
type CatalogGateReader struct {
	evidence *catalogcoverage.EvidenceStore
}

func NewCatalogGateReader(evidence *catalogcoverage.EvidenceStore) *CatalogGateReader {
	return &CatalogGateReader{evidence: evidence}
}

func (r *CatalogGateReader) GateVerdict(ctx context.Context, component components.Component, version, scenario, gate string) (string, error) {
	if r == nil || r.evidence == nil || strings.TrimSpace(component.CatalogID) == "" {
		return "not-measured", nil
	}
	rows, err := r.evidence.List(ctx)
	if err != nil {
		return "", err
	}
	for _, row := range rows {
		if row.AssetID != component.CatalogID || row.Gate != gate || row.Version != version {
			continue
		}
		if row.Target != "" && row.Target != "react-vite" && row.Target != scenario {
			continue
		}
		if row.Result == "pass" || row.Result == "fail" || row.Result == "unmeasured" || row.Result == "skipped" {
			return row.Result, nil
		}
		return "not-measured", nil
	}
	return "not-measured", nil
}

func (r *CatalogMaturityReader) Maturity(ctx context.Context, component components.Component, _ string, scenario string) (MaturityVerdict, error) {
	assets, err := catalogcoverage.LoadCatalog(filepath.Join(r.root, "scenarios", "react-component-library", "catalog"))
	if err != nil {
		return MaturityVerdict{}, err
	}
	impls, err := catalogcoverage.LoadImplementations(filepath.Join(r.root, "scenarios", "react-component-library", "library"))
	if err != nil {
		return MaturityVerdict{}, err
	}
	gates, err := catalogcoverage.LoadGateDefinitions(filepath.Join(r.root, "scenarios", "react-component-library", "catalog", "config.json"))
	if err != nil {
		return MaturityVerdict{}, err
	}
	floor, err := declaredMaturityFloor(filepath.Join(r.root, "scenarios", "react-component-library", "catalog", "config.json"))
	if err != nil {
		return MaturityVerdict{}, err
	}
	var evidence []catalogcoverage.GateEvidence
	if r.evidence != nil {
		evidence, err = r.evidence.List(ctx)
		if err != nil {
			return MaturityVerdict{}, err
		}
	}
	assetID := component.CatalogID
	if assetID == "" {
		return MaturityVerdict{Achieved: string(catalogcoverage.RungMissing), Floor: floor}, nil
	}
	report := catalogcoverage.ComputeWithEvidence(assets, impls, evidence, gates)
	for _, row := range report.Rows {
		if row.AssetID == assetID && (row.Platform == "" || row.Platform == "react-vite" || row.Platform == scenario) {
			return MaturityVerdict{Achieved: string(row.Achieved), Floor: floor}, nil
		}
	}
	return MaturityVerdict{}, fmt.Errorf("catalog maturity is unavailable for %s", assetID)
}

func declaredMaturityFloor(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read catalog maturity floor: %w", err)
	}
	var config struct {
		Floor string `json:"x-adoptionMaturityFloor"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		return "", fmt.Errorf("parse catalog maturity floor: %w", err)
	}
	if config.Floor == "" {
		return string(catalogcoverage.RungVerified), nil
	}
	return config.Floor, nil
}
