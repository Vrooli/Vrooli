// rest_handlers.go implements the transfer domain's two REST byte edges, the
// CLI twins of the API's documented REST exceptions:
//   - upload:   POST   /api/v1/transfer/items            (multipart file body)
//   - download: GET    /api/v1/transfer/items/{id}/content (streamed bytes)
//
// Both ride outside Connect because opaque/streamed bytes cannot travel in a
// proto message; the upload RESPONSE stays proto-typed (UploadItemResponse).
// Both are device-token authed via the X-Device-Token header. The upload streams
// the file through an io.Pipe so a multi-GB payload never buffers in CLI memory.
package transfer

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	transferv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// upload streams a local file to the transfer multipart endpoint and renders the
// proto-typed Item the server returns.
func (h *handlers) upload(ctx cliapp.RunContext) error {
	token, err := deviceToken(ctx)
	if err != nil {
		return err
	}
	path := strings.TrimSpace(ctx.Flag("file"))
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file %q: %w", path, err)
	}
	defer file.Close()

	fields := map[string]string{}
	if name := strings.TrimSpace(ctx.Flag("name")); name != "" {
		fields["name"] = name
	}
	if retention := strings.TrimSpace(ctx.Flag("retention")); retention != "" {
		// Validate locally so a typo fails fast with a clear message rather
		// than silently uploading with the global-default retention.
		if _, err := parseRetention(retention); err != nil {
			return err
		}
		fields["retention"] = strings.ToLower(retention)
	}
	if target := strings.TrimSpace(ctx.Flag("target")); target != "" {
		fields["target_device_id"] = target
	}

	base := strings.TrimRight(strings.TrimSpace(h.core.APIBase()), "/")
	if base == "" {
		return fmt.Errorf("api base URL is empty; configure an API base or set an API port")
	}
	endpoint := base + "/transfer/items"

	// Stream the multipart body through a pipe: the writer goroutine copies the
	// file into the multipart part while the HTTP client reads from the other
	// end, so the file is never fully buffered in memory.
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	go func() {
		var copyErr error
		defer func() { _ = pw.CloseWithError(copyErr) }()
		for k, v := range fields {
			if copyErr = writer.WriteField(k, v); copyErr != nil {
				return
			}
		}
		part, err := writer.CreateFormFile("file", filepath.Base(path))
		if err != nil {
			copyErr = err
			return
		}
		if _, copyErr = io.Copy(part, file); copyErr != nil {
			return
		}
		copyErr = writer.Close()
	}()

	req, err := http.NewRequest(http.MethodPost, endpoint, pr)
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set(deviceTokenHeader, token)

	resp, err := h.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read upload response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return cliutil.ParseAPIError(resp.StatusCode, body)
	}

	decoded, err := cliapp.DecodeUploadResponse[*transferv1.UploadItemResponse](body)
	if err != nil {
		return err
	}
	if decoded.Item == nil {
		return fmt.Errorf("server returned no item")
	}
	return cliapp.RenderProtoMutation(ctx, decoded, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Uploaded file as item %s.", decoded.Item.Id)},
		Changes: []string{formatItem(decoded.Item)},
		NextCommand: []string{
			"`transfer list` — see it from another device",
			fmt.Sprintf("`transfer download %s --out <path>` — pull it down", decoded.Item.Id),
		},
	})
}

// download streams an item's bytes to a local path, preserving the original
// filename from Content-Disposition when --out is a directory or omitted.
func (h *handlers) download(ctx cliapp.RunContext) error {
	token, err := deviceToken(ctx)
	if err != nil {
		return err
	}
	id := ctx.Positional("id")
	endpoint, err := h.itemContentURL(id, ctx.BoolFlag("thumb"))
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}
	req.Header.Set(deviceTokenHeader, token)

	resp, err := h.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return cliutil.ParseAPIError(resp.StatusCode, body)
	}

	suggested := filenameFromDisposition(resp.Header.Get("Content-Disposition"))
	outPath := resolveOutputPath(ctx.Flag("out"), suggested, id)

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create output file %q: %w", outPath, err)
	}
	written, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		return fmt.Errorf("write %q: %w", outPath, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close %q: %w", outPath, closeErr)
	}

	abs := outPath
	if a, err := filepath.Abs(outPath); err == nil {
		abs = a
	}
	return ctx.RenderMutation(cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Downloaded item %s.", id)},
		Changes: []string{fmt.Sprintf("%s (%d bytes)", abs, written)},
		NextCommand: []string{
			fmt.Sprintf("`transfer get %s` — show this item's metadata", id),
		},
	})
}
