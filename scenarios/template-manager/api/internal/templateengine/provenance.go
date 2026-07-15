package templateengine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func buildGenerationProvenance(info templatecontracts.TemplateInfo, design templatecontracts.ResolvedDesign, now time.Time) templatecontracts.GenerationProvenance {
	// Hash failures are non-fatal: a scenario must still generate even if the
	// hasher has a bug, and a stale/missing hash just makes drift unmeasurable
	// for that scenario — the surface degrades gracefully.
	manifestSha, contentSha, _ := computeTemplateHashes(info)
	return templatecontracts.GenerationProvenance{
		Template: templatecontracts.GenerationTemplate{
			ID:      info.Name,
			Version: strings.TrimSpace(info.Manifest.Version),
		},
		GeneratedAt: now.UTC().Format(time.RFC3339),
		Design: templatecontracts.GenerationDesign{
			ID:      strings.TrimSpace(design.KitID),
			Version: strings.TrimSpace(design.Version),
			Adapter: strings.TrimSpace(design.AdapterID),
		},
		ManifestSha: manifestSha,
		ContentSha:  contentSha,
	}
}

func injectScenarioProvenance(destination string, provenance templatecontracts.GenerationProvenance) error {
	servicePath := filepath.Join(destination, ".vrooli", "service.json")
	data, err := os.ReadFile(servicePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read service manifest: %w", err)
	}
	var manifest scenariomodel.ServiceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse service manifest: %w", err)
	}
	manifest.Generation = &scenariomodel.GenerationMetadata{
		Template: scenariomodel.GenerationTemplate{
			ID:      provenance.Template.ID,
			Version: provenance.Template.Version,
		},
		GeneratedAt: provenance.GeneratedAt,
		Design: scenariomodel.GenerationDesign{
			ID:      provenance.Design.ID,
			Version: provenance.Design.Version,
			Adapter: provenance.Design.Adapter,
		},
		ManifestSha: provenance.ManifestSha,
		ContentSha:  provenance.ContentSha,
	}
	rendered, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("render service manifest: %w", err)
	}
	rendered = append(rendered, '\n')
	return os.WriteFile(servicePath, rendered, 0o644)
}

func renderOrientationManifest(destination string, manifest templatecontracts.TemplateManifest, values map[string]string) error {
	if manifest.Orientation == nil {
		return nil
	}
	copyTo := strings.TrimSpace(manifest.Orientation.CopyTo)
	if copyTo == "" {
		return fmt.Errorf("orientation.copyTo is required")
	}
	cleanPath, err := cleanScenarioRelativePath(copyTo)
	if err != nil {
		return fmt.Errorf("orientation.copyTo: %w", err)
	}
	data, err := json.MarshalIndent(manifest.Orientation, "", "  ")
	if err != nil {
		return fmt.Errorf("render orientation manifest: %w", err)
	}
	data = []byte(renderTemplateString(string(data), values))
	target := filepath.Join(destination, cleanPath)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(target, data, 0o644)
}
