package cliapp

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/cli-core/cliutil"
)

// UploadedFile describes one file part for UploadFile.
type UploadedFile struct {
	Name        string
	ContentType string
	Reader      io.Reader
}

// UploadFile posts multipart/form-data to a versioned API path and returns the
// raw response body. It is intended for REST exceptions where the payload is
// opaque binary and Connect-RPC is deliberately not the right transport.
func UploadFile(app *ScenarioApp, path string, fields map[string]string, file UploadedFile) ([]byte, error) {
	if app == nil {
		return nil, fmt.Errorf("scenario app is nil")
	}
	if file.Reader == nil {
		return nil, fmt.Errorf("uploaded file reader is nil")
	}
	base := strings.TrimRight(strings.TrimSpace(app.APIBase()), "/")
	if base == "" {
		return nil, fmt.Errorf("api base URL is empty; configure an API base or set an API port")
	}
	if parsed, err := url.Parse(base); err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid api base URL %q", base)
	}

	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)
	writeDone := make(chan error, 1)
	go func() {
		var sourceErr error
		for k, v := range fields {
			if sourceErr = writer.WriteField(k, v); sourceErr != nil {
				break
			}
		}
		if sourceErr == nil {
			partHeader := make(textproto.MIMEHeader)
			partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, escapeMultipartQuote(defaultUploadFileName(file.Name))))
			partHeader.Set("Content-Type", defaultUploadContentType(file.ContentType))
			var part io.Writer
			part, sourceErr = writer.CreatePart(partHeader)
			if sourceErr == nil {
				_, sourceErr = io.Copy(part, file.Reader)
			}
		}
		if closeErr := writer.Close(); sourceErr == nil {
			sourceErr = closeErr
		}
		_ = pipeWriter.CloseWithError(sourceErr)
		writeDone <- sourceErr
	}()

	endpoint := base + normalizeAPIPath(path)
	req, err := http.NewRequest(http.MethodPost, endpoint, pipeReader)
	if err != nil {
		_ = pipeReader.Close()
		return nil, fmt.Errorf("create upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if app.HTTPClient != nil {
		app.HTTPClient.SetToken(strings.TrimSpace(app.tokenSource()))
		app.HTTPClient.ApplyRequestHeaders(req)
	}

	client := &http.Client{}
	if app.HTTPClient != nil {
		client.Timeout = app.HTTPClient.Timeout()
	}
	resp, err := client.Do(req)
	if err != nil {
		_ = pipeReader.CloseWithError(err)
		if sourceErr, complete := waitUploadWriter(writeDone, 250*time.Millisecond); complete && sourceErr != nil {
			return nil, fmt.Errorf("write multipart file: %w", sourceErr)
		}
		return nil, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read upload response: %w", err)
	}
	if resp.StatusCode >= 400 {
		_ = pipeReader.CloseWithError(fmt.Errorf("server rejected multipart upload with status %d", resp.StatusCode))
		return nil, cliutil.ParseAPIError(resp.StatusCode, data)
	}
	if sourceErr, complete := waitUploadWriter(writeDone, 5*time.Second); !complete {
		_ = pipeReader.CloseWithError(fmt.Errorf("upload response arrived before request body finished"))
		return nil, fmt.Errorf("multipart upload writer did not finish after the server response")
	} else if sourceErr != nil {
		return nil, fmt.Errorf("write multipart file: %w", sourceErr)
	}
	return data, nil
}

func waitUploadWriter(done <-chan error, timeout time.Duration) (error, bool) {
	select {
	case err := <-done:
		return err, true
	case <-time.After(timeout):
		return nil, false
	}
}

func defaultUploadFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "upload"
	}
	return name
}

func defaultUploadContentType(contentType string) string {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return "application/octet-stream"
	}
	return contentType
}

func escapeMultipartQuote(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	return strings.ReplaceAll(value, `"`, "\\\"")
}

// DecodeUploadResponse decodes a multipart upload response body into a fresh
// proto message of type T using protojson. Pairs with UploadFile, which
// returns the raw response body for REST-multipart exceptions where Connect-RPC
// is not the right transport.
//
// T is a pointer-to-proto-message type (e.g.
// *notesv1.UploadAttachmentResponse). On decode failure, the error includes
// the type name for easier debugging.
//
// Mirrors the proto-decode half of Call so attach handlers don't redo the
// `protojson.Unmarshal(body, &resp)` ribbon per multipart endpoint.
func DecodeUploadResponse[T proto.Message](body []byte) (T, error) {
	resp := newProto[T]()
	if err := protojson.Unmarshal(body, resp); err != nil {
		var zero T
		return zero, fmt.Errorf("decode %T: %w", resp, err)
	}
	return resp, nil
}
