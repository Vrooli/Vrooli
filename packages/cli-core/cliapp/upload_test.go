package cliapp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
