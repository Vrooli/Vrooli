package storage

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Minimal valid file headers for content-type detection.
var (
	// 8-byte PNG signature + minimal IHDR chunk (enough for DetectContentType).
	pngHeader = []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, // IHDR length
		0x49, 0x48, 0x44, 0x52, // "IHDR"
		0x00, 0x00, 0x00, 0x01, // width 1
		0x00, 0x00, 0x00, 0x01, // height 1
		0x08, 0x02, // bit depth 8, color type 2 (RGB)
		0x00, 0x00, 0x00, // compression, filter, interlace
		0x90, 0x77, 0x53, 0xDE, // CRC
	}
	jpegHeader = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00}
	gifHeader  = []byte("GIF89a" + strings.Repeat("\x00", 20))
	webpHeader = []byte("RIFF\x00\x00\x00\x00WEBPVP8 ")
)

// createTestFile builds a multipart.File and FileHeader suitable for Upload.
func createTestFile(t *testing.T, name string, content []byte, contentType string) (multipart.File, *multipart.FileHeader) {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+name+`"`)
	if contentType != "" {
		h.Set("Content-Type", contentType)
	}
	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("CreatePart: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("Write: %v", err)
	}
	writer.Close()

	reader := multipart.NewReader(&buf, writer.Boundary())
	form, err := reader.ReadForm(int64(len(content)) + 1024)
	if err != nil {
		t.Fatalf("ReadForm: %v", err)
	}

	fhs := form.File["file"]
	if len(fhs) == 0 {
		t.Fatal("no file in form")
	}
	fh := fhs[0]
	f, err := fh.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return f, fh
}

func TestUploadPNG(t *testing.T) {
	svc := NewLocalService(t.TempDir())
	f, fh := createTestFile(t, "test.png", pngHeader, "image/png")
	defer f.Close()

	meta, err := svc.Upload(context.Background(), f, fh)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if meta.ContentType != "image/png" {
		t.Errorf("ContentType = %q, want image/png", meta.ContentType)
	}
	if meta.FileName != "test.png" {
		t.Errorf("FileName = %q, want test.png", meta.FileName)
	}
	if !strings.HasSuffix(meta.StoragePath, ".png") {
		t.Errorf("StoragePath %q should end with .png", meta.StoragePath)
	}
	// Verify file exists on disk.
	if _, err := os.Stat(svc.GetFilePath(meta.StoragePath)); err != nil {
		t.Errorf("file not found on disk: %v", err)
	}
}

func TestUploadJPEG(t *testing.T) {
	svc := NewLocalService(t.TempDir())
	f, fh := createTestFile(t, "photo.jpg", jpegHeader, "image/jpeg")
	defer f.Close()

	meta, err := svc.Upload(context.Background(), f, fh)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if meta.ContentType != "image/jpeg" {
		t.Errorf("ContentType = %q, want image/jpeg", meta.ContentType)
	}
	if !strings.HasSuffix(meta.StoragePath, ".jpg") {
		t.Errorf("StoragePath %q should end with .jpg", meta.StoragePath)
	}
}

func TestUploadGIF(t *testing.T) {
	svc := NewLocalService(t.TempDir())
	f, fh := createTestFile(t, "anim.gif", gifHeader, "image/gif")
	defer f.Close()

	meta, err := svc.Upload(context.Background(), f, fh)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if meta.ContentType != "image/gif" {
		t.Errorf("ContentType = %q, want image/gif", meta.ContentType)
	}
	if !strings.HasSuffix(meta.StoragePath, ".gif") {
		t.Errorf("StoragePath %q should end with .gif", meta.StoragePath)
	}
}

func TestUploadWebP(t *testing.T) {
	svc := NewLocalService(t.TempDir())
	f, fh := createTestFile(t, "image.webp", webpHeader, "image/webp")
	defer f.Close()

	meta, err := svc.Upload(context.Background(), f, fh)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if meta.ContentType != "image/webp" {
		t.Errorf("ContentType = %q, want image/webp", meta.ContentType)
	}
	if !strings.HasSuffix(meta.StoragePath, ".webp") {
		t.Errorf("StoragePath %q should end with .webp", meta.StoragePath)
	}
}

func TestUploadDisallowedType(t *testing.T) {
	svc := NewLocalService(t.TempDir())
	htmlContent := []byte("<html><body>hello</body></html>")
	f, fh := createTestFile(t, "page.html", htmlContent, "text/html")
	defer f.Close()

	_, err := svc.Upload(context.Background(), f, fh)
	if err == nil {
		t.Fatal("expected error for disallowed type")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("error = %q, want 'not allowed'", err.Error())
	}
}

func TestUploadExceedsMaxSize(t *testing.T) {
	svc := NewLocalService(t.TempDir(), WithMaxFileSize(10))
	// PNG header is larger than 10 bytes.
	f, fh := createTestFile(t, "big.png", pngHeader, "image/png")
	defer f.Close()

	_, err := svc.Upload(context.Background(), f, fh)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("error = %q, want 'exceeds maximum'", err.Error())
	}
}

func TestGetReturnsUploaded(t *testing.T) {
	svc := NewLocalService(t.TempDir())
	f, fh := createTestFile(t, "test.png", pngHeader, "image/png")
	defer f.Close()

	meta, err := svc.Upload(context.Background(), f, fh)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	got, err := svc.Get(context.Background(), meta.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != meta.ID {
		t.Errorf("ID = %q, want %q", got.ID, meta.ID)
	}
}

func TestGetUnknownID(t *testing.T) {
	svc := NewLocalService(t.TempDir())
	_, err := svc.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown ID")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want 'not found'", err.Error())
	}
}

func TestGetMultiple(t *testing.T) {
	svc := NewLocalService(t.TempDir())
	ctx := context.Background()

	f1, fh1 := createTestFile(t, "a.png", pngHeader, "image/png")
	defer f1.Close()
	m1, _ := svc.Upload(ctx, f1, fh1)

	f2, fh2 := createTestFile(t, "b.png", pngHeader, "image/png")
	defer f2.Close()
	m2, _ := svc.Upload(ctx, f2, fh2)

	results, err := svc.GetMultiple(ctx, []string{m1.ID, "nonexistent", m2.ID})
	if err != nil {
		t.Fatalf("GetMultiple: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}

func TestGetFilePath(t *testing.T) {
	base := t.TempDir()
	svc := NewLocalService(base)
	got := svc.GetFilePath("2026/03/14/abc.png")
	want := filepath.Join(base, "2026/03/14/abc.png")
	if got != want {
		t.Errorf("GetFilePath = %q, want %q", got, want)
	}
}

func TestGetServingURL(t *testing.T) {
	svc := NewLocalService(t.TempDir())
	got := svc.GetServingURL("2026/03/14/abc.png")
	want := "/api/v1/uploads/2026/03/14/abc.png"
	if got != want {
		t.Errorf("GetServingURL = %q, want %q", got, want)
	}
}

func TestDelete(t *testing.T) {
	svc := NewLocalService(t.TempDir())
	ctx := context.Background()

	f, fh := createTestFile(t, "del.png", pngHeader, "image/png")
	defer f.Close()
	meta, _ := svc.Upload(ctx, f, fh)

	if err := svc.Delete(ctx, meta.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// File should be gone from disk.
	if _, err := os.Stat(svc.GetFilePath(meta.StoragePath)); !os.IsNotExist(err) {
		t.Error("file still exists on disk after delete")
	}

	// Index should not contain it.
	if _, err := svc.Get(ctx, meta.ID); err == nil {
		t.Error("Get succeeded after Delete")
	}
}

func TestDeleteUnknownID(t *testing.T) {
	svc := NewLocalService(t.TempDir())
	err := svc.Delete(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown ID")
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"path traversal", "../../../etc/passwd", "passwd"},
		{"backslash traversal", `..\..\secret.txt`, "secret.txt"},
		{"null byte", "file\x00name.png", "file_name.png"},
		{"empty", "", "unnamed"},
		{"dot", ".", "unnamed"},
		{"dotdot", "..", "unnamed"},
		{"normal", "photo.png", "photo.png"},
		{
			"long name",
			strings.Repeat("a", 300) + ".png",
			strings.Repeat("a", 251) + ".png",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFileName(tt.in)
			if got != tt.want {
				t.Errorf("sanitizeFileName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestConcurrentUploadSafety(t *testing.T) {
	svc := NewLocalService(t.TempDir())
	ctx := context.Background()
	const n = 20

	var wg sync.WaitGroup
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f, fh := createTestFile(t, "concurrent.png", pngHeader, "image/png")
			defer f.Close()
			_, err := svc.Upload(ctx, f, fh)
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent upload error: %v", err)
	}

	// Verify all uploads are in the index.
	svc.mu.RLock()
	count := len(svc.index)
	svc.mu.RUnlock()
	if count != n {
		t.Errorf("index has %d entries, want %d", count, n)
	}
}

// Verify the mock implements Service at compile time.
var (
	_ Service = (*MockService)(nil)
	_ Service = (*LocalService)(nil)
)

// Verify io.Seeker is required (multipart.File embeds it).
var _ io.Seeker = (multipart.File)(nil)
