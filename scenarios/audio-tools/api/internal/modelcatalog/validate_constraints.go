package modelcatalog

import (
	"fmt"
	"strings"
)

func validateHardware(m Model) []Issue {
	var issues []Issue
	h := m.Hardware
	if h.GPURequired && h.CPUCapable {
		issues = append(issues, Issue{
			Code: IssueInvalidHardware, ModelID: m.ID, Field: "hardware",
			Message: "gpu_required and cpu_capable are mutually exclusive",
		})
	}
	if !h.GPURequired && !h.CPUCapable {
		issues = append(issues, Issue{
			Code: IssueInvalidHardware, ModelID: m.ID, Field: "hardware",
			Message: "model must be cpu_capable or gpu_required (remote models declare cpu_capable for the provider host)",
		})
	}
	if h.GPURequired && h.MinVRAMGB <= 0 {
		issues = append(issues, Issue{
			Code: IssueInvalidHardware, ModelID: m.ID, Field: "hardware.min_vram_gb",
			Message: "gpu_required models must declare a positive min_vram_gb",
		})
	}
	if h.MinVRAMGB < 0 || h.MinRAMGB < 0 {
		issues = append(issues, Issue{
			Code: IssueInvalidHardware, ModelID: m.ID, Field: "hardware",
			Message: "min_vram_gb and min_ram_gb must not be negative",
		})
	}
	if m.SizeMBApprox < 0 {
		issues = append(issues, Issue{
			Code: IssueInvalidHardware, ModelID: m.ID, Field: "size_mb_approx",
			Message: "size_mb_approx must not be negative",
		})
	}
	for _, entry := range h.OSArch {
		osName, arch, ok := strings.Cut(entry, "/")
		if !ok || !oneOf(osName, "linux", "darwin", "windows", "any") || !oneOf(arch, "amd64", "arm64", "any") {
			issues = append(issues, Issue{
				Code: IssueInvalidHardware, ModelID: m.ID, Field: "hardware.os_arch",
				Message: fmt.Sprintf("invalid os/arch %q; use linux|darwin|windows|any / amd64|arm64|any", entry),
			})
		}
	}
	return issues
}

func validateBenchmarks(m Model) []Issue {
	var issues []Issue
	b := m.Benchmarks
	if b.WER != nil && *b.WER < 0 {
		issues = append(issues, Issue{
			Code: IssueInvalidBenchmark, ModelID: m.ID, Field: "benchmarks.wer",
			Message: "WER percentage must not be negative",
		})
	}
	if b.SIM != nil && (*b.SIM < 0 || *b.SIM > 1) {
		issues = append(issues, Issue{
			Code: IssueInvalidBenchmark, ModelID: m.ID, Field: "benchmarks.sim",
			Message: "SIM must be a 0..1 similarity score",
		})
	}
	if b.RTF != nil && *b.RTF < 0 {
		issues = append(issues, Issue{
			Code: IssueInvalidBenchmark, ModelID: m.ID, Field: "benchmarks.rtf",
			Message: "RTF must not be negative",
		})
	}
	return issues
}

func validatePricing(m Model) []Issue {
	if m.Pricing == nil {
		if m.Capability == CapabilityDuplex {
			return []Issue{{Code: IssueInvalidPricing, ModelID: m.ID, Field: "pricing", Message: "duplex record must declare published pricing"}}
		}
		return nil
	}
	p := m.Pricing
	var out []Issue
	add := func(field, message string) {
		out = append(out, Issue{Code: IssueInvalidPricing, ModelID: m.ID, Field: field, Message: message})
	}
	if p.Currency != "USD" {
		add("pricing.currency", "only explicitly sourced USD prices are supported")
	}
	if p.Unit != "minute" {
		add("pricing.unit", "pricing unit must be minute")
	}
	for _, rate := range []struct {
		field string
		value *int64
	}{
		{field: "active_micros_per_minute", value: p.ActiveMicrosPerMinute},
		{field: "standby_micros_per_minute", value: p.StandbyMicrosPerMinute},
	} {
		if rate.value == nil || *rate.value < 0 {
			add("pricing."+rate.field, "integer micro-USD rate must be present and non-negative")
		}
	}
	if p.BillingIncrementSeconds <= 0 {
		add("pricing.billing_increment_seconds", "billing increment must be positive")
	}
	if strings.TrimSpace(p.SourceURL) == "" {
		add("pricing.source_url", "pricing must cite its source")
	}
	return out
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
