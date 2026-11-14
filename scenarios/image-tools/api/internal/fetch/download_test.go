package fetch

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHTTPDownloaderStreamsAndReportsProgress(t *testing.T) {
	body := []byte("download-body")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "13")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	var calls int
	var finalDone, finalTotal int64
	dest := filepath.Join(t.TempDir(), "model.bin")
	err := HTTPDownloader{Client: srv.Client()}.Download(context.Background(), srv.URL+"/model.bin", dest, func(done, total int64) {
		calls++
		finalDone = done
		finalTotal = total
	})
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("downloaded body = %q, want %q", got, body)
	}
	if calls == 0 || finalDone != int64(len(body)) || finalTotal != int64(len(body)) {
		t.Fatalf("progress calls=%d done=%d total=%d", calls, finalDone, finalTotal)
	}
}

func TestHTTPDownloaderRejectsBadRequestAndStatus(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "model.bin")
	if err := (HTTPDownloader{}).Download(context.Background(), "://bad-url", dest, nil); err == nil {
		t.Fatal("expected invalid URL error")
	}

	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	if err := (HTTPDownloader{Client: srv.Client()}).Download(context.Background(), srv.URL, dest, nil); err == nil {
		t.Fatal("expected non-200 status error")
	}
}
