// Package resources loads and validates the resolved resource deployment plan
// included in a desktop bundle. It deliberately does not re-resolve manifests:
// runtime executes the exact, signed selection admitted by the pipeline.
package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

type Plan struct {
	SchemaVersion string `json:"schema_version"`
	Resources     []Item `json:"resources"`
}
type Item struct {
	RequestedResource string     `json:"requested_resource"`
	Resource          string     `json:"resource"`
	OS                string     `json:"os"`
	Architecture      string     `json:"architecture"`
	Mode              string     `json:"mode"`
	Support           string     `json:"support"`
	Requires          []string   `json:"requires,omitempty"`
	Limitations       []string   `json:"limitations,omitempty"`
	Artifact          string     `json:"artifact,omitempty"`
	Files             []Artifact `json:"files,omitempty"`
}
type Artifact struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

// Load validates all items selected for this runtime host. Bundled clients are
// ready for scenario services to invoke from the bundle; services/controllers
// require an explicit future lifecycle command contract and are rejected rather
// than being guessed or launched with unsafe default arguments.
func Load(bundleRoot string) (*Plan, error) {
	path := filepath.Join(bundleRoot, "resource-deployment-plan.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Plan{SchemaVersion: "v2"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read resource deployment plan: %w", err)
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("parse resource deployment plan: %w", err)
	}
	if plan.SchemaVersion != "v2" {
		return nil, fmt.Errorf("unsupported resource deployment plan version %q", plan.SchemaVersion)
	}
	for _, item := range plan.Resources {
		if item.OS != runtimeOS() || item.Architecture != runtime.GOARCH {
			continue
		}
		if item.Support == "unsupported" {
			return nil, fmt.Errorf("resolved resource %s is unsupported on this host", item.RequestedResource)
		}
		if item.Mode != "bundled-client" && item.Mode != "bundled-service" && item.Mode != "bundled-controller" {
			continue
		}
		if item.Mode != "bundled-client" {
			return nil, fmt.Errorf("resource %s uses %s but this bundle has no lifecycle adapter", item.Resource, item.Mode)
		}
		if err := verifyClient(bundleRoot, item); err != nil {
			return nil, err
		}
	}
	return &plan, nil
}

func runtimeOS() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}

func verifyClient(bundleRoot string, item Item) error {
	if !resourcedeployment.IsSafeArtifactName(item.Artifact) {
		return fmt.Errorf("resource %s has unsafe artifact identity", item.Resource)
	}
	if len(item.Files) != 3 {
		return fmt.Errorf("resource %s must include binary, manifest, and build metadata", item.Resource)
	}
	for _, artifact := range item.Files {
		if !resourcedeployment.IsSafeArtifactName(artifact.Name) {
			return fmt.Errorf("resource %s has unsafe artifact file", item.Resource)
		}
		data, err := os.ReadFile(filepath.Join(bundleRoot, "resources", item.Resource, artifact.Name))
		if err != nil {
			return fmt.Errorf("read resource artifact %s: %w", artifact.Name, err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != artifact.SHA256 {
			return fmt.Errorf("resource artifact hash mismatch for %s", artifact.Name)
		}
	}
	var contract struct {
		Name string `json:"name"`
	}
	manifestData, err := os.ReadFile(filepath.Join(bundleRoot, "resources", item.Resource, item.Artifact+".manifest.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(manifestData, &contract); err != nil {
		return fmt.Errorf("parse resource contract for %s: %w", item.Resource, err)
	}
	if contract.Name != item.Resource {
		return fmt.Errorf("resource artifact contract mismatch: plan selects %s, contract names %s", item.Resource, contract.Name)
	}
	return nil
}
