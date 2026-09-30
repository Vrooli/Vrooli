package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/modules"
)

// TestTransportDriftGate protects the two deliberate REST exceptions and the
// UI transport boundary. Every other onboarding operation must use Connect.
func TestTransportDriftGate(t *testing.T) {
	if err := validateTransportDrift(".."); err != nil {
		t.Fatal(err)
	}
}

func TestTransportDriftGateHasTeeth(t *testing.T) {
	t.Run("ui REST call", func(t *testing.T) {
		uiRoot := t.TempDir()
		if err := os.WriteFile(filepath.Join(uiRoot, "App.tsx"), []byte(`export const App = () => fetch("/api/v1/anything");`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := validateUITransport(uiRoot, modules.AllEndpoints()); err == nil {
			t.Fatal("expected an unlisted UI REST call to fail the drift gate")
		}
	})
	t.Run("untagged route", func(t *testing.T) {
		source := []byte(`s.router.HandleFunc("/api/v1/anything", handler).Methods("GET")`)
		if err := validateMainRESTRoutes(source, modules.AllEndpoints()); err == nil {
			t.Fatal("expected an untagged REST route to fail the drift gate")
		}
	})
}

func validateTransportDrift(scenarioRoot string) error {
	endpoints := modules.AllEndpoints()
	mainSource, err := os.ReadFile(filepath.Join(scenarioRoot, "api", "main.go"))
	if err != nil {
		return fmt.Errorf("read main.go: %w", err)
	}
	if err := validateMainRESTRoutes(mainSource, endpoints); err != nil {
		return err
	}
	return validateUITransport(filepath.Join(scenarioRoot, "ui", "src"), endpoints)
}

func validateMainRESTRoutes(source []byte, endpoints []module.EndpointDescriptor) error {
	pattern := regexp.MustCompile(`Handle(?:Func)?\("([^"]+)",.*?\)\.Methods\("([A-Z]+)"\)`)
	byRoute := map[string]module.EndpointDescriptor{}
	for _, endpoint := range endpoints {
		if endpoint.RESTException == nil {
			continue
		}
		byRoute[endpoint.Method+" "+endpoint.Path] = endpoint
	}
	for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
		key := match[2] + " " + match[1]
		endpoint, ok := byRoute[key]
		if !ok || endpoint.RESTException == nil || !isCanonicalRESTReason(endpoint.RESTException.Reason) {
			return fmt.Errorf("route %s is not backed by a tagged REST exception descriptor", key)
		}
	}
	return nil
}

func isCanonicalRESTReason(reason module.RESTReason) bool {
	switch reason {
	case module.RESTReasonMultipartUpload, module.RESTReasonWebhookReceiver, module.RESTReasonThirdPartyShape, module.RESTReasonOpsProbe:
		return true
	default:
		return false
	}
}

func validateUITransport(uiRoot string, endpoints []module.EndpointDescriptor) error {
	allowed := map[string]bool{}
	for _, endpoint := range endpoints {
		if endpoint.RESTException != nil {
			allowed[endpoint.Path] = true
		}
	}
	pathPattern := regexp.MustCompile("[`\\\"'](/[^`\\\"']+)[`\\\"']")
	var failures []string
	err := filepath.Walk(uiRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info == nil || info.IsDir() || !strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for lineNumber, line := range strings.Split(string(data), "\n") {
			if !strings.Contains(line, "fetch(") && !strings.Contains(line, "axios.") {
				continue
			}
			for _, match := range pathPattern.FindAllStringSubmatch(line, -1) {
				candidate := match[1]
				if (candidate == "/health" || strings.HasPrefix(candidate, "/api/")) && !allowed[candidate] {
					failures = append(failures, fmt.Sprintf("%s:%d uses unlisted REST path %s", path, lineNumber+1, candidate))
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(failures) > 0 {
		sort.Strings(failures)
		return fmt.Errorf("UI transport drift: %s", strings.Join(failures, "; "))
	}
	return nil
}
