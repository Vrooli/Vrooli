package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"landing-page-business-suite-api/internal/presentation"
	"landing-page-business-suite-api/internal/presentationqualification"
	"landing-page-business-suite-api/internal/presentationseed"
)

type compositionAssetVerifier struct{}

func (compositionAssetVerifier) VerifyPublication(context.Context, presentation.Document, *presentation.Document) error {
	return nil
}

func TestPresentationQualificationCompositionDoesNotDiscoverAtConstruction(t *testing.T) {
	called := false
	composition := NewPresentationQualificationComposition(PresentationQualificationCompositionOptions{
		ResolveURL: func(context.Context, string) (string, error) {
			called = true
			return "", errors.New("must not discover during construction")
		},
	})
	if composition == nil {
		t.Fatal("composition is nil")
	}
	if called {
		t.Fatal("owner discovery occurred during construction")
	}
}

func TestPresentationQualificationCompositionPreviewUsesInjectedAssetOwnerWithoutOwnerDiscovery(t *testing.T) {
	document, err := presentationseed.Recommended()
	if err != nil {
		t.Fatalf("Recommended: %v", err)
	}
	composition := NewPresentationQualificationComposition(PresentationQualificationCompositionOptions{
		Assets: compositionAssetVerifier{},
		ResolveURL: func(context.Context, string) (string, error) {
			t.Fatal("preview-only document attempted owner capability discovery")
			return "", nil
		},
	})
	if err := composition.VerifyPublication(context.Background(), document, nil); err != nil {
		t.Fatalf("VerifyPublication: %v", err)
	}
}

func TestPresentationQualificationCompositionRejectsNilContext(t *testing.T) {
	document, err := presentationseed.Recommended()
	if err != nil {
		t.Fatalf("Recommended: %v", err)
	}
	composition := composePresentationQualification(compositionAssetVerifier{}, nil)
	if err := composition.VerifyPublication(nil, document, nil); err == nil {
		t.Fatal("nil context unexpectedly accepted")
	}
}

func TestComposePresentationQualificationFailsClosedWithoutAvailableProof(t *testing.T) {
	document, err := presentationseed.Recommended()
	if err != nil {
		t.Fatalf("Recommended: %v", err)
	}
	document.Apps[0].Publication = presentation.PublicationPublished
	document.Apps[0].Capabilities[0].Status = presentation.CapabilityAvailable
	document.Apps[0].Capabilities[0].EvidenceRefs = []string{"test-genie:validation:receipt-1"}
	document.Apps[0].Capabilities[0].OwnerQualification = &presentation.OwnerQualification{
		Owner:       "web-console",
		EvidenceRef: "test-genie:validation:receipt-1",
		ReleaseRef:  "deployment-manager:release:release-1",
		Qualified:   true,
	}
	composition := composePresentationQualification(compositionAssetVerifier{}, nil)
	if err := composition.VerifyPublication(context.Background(), document, nil); err == nil {
		t.Fatal("available capability without server-owned bindings unexpectedly qualified")
	}
}

func TestPresentationQualificationCompositionDiscoversOwnersPerRead(t *testing.T) {
	document, err := presentationseed.Recommended()
	if err != nil {
		t.Fatalf("Recommended: %v", err)
	}
	document.Apps[0].Publication = presentation.PublicationPublished
	document.Apps[0].Capabilities[0].Status = presentation.CapabilityAvailable
	document.Apps[0].Capabilities[0].EvidenceRefs = []string{"test-genie:validation:receipt-1"}
	document.Apps[0].Capabilities[0].OwnerQualification = &presentation.OwnerQualification{
		Owner:       "web-console",
		EvidenceRef: "test-genie:validation:receipt-1",
		ReleaseRef:  "deployment-manager:release:release-1",
		Qualified:   true,
	}
	bindings := presentationqualification.NewStaticBindings(map[string]presentationqualification.CapabilityBinding{
		"web-console\x00sessions": {
			AppKey:             "web-console",
			CapabilityID:       "sessions",
			Owner:              "web-console",
			ReleaseProfileID:   "profile-1",
			ScenarioID:         "web-console",
			ValidationIntentID: "intent-sessions",
			ValidationRef:      "test-genie:validation:receipt-1",
			ReleaseRef:         "deployment-manager:release:release-1",
			RequiredEvidence: []presentationqualification.EvidenceSelector{{
				EvidenceID: "evidence-1",
				Kind:       "behavioral-test",
				Owner:      "web-console",
				SubjectID:  "web-console",
			}},
		},
	})
	var mu sync.Mutex
	var scenarios []string
	var requestContextSeen bool
	transport := &compositionTransport{}
	composition := NewPresentationQualificationComposition(PresentationQualificationCompositionOptions{
		Bindings:   bindings,
		Assets:     compositionAssetVerifier{},
		HTTPClient: &http.Client{Transport: transport},
		ResolveURL: func(ctx context.Context, scenario string) (string, error) {
			mu.Lock()
			defer mu.Unlock()
			scenarios = append(scenarios, scenario)
			requestContextSeen = ctx.Value(compositionContextKey{}) == "request"
			return "http://owner.invalid", nil
		},
	})
	ctx := database.WithTestMode(context.WithValue(context.Background(), compositionContextKey{}, "request"))
	if err := composition.VerifyPublication(ctx, document, nil); err == nil {
		t.Fatal("VerifyPublication unexpectedly succeeded with unavailable Test Genie")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(scenarios) != 1 || scenarios[0] != presentationTestGenieScenario {
		t.Fatalf("discovered scenarios = %#v, want one Test Genie read", scenarios)
	}
	if !requestContextSeen {
		t.Fatal("owner discovery did not receive the request context")
	}
	if got := transport.path(); got != "/vrooli.test_genie.v1.validation.ValidationService/GetValidation" {
		t.Fatalf("owner request path = %q, want generated Test Genie GetValidation", got)
	}
	if got := transport.header(apihttp.TestModeHeader); got != apihttp.TestModeValue {
		t.Fatalf("owner test-mode header = %q, want %q", got, apihttp.TestModeValue)
	}
}

func TestPresentationQualificationCompositionHonorsCancellation(t *testing.T) {
	document := availableSessionsDocument(t)
	composition := NewPresentationQualificationComposition(PresentationQualificationCompositionOptions{
		Bindings: sessionsBinding(),
		Assets:   compositionAssetVerifier{},
		ResolveURL: func(context.Context, string) (string, error) {
			return "http://owner.invalid", nil
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := composition.VerifyPublication(ctx, document, nil); err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestPresentationQualificationCompositionBoundsOwnerTimeout(t *testing.T) {
	document := availableSessionsDocument(t)
	composition := NewPresentationQualificationComposition(PresentationQualificationCompositionOptions{
		Bindings:       sessionsBinding(),
		Assets:         compositionAssetVerifier{},
		RequestTimeout: 20 * time.Millisecond,
		HTTPClient:     &http.Client{Transport: blockingCompositionTransport{}},
		ResolveURL: func(context.Context, string) (string, error) {
			return "http://owner.invalid", nil
		},
	})
	started := time.Now()
	err := composition.VerifyPublication(context.Background(), document, nil)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "deadline") {
		t.Fatalf("error = %v, want bounded deadline error", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("owner read took %s, want bounded timeout", elapsed)
	}
}

func TestResolveOwnerScenarioRejectsUntrustedEndpoints(t *testing.T) {
	for _, endpoint := range []string{
		"",
		"ftp://owner.invalid",
		"http://user:pass@owner.invalid",
		"http://owner.invalid/private",
		"http://owner.invalid?redirect=https://evil.invalid",
		"http://owner.invalid#fragment",
	} {
		_, err := resolveOwnerScenario(context.Background(), func(context.Context, string) (string, error) {
			return endpoint, nil
		}, presentationTestGenieScenario)
		if err == nil {
			t.Errorf("endpoint %q unexpectedly accepted", endpoint)
		}
	}
}

func availableSessionsDocument(t *testing.T) presentation.Document {
	t.Helper()
	document, err := presentationseed.Recommended()
	if err != nil {
		t.Fatalf("Recommended: %v", err)
	}
	document.Apps[0].Publication = presentation.PublicationPublished
	document.Apps[0].Capabilities[0].Status = presentation.CapabilityAvailable
	document.Apps[0].Capabilities[0].EvidenceRefs = []string{"test-genie:validation:receipt-1"}
	document.Apps[0].Capabilities[0].OwnerQualification = &presentation.OwnerQualification{
		Owner:       "web-console",
		EvidenceRef: "test-genie:validation:receipt-1",
		ReleaseRef:  "deployment-manager:release:release-1",
		Qualified:   true,
	}
	return document
}

func sessionsBinding() presentationqualification.BindingResolver {
	return presentationqualification.NewStaticBindings(map[string]presentationqualification.CapabilityBinding{
		"web-console\x00sessions": {
			AppKey:             "web-console",
			CapabilityID:       "sessions",
			Owner:              "web-console",
			ReleaseProfileID:   "profile-1",
			ScenarioID:         "web-console",
			ValidationIntentID: "intent-sessions",
			ValidationRef:      "test-genie:validation:receipt-1",
			ReleaseRef:         "deployment-manager:release:release-1",
			RequiredEvidence: []presentationqualification.EvidenceSelector{{
				EvidenceID: "evidence-1",
				Kind:       "behavioral-test",
				Owner:      "web-console",
				SubjectID:  "web-console",
			}},
		},
	})
}

type compositionContextKey struct{}

type compositionTransport struct {
	mu      sync.Mutex
	paths   []string
	headers http.Header
}

func (t *compositionTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.paths = append(t.paths, request.URL.Path)
	t.headers = request.Header.Clone()
	t.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Status:     "503 Service Unavailable",
		Body:       io.NopCloser(http.NoBody),
		Header:     make(http.Header),
		Request:    request,
	}, nil
}

func (t *compositionTransport) header(key string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.headers.Get(key)
}

func (t *compositionTransport) path() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.paths) == 0 {
		return ""
	}
	return t.paths[0]
}

type blockingCompositionTransport struct{}

func (blockingCompositionTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	<-request.Context().Done()
	return nil, request.Context().Err()
}
