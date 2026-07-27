package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/vrooli/api-core/blobstore"
	"signal-inbox/internal/signals"
)

type CommandRunner interface {
	RunOCR(context.Context, string) ([]byte, error)
}

type OSCommandRunner struct{}

func (OSCommandRunner) RunOCR(ctx context.Context, inputPath string) ([]byte, error) {
	// #nosec G204 -- executable/subcommand are fixed; inputPath is a CreateTemp
	// result owned by this process, never an operator-supplied shell fragment.
	return exec.CommandContext(ctx, "image-tools", "analyze", "ocr", inputPath, "--json").Output()
}

// ImageExtractor delegates OCR to image-tools. BlobStore bytes are copied to a
// private temporary file only because the measured image-tools CLI contract
// accepts a local path; retained signal media is never moved or deleted.
type ImageExtractor struct {
	store  blobstore.BlobStore
	runner CommandRunner
}

func NewImageExtractor(store blobstore.BlobStore, runner CommandRunner) *ImageExtractor {
	return &ImageExtractor{store: store, runner: runner}
}

func (e *ImageExtractor) Supports(kind signals.SourceKind) bool {
	return kind == signals.SourceKindImage
}

func (e *ImageExtractor) Extract(ctx context.Context, signal signals.Signal) (ExtractionResult, error) {
	reader, mimeType, err := e.store.Get(ctx, signal.RawPayloadRef)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("read retained image: %w", err)
	}
	defer reader.Close()

	extension := extensionForMIME(mimeType)
	temporary, err := os.CreateTemp("", "signal-inbox-ocr-*"+extension)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("create image-tools input: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o400); err != nil {
		if closeErr := temporary.Close(); closeErr != nil {
			return ExtractionResult{}, fmt.Errorf("protect image-tools input: %w (also close temporary input: %v)", err, closeErr)
		}
		return ExtractionResult{}, fmt.Errorf("protect image-tools input: %w", err)
	}
	if _, err := io.Copy(temporary, reader); err != nil {
		if closeErr := temporary.Close(); closeErr != nil {
			return ExtractionResult{}, fmt.Errorf("materialize image-tools input: %w (also close temporary input: %v)", err, closeErr)
		}
		return ExtractionResult{}, fmt.Errorf("materialize image-tools input: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return ExtractionResult{}, fmt.Errorf("close image-tools input: %w", err)
	}

	output, err := e.runner.RunOCR(ctx, temporaryPath)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("image-tools OCR: %w", err)
	}
	text, err := parseOCRText(output)
	if err != nil {
		return ExtractionResult{}, err
	}
	return ExtractionResult{Content: text, ContentUnits: len(strings.Fields(text))}, nil
}

func parseOCRText(output []byte) (string, error) {
	var response struct {
		Ocr *struct {
			FullText string `json:"fullText"`
		} `json:"ocr"`
		FullText string `json:"fullText"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return "", fmt.Errorf("decode image-tools OCR response: %w", err)
	}
	if response.Ocr != nil {
		return normalizeContent(response.Ocr.FullText), nil
	}
	return normalizeContent(response.FullText), nil
}

func extensionForMIME(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".png"
	}
}
