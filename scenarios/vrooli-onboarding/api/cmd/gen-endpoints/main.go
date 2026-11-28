// Command gen-endpoints validates the module-owned endpoint contract. During
// the incremental migration it preserves legacy endpoint metadata until its
// domain is replaced by a Connect module.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/modules"
)

type catalog struct {
	Schema      string            `json:"$schema"`
	Version     string            `json:"version"`
	Service     string            `json:"service"`
	Categories  []string          `json:"categories"`
	CLICommands []json.RawMessage `json:"cli_commands"`
	Endpoints   []json.RawMessage `json:"endpoints"`
}

func main() {
	output := flag.String("output", "../.vrooli/endpoints.json", "endpoint catalog to validate")
	check := flag.Bool("check", false, "validate without writing")
	flag.Parse()
	if err := generate(*output, *check); err != nil {
		fmt.Fprintf(os.Stderr, "gen-endpoints: %v\n", err)
		os.Exit(1)
	}
}

func generate(outputPath string, check bool) error {
	if err := validateTransport(modules.AllEndpoints()); err != nil {
		return err
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", outputPath, err)
	}
	var current catalog
	if err := json.Unmarshal(data, &current); err != nil {
		return fmt.Errorf("decode %s: %w", outputPath, err)
	}
	merged, err := mergeEndpoints(current.Endpoints, modules.AllEndpoints())
	if err != nil {
		return err
	}
	current.Endpoints = merged
	if check {
		want, marshalErr := json.Marshal(current)
		if marshalErr != nil {
			return fmt.Errorf("encode generated endpoint catalog: %w", marshalErr)
		}
		var wantJSON, gotJSON any
		if err := json.Unmarshal(want, &wantJSON); err != nil {
			return err
		}
		if err := json.Unmarshal(data, &gotJSON); err != nil {
			return fmt.Errorf("decode %s: %w", outputPath, err)
		}
		if !bytes.Equal(mustJSON(wantJSON), mustJSON(gotJSON)) {
			return fmt.Errorf("%s is stale; run gen-endpoints", outputPath)
		}
		return nil
	}
	data, err = json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", outputPath, err)
	}
	return os.WriteFile(outputPath, append(data, '\n'), 0o644)
}

type endpointCatalogRow struct {
	method string
	path   string
	raw    json.RawMessage
}

type endpointKey struct {
	method string
	path   string
}

var migratedEndpointIDs = []string{
	"operatorinputs_list", "operatorinputs_resolve", "readiness_get", "readiness_acknowledge",
	"apply_start", "apply_run", "apply_plan", "apply_review", "apply_cancel",
	"session_get", "session_advance", "session_model",
}

var migratedEndpointPrefixes = []string{
	"session_draft_", "selection_", "capabilities_", "credentials_", "host_", "operator_state_",
	"resources_", "profiles_",
}

var retiredLegacyPathPrefixes = []string{
	"/api/v2/operator-inputs", "/api/v2/readiness", "/api/v2/apply", "/api/v2/session",
	"/api/v2/steps", "/api/v2/scenarios", "/api/v2/core-set", "/api/v2/recommendation",
	"/api/v2/resources", "/api/v2/closure", "/api/v2/union", "/api/v2/handoff",
	"/api/v2/capabilities", "/api/v2/credentials", "/api/v2/host-requirements",
	"/api/v2/host-facts", "/api/v2/targets", "/api/v1/operator-state", "/api/v2/operator-state",
	"/api/v1/resources", "/api/v1/glossary", "/vrooli.vrooli_onboarding.v1.profiles.ProfileService/",
}

func decodeCatalogRow(raw json.RawMessage) (endpointCatalogRow, error) {
	var decoded struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return endpointCatalogRow{}, fmt.Errorf("decode endpoint: %w", err)
	}
	return endpointCatalogRow{method: decoded.Method, path: decoded.Path, raw: raw}, nil
}

func moduleOwnsEndpointMetadata(id string) bool {
	for _, exact := range migratedEndpointIDs {
		if id == exact {
			return true
		}
	}
	for _, prefix := range migratedEndpointPrefixes {
		if strings.HasPrefix(id, prefix) {
			return true
		}
	}
	return id == "glossary_search" || id == "configuration_search"
}

func isRetiredLegacyPath(path string) bool {
	for _, prefix := range retiredLegacyPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func mergeEndpoints(current []json.RawMessage, modules []module.EndpointDescriptor) ([]json.RawMessage, error) {
	merged := make([]endpointCatalogRow, 0, len(current)+len(modules))
	for _, raw := range current {
		row, err := decodeCatalogRow(raw)
		if err != nil {
			return nil, err
		}
		merged = append(merged, row)
	}
	for _, endpoint := range modules {
		encoded, err := json.Marshal(endpoint)
		if err != nil {
			return nil, fmt.Errorf("encode module endpoint %q: %w", endpoint.ID, err)
		}
		key := endpointKey{method: endpoint.Method, path: endpoint.Path}
		found := false
		for index := range merged {
			if (endpointKey{method: merged[index].method, path: merged[index].path}) == key {
				merged[index].raw = encoded
				found = true
			}
		}
		if found {
			continue
		}
		// As Connect modules replace domains, retire their old REST metadata only
		// when the catalog does not already contain this endpoint's exact route.
		if moduleOwnsEndpointMetadata(endpoint.ID) {
			filtered := merged[:0]
			for _, row := range merged {
				if !isRetiredLegacyPath(row.path) {
					filtered = append(filtered, row)
				}
			}
			merged = append(filtered, endpointCatalogRow{method: endpoint.Method, path: endpoint.Path, raw: encoded})
			continue
		}
		return nil, fmt.Errorf("module endpoint %s %s has no endpoint metadata; regenerate the catalog", endpoint.Method, endpoint.Path)
	}
	result := make([]json.RawMessage, 0, len(merged))
	for _, row := range merged {
		result = append(result, row.raw)
	}
	return result, nil
}

func mustJSON(value any) []byte {
	data, _ := json.Marshal(value)
	return data
}

func validateTransport(endpoints []module.EndpointDescriptor) error {
	for _, endpoint := range endpoints {
		if strings.HasPrefix(endpoint.Path, "/vrooli.") {
			continue
		}
		if endpoint.RESTException == nil || !validRESTReason(endpoint.RESTException.Reason) {
			return fmt.Errorf("endpoint %q has path %q without a canonical RESTException", endpoint.ID, endpoint.Path)
		}
	}
	return nil
}

func validRESTReason(reason module.RESTReason) bool {
	switch reason {
	case module.RESTReasonMultipartUpload, module.RESTReasonWebhookReceiver,
		module.RESTReasonThirdPartyShape, module.RESTReasonOpsProbe:
		return true
	default:
		return false
	}
}
