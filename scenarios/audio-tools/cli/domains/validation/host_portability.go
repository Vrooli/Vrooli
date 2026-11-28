package validation

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	experimentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment/experiment_v1connect"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models"
	modelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models/models_v1connect"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

// runHostPortability records inventory and owner-backed D1/D2 inference cells.
// Inventory is not inference evidence; unavailable cells remain explicit.
func (h *handlers) runHostPortability(ctx cliapp.RunContext) error {
	if h.core == nil {
		return fmt.Errorf("host portability validation requires the scenario CLI core")
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	sttClient := sttconnect.NewSTTServiceClient(httpClient, baseURL)
	modelsClient := modelsconnect.NewModelRegistryServiceClient(httpClient, baseURL)
	experimentClient := experimentconnect.NewExperimentServiceClient(httpClient, baseURL)
	if err := requireLabQualificationEvidenceAPI(context.Background(), experimentClient, "validation run-host-portability"); err != nil {
		return err
	}
	build, err := audioToolsBuildIdentity(httpClient, baseURL)
	if err != nil {
		return err
	}
	resp, err := sttClient.ListEngines(context.Background(), connect.NewRequest(&sttv1.ListEnginesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("host portability engine inventory", err, nil)
	}
	engines := make(map[string]*sttv1.EngineInfo, len(resp.Msg.GetEngines()))
	for _, engine := range resp.Msg.GetEngines() {
		if engine != nil {
			engines[engine.GetId()] = engine
		}
	}
	modelsResp, err := modelsClient.ListModels(context.Background(), connect.NewRequest(&modelsv1.ListModelsRequest{Capability: "stt"}))
	if err != nil {
		return cliapp.WrapAPIError("host portability model inventory", err, nil)
	}
	models := make(map[string]*modelsv1.ModelRecord, len(modelsResp.Msg.GetModels()))
	for _, model := range modelsResp.Msg.GetModels() {
		if model != nil {
			models[model.GetId()] = model
		}
	}
	modelResolution, err := collectModelResolutionReceipts(context.Background(), func(ctx context.Context, capability, engine string, platform *modelsv1.Platform) (*modelsv1.ModelRecord, string, error) {
		resp, resolveErr := modelsClient.ResolveModel(ctx, connect.NewRequest(&modelsv1.ResolveModelRequest{
			Capability: capability, Language: "en", Commercial: true, EngineId: engine, Platform: platform,
		}))
		if resolveErr != nil {
			return nil, "", resolveErr
		}
		if resp == nil || resp.Msg == nil || resp.Msg.GetModel() == nil {
			return nil, "", fmt.Errorf("resolver returned no model")
		}
		return resp.Msg.GetModel(), resp.Msg.GetExplanation(), nil
	})
	if err != nil {
		return cliapp.WrapAPIError("record model-resolution observations", err, nil)
	}
	d1CPU := runD1WhisperCPUInference(context.Background(), h)
	d2 := runD2SherpaInference(context.Background(), h)
	if engine := engines["sherpa-streaming"]; engine != nil {
		d2["route_available"] = engine.GetAvailable()
	} else {
		d2["route_available"] = false
	}
	memoryReceipts, err := recordHostMemoryEvidence(context.Background(), experimentClient, models, d1CPU, d2, h.Now().UTC())
	if err != nil {
		return cliapp.WrapAPIError("record model memory evidence", err, nil)
	}
	details := hostPortabilityDetails(engines, d1CPU, d2)
	details["model_memory_evidence"] = memoryReceipts
	details["model_resolution_receipts"] = modelResolution
	details["product_path_capture"] = hostPortabilityProductPathDiagnostic()
	missing := []string{}
	if d1CPU["inference"] != "passed" {
		missing = append(missing, "D1_CPU_inference_not_run")
	}
	if engines["kyutai"] == nil || !engines["kyutai"].GetAvailable() {
		missing = append(missing, "D1_accelerator_explicit_unavailable_and_product_fallback_evidence_missing")
	} else {
		missing = append(missing, "D1_accelerator_inference_not_run")
	}
	machine, err := json.Marshal(map[string]any{
		"target_revision": acceptanceTargetRevision, "method": "audio-tools-acceptance-v2",
		"cohort": "voice-pilot-en-v1", "build": build, "device_profile": "D1",
		"policy_revision": "policy-v1", "corpus_revision": "voice-pilot-en-v1",
		"execution_lane": "D1 host profile with bounded CPU container; D2 remains separately measured", "provider_cell": map[string]string{"engine_id": "whisper-local", "model_id": "whisper-small", "strategy": "vad_segment", "device_profile": "D1"},
		"acceptance": map[string]any{"host-portability": map[string]any{"in_band": false, "negative_controls_passed": true, "required_unavailable": missing}},
		"details":    details,
	})
	if err != nil {
		return fmt.Errorf("encode host portability receipt: %w", err)
	}
	artifact := fmt.Sprintf("audio-tools:host-portability:%s:%s", h.Now().UTC().Format("20060102T150405.000000000Z"), strings.ToLower(runtime.GOARCH))
	if _, err := experimentClient.RecordQualificationEvidence(context.Background(), connect.NewRequest(&experimentv1.RecordQualificationEvidenceRequest{Evidence: &experimentv1.QualificationEvidence{
		EngineId: "whisper-local", ModelId: "whisper-small", Strategy: "vad_segment", PolicyProfile: "policy-v1",
		Kind:   experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_BROWSER_PRODUCT_PATH,
		Passed: false, ArtifactRef: artifact, Notes: "D1 CPU Whisper inference and D2 bounded Sherpa inference are attempted when prerequisites are present; D1 Kyutai remains unqualified without explicit device-assignment reason and same-artifact product-path fallback proof; whole row remains incomplete; D3-D6 not claimed", MachineJson: string(machine),
	}})); err != nil {
		return cliapp.WrapAPIError("record host portability evidence", err, nil)
	}
	if ctx.JSON() {
		return json.NewEncoder(ctx.Stdout()).Encode(map[string]any{"passed": false, "artifact_ref": artifact, "details": details})
	}
	fmt.Fprintf(ctx.Stdout(), "host portability qualification incomplete (inference not run for required cells): %s\n", artifact)
	return nil
}

// hostPortabilityProductPathDiagnostic explains why this inventory/inference
// producer does not manufacture the browser receipts required by the board.
// Those receipts must be captured through the owner-managed BAS path, where
// Agent Manager identity, BAS grant, and Test Genie lease are verified before
// a fresh browser session is opened.
func hostPortabilityProductPathDiagnostic() map[string]any {
	return map[string]any{
		"state":               "not_attempted_by_host_inventory_producer",
		"reason":              "host inference and engine inventory do not establish visible Dictation Studio product-path behavior or browser provenance",
		"required_owner_path": "Audio Tools managed browser qualification through validation.soak and the BAS owner API",
		"required_prerequisites": []string{
			"request-scoped Agent Manager identity token",
			"BAS browser grant receipt",
			"active Test Genie demand-lease receipt verified for target audio-tools",
			"configured Test Genie lease verifier and BAS Agent Access endpoint",
		},
		"credential_boundary":         "credentials are supplied to the managed Audio Tools validation request; they are not CLI flags, environment fallbacks, or reusable local grants",
		"capture_contract":            "fresh BAS session, Audio Tools-selected Dictation Studio target and fixed fixture, visible transcript/fallback plus browser and host provenance",
		"registered_authorized_route": "test-genie execute audio-tools --phases soak",
		"registered_phase_scope":      "the current soak phase proves an accelerated virtual-replay cell; it does not prove the local D1 CPU or same-artifact D1 Sherpa fallback required by host portability",
		"remaining_owner_work":        "Audio Tools must add a managed D1 product-path cell that binds the local CPU host/browser provenance, visible fallback transcript, and exact inference artifact; current inventory producer cannot capture BAS sessions or receive owner credentials",
		"authorization_owner":         "Test Genie issues the demand lease; Agent Manager supplies request-scoped identity and browser grant; Audio Tools validation.soak verifies the lease and BAS uses the browser grant",
	}
}

type modelResolver func(context.Context, string, string, *modelsv1.Platform) (*modelsv1.ModelRecord, string, error)

// collectModelResolutionReceipts records policy decisions for the mandated
// D1 CPU and D2 constrained profiles. These are resolution observations only:
// quality, inference capacity and promotion remain governed by their own
// receipts, and no candidate status is changed here.
func collectModelResolutionReceipts(ctx context.Context, resolve modelResolver) (map[string]any, error) {
	profiles := []struct {
		id        string
		platform  *modelsv1.Platform
		sttEngine string
	}{
		{id: "D1-linux-amd64-cpu", platform: &modelsv1.Platform{Os: "linux", Arch: "amd64", Accelerated: false}, sttEngine: "whisper-local"},
		{id: "D2-linux-amd64-constrained", platform: &modelsv1.Platform{Os: "linux", Arch: "amd64", Accelerated: false, RamGb: 4, CpuLimit: 2}, sttEngine: "sherpa-streaming"},
	}
	capabilities := []string{"stt", "tts", "summarize", "speaker"}
	out := map[string]any{"schema_version": 1, "selection_policy": "commercial declared default; language=en; STT engine scoped per profile", "profile_semantics": "resolution against declared D1/D2 target constraints; this observation does not prove that the host or browser executed the selected model", "quality_evidence": "not established by resolution; join the matching corpus-quality receipt separately", "capacity_evidence": "OS/architecture, accelerator, VRAM and RAM constraints are evaluated by the resolver; CPU quota is recorded, but model-specific CPU minima are unknown and therefore not enforced; inference and measured memory remain separate evidence"}
	for _, profile := range profiles {
		rows := make([]map[string]any, 0, len(capabilities))
		for _, capability := range capabilities {
			engine := ""
			if capability == "stt" {
				engine = profile.sttEngine
			}
			model, explanation, err := resolve(ctx, capability, engine, profile.platform)
			row := map[string]any{"capability": capability, "engine_id": engine, "state": "unavailable"}
			cpuLimit := profile.platform.GetCpuLimit()
			cpuLimitState := "unknown"
			if cpuLimit > 0 {
				cpuLimitState = "recorded"
			}
			cpuEnvelope := map[string]any{"limit_cores": cpuLimit, "limit_state": cpuLimitState, "model_minimum_cores": "unknown", "qualification": "not_proven"}
			row["cpu_capacity"] = cpuEnvelope
			if err != nil {
				row["reason"] = err.Error()
			} else if model == nil {
				row["reason"] = "resolver returned no model"
			} else {
				row["state"] = "resolved"
				row["model_id"] = model.GetId()
				row["readiness"] = model.GetReadiness().String()
				row["commercial_use"] = model.GetCommercialUse().String()
				row["license"] = model.GetLicense()
				row["declared_size_mb"] = model.GetSizeMbApprox()
				row["explanation"] = explanation
			}
			rows = append(rows, row)
		}
		out[profile.id] = map[string]any{"profile": profile.id, "browser": "Chromium", "accelerator": "CPU only", "platform_constraints": profile.platform, "cpu_limit": int(profile.platform.GetCpuLimit()), "observations": rows}
	}
	return out, nil
}

// hostPortabilityDetails keeps host inventory facts separate from inference
// evidence. A listed/available engine says nothing about whether it ran an
// inference successfully for this qualification.
func hostPortabilityDetails(engines map[string]*sttv1.EngineInfo, d1CPU, d2 map[string]any) map[string]any {
	inventory := make(map[string]any, len(engines))
	for id, engine := range engines {
		if engine == nil {
			continue
		}
		inventory[id] = map[string]any{"available": engine.GetAvailable(), "verdict": engine.GetVerdict(), "reason": engine.GetReason()}
	}
	if d2 == nil {
		d2 = d2Unavailable("D2 runner did not return a result")
	}
	if d1CPU == nil {
		d1CPU = d1CPUUnavailable("D1 CPU runner did not return a result")
	}
	return map[string]any{
		"os": runtime.GOOS, "arch": runtime.GOARCH, "cpu_count": runtime.NumCPU(),
		"engine_inventory":   inventory,
		"d1_cpu":             d1CPU,
		"d1_accelerator":     d1AcceleratorStatus(engines["kyutai"], engines["sherpa-streaming"]),
		"d2":                 d2,
		"unclaimed_profiles": []string{"D3", "D4", "D5", "D6"},
	}
}

// d1AcceleratorStatus keeps a missing accelerator route actionable without
// treating its inventory verdict or an alternate engine as qualification.
func d1AcceleratorStatus(engine, fallback *sttv1.EngineInfo) map[string]any {
	result := map[string]any{"engine_id": "kyutai", "model_id": "kyutai/stt-1b-en_fr", "state": "unqualified", "inference": "not_run"}
	if engine == nil {
		result["reason"] = "Kyutai engine is absent from the live engine inventory"
	} else {
		result["available"] = engine.GetAvailable()
		result["verdict"] = engine.GetVerdict()
		result["reason"] = engine.GetReason()
	}
	if fallback != nil {
		result["fallback_candidate"] = fallback.GetId()
		result["fallback_available"] = fallback.GetAvailable()
		result["fallback_reason"] = fallback.GetReason()
	}
	return result
}

func d1CPUUnavailable(reason string) map[string]any {
	return map[string]any{
		"engine_id": "whisper-local", "model_id": "whisper-small", "state": "unavailable", "inference": "not_run",
		"selected_route": "whisper.cpp-cpu", "reason": reason,
		"profile": map[string]any{"os": runtime.GOOS, "arch": runtime.GOARCH, "host_cpu_count": runtime.NumCPU(), "accelerator": "none", "network": "none"},
	}
}

// runD1WhisperCPUInference exercises the mandate's declared whisper-local CPU
// cell with staged local weights and a digest-pinned runtime. The container is
// isolated from network/GPU access and bounded so the smoke cannot monopolize
// the host. This measures inference, not the managed whisper HTTP service or
// browser microphone path; those remain separate evidence.
func runD1WhisperCPUInference(parent context.Context, h *handlers) map[string]any {
	unavailable := func(reason string) map[string]any { return d1CPUUnavailable(reason) }
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return unavailable("pinned CPU runner is qualified only for the observed linux/amd64 host")
	}
	root, err := h.scenarioRoot()
	if err != nil {
		return unavailable("cannot locate audio-tools source tree: " + err.Error())
	}
	wav := filepath.Join(root, "bas", "fixtures", "dictation-reference.wav")
	modelRoot := strings.TrimSpace(h.getenv("WHISPER_MODELS_DIR"))
	if modelRoot == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return unavailable("cannot resolve home directory for staged Whisper model: " + homeErr.Error())
		}
		modelRoot = filepath.Join(home, ".local", "share", "vrooli", "resources", "whisper", "models")
	}
	model := filepath.Join(modelRoot, "ggml-small.bin")
	for _, path := range []string{wav, model} {
		if _, err := os.Stat(path); err != nil {
			return unavailable("required staged D1 CPU input is missing: " + path + ": " + err.Error())
		}
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return unavailable("docker executable is unavailable for the pinned whisper.cpp CPU runner")
	}
	probeCtx, probeCancel := context.WithTimeout(parent, 10*time.Second)
	defer probeCancel()
	if err := exec.CommandContext(probeCtx, "docker", "image", "inspect", whisperCppImage).Run(); err != nil {
		return unavailable("cached digest-pinned whisper.cpp image is unavailable (no pull attempted)")
	}
	const timeout = 3 * time.Minute
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	args := []string{
		"run", "--pull=never", "--rm", "--network=none", "--cpus=8", "--memory=8g", "--memory-swap=8g", "--pids-limit=128", "--read-only",
		"--mount", "type=bind,src=" + model + ",dst=/models/ggml-small.bin,readonly",
		"--mount", "type=bind,src=" + wav + ",dst=/fixtures/dictation-reference.wav,readonly",
		"--entrypoint", "/bin/bash", whisperCppImage,
		"-lc", "ulimit -c 0; /app/build/bin/whisper-cli -m /models/ggml-small.bin -f /fixtures/dictation-reference.wav --no-gpu -t 8 -nt -np; result=$?; if [ -r /sys/fs/cgroup/memory.peak ]; then printf '\\nD1_MEMORY_PEAK_BYTES='; cat /sys/fs/cgroup/memory.peak; fi; exit $result",
	}
	started := time.Now()
	output, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	duration := time.Since(started)
	if err != nil {
		return unavailable(fmt.Sprintf("bounded D1 CPU Whisper inference failed (%v; output_sha256=%s)", err, outputSHA256(output)))
	}
	if !strings.Contains(string(output), "ggml_vulkan: No devices found.") {
		return unavailable("D1 CPU runner did not confirm the accelerator-free runtime path (output_sha256=" + outputSHA256(output) + ")")
	}
	transcript := strings.TrimSpace(stripMarkedLine(string(output), "D1_MEMORY_PEAK_BYTES="))
	if index := strings.LastIndex(transcript, "\n"); index >= 0 {
		transcript = strings.TrimSpace(transcript[index+1:])
	}
	if !d1TranscriptEvidence(transcript) {
		return unavailable("D1 CPU inference did not match the expected fixture phrase (output_sha256=" + outputSHA256(output) + ")")
	}
	peakMemory, peakErr := parseCgroupV2MemoryPeak(string(output), "D1_MEMORY_PEAK_BYTES=")
	artifacts := map[string]string{}
	for _, path := range []string{wav, model} {
		digest, size, hashErr := sha256File(path)
		if hashErr != nil {
			return unavailable("D1 CPU succeeded but artifact receipt could not hash " + filepath.Base(path) + ": " + hashErr.Error())
		}
		artifacts[filepath.Base(path)] = fmt.Sprintf("sha256:%s,size_bytes:%d", digest, size)
	}
	return map[string]any{
		"engine_id": "whisper-local", "model_id": "whisper-small", "state": "available", "inference": "passed",
		"selected_route": "whisper.cpp-cpu", "execution_lane": "digest-pinned-container-cpu",
		"container_image": whisperCppImage, "test": "whisper-cli --no-gpu -t 8",
		"profile":   map[string]any{"os": runtime.GOOS, "arch": runtime.GOARCH, "host_cpu_count": runtime.NumCPU(), "container_cpu_limit": 8, "container_memory_limit_bytes": int64(8 * 1024 * 1024 * 1024), "accelerator": "none", "network": "none", "timeout_seconds": int(timeout.Seconds())},
		"artifacts": artifacts, "duration_ms": duration.Milliseconds(), "transcript_match": d1TranscriptEvidence(transcript),
		"peak_memory_bytes": peakMemory, "peak_memory_measured": peakErr == nil,
		"peak_memory_source":       "cgroup-v2-memory.peak",
		"memory_measurement_error": errorText(peakErr),
		"host_serviceable":         false, "limitation": "inference is proven in the pinned CPU runner; the managed whisper HTTP service is currently unserviceable",
	}
}

const whisperCppImage = "ghcr.io/ggml-org/whisper.cpp@sha256:4ebde8b234bbca79f304ffd482cbb252c55f8eb0bce18c5d502adcc78047419e"

var d2SherpaInt8ModelFiles = []string{
	"encoder-epoch-99-avg-1-chunk-16-left-128.int8.onnx",
	"decoder-epoch-99-avg-1-chunk-16-left-128.int8.onnx",
	"joiner-epoch-99-avg-1-chunk-16-left-128.int8.onnx",
	"tokens.txt",
}

const (
	d2ExpectedText        = "The quick brown, fox jumps."
	d2ExpectedFinalSuffix = "fox jumps."
	defaultD2SherpaRoot   = "/tmp/sherpa-root"
)

// d2SherpaSmokeExpectationArgs keeps producer expectations aligned with the
// server smoke's required full-text and final-suffix assertions.
func d2SherpaSmokeExpectationArgs() []string {
	return []string{
		"--env", "SHERPA_ONNX_STREAMING_EXPECT_TEXT=" + d2ExpectedText,
		"--env", "SHERPA_ONNX_STREAMING_EXPECT_FINAL_SUFFIX=" + d2ExpectedFinalSuffix,
	}
}

func d2Unavailable(reason string) map[string]any {
	return map[string]any{
		"state": "unavailable", "inference": "not_run", "selected_route": "sherpa-streaming-zipformer-en-2023-06-26",
		"quantization": "int8", "model_files": map[string]string{"encoder": d2SherpaInt8ModelFiles[0], "decoder": d2SherpaInt8ModelFiles[1], "joiner": d2SherpaInt8ModelFiles[2], "tokens": d2SherpaInt8ModelFiles[3]},
		"profile": map[string]any{"cpu_count": 2, "memory_limit_bytes": int64(4 * 1024 * 1024 * 1024), "memory_swap_limit_bytes": int64(4 * 1024 * 1024 * 1024), "accelerator": "none", "network": "none"},
		"reason":  reason,
	}
}

func d2SherpaModelPath(name string) string {
	return filepath.Join("/models/asr", name)
}

// runD2SherpaInference builds the existing opt-in native streaming smoke test
// outside the repository and runs it inside a cached, digest-pinned image with
// strict CPU, memory, PID, network, and timeout limits. Every input is mounted
// read-only; Docker cannot pull because --pull=never is explicit.
func runD2SherpaInference(parent context.Context, h *handlers) map[string]any {
	unavailable := func(reason string) map[string]any { return d2Unavailable(reason) }
	root, err := h.scenarioRoot()
	if err != nil {
		return unavailable("cannot locate audio-tools source tree: " + err.Error())
	}
	repoRoot := filepath.Dir(filepath.Dir(root))
	serverDir := filepath.Join(repoRoot, "resources", "sherpa-onnx", "server")
	runtimeRoot := d2SherpaRuntimeRoot(h.getenv)
	modelRoot := strings.TrimSpace(h.getenv("SHERPA_ONNX_MODEL_ROOT"))
	if modelRoot == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return unavailable("cannot resolve home directory for staged Sherpa model artifacts: " + homeErr.Error())
		}
		modelRoot = filepath.Join(home, ".local", "share", "vrooli", "resources", "sherpa-onnx", "models")
	}
	modelDir := filepath.Join(modelRoot, "asr", "sherpa-onnx-streaming-zipformer-en-2023-06-26")
	punctDir := filepath.Join(modelRoot, "punctuation", "sherpa-onnx-punct-ct-transformer-zh-en-vocab272727-2024-04-12-int8")
	wav := filepath.Join(root, "bas", "fixtures", "dictation-reference.wav")
	for _, path := range []string{serverDir, filepath.Join(runtimeRoot, "include"), filepath.Join(runtimeRoot, "lib"), modelDir, punctDir, wav} {
		if _, err := os.Stat(path); err != nil {
			return unavailable("required staged D2 input is missing: " + path + ": " + err.Error())
		}
	}
	for _, name := range d2SherpaInt8ModelFiles {
		if _, err := os.Stat(filepath.Join(modelDir, name)); err != nil {
			return unavailable("required Zipformer artifact is missing: " + filepath.Join(modelDir, name))
		}
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return unavailable("docker executable is unavailable")
	}
	probeCtx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	if err := exec.CommandContext(probeCtx, "docker", "image", "inspect", whisperCppImage).Run(); err != nil {
		return unavailable("cached digest-pinned D2 image is unavailable (no pull attempted)")
	}
	if err := exec.CommandContext(probeCtx, "go", "version").Run(); err != nil {
		return unavailable("Go toolchain is unavailable for building the native smoke test")
	}
	buildDir, err := os.MkdirTemp("", "audio-tools-d2-build-")
	if err != nil {
		return unavailable("create temporary D2 test build directory: " + err.Error())
	}
	defer os.RemoveAll(buildDir)
	testBinary := filepath.Join(buildDir, "sherpa-d2.test")
	buildCtx, buildCancel := context.WithTimeout(parent, 2*time.Minute)
	defer buildCancel()
	build := exec.CommandContext(buildCtx, "go", "test", "-c", "-tags", "sherpa_onnx", "-o", testBinary, ".")
	build.Dir = serverDir
	build.Env = append(os.Environ(), "GOWORK=off", "CGO_CFLAGS=-I"+filepath.Join(runtimeRoot, "include"), "CGO_LDFLAGS=-L"+filepath.Join(runtimeRoot, "lib"))
	if output, err := build.CombinedOutput(); err != nil {
		return unavailable(fmt.Sprintf("native Sherpa smoke-test build failed (%v; output_sha256=%s)", err, outputSHA256(output)))
	}
	if _, err := os.Stat(testBinary); err != nil {
		return unavailable("native Sherpa test binary was not produced: " + err.Error())
	}
	containerTimeout := 3 * time.Minute
	ctx, cancelRun := context.WithTimeout(parent, containerTimeout)
	defer cancelRun()
	args := []string{
		"run", "--pull=never", "--rm", "--network=none", "--cpus=2", "--memory=4g", "--memory-swap=4g", "--pids-limit=256", "--read-only",
		"--mount", "type=bind,src=" + testBinary + ",dst=/runner/sherpa-d2.test,readonly",
		"--mount", "type=bind,src=" + filepath.Join(runtimeRoot, "lib") + ",dst=/runtime/lib,readonly",
		"--mount", "type=bind,src=" + modelDir + ",dst=/models/asr,readonly",
		"--mount", "type=bind,src=" + punctDir + ",dst=/models/punctuation,readonly",
		"--mount", "type=bind,src=" + wav + ",dst=/fixtures/dictation-reference.wav,readonly",
		"--env", "LD_LIBRARY_PATH=/runtime/lib", "--env", "SHERPA_ONNX_THREADS=2", "--env", "SHERPA_ONNX_STREAMING_MODEL_DIR=/models/asr",
		"--env", "SHERPA_ONNX_STREAMING_ENCODER=" + d2SherpaModelPath(d2SherpaInt8ModelFiles[0]),
		"--env", "SHERPA_ONNX_STREAMING_DECODER=" + d2SherpaModelPath(d2SherpaInt8ModelFiles[1]),
		"--env", "SHERPA_ONNX_STREAMING_JOINER=" + d2SherpaModelPath(d2SherpaInt8ModelFiles[2]),
		"--env", "SHERPA_ONNX_STREAMING_TOKENS=" + d2SherpaModelPath(d2SherpaInt8ModelFiles[3]),
		"--env", "SHERPA_ONNX_PUNCTUATION_MODEL_DIR=/models/punctuation", "--env", "SHERPA_ONNX_STREAMING_TEST_WAV=/fixtures/dictation-reference.wav",
	}
	args = append(args, d2SherpaSmokeExpectationArgs()...)
	args = append(args, "--entrypoint", "/bin/bash", whisperCppImage,
		"-lc", "ulimit -c 0; /runner/sherpa-d2.test -test.v -test.run '^TestNativeStreamingModelSmoke$'; result=$?; if [ -r /sys/fs/cgroup/memory.peak ]; then printf 'D2_MEMORY_PEAK_BYTES='; cat /sys/fs/cgroup/memory.peak; fi; exit $result",
	)
	started := time.Now()
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	duration := time.Since(started)
	if err != nil {
		return unavailable(fmt.Sprintf("bounded D2 Sherpa inference failed (%v; output_sha256=%s)", err, outputSHA256(output)))
	}
	artifacts := map[string]string{}
	artifactPaths := append([]string{wav}, modelFilePaths(modelDir, d2SherpaInt8ModelFiles)...)
	artifactPaths = append(artifactPaths,
		filepath.Join(punctDir, "model.int8.onnx"),
		filepath.Join(runtimeRoot, "lib", "libsherpa-onnx-c-api.so"),
		filepath.Join(runtimeRoot, "lib", "libonnxruntime.so"),
		testBinary,
	)
	for _, path := range artifactPaths {
		digest, size, hashErr := sha256File(path)
		if hashErr != nil {
			return unavailable("D2 succeeded but artifact receipt could not hash " + path + ": " + hashErr.Error())
		}
		artifacts[filepath.Base(path)] = fmt.Sprintf("sha256:%s,size_bytes:%d", digest, size)
	}
	peakMemory := parseD2PeakMemory(string(output))
	return map[string]any{
		"state": "available", "inference": "passed", "selected_route": "sherpa-streaming-zipformer-en-2023-06-26",
		"profile":         map[string]any{"os": runtime.GOOS, "arch": runtime.GOARCH, "cpu_count": 2, "memory_limit_bytes": int64(4 * 1024 * 1024 * 1024), "memory_swap_limit_bytes": int64(4 * 1024 * 1024 * 1024), "accelerator": "none", "network": "none", "pids_limit": 256, "timeout_seconds": int(containerTimeout.Seconds())},
		"container_image": whisperCppImage, "test": "resources/sherpa-onnx/server:TestNativeStreamingModelSmoke", "build": "go test -c -tags sherpa_onnx",
		"model": "sherpa-onnx-streaming-zipformer-en-2023-06-26", "quantization": "int8",
		"model_files": map[string]string{"encoder": d2SherpaInt8ModelFiles[0], "decoder": d2SherpaInt8ModelFiles[1], "joiner": d2SherpaInt8ModelFiles[2], "tokens": d2SherpaInt8ModelFiles[3]},
		"artifacts":   artifacts,
		"duration_ms": duration.Milliseconds(), "smoke_test": d2SmokeEvidence(output),
		"peak_memory_bytes": peakMemory, "peak_memory_measured": peakMemory > 0,
	}
}

func d2SherpaRuntimeRoot(getenv func(string) string) string {
	if getenv != nil {
		if root := strings.TrimSpace(getenv("SHERPA_ONNX_ROOT")); root != "" {
			return root
		}
	}
	return defaultD2SherpaRoot
}

func parseD2PeakMemory(output string) int64 {
	value, err := parseCgroupV2MemoryPeak(output, "D2_MEMORY_PEAK_BYTES=")
	if err != nil {
		return 0
	}
	return value
}

func stripMarkedLine(output, marker string) string {
	lines := strings.Split(output, "\n")
	kept := lines[:0]
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), marker) {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func modelFilePaths(root string, files []string) []string {
	paths := make([]string, 0, len(files))
	for _, name := range files {
		paths = append(paths, filepath.Join(root, name))
	}
	return paths
}

func sha256File(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), n, nil
}

// outputSHA256 gives operators a stable correlation value for failed native
// runner output without persisting transcript-bearing stdout/stderr.
func outputSHA256(output []byte) string {
	digest := sha256.Sum256(output)
	return fmt.Sprintf("%x", digest)
}

// d1TranscriptEvidence records only the result of the fixture phrase check.
// The transcript itself is deliberately confined to the local comparison.
func d1TranscriptEvidence(transcript string) bool {
	return strings.Contains(strings.ToLower(transcript), "the quick brown fox jumps")
}

// d2SmokeEvidence keeps runner output correlatable without retaining any of
// the model-generated text emitted by the native test.
func d2SmokeEvidence(output []byte) map[string]any {
	return map[string]any{"passed": true, "output_sha256": outputSHA256(output)}
}
