package hostwireless

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqkit/grub"
	mt7927source "github.com/vrooli/vrooli/internal/tools/mt7927-source"
)

const maxArtifactBytes = 64 << 20

var bundleID = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

// FileIdentity records a file with measured bytes, or a reason it could not be measured.
type FileIdentity struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256,omitempty"`
	Issue  string `json:"issue,omitempty"`
}

// ActivationPlan is a read-only preview. A prepared bundle contains unsigned modules.
type ActivationPlan struct {
	Profile             string                        `json:"profile"`
	KernelRelease       string                        `json:"kernel_release,omitempty"`
	PreparationState    string                        `json:"preparation_state"`
	PreparationIssue    string                        `json:"preparation_issue,omitempty"`
	Bundle              string                        `json:"bundle,omitempty"`
	BundleReceiptSHA256 string                        `json:"bundle_receipt_sha256,omitempty"`
	KernelSymbolsSHA256 string                        `json:"prepared_kernel_symbols_sha256,omitempty"`
	Candidates          []FileIdentity                `json:"unsigned_candidates,omitempty"`
	Modaliases          []string                      `json:"modaliases,omitempty"`
	Originals           []FileIdentity                `json:"stock_modules,omitempty"`
	Preserved           []FileIdentity                `json:"preserved_files,omitempty"`
	RecoveryCandidates  []string                      `json:"recovery_candidates,omitempty"`
	RecoveryRelease     string                        `json:"recovery_release,omitempty"`
	WiredInterfaces     []string                      `json:"wired_default_route_interfaces,omitempty"`
	WiredDrivers        map[string]string             `json:"wired_drivers,omitempty"`
	NVIDIALoadedModules []string                      `json:"nvidia_loaded_modules,omitempty"`
	SecureBoot          hostinventory.SecureBootState `json:"secure_boot"`
	OverlayDirectory    string                        `json:"overlay_directory,omitempty"`
	ActivationEligible  bool                          `json:"activation_eligible"`
	Prerequisites       []string                      `json:"prerequisites"`
}

// PreviewActivation checks observed host state and an optional canonical bundle.
func PreviewActivation(ctx context.Context, profile, generation, recoveryRelease string) (ActivationPlan, error) {
	plan := ActivationPlan{Profile: profile, Bundle: generation}
	spec, ok := preparationProfiles[profile]
	if !ok {
		return plan, fmt.Errorf("unknown wireless preparation profile %q", profile)
	}
	if generation != "" && !bundleID.MatchString(generation) {
		return plan, errors.New("bundle must be a preparation generation ID")
	}
	if recoveryRelease != "" && (!nativeName.MatchString(recoveryRelease) || recoveryRelease == nobleABI) {
		return plan, errors.New("recovery kernel must identify a distinct installed release")
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	var inventory hostinventory.HostInventory
	prepared, err := prepare(ctx, profile, false, &inventory)
	if err != nil {
		return plan, err
	}
	plan.KernelRelease = prepared.KernelRelease
	plan.Modaliases = slices.Clone(prepared.Modaliases)
	plan.PreparationState, plan.PreparationIssue = prepared.State, prepared.Reason
	plan.SecureBoot = inventory.SecureBoot
	plan.Prerequisites = slices.Clone(spec.openItems)
	plan.Prerequisites = append(plan.Prerequisites,
		"Verify an independent, selectable stock boot entry and a console recovery path",
		"Link exact signed module bytes to the reviewed unsigned bundle and verify the signing key is trusted by the running kernel",
		"Qualify the live privileged overlay and index transaction before any activation",
	)
	if prepared.State != "eligible" {
		return plan, nil
	}
	plan.OverlayDirectory = overlayDirectory(profile)
	observeActivationHost(ctx, &plan, spec, inventory)
	if recoveryRelease != "" {
		if !slices.Contains(plan.RecoveryCandidates, recoveryRelease) {
			return plan, errors.New("recovery kernel image, initramfs or module tree unavailable")
		}
		plan.RecoveryRelease = recoveryRelease
		plan.Preserved = append(plan.Preserved,
			observeFile(ctx, "/boot/vmlinuz-"+recoveryRelease),
			observeFile(ctx, "/boot/initrd.img-"+recoveryRelease),
		)
	}
	if generation == "" {
		plan.Prerequisites = append(plan.Prerequisites, "Prepare and identify the exact unsigned bundle")
		return plan, nil
	}
	plan.Candidates, plan.BundleReceiptSHA256, err = verifyPreparedBundle(ctx, prepared, spec, generation)
	if err == nil {
		directory := filepath.Join(filepath.Dir(filepath.Dir(prepared.SourceDirectory)), "host-wireless", profile, generation)
		plan.KernelSymbolsSHA256, err = preparedKernelSymbols(directory)
		if err == nil && plan.KernelSymbolsSHA256 != preservedSHA(plan, "/"+abiTree+"/build/Module.symvers") {
			err = errors.New("prepared kernel symbol identity differs from the observed headers")
		}
	}
	return plan, err
}

func preparedKernelSymbols(directory string) (string, error) {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return "", err
	}
	defer root.Close()
	data, err := readSmallRegular(bundleFiles{root}, "kernel-inputs.sha256", 4096)
	if err != nil {
		return "", err
	}
	var hash string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == "/"+abiTree+"/build/Module.symvers" {
			if hash != "" || !validSHA(fields[0]) {
				return "", errors.New("ambiguous prepared kernel symbol identity")
			}
			hash = fields[0]
		}
	}
	if hash == "" {
		return "", errors.New("prepared kernel symbol identity unavailable")
	}
	return hash, nil
}

func overlayDirectory(profile string) string {
	return filepath.ToSlash(filepath.Join("lib", "modules", nobleABI, "updates", "vrooli-"+strings.TrimPrefix(profile, "noble-")))
}

func verifyPreparedBundle(ctx context.Context, expected Receipt, spec buildProfile, generation string) ([]FileIdentity, string, error) {
	if !bundleID.MatchString(generation) {
		return nil, "", errors.New("invalid preparation generation ID")
	}
	artifactRoot := filepath.Dir(filepath.Dir(expected.SourceDirectory))
	profileRoot := filepath.Join("host-wireless", expected.Profile)
	generationRoot := filepath.Join(profileRoot, generation)
	root, err := os.OpenRoot(artifactRoot)
	if err != nil {
		return nil, "", err
	}
	defer root.Close()
	for _, directory := range []string{"host-wireless", profileRoot, generationRoot} {
		info, err := root.Lstat(directory)
		if err != nil || !info.IsDir() {
			return nil, "", errors.New("preparation bundle directory unavailable")
		}
	}
	bundle, err := root.OpenRoot(generationRoot)
	if err != nil {
		return nil, "", err
	}
	defer bundle.Close()
	info, err := bundle.Lstat("receipt.json")
	if err != nil || !info.Mode().IsRegular() {
		return nil, "", errors.New("preparation receipt is not a regular file")
	}
	filesystem := bundleFiles{bundle}
	data, err := readSmallRegular(filesystem, "receipt.json", 64<<10)
	if err != nil {
		return nil, "", fmt.Errorf("read preparation receipt: %w", err)
	}
	var r Receipt
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, "", err
	}
	sourceSHA, err := binaryfetch.FileDigestContext(ctx, filepath.Join(expected.SourceDirectory, mt7927source.Manifest))
	if err != nil {
		return nil, "", err
	}
	if r.SchemaVersion != 1 || r.State != preparedArtifactState || r.Profile != expected.Profile || r.KernelRelease != expected.KernelRelease ||
		r.SourceDirectory != expected.SourceDirectory || r.SourceSHA256 != sourceSHA || r.PatchSHA256 != expected.PatchSHA256 ||
		r.BuildRecipeSHA256 != expected.BuildRecipeSHA256 || r.ArtifactDirectory != filepath.Join(artifactRoot, generationRoot) ||
		!r.CopiedTreeRollback || r.ActivationEligible || !slices.Equal(r.Modaliases, expected.Modaliases) ||
		r.StockABI.StockExports == 0 || r.StockABI.StockExports != r.CandidateABI.StockExports ||
		r.StockABI.Imports == 0 || r.CandidateABI.Imports == 0 || r.CandidateABI.NewExports != len(spec.newExports) {
		return nil, "", errors.New("preparation receipt does not match the observed host and reviewed profile")
	}
	entries, err := fs.ReadDir(bundle.FS(), ".")
	if err != nil {
		return nil, "", err
	}
	if len(entries) != len(r.ArtifactSHA256)+1 || len(entries) > 64 {
		return nil, "", errors.New("preparation bundle contains an unexpected set of files")
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			return nil, "", fmt.Errorf("unexpected bundle file type %s", entry.Name())
		}
		if entry.Name() != "receipt.json" {
			if err := verifyFile(ctx, filesystem, entry.Name(), r.ArtifactSHA256[entry.Name()]); err != nil {
				return nil, "", err
			}
		}
	}
	files := make([]FileIdentity, 0, len(spec.modules))
	for _, module := range spec.modules {
		name := module + ".ko"
		hash, ok := r.ArtifactSHA256[name]
		if !ok {
			return nil, "", fmt.Errorf("receipt lacks candidate %s", name)
		}
		files = append(files, FileIdentity{Path: name, SHA256: hash})
	}
	return files, digest(data), nil
}

type bundleFiles struct{ root *os.Root }

func (f bundleFiles) Open(name string) (fs.File, error) {
	return openBundleFile(f.root, name)
}

func bootFiles(release string) []string {
	return []string{
		"/boot/vmlinuz-" + release, "/boot/initrd.img-" + release,
		"/boot/grub/grub.cfg", "/boot/grub/grubenv", grub.DefaultConfigPath,
	}
}

func readSmallRegular(tree fs.FS, path string, maxBytes int64) ([]byte, error) {
	file, err := tree.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxBytes {
		return nil, fmt.Errorf("unexpected file type or size: %s", path)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err == nil && int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("file exceeds bound: %s", path)
	}
	return data, err
}
