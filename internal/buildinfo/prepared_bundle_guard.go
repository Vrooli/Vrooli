package buildinfo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

// PreparedBundleIntent is durable evidence of an unfinished bundle publication.
// It is not authority to publish a different root or manifest.
type PreparedBundleIntent struct {
	SourceRoot   string            `json:"source_root,omitempty"`
	SameContract bool              `json:"same_contract"`
	Version      int               `json:"version"`
	Manifest     string            `json:"manifest_sha256"`
	Root         string            `json:"root_sha256"`
	Prior        map[string]string `json:"prior_sha256"`
}

func PreparedBundlePaths(home string) (string, string, error) {
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", "", err
	}
	dir := filepath.Join(state, "root-rebaseline", "ordinary-bundle")
	return filepath.Join(dir, "intent.json"), filepath.Join(dir, "bundle"), nil
}

// WritePreparedBundleIntent records the v1 ordinary-bundle recovery protocol.
// Keeping path selection, validation and encoding here lets callers coordinate
// the transaction without owning the durable record format.
func WritePreparedBundleIntent(home string, intent PreparedBundleIntent) error {
	if err := validatePreparedBundleIntent(intent); err != nil {
		return err
	}
	path, _, err := PreparedBundlePaths(home)
	if err != nil {
		return err
	}
	data, err := json.Marshal(intent)
	if err != nil {
		return err
	}
	return config.WriteOwnedFileDurable(path, data, 0o600)
}

func ReadPreparedBundleIntent(home string) (*PreparedBundleIntent, error) {
	path, _, err := PreparedBundlePaths(home)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var intent PreparedBundleIntent
	if len(data) > 4096 || json.Unmarshal(data, &intent) != nil || intent.Version != 1 || !validPreparedBundleDigest(intent.Manifest) || !validPreparedBundleDigest(intent.Root) {
		return nil, fmt.Errorf("invalid ordinary bundle recovery intent at %s", path)
	}
	if intent.SourceRoot != "" && (!filepath.IsAbs(intent.SourceRoot) || filepath.Clean(intent.SourceRoot) != intent.SourceRoot) {
		return nil, fmt.Errorf("invalid source root in ordinary bundle recovery intent at %s", path)
	}
	return &intent, nil
}

func validatePreparedBundleIntent(intent PreparedBundleIntent) error {
	if intent.Version != 1 || !validPreparedBundleDigest(intent.Manifest) || !validPreparedBundleDigest(intent.Root) {
		return fmt.Errorf("invalid ordinary bundle recovery intent")
	}
	if intent.SourceRoot != "" && (!filepath.IsAbs(intent.SourceRoot) || filepath.Clean(intent.SourceRoot) != intent.SourceRoot) {
		return fmt.Errorf("invalid source root in ordinary bundle recovery intent")
	}
	return nil
}

func validPreparedBundleDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32 && hex.EncodeToString(decoded) == value
}

// GuardPreparedBundle blocks unrelated mutation until recovery completes. The
// retained root is accepted only by the activation admission owner, after a
// fresh digest check. Callers must hold a transition or root publication lock.
func GuardPreparedBundle(home, candidate string) error {
	intent, err := ReadPreparedBundleIntent(home)
	if err != nil || intent == nil {
		return err
	}
	_, dir, err := PreparedBundlePaths(home)
	if err != nil {
		return err
	}
	if candidate != "" && filepath.Clean(candidate) == filepath.Join(dir, "vrooli") {
		f, err := os.Open(candidate)
		if err != nil {
			return err
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		if hex.EncodeToString(h.Sum(nil)) == intent.Root {
			return nil
		}
	}
	bin, pathErr := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if pathErr != nil {
		return pathErr
	}
	return fmt.Errorf("an ordinary bundle install is incomplete; resume it with %q --resume-bundle %q, or, when the installed images still match the recorded prior state, discard it with %q --abort-bundle %q", filepath.Join(dir, "vrooli-atomic-install"), bin, filepath.Join(dir, "vrooli-atomic-install"), bin)
}
