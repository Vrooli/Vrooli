package artifactcodecs

import (
	"bytes"
	_ "embed"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
	fastxz "github.com/mikelolasagasti/xz"
	"github.com/vrooli/binaryfetch"
)

//go:embed testdata/tar-small.xz
var smallTarXZ []byte

func TestRegisteredTarXZDecoderInspectsAndExtracts(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "fixture.tar.xz")
	if err := os.WriteFile(archive, smallTarXZ, 0o600); err != nil {
		t.Fatal(err)
	}
	opts := binaryfetch.ExtractOptions{MaxEntries: 8, MaxExpandedBytes: 1 << 20}

	inspected, err := binaryfetch.InspectArchiveBounded(archive, "tar.xz", opts)
	if err != nil {
		t.Fatal(err)
	}
	if inspected.Files != 1 || inspected.ExpandedBytes == 0 {
		t.Fatalf("unexpected archive inspection: %+v", inspected)
	}

	dest := filepath.Join(t.TempDir(), "stage")
	extracted, err := binaryfetch.ExtractArchiveBounded(archive, "tar.xz", dest, opts)
	if err != nil {
		t.Fatal(err)
	}
	if extracted != inspected {
		t.Fatalf("extraction summary %+v differs from inspection %+v", extracted, inspected)
	}
	got, err := os.ReadFile(filepath.Join(dest, "payload.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "bounded xz archive fixture\n" {
		t.Fatalf("extracted payload = %q", got)
	}
}

func TestXZDecoderRejectsCorruptAndTruncatedStreams(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{name: "corrupt footer", data: corruptXZFooter(smallTarXZ)},
		{name: "truncated footer", data: smallTarXZ[:len(smallTarXZ)-4]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decoder, err := newXZDecoder(bytes.NewReader(tc.data))
			if err != nil {
				return // Header-level corruption is rejected eagerly.
			}
			_, err = io.Copy(io.Discard, decoder)
			if err == nil {
				t.Fatal("invalid xz stream decoded without an error")
			}
		})
	}
}

func TestRegisteredTarXZRejectsInvalidFooter(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{name: "corrupt footer", data: corruptXZFooter(smallTarXZ)},
		{name: "truncated footer", data: smallTarXZ[:len(smallTarXZ)-4]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			archive := filepath.Join(t.TempDir(), "invalid.tar.xz")
			if err := os.WriteFile(archive, tc.data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := binaryfetch.InspectArchiveBounded(archive, "tar.xz", binaryfetch.ExtractOptions{}); err == nil {
				t.Fatal("archive inspection accepted an invalid XZ footer")
			}
			if _, err := binaryfetch.ExtractArchiveBounded(archive, "tar.xz", filepath.Join(t.TempDir(), "stage"), binaryfetch.ExtractOptions{}); err == nil {
				t.Fatal("archive extraction accepted an invalid XZ footer")
			}
		})
	}
}

func TestRegisteredTarZstdFinalizesValidAndCorruptStreams(t *testing.T) {
	xzReader, err := fastxz.NewReader(bytes.NewReader(smallTarXZ), maxXZDictionaryBytes)
	if err != nil {
		t.Fatal(err)
	}
	tarBytes, err := io.ReadAll(xzReader)
	if err != nil {
		t.Fatal(err)
	}
	var compressed bytes.Buffer
	encoder, err := zstd.NewWriter(&compressed, zstd.WithEncoderCRC(true))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := encoder.Write(tarBytes); err != nil {
		t.Fatal(err)
	}
	if err := encoder.Close(); err != nil {
		t.Fatal(err)
	}

	validPath := filepath.Join(t.TempDir(), "valid.tar.zst")
	if err := os.WriteFile(validPath, compressed.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := binaryfetch.InspectArchiveBounded(validPath, "tar.zst", binaryfetch.ExtractOptions{}); err != nil {
		t.Fatalf("valid padded tar.zst rejected: %v", err)
	}

	corrupt := bytes.Clone(compressed.Bytes())
	corrupt[len(corrupt)-1] ^= 0x01
	corruptPath := filepath.Join(t.TempDir(), "corrupt.tar.zst")
	if err := os.WriteFile(corruptPath, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := binaryfetch.InspectArchiveBounded(corruptPath, "tar.zst", binaryfetch.ExtractOptions{}); err == nil {
		t.Fatal("archive inspection accepted a corrupt zstd checksum")
	}
	if _, err := binaryfetch.ExtractArchiveBounded(corruptPath, "tar.zst", filepath.Join(t.TempDir(), "stage"), binaryfetch.ExtractOptions{}); err == nil {
		t.Fatal("archive extraction accepted a corrupt zstd checksum")
	}
}

func TestXZDecoderRejectsDictionaryAboveConfiguredLimit(t *testing.T) {
	if maxXZDictionaryBytes != 64<<20 {
		t.Fatalf("production dictionary limit = %d, want 64 MiB", maxXZDictionaryBytes)
	}
	decoder, err := newXZDecoderWithLimit(bytes.NewReader(smallTarXZ), 512<<10)
	if err == nil {
		_, err = io.Copy(io.Discard, decoder)
	}
	if !errors.Is(err, fastxz.ErrMemlimit) {
		t.Fatalf("over-limit stream error = %v, want %v", err, fastxz.ErrMemlimit)
	}
}

func corruptXZFooter(input []byte) []byte {
	corrupt := bytes.Clone(input)
	if len(corrupt) > 0 {
		corrupt[len(corrupt)-1] ^= 0x01
	}
	return corrupt
}
