package imageengine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/vrooli/api-core/discovery"
)

// Executor is the only raster seam Backdrop Studio depends on. Implementations
// must send every treatment to image-tools; Backdrop Studio deliberately has no
// local pixel implementation.
type Executor interface {
	Apply(ctx context.Context, input []byte, treatments []string, palette map[string]string) ([]byte, error)
}

// Client calls image-tools' synchronous deterministic operation edge. The
// resolver is injectable so unit tests can exercise the transport without
// depending on a running scenario.
type Client struct {
	HTTPClient *http.Client
	Resolve    func(context.Context) (string, error)
}

func NewClient() *Client {
	return &Client{
		HTTPClient: &http.Client{},
		Resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "image-tools")
		},
	}
}

func (c *Client) Apply(ctx context.Context, input []byte, treatments []string, palette map[string]string) ([]byte, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("image-tools: input image is empty")
	}
	if len(treatments) == 0 {
		return nil, fmt.Errorf("image-tools: treatment chain is empty")
	}
	resolve := c.Resolve
	if resolve == nil {
		return nil, fmt.Errorf("image-tools: URL resolver is not configured")
	}
	baseURL, err := resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("image-tools: resolve scenario: %w", err)
	}
	baseURL = strings.TrimRight(baseURL, "/")
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	result := append([]byte(nil), input...)
	for _, treatment := range treatments {
		treatment = strings.TrimSpace(treatment)
		if treatment == "" || strings.HasPrefix(treatment, "$brand.") {
			return nil, fmt.Errorf("image-tools: invalid treatment %q", treatment)
		}
		result, err = run(ctx, httpClient, baseURL, treatment, result, paramsFor(treatment, palette))
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func run(ctx context.Context, client *http.Client, baseURL, operation string, input []byte, params string) ([]byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "backdrop.png")
	if err != nil {
		return nil, fmt.Errorf("image-tools: create multipart image: %w", err)
	}
	if _, err := part.Write(input); err != nil {
		return nil, fmt.Errorf("image-tools: write multipart image: %w", err)
	}
	if err := writer.WriteField("params", params); err != nil {
		return nil, fmt.Errorf("image-tools: write params: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("image-tools: close multipart request: %w", err)
	}
	u := baseURL + "/api/v1/ops/" + url.PathEscape(operation) + "?output=bytes"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, &body)
	if err != nil {
		return nil, fmt.Errorf("image-tools: create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("image-tools: execute %s: %w", operation, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return nil, fmt.Errorf("image-tools: %s returned %s: %s", operation, resp.Status, strings.TrimSpace(string(detail)))
	}
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("image-tools: read %s result: %w", operation, err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("image-tools: %s returned an empty image", operation)
	}
	return out, nil
}

func paramsFor(operation string, palette map[string]string) string {
	dark := palette["$brand.primary"]
	if dark == "" {
		dark = "#0f172a"
	}
	light := palette["$brand.background"]
	if light == "" {
		light = "#e0f2fe"
	}
	switch operation {
	case "duotone":
		return fmt.Sprintf(`{"duotone":{"dark":%q,"light":%q}}`, dark, light)
	case "posterize":
		return fmt.Sprintf(`{"posterize":{"levels":5,"dark":%q,"light":%q}}`, dark, light)
	case "halftone":
		return fmt.Sprintf(`{"halftone":{"lpi":18,"angle":15,"dot":"circle","dark":%q,"light":%q}}`, dark, light)
	case "dither_ordered":
		return fmt.Sprintf(`{"dither_ordered":{"dark":%q,"light":%q}}`, dark, light)
	case "dither_diffusion":
		return fmt.Sprintf(`{"dither_diffusion":{"dark":%q,"light":%q}}`, dark, light)
	case "grain":
		return `{"grain":{"seed":1,"amount":0.12,"contrast_multiplier":1.05}}`
	case "scrim":
		return fmt.Sprintf(`{"scrim":{"color":%q,"opacity":0.34,"direction":"top"}}`, dark)
	default:
		return `{}`
	}
}
