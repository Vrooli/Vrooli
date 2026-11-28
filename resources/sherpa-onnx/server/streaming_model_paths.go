package main

import (
	"os"
	"path/filepath"
)

// resolveStreamingModelFile supports Sherpa model archives that use either
// the dated upstream filenames or the generic encoder.onnx/decoder.onnx/
// joiner.onnx layout. An explicit path always wins; the caller remains
// responsible for reporting a missing or invalid override.
func resolveStreamingModelFile(modelDir, override, preferredName string, alternateNames ...string) string {
	if override != "" {
		return override
	}
	for _, name := range append([]string{preferredName}, alternateNames...) {
		path := filepath.Join(modelDir, name)
		info, err := os.Stat(path)
		if err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return filepath.Join(modelDir, preferredName)
}
