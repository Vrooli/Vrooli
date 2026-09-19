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
	byKey := make(map[string]bool, len(current.Endpoints))
	for _, raw := range current.Endpoints {
		var endpoint struct {
			Path   string `json:"path"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(raw, &endpoint); err != nil {
			return fmt.Errorf("decode endpoint: %w", err)
		}
		byKey[endpoint.Method+" "+endpoint.Path] = true
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
		original, readErr := os.ReadFile(outputPath)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", outputPath, readErr)
		}
		var wantJSON, gotJSON any
		if err := json.Unmarshal(want, &wantJSON); err != nil {
			return err
		}
		if err := json.Unmarshal(original, &gotJSON); err != nil {
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

func mergeEndpoints(current []json.RawMessage, modules []module.EndpointDescriptor) ([]json.RawMessage, error) {
	merged := append([]json.RawMessage(nil), current...)
	for _, endpoint := range modules {
		encoded, err := json.Marshal(endpoint)
		if err != nil {
			return nil, fmt.Errorf("encode module endpoint %q: %w", endpoint.ID, err)
		}
		key := endpoint.Method + " " + endpoint.Path
		found := false
		for index, raw := range merged {
			var currentEndpoint struct {
				Path   string `json:"path"`
				Method string `json:"method"`
			}
			if err := json.Unmarshal(raw, &currentEndpoint); err != nil {
				return nil, fmt.Errorf("decode endpoint: %w", err)
			}
			if currentEndpoint.Method+" "+currentEndpoint.Path == key {
				merged[index] = encoded
				found = true
			}
		}
		if found {
			continue
		}
		// The first domain migration replaces the two legacy REST entries.
		// Later migrations add their explicit replacement here before deleting
		// the corresponding legacy route metadata.
		if endpoint.ID == "operatorinputs_list" || endpoint.ID == "operatorinputs_resolve" || endpoint.ID == "readiness_get" || endpoint.ID == "readiness_acknowledge" || endpoint.ID == "apply_start" || endpoint.ID == "apply_run" || endpoint.ID == "apply_plan" || endpoint.ID == "apply_review" || endpoint.ID == "apply_cancel" || endpoint.ID == "session_get" || endpoint.ID == "session_advance" || endpoint.ID == "session_model" || strings.HasPrefix(endpoint.ID, "session_draft_") || strings.HasPrefix(endpoint.ID, "selection_") || strings.HasPrefix(endpoint.ID, "capabilities_") || strings.HasPrefix(endpoint.ID, "credentials_") || strings.HasPrefix(endpoint.ID, "host_") || strings.HasPrefix(endpoint.ID, "operator_state_") || strings.HasPrefix(endpoint.ID, "resources_") || strings.HasPrefix(endpoint.ID, "profiles_") || endpoint.ID == "glossary_search" || endpoint.ID == "configuration_search" {
			filtered := merged[:0]
			for _, raw := range merged {
				var currentEndpoint struct {
					Path string `json:"path"`
				}
				if err := json.Unmarshal(raw, &currentEndpoint); err != nil {
					return nil, fmt.Errorf("decode endpoint: %w", err)
				}
				if strings.HasPrefix(currentEndpoint.Path, "/api/v2/operator-inputs") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/readiness") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/apply") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/session") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/steps") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/scenarios") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/core-set") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/recommendation") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/resources") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/closure") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/union") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/handoff") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/capabilities") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/credentials") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/host-requirements") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/host-facts") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/targets") || strings.HasPrefix(currentEndpoint.Path, "/api/v1/operator-state") || strings.HasPrefix(currentEndpoint.Path, "/api/v2/operator-state") || strings.HasPrefix(currentEndpoint.Path, "/api/v1/resources") || strings.HasPrefix(currentEndpoint.Path, "/api/v1/glossary") || strings.HasPrefix(currentEndpoint.Path, "/vrooli.vrooli_onboarding.v1.profiles.ProfileService/") {
					continue
				}
				filtered = append(filtered, raw)
			}
			merged = append(filtered, encoded)
			continue
		}
		return nil, fmt.Errorf("module endpoint %s %s has no endpoint metadata; regenerate the catalog", endpoint.Method, endpoint.Path)
	}
	return merged, nil
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
