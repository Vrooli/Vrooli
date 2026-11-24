package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSetpointRejectsMalformedInput(t *testing.T) {
	p := filepath.Join(t.TempDir(), "setpoint.json")
	if err := os.WriteFile(p, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSetpoint(p, nil); err == nil {
		t.Fatal("expected loud parse failure")
	}
}

func TestLoadSetpointRejectsTrailingJSON(t *testing.T) {
	for name, trailing := range map[string]string{
		"second value": ` {"schemaVersion":"1.0.0","bars":[]}`,
		"garbage":      ` trailing`,
	} {
		t.Run(name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "setpoint.json")
			body := `{"schemaVersion":"1.0.0","bars":[],"notGradeable":[]}` + trailing
			if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadSetpoint(p, nil); err == nil {
				t.Fatal("expected trailing data to be rejected")
			}
		})
	}
}

func TestLoadSetpointAcceptsTrailingWhitespace(t *testing.T) {
	p := filepath.Join(t.TempDir(), "setpoint.json")
	if err := os.WriteFile(p, []byte("{\"$schema\":\"./setpoint.schema.json\",\"schemaVersion\":\"1.0.0\",\"bars\":[],\"notGradeable\":[]} \n\t"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSetpoint(p, nil); err != nil {
		t.Fatalf("LoadSetpoint rejected valid trailing whitespace: %v", err)
	}
}

func TestCheckedInSetpointLoadsAgainstCheckedInRegistry(t *testing.T) {
	registry, err := LoadRegistry(filepath.Join("..", "config", "outcome-registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := LoadSplitRoomCatalog(registry); err != nil {
		t.Fatal(err)
	}
	if err := LoadSplitSignalCatalog(registry); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSetpoint(filepath.Join("..", "config", "setpoint.json"), registry); err != nil {
		t.Fatalf("checked-in setpoint did not validate against registry: %v", err)
	}
}

func TestLoadSetpointRejectsBarEqualToAuthoredValue(t *testing.T) {
	p := filepath.Join(t.TempDir(), "setpoint.json")
	body := `{"schemaVersion":"1.0.0","bars":[{"metricId":"m","direction":"up","bar":10,"unit":"count","authoredAgainst":10,"decisionRef":"d"}]}`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSetpoint(p, nil); err == nil {
		t.Fatal("expected equal-bar failure")
	}
}

func TestLoadSetpointRequiresTypedBarFields(t *testing.T) {
	for name, bar := range map[string]string{
		"string value":     `{"metricId":"m","direction":"up","bar":"12","unit":"count","authoredAgainst":10,"decisionRef":"decision:test"}`,
		"missing baseline": `{"metricId":"m","direction":"up","bar":12,"unit":"count","decisionRef":"decision:test"}`,
		"missing unit":     `{"metricId":"m","direction":"up","bar":12,"authoredAgainst":10,"decisionRef":"decision:test"}`,
	} {
		t.Run(name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "setpoint.json")
			body := `{"schemaVersion":"1.0.0","bars":[` + bar + `]}`
			if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadSetpoint(p, nil); err == nil {
				t.Fatal("expected malformed target definition to be rejected")
			}
		})
	}
}

func TestLoadSetpointRejectsUnsupportedSchemaVersion(t *testing.T) {
	p := filepath.Join(t.TempDir(), "setpoint.json")
	if err := os.WriteFile(p, []byte(`{"schemaVersion":"2.0.0","bars":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSetpoint(p, nil); err == nil {
		t.Fatal("expected unsupported schema version to be rejected")
	}
}

func TestLoadSetpointRejectsUnknownDirection(t *testing.T) {
	p := filepath.Join(t.TempDir(), "setpoint.json")
	body := `{"schemaVersion":"1.0.0","bars":[{"metricId":"m","direction":"sideways","bar":12,"unit":"count","authoredAgainst":10,"decisionRef":"decision:test"}]}`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSetpoint(p, nil); err == nil {
		t.Fatal("expected unknown target direction to be rejected")
	}
}

func TestLoadSetpointRequiresRegisteredMetricAndMatchingUnit(t *testing.T) {
	registry := &Registry{Metrics: []MetricEntry{{ID: "m", Unit: "count", Coverage: CoverageNow}}}
	for name, bar := range map[string]string{
		"unknown metric": `{"metricId":"other","direction":"up","bar":12,"unit":"count","authoredAgainst":10,"decisionRef":"decision:test"}`,
		"wrong unit":     `{"metricId":"m","direction":"up","bar":12,"unit":"usd","authoredAgainst":10,"decisionRef":"decision:test"}`,
	} {
		t.Run(name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "setpoint.json")
			body := `{"schemaVersion":"1.0.0","bars":[` + bar + `],"notGradeable":[]}`
			if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadSetpoint(p, registry); err == nil {
				t.Fatal("expected invalid target binding to be rejected")
			}
		})
	}
}
