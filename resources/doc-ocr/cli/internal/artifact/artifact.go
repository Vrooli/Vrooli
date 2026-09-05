package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

const ModelSHA256 = "98d33ace6e54a09f43eb2671950324f50b49861b53617742269f8e54723482a5"

type Resolver struct{ DataDir, SourceRoot string }

func (r Resolver) ModelPath() string {
	if r.DataDir != "" {
		return filepath.Join(r.DataDir, "ocr-model.json")
	}
	candidates := []string{
		filepath.Join(r.SourceRoot, "artifacts", "ocr-model.json"),
		filepath.Join(r.SourceRoot, "..", "artifacts", "ocr-model.json"),
		filepath.Join("..", "artifacts", "ocr-model.json"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

func (r Resolver) Verify() (string, error) {
	path := r.ModelPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return path, fmt.Errorf("OCR model not found at %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if actual != ModelSHA256 {
		return path, fmt.Errorf("OCR model checksum mismatch: expected %s, got %s", ModelSHA256, actual)
	}
	return path, nil
}
