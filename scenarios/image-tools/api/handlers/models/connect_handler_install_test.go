package models

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	apidb "github.com/vrooli/api-core/database"

	"image-tools/internal/capabilities"
	internaljobs "image-tools/internal/jobs"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/smoke"

	db "github.com/vrooli/api-core/databasetest"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// fakeSubmitter records install-job submissions without running them.
type fakeSubmitter struct {
	last      internaljobs.Spec
	submitted int
}

const handlerInstallTestModelID = "handler-install-test-model"

func (f *fakeSubmitter) Submit(_ context.Context, spec internaljobs.Spec) (internaljobs.Job, error) {
	f.submitted++
	f.last = spec
	return internaljobs.Job{ID: "job-1", Operation: spec.Operation}, nil
}

// newInstallHandler builds a handler with a full Installer (fake downloader +
// generous fake disk) and a fake job submitter.
func newInstallHandler(t *testing.T) (*connectHandler, *internalmodels.Installer, *fakeSubmitter) {
	t.Helper()
	reg, err := internalmodels.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(internalmodels.Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	installer := &internalmodels.Installer{
		Root:   t.TempDir(),
		Reg:    reg,
		Custom: internalmodels.NewCustomStore(d),
		State:  internalmodels.NewInstallStore(d),
		Download: downloaderTo(func(dest string) error {
			// Above the installer's 1 KiB artifact-validation floor and not HTML,
			// so it passes validation (the bytes stand in for a real weight).
			return os.WriteFile(dest, bytes.Repeat([]byte("weights\n"), 256), 0o644)
		}),
		DiskAvail: func(string) (int64, error) { return int64(100) << 30, nil },
	}
	if err := installer.AddCustom(context.Background(), internalmodels.Model{
		ID:           handlerInstallTestModelID,
		Name:         "Handler Install Test Model",
		Operations:   []string{"upscale"},
		Tier:         internalmodels.TierNiceToHave,
		Backend:      "onnxruntime",
		SizeMBApprox: 16,
		Enabled:      true,
		Hardware:     internalmodels.Hardware{CPUCapable: true},
		CapabilityLabels: internalmodels.CapabilityLabels{
			License:       "MIT",
			CommercialUse: internalmodels.CommercialUseYes,
		},
		Source: internalmodels.Source{DownloadURL: "https://example.test/handler-install-test-model.bin"},
	}); err != nil {
		t.Fatalf("add handler install test model: %v", err)
	}
	sub := &fakeSubmitter{}
	h := NewConnectHandler(Deps{
		Registry:   reg,
		Store:      internalmodels.NewStore(d),
		Probe:      capabilities.FakeProbe{Host: cpuOnlyHost},
		Installer:  installer,
		Jobs:       sub,
		OpDefaults: internalmodels.NewOpDefaultStore(d),
	})
	return h, installer, sub
}

type installListQueryCounter struct {
	internalmodels.SQLExecutor
	listReads int
	rowReads  int
}

func (c *installListQueryCounter) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if strings.Contains(query, "FROM model_install") {
		c.listReads++
	}
	return c.SQLExecutor.QueryContext(ctx, query, args...)
}

func (c *installListQueryCounter) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if strings.Contains(query, "FROM model_install") {
		c.rowReads++
	}
	return c.SQLExecutor.QueryRowContext(ctx, query, args...)
}

func TestInstallStatesProjectsBuildReadinessFromListedRecord(t *testing.T) {
	h, installer, _ := newInstallHandler(t)
	var buildModel internalmodels.Model
	for _, model := range h.deps.Registry.Models() {
		if model.BuildProfile != nil {
			buildModel = model
			break
		}
	}
	if buildModel.ID == "" {
		t.Fatal("fixture catalog has no build-profile model")
	}

	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(internalmodels.Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	counter := &installListQueryCounter{SQLExecutor: d}
	installer.State = internalmodels.NewInstallStore(counter)
	if err := installer.State.Set(context.Background(), internalmodels.InstallRecord{
		ID: buildModel.ID, Installed: true, Path: t.TempDir(),
	}); err != nil {
		t.Fatalf("persist listed build assets: %v", err)
	}

	states, err := h.installStates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	record, found := states[buildModel.ID]
	if !found || !record.Installed {
		t.Fatalf("bulk install state = %+v found=%t, want raw durable source record", record, found)
	}
	publicRecord, publicFound := h.publicInstallState(buildModel, record, found)
	if !publicFound || publicRecord.Installed {
		t.Fatalf("public install state = %+v found=%t, want build output to gate readiness", publicRecord, publicFound)
	}
	if counter.listReads != 1 || counter.rowReads != 0 {
		t.Fatalf("install state reads = list:%d row:%d, want one bulk list and no repeated per-model lookup", counter.listReads, counter.rowReads)
	}
}

func TestListOperationModelsReusesBulkInstallRecords(t *testing.T) {
	h, installer, _ := newInstallHandler(t)
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(internalmodels.Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	counter := &installListQueryCounter{SQLExecutor: d}
	installer.State = internalmodels.NewInstallStore(counter)

	resp, err := h.ListOperationModels(context.Background(), connect.NewRequest(&modelsv1.ListOperationModelsRequest{Operation: "text_to_image"}))
	if err != nil {
		t.Fatalf("list operation models: %v", err)
	}
	if len(resp.Msg.GetCandidates()) == 0 {
		t.Fatal("expected text_to_image candidates")
	}
	if counter.listReads != 1 || counter.rowReads != 0 {
		t.Fatalf("install record reads = list:%d row:%d, want one bulk read and no per-candidate install lookups", counter.listReads, counter.rowReads)
	}
}

func TestSetAndListDefaults(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	ctx := context.Background()

	// Pin a non-seed-default model that serves text_to_image (sd-1.5 serves it).
	if _, err := h.SetDefaultModel(ctx, connect.NewRequest(&modelsv1.SetDefaultModelRequest{
		Operation: "text_to_image", ModelId: "sd-1.5",
	})); err != nil {
		t.Fatalf("set default: %v", err)
	}
	resp, err := h.ListDefaults(ctx, connect.NewRequest(&modelsv1.ListDefaultsRequest{}))
	if err != nil {
		t.Fatalf("list defaults: %v", err)
	}
	var found *modelsv1.OpDefault
	for _, d := range resp.Msg.Defaults {
		if d.Operation == "text_to_image" {
			found = d
		}
	}
	if found == nil || found.ModelId != "sd-1.5" || found.Source != "override" {
		t.Fatalf("expected text_to_image override sd-1.5, got %+v", found)
	}
}

func TestSetDefaultRejectsModelNotServingOp(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	// tesseract serves ocr, not text_to_image.
	_, err := h.SetDefaultModel(context.Background(), connect.NewRequest(&modelsv1.SetDefaultModelRequest{
		Operation: "text_to_image", ModelId: "tesseract",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("got %v, want InvalidArgument", connect.CodeOf(err))
	}
}

type downloaderTo func(dest string) error

func (fn downloaderTo) Download(_ context.Context, _, dest string, _ func(int64, int64)) error {
	return fn(dest)
}

func TestInstallModelSubmitsJob(t *testing.T) {
	h, _, sub := newInstallHandler(t)
	resp, err := h.InstallModel(context.Background(), connect.NewRequest(&modelsv1.InstallModelRequest{Id: handlerInstallTestModelID}))
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if resp.Msg.AlreadyInstalled {
		t.Fatal("fresh model should not report already installed")
	}
	if resp.Msg.JobId == "" {
		t.Fatal("expected a job id")
	}
	if sub.submitted != 1 || sub.last.Operation != internalmodels.InstallJobOperation {
		t.Fatalf("expected one install-job submit, got %d op=%q", sub.submitted, sub.last.Operation)
	}
}

func TestInstallModelUsesConfiguredETA(t *testing.T) {
	h, _, sub := newInstallHandler(t)
	h.deps.EstimateInstallSeconds = func(sizeMBApprox int) int {
		if sizeMBApprox != 16 {
			t.Fatalf("ETA estimator saw size %d, want 16", sizeMBApprox)
		}
		return 123
	}

	resp, err := h.InstallModel(context.Background(), connect.NewRequest(&modelsv1.InstallModelRequest{Id: handlerInstallTestModelID}))
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if resp.Msg.EtaSeconds != 123 {
		t.Fatalf("response eta = %d, want 123", resp.Msg.EtaSeconds)
	}
	if sub.last.EstimatedSeconds != 123 {
		t.Fatalf("submitted eta = %d, want 123", sub.last.EstimatedSeconds)
	}
}

func TestInstallModelAlreadyInstalledSkipsJob(t *testing.T) {
	h, installer, sub := newInstallHandler(t)
	ctx := context.Background()
	_, _, _, python := prepareSmokeTestModel(t, installer, handlerInstallTestModelID)
	installer.Smoke = &internalmodels.SmokeConfig{Python: python, LockHash: "ready-lock", Run: func(context.Context, string, []string, []string) ([]byte, error) {
		return []byte("load passed"), nil
	}}
	if _, err := installer.Install(ctx, handlerInstallTestModelID, nil); err != nil {
		t.Fatalf("pre-install: %v", err)
	}
	resp, err := h.InstallModel(ctx, connect.NewRequest(&modelsv1.InstallModelRequest{Id: handlerInstallTestModelID}))
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !resp.Msg.AlreadyInstalled {
		t.Fatal("expected already_installed=true")
	}
	if sub.submitted != 0 {
		t.Fatalf("no job should be submitted for an installed model, got %d", sub.submitted)
	}
}

func TestInstallModelRevalidatesInstalledSmokePendingModelWithoutDownload(t *testing.T) {
	h, installer, sub := newInstallHandler(t)
	ctx := context.Background()
	modelDir, weightPath, weight, python := prepareSmokeTestModel(t, installer, handlerInstallTestModelID)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatalf("create installed model dir: %v", err)
	}
	if err := os.WriteFile(weightPath, weight, 0o644); err != nil {
		t.Fatalf("write installed artifact: %v", err)
	}
	if err := installer.State.Set(ctx, internalmodels.InstallRecord{
		ID: handlerInstallTestModelID, Installed: true, Path: modelDir, Checksum: checksum(weight),
	}); err != nil {
		t.Fatalf("record installed model: %v", err)
	}
	smokeCalls := 0
	installer.Smoke = &internalmodels.SmokeConfig{
		Python: python, LockHash: "test-lock",
		Run: func(_ context.Context, gotPython string, args, _ []string) ([]byte, error) {
			smokeCalls++
			if gotPython != python {
				t.Fatalf("smoke interpreter = %q, want %q", gotPython, python)
			}
			if len(args) < 2 || args[0] != "-m" || args[1] != internalmodels.SmokeModule {
				t.Fatalf("smoke args = %v, want probe module invocation", args)
			}
			return []byte("onnx load smoke passed"), nil
		},
	}
	if ready, state := installer.RuntimeReadiness(ctx, handlerInstallTestModelID); ready || state != "smoke_pending" {
		t.Fatalf("initial readiness = %t/%q, want smoke_pending", ready, state)
	}

	response, err := h.InstallModel(ctx, connect.NewRequest(&modelsv1.InstallModelRequest{Id: handlerInstallTestModelID}))
	if err != nil {
		t.Fatalf("request smoke revalidation: %v", err)
	}
	if response.Msg.AlreadyInstalled || response.Msg.JobId == "" {
		t.Fatalf("pending smoke must schedule a durable revalidation job: %+v", response.Msg)
	}
	if sub.submitted != 1 || sub.last.Operation != internalmodels.InstallJobOperation {
		t.Fatalf("revalidation must use one install job, got submissions=%d operation=%q", sub.submitted, sub.last.Operation)
	}

	if _, err := installer.InstallForJob(ctx, handlerInstallTestModelID, response.Msg.JobId, nil); err != nil {
		t.Fatalf("run installed-model revalidation: %v", err)
	}
	if smokeCalls != 1 {
		t.Fatalf("smoke calls = %d, want 1", smokeCalls)
	}
	if ready, state := installer.RuntimeReadiness(ctx, handlerInstallTestModelID); !ready || state != "ready" {
		t.Fatalf("readiness after smoke = %t/%q, want ready", ready, state)
	}
	gotWeight, err := os.ReadFile(weightPath)
	if err != nil {
		t.Fatalf("read installed artifact: %v", err)
	}
	if !bytes.Equal(gotWeight, weight) {
		t.Fatal("revalidation changed the installed model artifact")
	}
}

func prepareSmokeTestModel(t *testing.T, installer *internalmodels.Installer, id string) (modelDir, weightPath string, weight []byte, python string) {
	t.Helper()
	model, err := installer.Resolve(context.Background(), id)
	if err != nil {
		t.Fatalf("resolve smoke test model: %v", err)
	}
	model.Source = internalmodels.Source{Assets: []internalmodels.Asset{{URL: "https://example.test/weights.onnx", Filename: "weights.onnx"}}}
	if err := installer.Custom.Add(context.Background(), model); err != nil {
		t.Fatalf("update smoke test model source: %v", err)
	}
	modelDir = filepath.Join(installer.Root, "models", id)
	weightPath = filepath.Join(modelDir, "weights.onnx")
	weight = bytes.Repeat([]byte("already-installed-onnx-artifact\n"), 128)
	python = filepath.Join(installer.Root, "venv", "bin", "python")
	if err := os.MkdirAll(filepath.Dir(python), 0o755); err != nil {
		t.Fatalf("create fake venv dir: %v", err)
	}
	if err := os.WriteFile(python, []byte("fake python"), 0o755); err != nil {
		t.Fatalf("create fake interpreter: %v", err)
	}
	return modelDir, weightPath, weight, python
}

func checksum(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func TestInstallModelDoesNotScheduleSmokeWhenRuntimeEnvironmentIsMissing(t *testing.T) {
	h, installer, sub := newInstallHandler(t)
	ctx := context.Background()
	const modelID = "isnet-general-use"
	modelDir := filepath.Join(installer.Root, "models", modelID)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatalf("create installed model dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, "weights.onnx"), []byte("weights"), 0o644); err != nil {
		t.Fatalf("write installed artifact: %v", err)
	}
	if err := installer.State.Set(ctx, internalmodels.InstallRecord{
		ID: modelID, Installed: true, Path: modelDir, Checksum: "model-checksum",
	}); err != nil {
		t.Fatalf("record installed model: %v", err)
	}
	installer.Smoke = &internalmodels.SmokeConfig{LockHash: "test-lock"}

	_, err := h.InstallModel(ctx, connect.NewRequest(&modelsv1.InstallModelRequest{Id: modelID}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("install error = %v, want FailedPrecondition", err)
	}
	if sub.submitted != 0 {
		t.Fatalf("must not schedule futile smoke job without a Python environment, got %d", sub.submitted)
	}
}

func TestInstallModelAlreadyInstalledSmokeReadySkipsJob(t *testing.T) {
	h, installer, sub := newInstallHandler(t)
	ctx := context.Background()
	const modelID = "isnet-general-use"
	modelDir := filepath.Join(installer.Root, "models", modelID)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatalf("create installed model dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, "weights.onnx"), []byte("weights"), 0o644); err != nil {
		t.Fatalf("write installed artifact: %v", err)
	}
	if err := installer.State.Set(ctx, internalmodels.InstallRecord{
		ID: modelID, Installed: true, Path: modelDir, Checksum: "model-checksum",
	}); err != nil {
		t.Fatalf("record installed model: %v", err)
	}
	python := filepath.Join(installer.Root, "venv", "bin", "python")
	if err := os.MkdirAll(filepath.Dir(python), 0o755); err != nil {
		t.Fatalf("create fake venv dir: %v", err)
	}
	if err := os.WriteFile(python, []byte("fake python"), 0o755); err != nil {
		t.Fatalf("create fake interpreter: %v", err)
	}
	if err := smoke.WriteVerdict(modelDir, smoke.Verdict{
		Pass: true, ModelHash: "model-checksum", LockHash: "test-lock", Kind: "onnx",
	}); err != nil {
		t.Fatalf("write cached smoke verdict: %v", err)
	}
	installer.Smoke = &internalmodels.SmokeConfig{Python: python, LockHash: "test-lock"}

	response, err := h.InstallModel(ctx, connect.NewRequest(&modelsv1.InstallModelRequest{Id: modelID}))
	if err != nil {
		t.Fatalf("install ready model: %v", err)
	}
	if !response.Msg.AlreadyInstalled {
		t.Fatal("runtime-ready installed model should remain already_installed")
	}
	if sub.submitted != 0 {
		t.Fatalf("runtime-ready model must not schedule a revalidation job, got %d", sub.submitted)
	}
}

func TestInstallModelBlocksConditionalBuildDespiteStoredAssets(t *testing.T) {
	h, installer, sub := newInstallHandler(t)
	const modelID = "big-lama-1024-export"
	modelDir := filepath.Join(installer.Root, "models", modelID)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := installer.State.Set(context.Background(), internalmodels.InstallRecord{
		ID: modelID, Installed: true, Path: modelDir, Checksum: "pinned-assets",
	}); err != nil {
		t.Fatal(err)
	}

	_, err := h.InstallModel(context.Background(), connect.NewRequest(&modelsv1.InstallModelRequest{Id: modelID}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("conditional install error = %v, want FailedPrecondition", err)
	}
	if sub.submitted != 0 {
		t.Fatalf("conditional model submitted %d build/install jobs", sub.submitted)
	}
	// Existing assets remain on disk, but they do not bypass the rights gate.
	stored, ok, err := installer.State.Get(context.Background(), modelID)
	if err != nil || !ok || !stored.Installed {
		t.Fatalf("stored source asset record should remain durable: record=%+v ok=%v err=%v", stored, ok, err)
	}
	got, err := h.GetModel(context.Background(), connect.NewRequest(&modelsv1.GetModelRequest{Id: modelID}))
	if err != nil {
		t.Fatalf("GetModel: %v", err)
	}
	if got.Msg.GetModel().GetInstall().GetInstalled() {
		t.Fatal("GetModel must not report source-only build assets as an installed runtime artifact")
	}
	listed, err := h.ListModels(context.Background(), connect.NewRequest(&modelsv1.ListModelsRequest{}))
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	for _, model := range listed.Msg.GetModels() {
		if model.GetId() == modelID && model.GetInstall().GetInstalled() {
			t.Fatal("ListModels must not report source-only build assets as installed")
		}
	}
	picker, err := h.ListOperationModels(context.Background(), connect.NewRequest(&modelsv1.ListOperationModelsRequest{Operation: "object_removal"}))
	if err != nil {
		t.Fatalf("ListOperationModels: %v", err)
	}
	candidate := findCandidate(picker.Msg, modelID)
	if candidate == nil || candidate.GetModel().GetInstall().GetInstalled() || candidate.GetReadyState() == "ready" {
		t.Fatalf("picker must not present a source-only build candidate as installed or ready: candidate=%+v", candidate)
	}
}

func TestInstallModelUnknownNotFound(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	_, err := h.InstallModel(context.Background(), connect.NewRequest(&modelsv1.InstallModelRequest{Id: "nope"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("got %v, want NotFound", connect.CodeOf(err))
	}
}

func TestInstallModelCannotAcquireUnverifiedCommercialWeights(t *testing.T) {
	h, _, submitter := newInstallHandler(t)
	var conditionalID string
	for _, model := range h.deps.Registry.Models() {
		if model.CapabilityLabels.CommercialUse == internalmodels.CommercialUseConditional {
			conditionalID = model.ID
			break
		}
	}
	if conditionalID == "" {
		t.Fatal("seed should include a conditional-rights model for this regression")
	}
	_, err := h.InstallModel(context.Background(), connect.NewRequest(&modelsv1.InstallModelRequest{Id: conditionalID}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("install conditional model error = %v, want FailedPrecondition", err)
	}
	if submitter.submitted != 0 {
		t.Fatalf("conditional model submitted %d install jobs", submitter.submitted)
	}
}

func TestInstallStateSurfacesInListAndGet(t *testing.T) {
	h, installer, _ := newInstallHandler(t)
	ctx := context.Background()
	if _, err := installer.Install(ctx, handlerInstallTestModelID, nil); err != nil {
		t.Fatalf("install: %v", err)
	}
	got, err := h.GetModel(ctx, connect.NewRequest(&modelsv1.GetModelRequest{Id: handlerInstallTestModelID}))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Msg.Model.Install == nil || !got.Msg.Model.Install.Installed {
		t.Fatal("install state should report installed in GetModel")
	}
	if got.Msg.Model.Install.Checksum == "" {
		t.Fatal("installed model should carry a pinned checksum")
	}
}

func TestRemoveModelClearsInstall(t *testing.T) {
	h, installer, _ := newInstallHandler(t)
	ctx := context.Background()
	if _, err := installer.Install(ctx, handlerInstallTestModelID, nil); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := h.RemoveModel(ctx, connect.NewRequest(&modelsv1.RemoveModelRequest{Id: handlerInstallTestModelID})); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if installer.Installed(ctx, handlerInstallTestModelID) {
		t.Fatal("model should not be installed after remove")
	}
}

func TestRemoveModelReturnsFailedPreconditionWhileInUse(t *testing.T) {
	h, installer, _ := newInstallHandler(t)
	ctx := context.Background()
	installer.UseGate = internalmodels.NewModelUseGate(t.TempDir())
	if _, err := installer.Install(ctx, handlerInstallTestModelID, nil); err != nil {
		t.Fatalf("install: %v", err)
	}
	release, err := installer.UseGate.Acquire(handlerInstallTestModelID)
	if err != nil {
		t.Fatalf("acquire use lease: %v", err)
	}
	defer release()

	_, err = h.RemoveModel(ctx, connect.NewRequest(&modelsv1.RemoveModelRequest{Id: handlerInstallTestModelID}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("RemoveModel error = %v, want FailedPrecondition", err)
	}
	if !installer.Installed(ctx, handlerInstallTestModelID) {
		t.Fatal("RemoveModel deleted the model while it was in use")
	}
}

func TestAddCustomModelRoundTrip(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	ctx := context.Background()

	weights := filepath.Join(t.TempDir(), "weights.bin")
	if err := os.WriteFile(weights, []byte("w"), 0o644); err != nil {
		t.Fatalf("write weights: %v", err)
	}
	resp, err := h.AddCustomModel(ctx, connect.NewRequest(&modelsv1.AddCustomModelRequest{
		Model: &modelsv1.Model{
			Id:           "my-model",
			Name:         "My Model",
			Operations:   []string{"upscale"},
			Tier:         string(internalmodels.TierNiceToHave),
			Backend:      "onnxruntime",
			SizeMbApprox: 16,
			Hardware:     &modelsv1.Hardware{CpuCapable: true},
			CapabilityLabels: &modelsv1.CapabilityLabels{
				License:       "MIT",
				CommercialUse: modelsv1.CommercialUse_COMMERCIAL_USE_YES,
			},
		},
		LocalPath: weights,
	}))
	if err != nil {
		t.Fatalf("add custom: %v", err)
	}
	if !resp.Msg.Model.Custom {
		t.Fatal("custom entry should be flagged custom")
	}

	// It now appears in ListModels.
	all, err := h.ListModels(ctx, connect.NewRequest(&modelsv1.ListModelsRequest{Operation: "upscale"}))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, m := range all.Msg.Models {
		if m.Id == "my-model" {
			found = true
		}
	}
	if !found {
		t.Fatal("custom model not surfaced in ListModels")
	}
}

func TestSelectModelResolvesEnabledCustomOverride(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	resp, err := h.SelectModel(context.Background(), connect.NewRequest(&modelsv1.SelectModelRequest{
		Operation:  "upscale",
		OverrideId: handlerInstallTestModelID,
	}))
	if err != nil {
		t.Fatalf("select custom model: %v", err)
	}
	if resp.Msg.GetModel().GetId() != handlerInstallTestModelID {
		t.Fatalf("selected model = %q, want %q", resp.Msg.GetModel().GetId(), handlerInstallTestModelID)
	}
	if !resp.Msg.GetModel().GetCustom() || !resp.Msg.GetModel().GetEnabled() {
		t.Fatalf("custom model metadata = custom:%t enabled:%t, want true/true", resp.Msg.GetModel().GetCustom(), resp.Msg.GetModel().GetEnabled())
	}
}

func TestListOperationModelsIncludesAndSelectsCustomCandidate(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	resp, err := h.ListOperationModels(context.Background(), connect.NewRequest(&modelsv1.ListOperationModelsRequest{
		Operation: "upscale",
	}))
	if err != nil {
		t.Fatalf("list custom candidates: %v", err)
	}
	if resp.Msg.GetSelectedId() != handlerInstallTestModelID {
		t.Fatalf("selected id = %q, want custom model %q", resp.Msg.GetSelectedId(), handlerInstallTestModelID)
	}
	for _, candidate := range resp.Msg.GetCandidates() {
		if candidate.GetModel().GetId() != handlerInstallTestModelID {
			continue
		}
		if !candidate.GetModel().GetCustom() || !candidate.GetSelected() {
			t.Fatalf("custom candidate = %+v, want custom and selected", candidate)
		}
		return
	}
	t.Fatalf("custom model %q missing from upscale candidates", handlerInstallTestModelID)
}

func TestAddCustomModelSeedCollisionInvalid(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	_, err := h.AddCustomModel(context.Background(), connect.NewRequest(&modelsv1.AddCustomModelRequest{
		Model: &modelsv1.Model{Id: "sd-1.5", Operations: []string{"text_to_image"}, Backend: "x"},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("got %v, want InvalidArgument", connect.CodeOf(err))
	}
}
