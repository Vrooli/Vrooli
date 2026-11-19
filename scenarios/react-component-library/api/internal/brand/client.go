package brand

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	assignmentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assignments"
	assignmentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assignments/assignments_v1connect"
	brandsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands"
	brandsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands/brands_v1connect"
)

// brandManagerScenario is the owning scenario's id for discovery.
const brandManagerScenario = "brand-manager"

// maxMarkBytes bounds the mark download. A mark is an SVG or small raster;
// anything past this is not a logo and should not be inlined in a UI response.
const maxMarkBytes = 2 << 20

// Client reads the assignment and brand through Brand Manager's supported
// surface: its Connect-RPC AssignmentsService/BrandsService plus the same
// mark-bytes route its own UI uses. It never opens Brand Manager's database.
type Client struct {
	Scenario       string
	HTTPClient     connect.HTTPClient
	resolveBaseURL func(context.Context) (string, error)
}

// NewClient returns the production resolver. Discovery finds brand-manager the
// way other cross-scenario calls do; BRAND_MANAGER_URL pins the address for a
// deployment that needs it.
func NewClient() *Client {
	return &Client{
		Scenario:   ScenarioName,
		HTTPClient: &http.Client{Timeout: 3 * time.Second},
		resolveBaseURL: func(ctx context.Context) (string, error) {
			return ResolveBaseURL(ctx)
		},
	}
}

// ResolveBaseURL locates Brand Manager, preferring an explicit override.
func ResolveBaseURL(ctx context.Context) (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("BRAND_MANAGER_URL")); explicit != "" {
		return explicit, nil
	}
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, brandManagerScenario)
	if err != nil {
		return "", fmt.Errorf("brand: brand-manager is not reachable: %w", err)
	}
	return baseURL, nil
}

// Resolve returns the assigned brand, or Brand{} when Brand Manager is
// unavailable or has no record for this scenario.
func (c *Client) Resolve(ctx context.Context) (Brand, error) {
	scenario := strings.TrimSpace(c.Scenario)
	if scenario == "" {
		scenario = ScenarioName
	}
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil || strings.TrimSpace(baseURL) == "" {
		return Brand{}, nil
	}
	baseURL = strings.TrimRight(baseURL, "/")
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	assignments := assignmentsconnect.NewAssignmentsServiceClient(httpClient, baseURL)
	statusResponse, err := assignments.GetScenarioStatus(ctx, connect.NewRequest(&assignmentsv1.GetScenarioStatusRequest{ScenarioName: scenario}))
	if err != nil {
		return Brand{}, nil
	}
	status := statusResponse.Msg.GetStatus()
	brandID := strings.TrimSpace(status.GetBrandId())
	if !status.GetHasBrand() || brandID == "" {
		return Brand{}, nil
	}

	brands := brandsconnect.NewBrandsServiceClient(httpClient, baseURL)
	brandResponse, err := brands.GetBrand(ctx, connect.NewRequest(&brandsv1.GetBrandRequest{Id: brandID}))
	if err != nil {
		return Brand{}, nil
	}
	record := brandResponse.Msg.GetBrand()
	if record == nil {
		return Brand{}, nil
	}

	out := Brand{
		Assigned: true,
		Name:     strings.TrimSpace(record.GetName()),
		Accent:   stringPtr(record.GetColors().GetAccent()),
		Wordmark: stringPtr(record.GetIdentity().GetDisplayName()),
	}
	if out.Name == "" {
		out.Name = strings.TrimSpace(record.GetIdentity().GetDisplayName())
	}
	out.Mark = c.resolveMark(ctx, httpClient, baseURL, record)
	return out, nil
}

// resolveMark prefers the picked mark asset (inlined as a data URI so the
// shell is unaffected by Brand Manager's origin policy), then an absolute
// logo URL. Relative logo paths are not browser-reachable from this API and
// are deliberately ignored rather than guessed.
func (c *Client) resolveMark(ctx context.Context, httpClient connect.HTTPClient, baseURL string, record *brandsv1.Brand) *string {
	if assetID := strings.TrimSpace(record.GetMarkAssetId()); assetID != "" {
		if dataURI, ok := downloadMark(ctx, httpClient, baseURL, assetID); ok {
			return &dataURI
		}
	}
	if logo := strings.TrimSpace(record.GetIdentity().GetLogoPath()); strings.HasPrefix(logo, "http://") || strings.HasPrefix(logo, "https://") {
		return &logo
	}
	return nil
}

func downloadMark(ctx context.Context, httpClient connect.HTTPClient, baseURL, brandID string) (string, bool) {
	endpoint := fmt.Sprintf("%s/api/v1/brand-manager/brands/mark?brand_id=%s", baseURL, url.QueryEscape(brandID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", false
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxMarkBytes+1))
	if err != nil || len(raw) == 0 || len(raw) > maxMarkBytes {
		return "", false
	}
	mime := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	if !strings.HasPrefix(mime, "image/") {
		mime = http.DetectContentType(raw)
		if !strings.HasPrefix(mime, "image/") && !strings.Contains(mime, "svg") {
			return "", false
		}
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw), true
}
