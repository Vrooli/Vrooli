package validationcatalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
)

type staticResolver string

func (r staticResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return string(r), nil
}

func TestResolveMapsProviderOwnedJourneyCatalog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/validation/catalog" || r.URL.Query().Get("scenario") != "sample" {
			t.Fatalf("unexpected catalog request: %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"journeys":[{"journey_id":"sample/bas/cases/login.json","display_name":"Login","source_path":"bas/cases/login.json","execution_mode":"observer","required":true,"category":"existing-bas-case","requirements":["auth"],"estimated_duration_seconds":12,"safety":{"mutating":false,"requires_isolation":false,"requires_confirmation":false}}]}`))
	}))
	defer server.Close()

	catalog, err := NewClient(staticResolver(server.URL), server.Client()).Resolve(context.Background(), "sample")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Journeys) != 1 || catalog.Journeys[0].JourneyID != "sample/bas/cases/login.json" || catalog.Journeys[0].Safety.Mutating {
		t.Fatalf("unexpected catalog: %+v", catalog)
	}
}

func TestResolveFailsClosedOnProviderError(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	_, err := NewClient(staticResolver(server.URL), server.Client()).Resolve(context.Background(), "sample")
	if err == nil {
		t.Fatal("expected catalog provider error")
	}
}

var _ validationmatrix.CatalogResolver = (*Client)(nil)
