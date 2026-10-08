package generation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type templateGeneratorBuildStamp struct {
	Schema     string `json:"schema"`
	InputsHash string `json:"inputs_sha256"`
}

func templateGeneratorInputFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == "dist" || entry.Name() == "node_modules") && path != root {
			return filepath.SkipDir
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".test.ts") {
			files = append(files, path)
			return nil
		}
		if path == filepath.Join(root, "package.json") || path == filepath.Join(root, "tsconfig.json") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return filepath.ToSlash(files[i]) < filepath.ToSlash(files[j]) })
	return files, nil
}

func templateGeneratorInputsHash(root string) (string, error) {
	files, err := templateGeneratorInputFiles(root)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	for _, file := range files {
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return "", err
		}
		contents, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		_, _ = h.Write([]byte(filepath.ToSlash(rel)))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(contents)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func verifyTemplateGeneratorStamp(templateDir string) error {
	buildTools := filepath.Join(templateDir, "build-tools")
	stampPath := filepath.Join(buildTools, "dist", ".build-stamp.json")
	contents, err := os.ReadFile(stampPath)
	if err != nil {
		return fmt.Errorf("template generator build stamp is missing: %w; run vrooli scenario setup scenario-to-desktop", err)
	}
	var stamp templateGeneratorBuildStamp
	if err := json.Unmarshal(contents, &stamp); err != nil {
		return fmt.Errorf("template generator build stamp is invalid: %w; run vrooli scenario setup scenario-to-desktop", err)
	}
	actual, err := templateGeneratorInputsHash(buildTools)
	if err != nil {
		return fmt.Errorf("cannot compute template generator build stamp: %w; run vrooli scenario setup scenario-to-desktop", err)
	}
	if stamp.Schema != "scenario-to-desktop-template-generator-build-v1" || stamp.InputsHash != actual {
		return fmt.Errorf("template generator is stale: recorded hash %q, current hash %q; run vrooli scenario setup scenario-to-desktop", stamp.InputsHash, actual)
	}
	return nil
}
