package domain

import (
	"encoding/json"
	"fmt"
	"os"

	"resource-doc-ocr/cli/internal/artifact"
	"resource-doc-ocr/cli/internal/discovery"
	"resource-doc-ocr/cli/internal/env"
	"resource-doc-ocr/cli/internal/ocr"
	"resource-doc-ocr/cli/internal/version"
)

type Service struct {
	Config   env.Config
	Runtime  discovery.Runtime
	Manifest version.Manifest
}

func NewService(cfg env.Config, runtime discovery.Runtime) Service {
	return Service{Config: cfg, Runtime: runtime, Manifest: version.Manifest{InstalledPath: runtime.InstalledManifest, SourcePath: runtime.SourceManifestPath}}
}

func (s Service) PrintInfo(name, version, description string) error {
	fmt.Printf("%s %s\n%s\n", name, version, description)
	if s.Runtime.InstalledManifest != "" {
		fmt.Printf("manifest: %s\n", s.Runtime.InstalledManifest)
	}
	return nil
}

func (s Service) resolver() artifact.Resolver {
	return artifact.Resolver{DataDir: s.Config.DataDir, SourceRoot: s.Runtime.SourceRoot}
}

func (s Service) Health() error {
	path, err := s.resolver().Verify()
	if err != nil {
		return fmt.Errorf("unhealthy: %w", err)
	}
	fixture, err := os.CreateTemp("", "vrooli-doc-ocr-health-*.txt")
	if err != nil {
		return fmt.Errorf("unhealthy: create fixture: %w", err)
	}
	defer os.Remove(fixture.Name())
	if _, err := fixture.WriteString("health probe"); err != nil {
		return fmt.Errorf("unhealthy: write fixture: %w", err)
	}
	if err := fixture.Close(); err != nil {
		return fmt.Errorf("unhealthy: close fixture: %w", err)
	}
	result, err := ocr.Recognize(fixture.Name(), "eng")
	if err != nil || len(result.Runs) == 0 {
		return fmt.Errorf("unhealthy: recognition probe: %v", err)
	}
	fmt.Printf("healthy model=%s sha256=%s mode=cpu\n", path, artifact.ModelSHA256)
	return nil
}

func (s Service) PrintStatus() error     { return s.Health() }
func (s Service) PrintDomainHelp() error { return s.Capabilities() }

func (s Service) Capabilities() error {
	return printJSON(map[string]any{"resource": "doc-ocr", "engine": "embedded-text-baseline", "mode": "cpu", "languages": []string{"eng"}, "positioned_runs": true, "confidence": true, "network": "none"})
}

func (s Service) Version(name, version string) error {
	path, err := s.resolver().Verify()
	result := map[string]any{"name": name, "version": version, "engine": "embedded-text-baseline", "mode": "cpu"}
	if err != nil {
		result["model_error"] = err.Error()
	} else {
		result["model"] = path
		result["model_sha256"] = artifact.ModelSHA256
	}
	return printJSON(result)
}

func (s Service) Languages() error {
	return printJSON(map[string]any{"languages": []string{"eng"}, "default": "eng"})
}

func (s Service) OCR(input, language string) error {
	if _, err := s.resolver().Verify(); err != nil {
		return fmt.Errorf("unhealthy: %w", err)
	}
	result, err := ocr.Recognize(input, language)
	if err != nil {
		return err
	}
	return printJSON(result)
}

func printJSON(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
