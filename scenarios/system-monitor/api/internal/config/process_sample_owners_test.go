package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseProcSampleOwnersTrimsAndDeduplicates(t *testing.T) {
	got, err := ParseProcSampleOwners(" device-control, ,system-monitor,device-control ")
	if err != nil {
		t.Fatalf("ParseProcSampleOwners returned error: %v", err)
	}
	want := []string{"device-control", "system-monitor"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseProcSampleOwners = %#v, want %#v", got, want)
	}
}

func TestParseProcSampleOwnersRejectsMoreThanBoundedOwnerCount(t *testing.T) {
	owners := make([]string, maxProcSampleOwners+1)
	for i := range owners {
		owners[i] = "scenario-" + strings.Repeat("x", i+1)
	}
	if _, err := ParseProcSampleOwners(strings.Join(owners, ",")); err == nil {
		t.Fatal("expected owner-list bound error")
	}
}

func TestLoadReadsConfiguredProcessSampleOwners(t *testing.T) {
	values := map[string]string{
		"API_PORT":                          "1234",
		"SYSTEM_MONITOR_PROC_SAMPLE_OWNERS": "device-control,system-monitor",
	}
	cfg, err := LoadFrom(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	want := []string{"device-control", "system-monitor"}
	if !reflect.DeepEqual(cfg.Monitoring.ProcSampleOwners, want) {
		t.Fatalf("configured owners = %#v, want %#v", cfg.Monitoring.ProcSampleOwners, want)
	}
}

func TestLoadFromRequiresAPIPortWithoutExiting(t *testing.T) {
	_, err := LoadFrom(func(string) (string, bool) { return "", false })
	if err == nil || !strings.Contains(err.Error(), "API_PORT is required") {
		t.Fatalf("LoadFrom error = %v, want an API_PORT validation error", err)
	}
}

func TestLoadFromUsesInjectedValuesAndDefaults(t *testing.T) {
	values := map[string]string{"PORT": "8080", "METRICS_INTERVAL_SECONDS": "45", "UI_PORT": "3000"}
	cfg, err := LoadFrom(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Server.APIPort != "8080" || cfg.Monitoring.MetricsInterval.Seconds() != 45 {
		t.Fatalf("loaded config = %+v, want injected port and interval", cfg)
	}
	if cfg.Server.Environment != "development" || len(cfg.Health.Endpoints) != 2 {
		t.Fatalf("defaults/endpoints = %+v, want development and API/UI endpoints", cfg)
	}
}
