package filepreview

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitLineSuffix(t *testing.T) {
	path, line := splitLineSuffix("/tmp/file.ts:42")
	if path != "/tmp/file.ts" || line == nil || *line != 42 {
		t.Fatalf("got path=%q line=%v", path, line)
	}
	path, line = splitLineSuffix("docs/plan.md")
	if path != "docs/plan.md" || line != nil {
		t.Fatalf("got path=%q line=%v", path, line)
	}
	path, line = splitLineSuffix("scenarios/api/main.go:42`")
	if path != "scenarios/api/main.go" || line == nil || *line != 42 {
		t.Fatalf("got path=%q line=%v", path, line)
	}
	// A bare drive-like colon prefix or trailing colon is not a line suffix.
	path, line = splitLineSuffix("foo:")
	if path != "foo:" || line != nil {
		t.Fatalf("got path=%q line=%v", path, line)
	}
}

func TestNormalizePath(t *testing.T) {
	path, line, err := normalizePath("`scenarios/web-console/api/main.go:42`")
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if path != "scenarios/web-console/api/main.go" || line == nil || *line != 42 {
		t.Fatalf("got path=%q line=%v", path, line)
	}

	path, line, err = normalizePath(`"docs/plan.md"`)
	if err != nil {
		t.Fatalf("normalize quoted: %v", err)
	}
	if path != "docs/plan.md" || line != nil {
		t.Fatalf("got path=%q line=%v", path, line)
	}

	// file:// with non-local host is rejected.
	if _, _, err := normalizePath("file://example.com/etc/passwd"); err == nil {
		t.Fatal("expected non-local file URL to be rejected")
	}

	// file://localhost path decodes.
	path, _, err = normalizePath("file://localhost/tmp/a%20b.txt")
	if err != nil || path != "/tmp/a b.txt" {
		t.Fatalf("got path=%q err=%v", path, err)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		kind Kind
	}{
		{"a.md", KindMarkdown},
		{"a.markdown", KindMarkdown},
		{"a.svg", KindSVG},
		{"a.png", KindImage},
		{"a.JPEG", KindImage},
		{"a.avif", KindImage},
		{"a.pdf", KindPDF},
		{"a.mp4", KindVideo},
		{"a.webm", KindVideo},
		{"a.ogv", KindVideo},
		{"a.mp3", KindAudio},
		{"a.ogg", KindAudio},
		{"a.flac", KindAudio},
		{"a.csv", KindCSV},
		{"a.tsv", KindCSV},
		{"a.diff", KindDiff},
		{"a.patch", KindDiff},
		{"a.go", KindCode},
		{"a.tsx", KindCode},
		{"a.txt", KindText},
	}
	for _, c := range cases {
		got := classify(c.name, func() ([]byte, error) { return nil, nil })
		if got.kind != c.kind {
			t.Errorf("%s: kind=%q want %q", c.name, got.kind, c.kind)
		}
		if got.mimeType == "" {
			t.Errorf("%s: empty mime", c.name)
		}
	}
}

func TestClassifyUnknownExtension(t *testing.T) {
	// UTF-8 text content → text.
	got := classify("notes.unknownext", func() ([]byte, error) { return []byte("hello world"), nil })
	if got.kind != KindText {
		t.Fatalf("text sniff kind=%q", got.kind)
	}
	// NUL byte → unsupported (binary, unrecognized MIME).
	got = classify("blob.unknownext", func() ([]byte, error) { return []byte{0x00, 0x01, 0x02, 0x03}, nil })
	if got.kind != KindUnsupported {
		t.Fatalf("binary sniff kind=%q", got.kind)
	}
	// PNG magic with no extension → image.
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0}
	got = classify("nameonly", func() ([]byte, error) { return png, nil })
	if got.kind != KindImage {
		t.Fatalf("png sniff kind=%q", got.kind)
	}
}

func newResolver(root string) *Resolver { return &Resolver{ProjectRoot: root} }

func TestResolveProjectRootRelative(t *testing.T) {
	root := t.TempDir()
	fp := filepath.Join(root, "docs", "plan.md")
	mustWrite(t, fp, "# plan\n")

	r := newResolver(root)
	target, err := r.Resolve("", nil, "docs/plan.md")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if target.ResolvedPath != fp {
		t.Fatalf("resolved=%q want %q", target.ResolvedPath, fp)
	}
	if target.ResolutionBasis != "project_root" {
		t.Fatalf("basis=%q", target.ResolutionBasis)
	}
	if target.Kind != KindMarkdown || !target.TextContentAvailable {
		t.Fatalf("kind=%q textAvail=%v", target.Kind, target.TextContentAvailable)
	}
	if target.Basename != "plan.md" {
		t.Fatalf("basename=%q", target.Basename)
	}
}

func TestResolveSessionCwdPrecedence(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "nested")
	mustWrite(t, filepath.Join(cwd, "a.txt"), "from cwd")
	mustWrite(t, filepath.Join(root, "a.txt"), "from root")

	r := newResolver(root)
	target, err := r.Resolve(cwd, nil, "a.txt")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if target.ResolutionBasis != "session_cwd" {
		t.Fatalf("basis=%q want session_cwd", target.ResolutionBasis)
	}
}

func TestResolveAbsolute(t *testing.T) {
	root := t.TempDir()
	fp := filepath.Join(root, "x.go")
	mustWrite(t, fp, "package x\n")
	r := newResolver(root)
	target, err := r.Resolve("", nil, fp)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if target.ResolutionBasis != "absolute" || target.Kind != KindCode {
		t.Fatalf("basis=%q kind=%q", target.ResolutionBasis, target.Kind)
	}
}

func TestResolveDirectoryRejected(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "adir"))
	r := newResolver(root)
	_, err := r.Resolve("", nil, "adir")
	var fe *Error
	if !errors.As(err, &fe) || fe.Code != CodeNotPreviewable {
		t.Fatalf("expected not_previewable, got %v", err)
	}
}

func TestResolveNotFound(t *testing.T) {
	root := t.TempDir()
	r := newResolver(root)
	_, err := r.Resolve("", nil, "missing.txt")
	var fe *Error
	if !errors.As(err, &fe) || fe.Code != CodeNotFound {
		t.Fatalf("expected not_found, got %v", err)
	}
}

func TestResolveEmptyPath(t *testing.T) {
	r := newResolver(t.TempDir())
	_, err := r.Resolve("", nil, "   ")
	var fe *Error
	if !errors.As(err, &fe) || fe.Code != CodeInvalid {
		t.Fatalf("expected invalid, got %v", err)
	}
}

func TestResolveMediaMetadataNoTextContent(t *testing.T) {
	root := t.TempDir()
	mustWriteBytes(t, filepath.Join(root, "clip.mp4"), []byte("\x00\x00\x00\x18ftypmp42"))
	r := newResolver(root)
	target, err := r.Resolve("", nil, "clip.mp4")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if target.Kind != KindVideo {
		t.Fatalf("kind=%q", target.Kind)
	}
	if target.TextContentAvailable {
		t.Fatal("video should not offer text content")
	}
	if !target.SupportsRange || !target.CanPreview {
		t.Fatalf("supportsRange=%v canPreview=%v", target.SupportsRange, target.CanPreview)
	}
}

func TestResolveOversizeTextDowngrades(t *testing.T) {
	root := t.TempDir()
	big := strings.Repeat("a", 2048)
	mustWrite(t, filepath.Join(root, "big.txt"), big)
	r := &Resolver{ProjectRoot: root, MaxTextBytes: 1024}
	target, err := r.Resolve("", nil, "big.txt")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if target.TextContentAvailable {
		t.Fatal("oversize text should drop TextContentAvailable")
	}
	if len(target.Warnings) == 0 {
		t.Fatal("expected an oversize warning")
	}
}

func TestReadTextValidatesUTF8(t *testing.T) {
	root := t.TempDir()
	mustWriteBytes(t, filepath.Join(root, "bad.txt"), []byte{0xff, 0xfe, 0x00})
	r := newResolver(root)
	target, err := r.Resolve("", nil, "bad.txt")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// A NUL/invalid-UTF8 .txt classified by extension as text still fails ReadText.
	if _, _, err := r.ReadText(target); err == nil {
		t.Fatal("expected ReadText to reject non-UTF8 content")
	}
}

func TestReadTextHappy(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.md"), "# hi\nbody\n")
	r := newResolver(root)
	target, _ := r.Resolve("", nil, "a.md")
	content, truncated, err := r.ReadText(target)
	if err != nil || truncated || !strings.Contains(content, "# hi") {
		t.Fatalf("content=%q truncated=%v err=%v", content, truncated, err)
	}
}

// --- helpers ---

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	mustWriteBytes(t, path, []byte(content))
}

func mustWriteBytes(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
}
