package modelcatalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCapabilityDefaultsFitResourceBudget makes "small by default" a build
// rule. Each promoted default must declare a real size, fit the capability
// budget, and state the RAM envelope it needs. Without this, promoting a new
// model is a silent machine-wide cost.
func TestCapabilityDefaultsFitResourceBudget(t *testing.T) {
	catalog := Default()

	for _, capability := range Capabilities {
		t.Run(string(capability), func(t *testing.T) {
			budget := FootprintBudget(capability)
			require.Positive(t, budget, "capability %s has no declared footprint budget", capability)

			def, err := catalog.DefaultFor(capability, "", Platform{})
			require.NoError(t, err)
			if !UsesLocalWeights(def.Runtime) {
				return
			}
			require.Positive(t, def.SizeMBApprox,
				"default %q runs locally but declares no size; an unmeasured default cannot be budgeted", def.ID)
			require.LessOrEqual(t, def.SizeMBApprox, float64(budget),
				"default %q declares %d MB, over the %s budget of %d MB; raise the budget deliberately or choose a smaller model",
				def.ID, def.SizeMBApprox, capability, budget)
			require.Positive(t, def.Hardware.MinRAMGB,
				"default %q must declare the RAM envelope it needs", def.ID)
		})
	}
}

// TestPromotedDefaultFootprintFitsBudgets keeps the aggregate default
// footprint within the sum of the per-capability budgets, so a default cannot
// be "within budget" only because it sits in an unbudgeted capability.
func TestPromotedDefaultFootprintFitsBudgets(t *testing.T) {
	catalog := Default()
	budget := 0
	for _, capability := range Capabilities {
		budget += FootprintBudget(capability)
	}
	got := catalog.PromotedDefaultFootprintMB()
	require.Positive(t, got, "promoted default footprint is zero; the registry is not exercising the defaults")
	require.LessOrEqual(t, got, float64(budget),
		"promoted defaults declare %s MB, over the aggregate budget of %d MB", formatSize(got), budget)
}

// TestCommercialDefaultsFitResourceBudget applies the same budget to the
// fail-closed commercial selection, so a licensed default cannot quietly be
// larger than the runtime default.
func TestCommercialDefaultsFitResourceBudget(t *testing.T) {
	catalog := Default()

	for _, capability := range Capabilities {
		commercial, err := catalog.CommercialDefaultFor(capability, "", Platform{})
		if err != nil {
			continue
		}
		if !UsesLocalWeights(commercial.Runtime) {
			continue
		}
		require.Positive(t, commercial.SizeMBApprox, "commercial default %q declares no size", commercial.ID)
		require.LessOrEqual(t, commercial.SizeMBApprox, float64(FootprintBudget(capability)),
			"commercial default %q exceeds the %s footprint budget", commercial.ID, capability)
	}
}

// TestLocalDefaultArtifactsAreEager generalizes the whisper runtime-truth gate:
// the artifact a default model names must be installed by a fresh resource
// install, not parked in an on-demand group. Otherwise the declared default is
// not actually present after setup.
func TestLocalDefaultArtifactsAreEager(t *testing.T) {
	catalog := Default()

	checked := 0
	for _, capability := range Capabilities {
		def, err := catalog.DefaultFor(capability, "", Platform{})
		if err != nil || def.Artifact == nil {
			continue
		}
		manifest := loadResourceManifest(t, def.Artifact.Resource)
		found := false
		for _, artifact := range manifest.ManagedService.DataArtifacts {
			if artifact.Name != def.Artifact.Name {
				continue
			}
			found = true
			require.NotEqual(t, "on-demand", artifact.AcquisitionMode,
				"default %q names on-demand artifact %q; a fresh install would not provide the default",
				def.ID, artifact.Name)
		}
		require.Truef(t, found, "default %q names artifact %q, which resources/%s does not declare",
			def.ID, def.Artifact.Name, def.Artifact.Resource)
		checked++
	}
	require.Positive(t, checked, "no local default artifacts were checked; the gate is not exercising the registry")
}

// TestSummarizeDefaultMovedOffNineB locks the 2026-11-19 resource decision. The
// 9B tag may remain a quality tier, but it must not be the resolved default
// again without an explicit, measured reversal.
func TestSummarizeDefaultMovedOffNineB(t *testing.T) {
	catalog := Default()

	def, err := catalog.DefaultFor(CapabilitySummarize, "", Platform{})
	require.NoError(t, err)
	require.NotEqual(t, "qwen3.5:9b", def.ID, "the 9B summarize tag reserves ~6 GB of VRAM and must not be the default")

	nine, ok := catalog.ByID("qwen3.5:9b")
	require.True(t, ok)
	require.False(t, nine.HasRole("default"), "the 9B tag must stay a quality tier, not a default candidate")
}
