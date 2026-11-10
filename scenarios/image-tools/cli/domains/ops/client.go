// Package ops is the CLI's deterministic-operation command surface. `ops list`
// mirrors the Connect OpsService discovery RPC; the per-operation run commands
// (resize, crop, …) drive the REST multipart execution edge
// (POST /api/v1/ops/{operation}) — image bytes can't ride a Connect call, so
// these are hand-built commands rather than manifest connect-rpc bindings.
package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
)

// GroupName is the manifest group name this package owns.
const GroupName = "ops"

// runResult holds what the CLI needs from a run: the output bytes and the
// metadata headers the REST edge returns.
type runResult struct {
	body   []byte
	jobID  string
	format string
	width  string
	height string
}

// extToFormat maps a result filename extension to the codec format name so the
// CLI can request the right output encoding from any op via the `?format=`
// query — `--out result.webp` yields WebP even for ops without a format param.
func extToFormat(outPath string) string {
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(outPath), ".")) {
	case "png":
		return "png"
	case "jpg", "jpeg", "jpe":
		return "jpeg"
	case "gif":
		return "gif"
	case "webp":
		return "webp"
	case "tif", "tiff":
		return "tiff"
	case "bmp":
		return "bmp"
	case "avif":
		return "avif"
	default:
		return ""
	}
}

// runOp posts an image + protojson params to the REST run edge and returns the
// streamed result bytes (output=bytes). overlayPath, when non-empty, is sent as
// the `overlay` multipart part (image watermark). outFormat, when non-empty,
// requests that output encoding via the `?format=` query.
func runOp(core *cliapp.ScenarioApp, operation, inputPath, overlayPath, outFormat string, params *opsv1.OpParams) (runResult, error) {
	return runOpWithPlates(core, operation, inputPath, overlayPath, outFormat, params, nil, "")
}

type importedImage struct {
	InputRef string `json:"input_ref"`
	Format   string `json:"format"`
	MIME     string `json:"mime_type"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

func importImageLink(core *cliapp.ScenarioApp, rawURL string) (importedImage, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return importImageLinkWithClient(httpClient, baseURL, rawURL)
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func importImageLinkWithClient(httpClient httpDoer, baseURL, rawURL string) (importedImage, error) {
	body, err := json.Marshal(map[string]string{"url": rawURL})
	if err != nil {
		return importedImage{}, err
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/api/v1/import/link"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return importedImage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return importedImage{}, fmt.Errorf("import image link: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	response, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return importedImage{}, err
	}
	if resp.StatusCode != http.StatusCreated {
		return importedImage{}, fmt.Errorf("import image link: API returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(response)))
	}
	var result importedImage
	if err := json.Unmarshal(response, &result); err != nil {
		return importedImage{}, fmt.Errorf("decode import image link response: %w", err)
	}
	if result.InputRef == "" {
		return importedImage{}, fmt.Errorf("import image link: API returned no input reference")
	}
	return result, nil
}

func runOpWithPlates(core *cliapp.ScenarioApp, operation, inputPath, overlayPath, outFormat string, params *opsv1.OpParams, plates []string, operationID string) (runResult, error) {
	imgBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return runResult{}, fmt.Errorf("read input %q: %w", inputPath, err)
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", filepath.Base(inputPath))
	if err != nil {
		return runResult{}, err
	}
	if _, err := fw.Write(imgBytes); err != nil {
		return runResult{}, err
	}
	if params != nil {
		raw, merr := protojson.Marshal(params)
		if merr != nil {
			return runResult{}, fmt.Errorf("marshal params: %w", merr)
		}
		if err := mw.WriteField("params", string(raw)); err != nil {
			return runResult{}, err
		}
	}
	if overlayPath != "" {
		ov, oerr := os.ReadFile(overlayPath)
		if oerr != nil {
			return runResult{}, fmt.Errorf("read overlay %q: %w", overlayPath, oerr)
		}
		ow, oerr := mw.CreateFormFile("overlay", filepath.Base(overlayPath))
		if oerr != nil {
			return runResult{}, oerr
		}
		if _, oerr := ow.Write(ov); oerr != nil {
			return runResult{}, oerr
		}
	}
	for i, path := range plates {
		data, err := os.ReadFile(path)
		if err != nil {
			return runResult{}, fmt.Errorf("read plate %d: %w", i, err)
		}
		part, err := mw.CreateFormFile(fmt.Sprintf("plate%d", i), filepath.Base(path))
		if err != nil {
			return runResult{}, err
		}
		if _, err = part.Write(data); err != nil {
			return runResult{}, err
		}
	}
	if err := mw.Close(); err != nil {
		return runResult{}, err
	}

	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	url := strings.TrimRight(baseURL, "/") + "/api/v1/ops/" + operation + "?output=bytes"
	if outFormat != "" {
		url += "&format=" + outFormat
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, &body)
	if err != nil {
		return runResult{}, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if operationID != "" {
		req.Header.Set("Idempotency-Key", operationID)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return runResult{}, fmt.Errorf("call %s: %w", operation, err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return runResult{}, err
	}
	if resp.StatusCode == http.StatusAccepted {
		var accepted opsv1.RunOpResponse
		if err := protojson.Unmarshal(out, &accepted); err != nil {
			return runResult{}, err
		}
		return runResult{jobID: accepted.GetJobId(), format: "queued"}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return runResult{}, fmt.Errorf("%s failed (%d): %s", operation, resp.StatusCode, strings.TrimSpace(string(out)))
	}
	return runResult{
		body:   out,
		jobID:  resp.Header.Get("X-Image-Tools-Job-Id"),
		format: resp.Header.Get("X-Image-Tools-Format"),
		width:  resp.Header.Get("X-Image-Tools-Width"),
		height: resp.Header.Get("X-Image-Tools-Height"),
	}, nil
}

// writeOutput writes a run result to --out (or a derived path) and reports it.
// For metadata reads (JSON), it prints to stdout when no --out is given.
func writeOutput(ctx cliapp.RunContext, res runResult, outPath string) error {
	if outPath == "" && res.format == "json" {
		_, _ = ctx.Stdout().Write(res.body)
		_, _ = ctx.Stdout().Write([]byte("\n"))
		return nil
	}
	if outPath == "" {
		return fmt.Errorf("--out is required (where should the result be written?)")
	}
	if err := os.WriteFile(outPath, res.body, 0o644); err != nil {
		return fmt.Errorf("write output %q: %w", outPath, err)
	}
	summary := fmt.Sprintf("Wrote %s (%d bytes)", outPath, len(res.body))
	if res.width != "" {
		summary += fmt.Sprintf(", %sx%s %s", res.width, res.height, res.format)
	}
	return ctx.RenderMutation(cliapp.MutationReport{
		Result:  []string{summary},
		Changes: []string{fmt.Sprintf("job=%s ref=%s", res.jobID, outPath)},
	})
}
