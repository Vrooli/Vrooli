package cliapp

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/vrooli/cli-core/cliutil"
)

func TestUploadFilePostsMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/notes/42/attachments" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer upload-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		if got := r.FormValue("note_id"); got != "42" {
			t.Fatalf("note_id = %q", got)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if header.Filename != "note.txt" || string(data) != "hello" {
			t.Fatalf("file = %q %q", header.Filename, data)
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	app := callTestApp(t, server)
	app.options.APIPrefix = "/api/v1"
	app.tokenSource = func() string { return "upload-token" }
	body, err := UploadFile(app, "/notes/42/attachments", map[string]string{"note_id": "42"}, UploadedFile{
		Name:        "note.txt",
		ContentType: "text/plain",
		Reader:      strings.NewReader("hello"),
	})
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("body = %q", body)
	}
}

func TestUploadFileWrapsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"invalid_argument","message":"bad file"}`)
	}))
	defer server.Close()

	app := callTestApp(t, server)
	_, err := UploadFile(app, "/upload", nil, UploadedFile{Reader: strings.NewReader("x")})
	if err == nil {
		t.Fatal("expected error")
	}
	wrapped := WrapAPIError("attach file", err, nil)
	if !strings.Contains(wrapped.Error(), "invalid_argument: bad file") {
		t.Fatalf("wrapped = %v", wrapped)
	}
}

func TestUploadFileDefaultsAndEscapesFileMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer file.Close()
		if header.Filename != `quote"slash\name.txt` {
			t.Fatalf("filename = %q", header.Filename)
		}
		if got := header.Header.Get("Content-Type"); got != "application/octet-stream" {
			t.Fatalf("content type = %q", got)
		}
		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()

	app := callTestApp(t, server)
	body, err := UploadFile(app, "/upload", nil, UploadedFile{
		Name:   `quote"slash\name.txt`,
		Reader: strings.NewReader("x"),
	})
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q", body)
	}
}

func TestUploadFileLargeBoundedPayload(t *testing.T) {
	const size = 33 << 20
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer file.Close()
		n, err := io.Copy(io.Discard, file)
		if err != nil {
			t.Fatalf("copy upload: %v", err)
		}
		if n != size {
			t.Fatalf("uploaded size = %d, want %d", n, size)
		}
		_, _ = fmt.Fprintf(w, `{"size":%d}`, n)
	}))
	defer server.Close()

	app := callTestApp(t, server)
	body, err := UploadFile(app, "/upload", nil, UploadedFile{
		Name:        "large.bin",
		ContentType: "application/octet-stream",
		Reader:      bytes.NewReader(bytes.Repeat([]byte("x"), size)),
	})
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if !strings.Contains(string(body), fmt.Sprintf(`"size":%d`, size)) {
		t.Fatalf("body = %q", body)
	}
}

func TestUploadFileRejectsInvalidInputs(t *testing.T) {
	if _, err := UploadFile(nil, "/upload", nil, UploadedFile{Reader: strings.NewReader("x")}); err == nil {
		t.Fatal("expected nil app error")
	}
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	app := callTestApp(t, server)
	if _, err := UploadFile(app, "/upload", nil, UploadedFile{}); err == nil {
		t.Fatal("expected nil reader error")
	}

	app.baseOptions = func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{} }
	if _, err := UploadFile(app, "/upload", nil, UploadedFile{Reader: strings.NewReader("x")}); err == nil {
		t.Fatal("expected empty base URL error")
	}

	app.baseOptions = func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{Override: "://bad"}
	}
	if _, err := UploadFile(app, "/upload", nil, UploadedFile{Reader: strings.NewReader("x")}); err == nil {
		t.Fatal("expected invalid base URL error")
	}
}

func TestDecodeUploadResponseDecodesProto(t *testing.T) {
	// wrapperspb.StringValue's protojson form is the bare JSON string "hello".
	resp, err := DecodeUploadResponse[*wrapperspb.StringValue]([]byte(`"hello"`))
	if err != nil {
		t.Fatalf("DecodeUploadResponse: %v", err)
	}
	if resp.GetValue() != "hello" {
		t.Fatalf("value = %q, want %q", resp.GetValue(), "hello")
	}
}

func TestDecodeUploadResponseRejectsInvalidJSON(t *testing.T) {
	if _, err := DecodeUploadResponse[*wrapperspb.StringValue]([]byte("not json")); err == nil {
		t.Fatal("expected decode error for invalid JSON")
	}
}
