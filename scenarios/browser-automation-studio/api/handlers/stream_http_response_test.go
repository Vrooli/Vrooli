package handlers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
)

type countedStreamReader struct {
	reader *bytes.Reader
	reads  int
	bytes  int
	closes int
}

func (r *countedStreamReader) Read(p []byte) (int, error) {
	r.reads++
	n, err := r.reader.Read(p)
	r.bytes += n
	return n, err
}

func (r *countedStreamReader) Close() error {
	r.closes++
	return nil
}

type countedResponseWriter struct {
	header     http.Header
	writeCalls int
	bytes      int
}

func (w *countedResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (*countedResponseWriter) WriteHeader(int) {}

func (w *countedResponseWriter) Write(p []byte) (int, error) {
	w.writeCalls++
	w.bytes += len(p)
	return len(p), nil
}

func TestStreamHTTPResponseCopiesOnePassAndClosesOnce(t *testing.T) {
	payload := bytes.Repeat([]byte("fixture"), 256)
	reader := &countedStreamReader{reader: bytes.NewReader(payload)}
	writer := &countedResponseWriter{}
	var copyErr error

	streamHTTPResponse(writer, reader, "application/octet-stream", int64(len(payload)), "private, max-age=60", func(err error) {
		copyErr = err
	})

	if copyErr != nil {
		t.Fatalf("copy error = %v", copyErr)
	}
	if reader.bytes != len(payload) || writer.bytes != len(payload) || reader.reads != 2 || writer.writeCalls != 1 {
		t.Fatalf("stream counts read=%d/%d bytes, write=%d/%d bytes; want 2 reads incl EOF, one full read pass and one write", reader.reads, reader.bytes, writer.writeCalls, writer.bytes)
	}
	if reader.closes != 1 {
		t.Fatalf("reader closed %d times, want 1", reader.closes)
	}
	if got := writer.Header().Get("Content-Length"); got != "1792" {
		t.Fatalf("Content-Length = %q, want 1792", got)
	}
	if got := writer.Header().Get("Cache-Control"); got != "private, max-age=60" {
		t.Fatalf("Cache-Control = %q", got)
	}
}

type failedResponseWriter struct{ header http.Header }

func (w *failedResponseWriter) Header() http.Header     { return w.header }
func (w *failedResponseWriter) WriteHeader(int)         {}
func (*failedResponseWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestStreamHTTPResponseClosesOnWriteFailure(t *testing.T) {
	reader := &countedStreamReader{reader: bytes.NewReader([]byte("payload"))}
	writer := &failedResponseWriter{header: make(http.Header)}
	var gotErr error

	streamHTTPResponse(writer, reader, "application/octet-stream", 7, "private, max-age=60", func(err error) { gotErr = err })

	if gotErr == nil || gotErr.Error() != "write failed" {
		t.Fatalf("copy error = %v, want write failed", gotErr)
	}
	if reader.closes != 1 {
		t.Fatalf("reader closed %d times, want 1", reader.closes)
	}
}

func BenchmarkStreamHTTPResponseFixedFixture(b *testing.B) {
	const fixtureBytes = 256 * 1024
	fixture := bytes.Repeat([]byte("x"), fixtureBytes)
	b.Run("shared-owner", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(fixtureBytes)
		for i := 0; i < b.N; i++ {
			w := &countedResponseWriter{}
			r := &countedStreamReader{reader: bytes.NewReader(fixture)}
			streamHTTPResponse(w, r, "application/octet-stream", fixtureBytes, "private, max-age=60", nil)
		}
	})
	b.Run("duplicated-lifecycle-baseline", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(fixtureBytes)
		for i := 0; i < b.N; i++ {
			w := &countedResponseWriter{}
			r := &countedStreamReader{reader: bytes.NewReader(fixture)}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", "262144")
			w.Header().Set("Cache-Control", "private, max-age=60")
			_, _ = io.Copy(w, r)
			r.Close()
		}
	})
}
