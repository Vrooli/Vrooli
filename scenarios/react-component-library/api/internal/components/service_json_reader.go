package components

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FSServiceJSONReader resolves <scenariosRoot>/<scenario>/.vrooli/service.json
// with the same traversal guard shape as deps.FSPackageJSONReader.
type FSServiceJSONReader struct {
	Root string
}

func NewFSServiceJSONReader(root string) *FSServiceJSONReader {
	return &FSServiceJSONReader{Root: root}
}

var _ ServiceJSONReader = (*FSServiceJSONReader)(nil)

// templateScenarioPrefix is the scenario-key form adoption records use for
// vendored template copies: "../templates/scenarios/<id>", resolved relative
// to the scenarios root (matching adoptions.FSScenarioFileReader).
const templateScenarioPrefix = "../templates/scenarios/"

func (r *FSServiceJSONReader) Read(_ context.Context, scenario string) ([]byte, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, fmt.Errorf("scenario required")
	}
	base := r.Root
	name := scenario
	if tmpl, ok := strings.CutPrefix(scenario, templateScenarioPrefix); ok {
		base = filepath.Clean(filepath.Join(r.Root, "..", "templates", "scenarios"))
		name = tmpl
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return nil, fmt.Errorf("invalid scenario name %q", scenario)
	}
	full := filepath.Join(base, name, ".vrooli", "service.json")
	cleaned := filepath.Clean(full)
	rootClean := filepath.Clean(base) + string(filepath.Separator)
	if !strings.HasPrefix(cleaned, rootClean) {
		return nil, fmt.Errorf("resolved path escapes root")
	}
	return os.ReadFile(cleaned)
}

func readScenarioDesignStyle(ctx context.Context, reader ServiceJSONReader, scenario string) (string, error) {
	raw, err := reader.Read(ctx, scenario)
	if err != nil {
		return "", fmt.Errorf("read service.json for scenario %q: %w", scenario, err)
	}
	var service struct {
		Generation struct {
			Design struct {
				ID string `json:"id"`
			} `json:"design"`
		} `json:"generation"`
	}
	if err := json.Unmarshal(raw, &service); err != nil {
		return "", fmt.Errorf("parse service.json for scenario %q: %w", scenario, err)
	}
	return strings.TrimSpace(service.Generation.Design.ID), nil
}

func styleFitDetail(libraryID, styleID string, affinity ComponentDesignAffinity) string {
	reason := strings.TrimSpace(affinity.Reason)
	switch affinity.Affinity {
	case DesignAffinityNative:
		if reason != "" {
			return fmt.Sprintf("component %q is native to design style %q: %s", libraryID, styleID, reason)
		}
		return fmt.Sprintf("component %q is native to design style %q", libraryID, styleID)
	case DesignAffinityCompatible:
		if reason != "" {
			return fmt.Sprintf("component %q is compatible with design style %q: %s", libraryID, styleID, reason)
		}
		return fmt.Sprintf("component %q is compatible with design style %q", libraryID, styleID)
	case DesignAffinityDiscouraged:
		if reason != "" {
			return fmt.Sprintf("component %q is discouraged for design style %q: %s", libraryID, styleID, reason)
		}
		return fmt.Sprintf("component %q is discouraged for design style %q", libraryID, styleID)
	default:
		return fmt.Sprintf("component %q has an unknown affinity for design style %q", libraryID, styleID)
	}
}
