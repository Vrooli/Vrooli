// Package selection is the CLI's smart-select command surface. `select classes`
// and `select suggest` mirror the Connect SelectionService discovery + compiler
// RPCs; `select segment` drives the REST multipart segment edge
// (POST /api/v1/selection/segment) — image bytes can't ride a Connect call —
// and writes the produced mask to --out.
package selection

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

	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/selection"
)

// GroupName is the manifest group name this package owns.
const GroupName = "select"

// segment posts an image + SegmentParams to the REST segment edge and returns
// the parsed result.
func segment(core *cliapp.ScenarioApp, inputPath string, params *selectionv1.SegmentParams) (*selectionv1.SegmentResult, error) {
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
	if params != nil {
		raw, merr := protojson.Marshal(params)
		if merr != nil {
			return nil, merr
		}
		if err := mw.WriteField("params", string(raw)); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	url := strings.TrimRight(baseURL, "/") + "/api/v1/selection/segment"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("segment: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("segment failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(out)))
	}
	parsed := &selectionv1.SegmentResult{}
	if err := protojson.Unmarshal(out, parsed); err != nil {
		return nil, fmt.Errorf("decode segment response: %w", err)
	}
	return parsed, nil
}

// downloadBlob fetches a stored blob (the produced mask) to a local path.
func downloadBlob(core *cliapp.ScenarioApp, ref, outPath string) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	url := strings.TrimRight(baseURL, "/") + "/api/v1/blobs/" + ref
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download mask: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download mask failed (%d)", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("write mask %q: %w", outPath, err)
	}
	return nil
}
