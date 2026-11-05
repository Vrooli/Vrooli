// Package diff is the CLI's image-comparison command surface. `diff modes`
// mirrors the Connect DiffService.ListDiffModes discovery RPC; `diff compare`
// drives the REST multipart compare edge (POST /api/v1/diff/compare) — two image
// payloads can't ride a Connect call — and optionally writes the heat-map to
// --out.
package diff

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

	diffv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff"
)

// GroupName is the manifest group name this package owns.
const GroupName = "diff"

// compare posts the base + compare images and DiffParams to the REST compare
// edge and returns the parsed result.
func compare(core *cliapp.ScenarioApp, basePath, comparePath string, params *diffv1.DiffParams) (*diffv1.DiffResult, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := attachFile(mw, "base", basePath); err != nil {
		return nil, err
	}
	if err := attachFile(mw, "compare", comparePath); err != nil {
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
	url := strings.TrimRight(baseURL, "/") + "/api/v1/diff/compare"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("compare: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("compare failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(out)))
	}
	parsed := &diffv1.DiffResult{}
	if err := protojson.Unmarshal(out, parsed); err != nil {
		return nil, fmt.Errorf("decode compare response: %w", err)
	}
	return parsed, nil
}

func attachFile(mw *multipart.Writer, field, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s %q: %w", field, path, err)
	}
	fw, err := mw.CreateFormFile(field, filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := fw.Write(data); err != nil {
		return err
	}
	return nil
}

// downloadBlob fetches a stored blob (the produced heat-map) to a local path.
func downloadBlob(core *cliapp.ScenarioApp, ref, outPath string) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	url := strings.TrimRight(baseURL, "/") + "/api/v1/blobs/" + ref
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download heat-map: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download heat-map failed (%d)", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("write heat-map %q: %w", outPath, err)
	}
	return nil
}
