package lifecycle

import (
	"testing"

	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
)

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
