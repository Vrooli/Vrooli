package models

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/capacity"
)

const (
	largePIIModelID       = "ab-ai/pii_model_based_on_distilbert"
	largePIIDiskOwner     = "resource-pii-detector:model-export"
	largePIIDiskFloor     = int64(120 << 30)
	largePIIHeartbeatTick = 15 * time.Second
	largePIIOutputSlack   = int64(512 << 20)
	largePIITempPeak      = int64(1 << 30)
)

type largeExportPath struct {
	Path string
	Role string
	Peak int64
}

type largeExportPlan struct {
	Request   capacity.DiskJobRequest
	Paths     []largeExportPath
	OutputDir string
	HubCache  string
	TempDir   string
}

type largeExportRunFunc func(context.Context, string, string, string, string, []string) error

// planLargePIIExport describes every path that the exporter may grow before
// it contacts Hugging Face or writes its output. It uses the pinned policy
// artifact size plus explicit export/temp margins; live path samples extend
// the reservation atomically if the observed physical growth exceeds them.
func planLargePIIExport(outDir string, policyBytes int64, getEnv func(string) string, home, jobID string) (largeExportPlan, error) {
	if policyBytes <= 0 || policyBytes > (math.MaxInt64-largePIIOutputSlack)/3 {
		return largeExportPlan{}, errors.New("large PII model has no safe positive disk estimate")
	}
	if strings.TrimSpace(jobID) == "" {
		return largeExportPlan{}, errors.New("large PII disk job identity is required")
	}
	if getEnv == nil {
		getEnv = os.Getenv
	}
	if strings.TrimSpace(home) == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return largeExportPlan{}, err
		}
	}
	output, err := canonicalFutureDirectory(outDir)
	if err != nil {
		return largeExportPlan{}, fmt.Errorf("resolve large PII output path: %w", err)
	}
	if _, err := os.Lstat(output); err == nil {
		return largeExportPlan{}, fmt.Errorf("large PII output directory already exists: %s", output)
	} else if !os.IsNotExist(err) {
		return largeExportPlan{}, fmt.Errorf("inspect large PII output directory: %w", err)
	}
	parentInfo, err := os.Stat(filepath.Dir(output))
	if err != nil || !parentInfo.IsDir() {
		return largeExportPlan{}, fmt.Errorf("large PII output parent must already exist as a directory: %s", filepath.Dir(output))
	}

	hfHome := strings.TrimSpace(getEnv("HF_HOME"))
	if hfHome == "" {
		hfHome = filepath.Join(home, ".cache", "huggingface")
	}
	hubCache := firstNonEmpty(getEnv("HF_HUB_CACHE"), getEnv("HUGGINGFACE_HUB_CACHE"), filepath.Join(hfHome, "hub"))
	hubCache, err = canonicalFutureDirectory(hubCache)
	if err != nil {
		return largeExportPlan{}, fmt.Errorf("resolve Hugging Face cache path: %w", err)
	}
	tempDir, err := canonicalFutureDirectory(firstNonEmpty(getEnv("TMPDIR"), os.TempDir()))
	if err != nil {
		return largeExportPlan{}, fmt.Errorf("resolve model exporter temporary path: %w", err)
	}
	if !filepath.IsAbs(output) || !filepath.IsAbs(hubCache) || !filepath.IsAbs(tempDir) {
		return largeExportPlan{}, errors.New("large PII export paths must resolve to absolute paths")
	}

	paths := []largeExportPath{
		{Path: output, Role: "pii-large-output-and-onnx-build", Peak: 2*policyBytes + largePIIOutputSlack},
		{Path: hubCache, Role: "pii-large-huggingface-weight-cache", Peak: policyBytes},
		{Path: tempDir, Role: "pii-large-export-temporary-files", Peak: largePIITempPeak},
	}
	paths = mergeLargeExportPaths(paths)
	requestPaths := make([]capacity.DiskJobPath, 0, len(paths))
	for _, path := range paths {
		requestPaths = append(requestPaths, capacity.DiskJobPath{Path: path.Path, Role: path.Role, PeakBytes: path.Peak})
	}
	return largeExportPlan{
		Request: capacity.DiskJobRequest{OwnerID: largePIIDiskOwner, JobID: jobID, Paths: requestPaths, FloorBytes: largePIIDiskFloor},
		Paths: paths, OutputDir: output, HubCache: hubCache, TempDir: tempDir,
	}, nil
}

func mergeLargeExportPaths(paths []largeExportPath) []largeExportPath {
	byPath := make(map[string]largeExportPath, len(paths))
	for _, path := range paths {
		if prior, ok := byPath[path.Path]; ok {
			prior.Peak += path.Peak
			if !strings.Contains(prior.Role, path.Role) {
				prior.Role += "," + path.Role
			}
			byPath[path.Path] = prior
		} else {
			byPath[path.Path] = path
		}
	}
	out := make([]largeExportPath, 0, len(byPath))
	for _, path := range byPath {
		out = append(out, path)
	}
	return out
}

func canonicalFutureDirectory(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("empty path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	candidate := filepath.Clean(abs)
	var suffix []string
	for {
		info, statErr := os.Stat(candidate)
		if statErr == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("path %s is not a directory", candidate)
			}
			resolved, err := filepath.EvalSymlinks(candidate)
			if err != nil {
				return "", err
			}
			parts := append([]string{resolved}, suffix...)
			return filepath.Clean(filepath.Join(parts...)), nil
		}
		if !os.IsNotExist(statErr) {
			return "", statErr
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return "", fmt.Errorf("path %s has no existing parent", path)
		}
		suffix = append([]string{filepath.Base(candidate)}, suffix...)
		candidate = parent
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

// runLargePIIExport holds the shared lease for the whole Python export,
// samples allocated path bytes, heartbeats while the child is running, and
// completes only after the child exits and final path samples succeed.
func runLargePIIExport(ctx context.Context, plan largeExportPlan, python, script string, admitter capacity.DiskJobAdmitter, run largeExportRunFunc) (capacity.DiskJobReceipt, error) {
	if admitter == nil {
		return capacity.DiskJobReceipt{}, errors.New("shared capacity disk admission is unavailable")
	}
	if run == nil {
		return capacity.DiskJobReceipt{}, errors.New("large PII exporter runner is unavailable")
	}
	if python == "" || script == "" {
		return capacity.DiskJobReceipt{}, errors.New("Python executable and export script are required")
	}
	if _, err := os.Stat(script); err != nil {
		return capacity.DiskJobReceipt{}, fmt.Errorf("inspect large PII export script: %w", err)
	}
	baseline := make(map[string]int64, len(plan.Paths))
	for _, path := range plan.Paths {
		bytes, err := allocatedDirectoryBytes(path.Path)
		if err != nil {
			return capacity.DiskJobReceipt{}, fmt.Errorf("sample %s before capacity admission: %w", path.Path, err)
		}
		baseline[path.Path] = bytes
	}
	lease, err := admitter.ReserveDiskJob(ctx, plan.Request)
	if err != nil {
		return capacity.DiskJobReceipt{}, fmt.Errorf("reserve complete PII model-export peak before write: %w", err)
	}
	last := make(map[string]int64, len(baseline))
	for path, bytes := range baseline {
		last[path] = bytes
		if _, err := lease.SamplePath(ctx, path, bytes, 0); err != nil {
			return capacity.DiskJobReceipt{}, fmt.Errorf("sample PII exporter baseline; lease %s remains active: %w", lease.Reservation().ID, err)
		}
	}
	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	childDone := make(chan error, 1)
	env := []string{
		"HF_HOME=" + filepath.Dir(plan.HubCache),
		"HF_HUB_CACHE=" + plan.HubCache,
		"HUGGINGFACE_HUB_CACHE=" + plan.HubCache,
		"TRANSFORMERS_CACHE=" + plan.HubCache,
		"TMPDIR=" + plan.TempDir,
		"PII_DETECTOR_MODEL_EXPORT_ADMITTED=true",
	}
	if err := os.Mkdir(plan.OutputDir, 0o700); err != nil {
		return capacity.DiskJobReceipt{}, fmt.Errorf("create new owner-controlled PII output directory after admission; lease %s remains active: %w", lease.Reservation().ID, err)
	}
	outputIdentity, err := os.Lstat(plan.OutputDir)
	if err != nil {
		return capacity.DiskJobReceipt{}, fmt.Errorf("identify new PII output directory; lease %s remains active: %w", lease.Reservation().ID, err)
	}
	go func() { childDone <- run(workCtx, python, script, plan.OutputDir, largePIIModelID, env) }()
	ticker := time.NewTicker(largePIIHeartbeatTick)
	defer ticker.Stop()
	var childErr error
	var ownerFailure error
	for {
		select {
		case childErr = <-childDone:
			goto childFinished
		case <-ctx.Done():
			cancel()
			childErr = <-childDone
			ownerFailure = fmt.Errorf("PII model export cancelled: %w", ctx.Err())
			goto childFinished
		case <-ticker.C:
			if err := sampleLargeExportPaths(workCtx, lease, plan.Paths, baseline, last); err != nil {
				cancel()
				_ = <-childDone
				return capacity.DiskJobReceipt{}, fmt.Errorf("sample PII model export; lease %s remains active for owner recovery: %w", lease.Reservation().ID, err)
			}
			if err := lease.Heartbeat(workCtx); err != nil {
				cancel()
				_ = <-childDone
				return capacity.DiskJobReceipt{}, fmt.Errorf("heartbeat PII model export; lease %s remains active for owner recovery: %w", lease.Reservation().ID, err)
			}
		}
	}

childFinished:
	if childErr == nil {
		if err := validateLargePIIExportOutput(plan.OutputDir); err != nil {
			childErr = err
		}
	}
	if childErr != nil {
		if err := removeOwnedExportOutput(plan.OutputDir, outputIdentity); err != nil {
			return capacity.DiskJobReceipt{}, fmt.Errorf("remove failed PII export output; lease %s remains active for owner recovery: %w (export failure: %v)", lease.Reservation().ID, err, childErr)
		}
	}
	if err := sampleLargeExportPaths(context.WithoutCancel(ctx), lease, plan.Paths, baseline, last); err != nil {
		return capacity.DiskJobReceipt{}, fmt.Errorf("final PII exporter path sample; lease %s remains active: %w", lease.Reservation().ID, err)
	}
	receipt, err := lease.Complete(context.WithoutCancel(ctx))
	if err != nil {
		return capacity.DiskJobReceipt{}, fmt.Errorf("complete PII model export disk job %s: %w", lease.Reservation().ID, err)
	}
	if ownerFailure != nil {
		return receipt, fmt.Errorf("PII model export stopped, its new output was cleaned, and retained cache writes were sampled (disk job %s): %w", receipt.ID, ownerFailure)
	}
	if childErr != nil {
		return receipt, fmt.Errorf("PII large model export failed after its writes stopped and were sampled (disk job %s): %w", receipt.ID, childErr)
	}
	return receipt, nil
}

func removeOwnedExportOutput(path string, identity os.FileInfo) error {
	current, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if identity == nil || !os.SameFile(identity, current) || current.Mode()&os.ModeSymlink != 0 || !current.IsDir() {
		return errors.New("PII output directory identity changed; retaining it for owner review")
	}
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		if err == nil {
			return errors.New("PII export output remains after owner cleanup")
		}
		return err
	}
	return nil
}

func validateLargePIIExportOutput(root string) error {
	for _, required := range []string{"model.onnx", "config.json", "vocab.txt", "tokenizer_config.json"} {
		info, err := os.Stat(filepath.Join(root, required))
		if err != nil {
			return fmt.Errorf("large PII exporter did not produce %s: %w", required, err)
		}
		if !info.Mode().IsRegular() || info.Size() <= 0 {
			return fmt.Errorf("large PII export artifact %s is not a nonempty regular file", required)
		}
	}
	return nil
}

func sampleLargeExportPaths(ctx context.Context, lease capacity.DiskJobLease, paths []largeExportPath, baseline, last map[string]int64) error {
	for _, path := range paths {
		bytes, err := allocatedDirectoryBytes(path.Path)
		if err != nil {
			return fmt.Errorf("measure %s: %w", path.Path, err)
		}
		if bytes < last[path.Path] {
			bytes = last[path.Path]
		}
		last[path.Path] = bytes
		delta := bytes - baseline[path.Path]
		largerPeak := int64(0)
		if delta > path.Peak {
			largerPeak = delta
		}
		if _, err := lease.SamplePath(ctx, path.Path, bytes, largerPeak); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handlers) ExportLarge(args []string) error {
	fs := flag.NewFlagSet("models export-large", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	outDir := fs.String("out", "", "new output directory for the staged large model")
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || strings.TrimSpace(*outDir) == "" {
		return errors.New("models export-large requires --out <new-directory> and no positional arguments")
	}
	policy, err := Load(policyPath(h.GetEnv))
	if err != nil {
		return err
	}
	model, ok := policy.Models[largePIIModelID]
	if !ok || model.DiskBytes <= 0 {
		return fmt.Errorf("pinned model policy has no positive disk estimate for %s", largePIIModelID)
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return fmt.Errorf("create large PII disk-job identity: %w", err)
	}
	plan, err := planLargePIIExport(*outDir, model.DiskBytes, h.GetEnv, "", hex.EncodeToString(nonce[:]))
	if err != nil {
		return err
	}
	admitter := h.DiskAdmission
	var closeBroker func() error
	if admitter == nil {
		broker, err := capacity.NewBroker(context.Background(), "")
		if err != nil {
			return fmt.Errorf("open shared capacity broker: %w", err)
		}
		admitter, closeBroker = broker, broker.Close
	}
	if closeBroker != nil {
		defer closeBroker()
	}
	sourceRoot := firstNonEmpty(h.GetEnv("VROOLI_SOURCE_ROOT"), h.GetEnv("VROOLI_CLI_SOURCE_ROOT"), h.GetEnv("RESOURCE_ROOT"))
	if sourceRoot == "" {
		return errors.New("Vrooli source root is unavailable; refusing unowned PII export")
	}
	script := largePIIExportScriptPath(sourceRoot)
	python := "python3"
	if resolved, err := exec.LookPath(python); err == nil {
		python = resolved
	} else {
		return fmt.Errorf("locate python3 for large PII export: %w", err)
	}
	runner := h.RunExporter
	if runner == nil {
		runner = runLargePIIExporter
	}
	ctx := context.Background()
	receipt, err := runLargePIIExport(ctx, plan, python, script, admitter, runner)
	if err != nil {
		return err
	}
	physicalBytes, err := allocatedDirectoryBytes(plan.OutputDir)
	if err != nil {
		return fmt.Errorf("measure completed large PII export output: %w", err)
	}
	artifacts, err := inspectExportArtifacts(plan.OutputDir)
	if err != nil {
		return fmt.Errorf("hash completed large PII export artifacts: %w", err)
	}
	result := map[string]any{"model": largePIIModelID, "output_dir": plan.OutputDir, "disk_job_receipt": receipt, "physical_bytes": physicalBytes, "artifacts": artifacts}
	return writeResult(h.Stdout, *jsonOutput, result, func() {
		fmt.Fprintf(h.Stdout, "exported %s to %s\ndisk job: %s (%s)\n", largePIIModelID, plan.OutputDir, receipt.ID, receipt.Status)
	})
}

func largePIIExportScriptPath(root string) string {
	for _, candidate := range []string{
		filepath.Join(root, "resources", "pii-detector", "scripts", "export_large_model.py"),
		filepath.Join(root, "scripts", "export_large_model.py"),
	} {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate
		}
	}
	return filepath.Join(root, "resources", "pii-detector", "scripts", "export_large_model.py")
}

func runLargePIIExporter(ctx context.Context, python, script, outDir, repo string, env []string) error {
	cmd := exec.CommandContext(ctx, python, script, "--out", outDir, "--repo", repo)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.Env = mergeEnvironment(os.Environ(), env)
	return cmd.Run()
}

func mergeEnvironment(base, overrides []string) []string {
	values := make(map[string]string, len(base)+len(overrides))
	for _, item := range base {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			values[key] = value
		}
	}
	for _, item := range overrides {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			values[key] = value
		}
	}
	out := make([]string, 0, len(values))
	for key, value := range values {
		out = append(out, key+"="+value)
	}
	return out
}

func inspectExportArtifacts(root string) ([]map[string]any, error) {
	var out []map[string]any
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			_ = f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		out = append(out, map[string]any{"path": path, "logical_bytes": info.Size(), "sha256": hex.EncodeToString(h.Sum(nil))})
		return nil
	})
	return out, err
}
