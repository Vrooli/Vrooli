package system

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

// A healthy card is not the same thing as a used card. An idle GPU at 35
// degrees with 2% utilisation passes every threshold this check had, and says
// nothing about whether the resources that asked for it are actually on it.
// That gap is why a fleet ran on the CPU for fifteen hours with every signal
// green.

// PlacementReporter answers which accelerated resources are running below the
// backend they declared. It is a seam so the GPU check is testable without a
// device or a running fleet.
type PlacementReporter interface {
	DriftedResources(ctx context.Context) ([]DriftedResource, error)
}

// DriftedResource is one resource serving below its declared backend.
type DriftedResource struct {
	Name     string `json:"name"`
	Declared string `json:"declared"`
	Observed string `json:"observed"`
}

// String renders one resource for an operator message.
func (d DriftedResource) String() string {
	observed := strings.TrimSpace(d.Observed)
	if observed == "" {
		observed = "an unknown backend"
	}
	return fmt.Sprintf("%s (declared %s, on %s)", d.Name, d.Declared, observed)
}

// applyPlacement folds the placement report into a GPU check result. It can
// only ever raise the verdict to a warning: a drifted resource is a real
// problem and a serving one, and it must not mask a critical temperature.
func applyPlacement(ctx context.Context, reporter PlacementReporter, result checks.Result) checks.Result {
	if reporter == nil {
		return result
	}
	drifted, err := reporter.DriftedResources(ctx)
	if err != nil {
		result.Details["placementError"] = err.Error()
		return result
	}

	names := make([]string, 0, len(drifted))
	descriptions := make([]string, 0, len(drifted))
	for _, resource := range drifted {
		names = append(names, resource.Name)
		descriptions = append(descriptions, resource.String())
	}
	sort.Strings(names)
	sort.Strings(descriptions)

	result.Details["driftedResources"] = names
	result.Details["driftedResourceCount"] = len(names)
	if len(drifted) == 0 {
		if result.Metrics != nil {
			result.Metrics.SubChecks = append(result.Metrics.SubChecks, checks.SubCheck{
				Name:   "accelerator-placement",
				Passed: true,
				Detail: "every accelerator-declaring resource is on its declared backend",
			})
		}
		return result
	}

	message := fmt.Sprintf("%d resource(s) declared an accelerator and are not on it: %s", len(drifted), strings.Join(descriptions, ", "))
	if result.Metrics != nil {
		result.Metrics.SubChecks = append(result.Metrics.SubChecks, checks.SubCheck{
			Name:   "accelerator-placement",
			Passed: false,
			Detail: message,
		})
	}
	if result.Status == checks.StatusOK {
		result.Status = checks.StatusWarning
		result.Message = message
		return result
	}
	// A card that is already critical stays critical; the placement finding is
	// appended so it is not lost behind the more urgent one.
	result.Message = result.Message + "; " + message
	return result
}
