package securestore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/config"
)

// StoreGeneration returns a stable, non-secret identifier for the current
// passphrase wrap. It deliberately hashes only public envelope material and
// never derives an identifier from the passphrase itself.
func StoreGeneration(path string) (string, error) {
	file, err := readSealedFile(path)
	if err != nil {
		return "", err
	}
	for _, wrap := range file.Wraps {
		if wrap.Provider != providerPassphrase {
			continue
		}
		generation, err := passphraseWrapGeneration(wrap)
		if err != nil {
			return "", err
		}
		// Keep a digest of the public wrap material alongside the counter. The
		// counter is what makes rotation observable; the digest distinguishes
		// legacy or manually-rewritten envelopes that happen to reuse it.
		h := sha256.New()
		_, _ = h.Write(wrap.Params)
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(wrap.Ciphertext))
		return fmt.Sprintf("%d:%s", generation, hex.EncodeToString(h.Sum(nil))), nil
	}
	return "", fmt.Errorf("credential store has no passphrase wrap")
}

func passphraseWrapGeneration(wrap wrappedKey) (uint64, error) {
	var params passphraseParams
	if len(wrap.Params) != 0 {
		if err := json.Unmarshal(wrap.Params, &params); err != nil {
			return 0, fmt.Errorf("%w: passphrase wrap has unreadable parameters", errSealedCorrupt)
		}
	}
	return normalizedPassphraseGeneration(params.Generation), nil
}

// CopyStore atomically copies an already encrypted store into sink. The sink
// is a directory and receives secrets.enc.json plus non-secret receipt
// metadata. repositoryPaths must contain every local repository root known to
// the control plane; a copy inside one is refused.
func CopyStore(source, sink, receiptPath string, repositoryPaths []string) (CopyStatus, error) {
	return CopyStoreWithPolicy(source, sink, receiptPath, CopyPolicy{RepositoryPaths: repositoryPaths})
}

// CopyStoreWithPolicy validates the complete sink policy before mutation and
// verifies the copied encrypted envelope before recording evidence. The
// compatibility CopyStore wrapper above preserves the older repository-only
// API for low-level callers; production onboarding supplies the stronger
// policy explicitly.
func CopyStoreWithPolicy(source, sink, receiptPath string, policy CopyPolicy) (CopyStatus, error) {
	source, err := filepath.Abs(filepath.Clean(source))
	if err != nil {
		return CopyStatus{}, fmt.Errorf("resolve credential store path: %w", err)
	}
	sink, err = filepath.Abs(filepath.Clean(sink))
	if err != nil {
		return CopyStatus{}, fmt.Errorf("resolve credential copy sink: %w", err)
	}
	for _, repository := range policy.RepositoryPaths {
		resolved, resolveErr := resolveExistingOrParent(repository)
		if resolveErr != nil || resolved == "" {
			continue
		}
		resolvedSink, sinkResolveErr := resolveExistingOrParent(sink)
		if sinkResolveErr != nil {
			return CopyStatus{}, fmt.Errorf("resolve credential copy sink for safety check: %w", sinkResolveErr)
		}
		rel, relErr := filepath.Rel(resolved, resolvedSink)
		if relErr == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return CopyStatus{}, &SinkConflictError{Sink: resolvedSink, Repository: resolved}
		}
	}
	if err := validateCopySink(source, sink, policy); err != nil {
		return CopyStatus{}, err
	}
	if _, err := readSealedFile(source); err != nil {
		return CopyStatus{}, fmt.Errorf("read encrypted credential store: %w", err)
	}
	generation, err := StoreGeneration(source)
	if err != nil {
		return CopyStatus{}, err
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return CopyStatus{}, fmt.Errorf("read encrypted credential store: %w", err)
	}
	if err := os.MkdirAll(sink, sealedDirPerm); err != nil {
		return CopyStatus{}, fmt.Errorf("create credential copy sink: %w", err)
	}
	if err := restrictCredentialDirectory(sink); err != nil {
		return CopyStatus{}, fmt.Errorf("restrict credential copy sink: %w", err)
	}
	destination := filepath.Join(sink, filepath.Base(source))
	if err := atomicCopy(destination, data); err != nil {
		return CopyStatus{}, err
	}
	verified, err := verifyCopiedStore(destination, data, generation)
	if err != nil {
		return CopyStatus{}, err
	}
	status := CopyStatus{Path: destination, Sink: sink, SinkIdentity: stableSinkIdentity(sink), CopiedAt: time.Now().UTC(), Generation: generation, Checksum: verified, VerifiedAt: time.Now().UTC(), Verification: "readback"}
	if err := writeCopyReceipt(receiptPath, status); err != nil {
		return CopyStatus{}, err
	}
	return status, nil
}

func validateCopySink(source, sink string, policy CopyPolicy) error {
	resolvedSource, err := resolveExistingOrParent(source)
	if err != nil {
		return fmt.Errorf("resolve encrypted credential store for sink policy: %w", err)
	}
	resolvedSink, err := resolveExistingOrParent(sink)
	if err != nil {
		return fmt.Errorf("resolve credential copy sink for policy: %w", err)
	}
	if (len(policy.ProtectedRoots) > 0 || policy.RequireIndependentDevice) && pathContainedBy(resolvedSink, filepath.Dir(resolvedSource)) {
		return &SinkConflictError{Sink: resolvedSink, Repository: filepath.Dir(resolvedSource)}
	}
	for _, root := range policy.ProtectedRoots {
		resolvedRoot, rootErr := resolveExistingOrParent(root)
		if rootErr != nil {
			return fmt.Errorf("resolve protected root %q: %w", root, rootErr)
		}
		if pathContainedBy(resolvedSink, resolvedRoot) {
			return &SinkConflictError{Sink: resolvedSink, Repository: resolvedRoot}
		}
	}
	if policy.RequireIndependentDevice {
		sourceDevice, sourceKnown := pathDeviceIdentity(filepath.Dir(resolvedSource))
		sinkDevice, sinkKnown := pathDeviceIdentity(resolvedSink)
		if !sinkKnown {
			sinkDevice, sinkKnown = pathDeviceIdentity(filepath.Dir(resolvedSink))
		}
		if !sourceKnown || !sinkKnown {
			return fmt.Errorf("credential copy sink physical independence is unknown")
		}
		if sourceDevice == sinkDevice {
			return fmt.Errorf("credential copy sink is on the same physical device as the encrypted store")
		}
	}
	return nil
}

func pathContainedBy(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func verifyCopiedStore(destination string, expected []byte, generationData string) (string, error) {
	data, err := os.ReadFile(destination)
	if err != nil {
		return "", fmt.Errorf("read back encrypted credential-store copy: %w", err)
	}
	if _, err := readSealedFile(destination); err != nil {
		return "", fmt.Errorf("verify encrypted credential-store copy: %w", err)
	}
	actual := sha256.Sum256(data)
	expectedSum := sha256.Sum256(expected)
	if actual != expectedSum {
		return "", fmt.Errorf("verify encrypted credential-store copy: checksum mismatch")
	}
	actualGeneration, err := StoreGeneration(destination)
	if err != nil {
		return "", fmt.Errorf("verify encrypted credential-store generation: %w", err)
	}
	if actualGeneration != generationData {
		return "", fmt.Errorf("verify encrypted credential-store generation: expected %s, got %s", generationData, actualGeneration)
	}
	return hex.EncodeToString(actual[:]), nil
}

func stableSinkIdentity(path string) string {
	resolved, err := resolveExistingOrParent(path)
	if err != nil || resolved == "" {
		resolved = filepath.Clean(path)
	}
	hash := sha256.Sum256([]byte(resolved))
	return hex.EncodeToString(hash[:])
}

// resolveExistingOrParent resolves symlinks for a path that may not exist yet.
// Checking only lexical absolute paths would allow a symlinked sink to land
// inside a repository after the containment check.
func resolveExistingOrParent(path string) (string, error) {
	path, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(resolved), nil
	}
	parent := filepath.Dir(path)
	if parent == path {
		return path, nil
	}
	resolvedParent, err := resolveExistingOrParent(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolvedParent, filepath.Base(path)), nil
}

func atomicCopy(destination string, data []byte) error {
	if err := config.WriteOwnedFileAtomic(destination, data, sealedFilePerm); err != nil {
		return fmt.Errorf("replace credential copy: %w", err)
	}
	if err := RestrictCredentialFile(destination); err != nil {
		return fmt.Errorf("restrict credential copy: %w", err)
	}
	return nil
}

func writeCopyReceipt(path string, status CopyStatus) error {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("encode credential copy receipt: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), sealedDirPerm); err != nil {
		return fmt.Errorf("create credential copy receipt directory: %w", err)
	}
	if err := atomicCopy(path, data); err != nil {
		return fmt.Errorf("write credential copy receipt: %w", err)
	}
	return nil
}

// WriteCopyReceipt republishes an already verified copy status after a caller
// adds non-secret schedule metadata to the same receipt.
func WriteCopyReceipt(path string, status CopyStatus) error { return writeCopyReceipt(path, status) }
