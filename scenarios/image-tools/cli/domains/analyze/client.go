// Package analyze is the CLI's image→data command surface. `analyze list`
// mirrors the Connect AnalysisService discovery RPC; the per-operation commands
// (probe, ocr, nsfw) drive the REST multipart analyze edge
// (POST /api/v1/analysis/{operation}) — image bytes can't ride a Connect call.
package analyze

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"

	analysisv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/analysis"
)

// GroupName is the manifest group name this package owns.
const GroupName = "analyze"

// analyze posts an image to the REST analyze edge and returns the parsed result.
func analyze(core *cliapp.ScenarioApp, operation, inputPath string) (*analysisv1.AnalyzeResponse, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("read input %q: %w", inputPath, err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", filepath.Base(inputPath))
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(data); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	url := strings.TrimRight(baseURL, "/") + "/api/v1/analysis/" + operation
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("analyze %s: %w", operation, err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s failed (%d): %s", operation, resp.StatusCode, strings.TrimSpace(string(out)))
	}
	parsed := &analysisv1.AnalyzeResponse{}
	if err := protojson.Unmarshal(out, parsed); err != nil {
		return nil, fmt.Errorf("decode analyze response: %w", err)
	}
	return parsed, nil
}
