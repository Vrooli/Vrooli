package buildinfo

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const harnessStateContractAsset = "packages/cli-core/agentcatalog/harness/builtin.json"

type harnessStateCompatibilityDescriptor struct {
	Resource      string   `json:"resource"`
	ID            string   `json:"id"`
	SessionEnv    []string `json:"session_env,omitempty"`
	HomeIsolation struct {
		Kind        string `json:"kind"`
		Env         string `json:"env,omitempty"`
		SharedDir   string `json:"shared_dir,omitempty"`
		StateDir    string `json:"state_dir,omitempty"`
		SessionsEnv string `json:"sessions_env,omitempty"`
		Layout      string `json:"layout,omitempty"`
	} `json:"home_isolation"`
	Features struct {
		ContextWindowSource string `json:"context_window_source,omitempty"`
	} `json:"features"`
}

// harnessStateContractAssetDigest covers the embedded descriptor fields that
// select persisted harness state readers and isolated-home layouts. Display
// metadata and other launch behavior do not force a state-contract change.
func harnessStateContractAssetDigest(root string) (string, error) {
	path := filepath.Join(root, filepath.FromSlash(harnessStateContractAsset))
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	const maxAssetBytes = 1 << 20
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxAssetBytes {
		return "", fmt.Errorf("%s is not a bounded regular file", harnessStateContractAsset)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var descriptors []harnessStateCompatibilityDescriptor
	if err := json.Unmarshal(data, &descriptors); err != nil {
		return "", fmt.Errorf("decode %s: %w", harnessStateContractAsset, err)
	}
	if len(descriptors) == 0 {
		return "", fmt.Errorf("%s contains no descriptors", harnessStateContractAsset)
	}
	for i, descriptor := range descriptors {
		if descriptor.Resource == "" || descriptor.ID == "" || descriptor.HomeIsolation.Kind == "" {
			return "", fmt.Errorf("%s descriptor %d lacks state-compatibility identity", harnessStateContractAsset, i)
		}
	}
	sort.Slice(descriptors, func(i, j int) bool {
		if descriptors[i].Resource == descriptors[j].Resource {
			return descriptors[i].ID < descriptors[j].ID
		}
		return descriptors[i].Resource < descriptors[j].Resource
	})
	canonical, err := json.Marshal(descriptors)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return fmt.Sprintf("%x", digest[:16]), nil
}
