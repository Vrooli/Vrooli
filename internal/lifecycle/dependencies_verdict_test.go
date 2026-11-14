package lifecycle

import (
	"testing"

	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
)

func TestBuildConcurrencyBudgetFitsReservations(t *testing.T) {
	if got := buildConcurrencyBudget(10, []int64{6, 4, 4}); got != 2 {
		t.Fatalf("buildConcurrencyBudget() = %d, want 2", got)
	}
	if got := buildConcurrencyBudget(100, []int64{1, 1, 1, 1, 1, 1, 1, 1, 1}); got != dependenciesParameterC {
		t.Fatalf("buildConcurrencyBudget() = %d, want cap %d", got, dependenciesParameterC)
	}
}

func TestBuildConcurrencyBudgetAlwaysMakesProgress(t *testing.T) {
	for _, tc := range []struct {
		available    int64
		reservations []int64
	}{
		{available: 0, reservations: nil},
		{available: 1, reservations: []int64{100}},
	} {
		if got := buildConcurrencyBudget(tc.available, tc.reservations); got < 1 {
			t.Fatalf("buildConcurrencyBudget(%d, %v) = %d, want at least one worker", tc.available, tc.reservations, got)
		}
	}
}

func TestResourceDependencyReadyAcceptsHealthyPlacementUndetermined(t *testing.T) {
	healthy := true
	status := resourcecontrol.Status{
		Running:    true,
		Healthy:    &healthy,
		Health:     "healthy",
		StatusCode: resourcecontrol.StatusCodePlacementUndetermined,
	}
	if !resourceDependencyReady(status) {
		t.Fatalf("resourceDependencyReady(%+v) = false, want true", status)
	}
}

func TestResourceDependencyStartReasonDoesNotTreatPlacementUndeterminedAsFailure(t *testing.T) {
	healthy := true
	status := resourcecontrol.Status{
		Running:    true,
		Healthy:    &healthy,
		Health:     "healthy",
		StatusCode: resourcecontrol.StatusCodePlacementUndetermined,
	}
	if got := resourceDependencyStartReason(status); got != "not ready" {
		t.Fatalf("resourceDependencyStartReason() = %q, want not ready", got)
	}
}

func TestResourceDependencyReadyAcceptsServingModeDrift(t *testing.T) {
	healthy := false
	serving := true
	status := resourcecontrol.Status{
		Running:    true,
		Healthy:    &healthy,
		Serving:    &serving,
		Health:     "degraded",
		StatusCode: resourcecontrol.StatusCodeModeDrift,
	}
	if !resourceDependencyReady(status) {
		t.Fatalf("resourceDependencyReady(%+v) = false, want true for a serving degraded resource", status)
	}
}

func TestResourceDependencyReadyRejectsUnavailableResource(t *testing.T) {
	healthy := false
	serving := false
	status := resourcecontrol.Status{
		Running:    true,
		Healthy:    &healthy,
		Serving:    &serving,
		Health:     "unhealthy",
		StatusCode: resourcecontrol.StatusCodeUnavailable,
	}
	if resourceDependencyReady(status) {
		t.Fatalf("resourceDependencyReady(%+v) = true, want false for an unavailable resource", status)
	}
}
