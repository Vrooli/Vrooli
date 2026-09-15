package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/discovery"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	releasesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases/releasesv1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation/validation_v1connect"
	"landing-page-business-suite-api/internal/presentation"
	"landing-page-business-suite-api/internal/presentationqualification"
)

const (
	presentationTestGenieScenario         = "test-genie"
	presentationDeploymentManagerScenario = "deployment-manager"
	defaultPresentationOwnerTimeout       = 10 * time.Second
)

// PresentationScenarioURLResolver resolves an owner scenario for one request.
// The default implementation uses api-core discovery without demand acquisition
// or lifecycle startup. Tests and the composition root may inject a resolver.
type PresentationScenarioURLResolver func(context.Context, string) (string, error)

// PresentationQualificationComposition is the LPBS publication-verifier
// composition root. It contains only server-owned dependencies. Test Genie and
// Deployment Manager clients are constructed lazily during an owner read, so a
// draft/preview document with no public available claims does not require an
// owner service at startup or during an unrelated request.
type PresentationQualificationComposition struct {
	bindings   presentationqualification.BindingResolver
	assets     presentationqualification.AssetPublicationVerifier
	resolveURL PresentationScenarioURLResolver
	httpClient *http.Client
	timeout    time.Duration
}

// PresentationQualificationCompositionOptions supplies server-owned seams.
// Bindings must come from an owner-maintained mapping; a presentation document
// cannot establish receipt membership by itself. Assets is the Backdrop/asset
// owner verifier supplied by the Schrodinger-owned composition.
type PresentationQualificationCompositionOptions struct {
	Bindings   presentationqualification.BindingResolver
	Assets     presentationqualification.AssetPublicationVerifier
	ResolveURL PresentationScenarioURLResolver
	HTTPClient *http.Client
	// RequestTimeout bounds each read from a discovered owner. A zero value
	// uses the bounded default; caller cancellation still wins immediately.
	RequestTimeout time.Duration
}

// NewPresentationQualificationComposition creates a pure request-time
// composition. It performs no discovery, network I/O, writes, deployment,
// publication, payment, or owner lifecycle operation.
func NewPresentationQualificationComposition(opts PresentationQualificationCompositionOptions) *PresentationQualificationComposition {
	resolveURL := opts.ResolveURL
	if resolveURL == nil {
		resolveURL = func(ctx context.Context, scenario string) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, scenario)
		}
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	httpClient = apihttp.NewTestModeClient(httpClient)
	timeout := opts.RequestTimeout
	if timeout <= 0 {
		timeout = defaultPresentationOwnerTimeout
	}
	return &PresentationQualificationComposition{
		bindings:   opts.Bindings,
		assets:     opts.Assets,
		resolveURL: resolveURL,
		httpClient: httpClient,
		timeout:    timeout,
	}
}

// composePresentationQualification is the composition-root seam used by
// LPBS. The asset verifier is supplied by Schrodinger and remains the owner of
// released asset bytes. A nil binding resolver is allowed for draft bootstrap,
// but any public available capability then fails closed during verification.
func composePresentationQualification(
	assetVerifier presentationqualification.AssetPublicationVerifier,
	bindings presentationqualification.BindingResolver,
) *PresentationQualificationComposition {
	return NewPresentationQualificationComposition(PresentationQualificationCompositionOptions{
		Assets:   assetVerifier,
		Bindings: bindings,
	})
}

// VerifyPublication implements the ConfigStore publication-verifier shape.
// Owner reads use the caller's context and resolve each owner endpoint at the
// point of the read. The generated clients expose only read methods through
// the adapters; no write-capable owner operation is invoked here.
func (c *PresentationQualificationComposition) VerifyPublication(ctx context.Context, document presentation.Document) error {
	if c == nil {
		return fmt.Errorf("presentation qualification composition is unavailable")
	}
	if ctx == nil {
		return errors.New("presentation qualification requires a non-nil request context")
	}
	validations := &requestValidationReader{resolveURL: c.resolveURL, httpClient: c.httpClient, timeout: c.timeout}
	releases := &requestReleaseReader{resolveURL: c.resolveURL, httpClient: c.httpClient, timeout: c.timeout}
	verifier := &presentationqualification.Verifier{
		Validations: validations,
		Releases:    releases,
		Bindings:    c.bindings,
		Assets:      c.assets,
	}
	return verifier.VerifyPublication(ctx, document)
}

type requestValidationReader struct {
	resolveURL PresentationScenarioURLResolver
	httpClient *http.Client
	timeout    time.Duration
}

func (r *requestValidationReader) ReadValidation(ctx context.Context, id string) (*validationReceipt, error) {
	if r == nil || r.resolveURL == nil {
		return nil, errors.New("Test Genie discovery is unavailable")
	}
	ownerContext, cancel, err := boundedOwnerContext(ctx, r.timeout)
	if err != nil {
		return nil, err
	}
	defer cancel()
	baseURL, err := resolveOwnerScenario(ownerContext, r.resolveURL, presentationTestGenieScenario)
	if err != nil {
		return nil, err
	}
	client := validationconnect.NewValidationServiceClient(r.httpClient, baseURL)
	reader, err := presentationqualification.NewTestGenieValidationReader(client)
	if err != nil {
		return nil, err
	}
	return reader.ReadValidation(ownerContext, id)
}

type requestReleaseReader struct {
	resolveURL PresentationScenarioURLResolver
	httpClient *http.Client
	timeout    time.Duration
}

func (r *requestReleaseReader) ReadRelease(ctx context.Context, id string) (*releaseView, error) {
	if r == nil || r.resolveURL == nil {
		return nil, errors.New("Deployment Manager discovery is unavailable")
	}
	ownerContext, cancel, err := boundedOwnerContext(ctx, r.timeout)
	if err != nil {
		return nil, err
	}
	defer cancel()
	baseURL, err := resolveOwnerScenario(ownerContext, r.resolveURL, presentationDeploymentManagerScenario)
	if err != nil {
		return nil, err
	}
	client := releasesconnect.NewReleasesServiceClient(r.httpClient, baseURL)
	reader, err := presentationqualification.NewDeploymentManagerReleaseReader(client)
	if err != nil {
		return nil, err
	}
	return reader.ReadRelease(ownerContext, id)
}

func resolveOwnerScenario(ctx context.Context, resolve PresentationScenarioURLResolver, scenario string) (string, error) {
	if ctx == nil {
		return "", errors.New("owner discovery requires a non-nil request context")
	}
	if resolve == nil {
		return "", errors.New("owner discovery resolver is unavailable")
	}
	baseURL, err := resolve(ctx, scenario)
	if err != nil {
		return "", fmt.Errorf("discover %s: %w", scenario, err)
	}
	parsed, err := safeOwnerOrigin(baseURL)
	if err != nil {
		return "", fmt.Errorf("discover %s: %w", scenario, err)
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func boundedOwnerContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc, error) {
	if ctx == nil {
		return nil, nil, errors.New("owner read requires a non-nil request context")
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if timeout <= 0 {
		timeout = defaultPresentationOwnerTimeout
	}
	derived, cancel := context.WithTimeout(ctx, timeout)
	return derived, cancel, nil
}

func safeOwnerOrigin(raw string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	parsed, err := url.Parse(trimmed)
	if err != nil || trimmed == "" || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, errors.New("owner endpoint must be a server-owned http(s) origin without credentials, path, query, or fragment")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("owner endpoint must use HTTP or HTTPS")
	}
	return parsed, nil
}

// These aliases keep the request readers' method declarations readable while
// retaining the generated owner types at the adapter boundary.
type validationReceipt = validationv1.ValidationReceipt
type releaseView = releasesv1.ReleaseView
