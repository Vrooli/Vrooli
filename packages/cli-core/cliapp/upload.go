package cliapp

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

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

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("write multipart field %q: %w", k, err)
		}
	}
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, escapeMultipartQuote(defaultUploadFileName(file.Name))))
	partHeader.Set("Content-Type", defaultUploadContentType(file.ContentType))
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return nil, fmt.Errorf("create multipart file: %w", err)
	}
	if _, err := io.Copy(part, file.Reader); err != nil {
		return nil, fmt.Errorf("write multipart file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finish multipart body: %w", err)
	}

	endpoint := base + normalizeAPIPath(path)
	req, err := http.NewRequest(http.MethodPost, endpoint, &body)
	if err != nil {
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
		return nil, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read upload response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, cliutil.ParseAPIError(resp.StatusCode, data)
	}
	return data, nil
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
