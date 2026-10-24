package rendering

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/platform-go"
	"github.com/vrooli/repo-contract-go/cliinvoke"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed testdata/blender_backend_probe.py
var blenderBackendProbe []byte

const (
	blenderProfileID = "blender-5.2.2-linux-x64"
	blenderTreeSHA   = "5fca420feab408276778511a84d7a2f5659bbc09aa5cb43b6d24ae59b220cb55"
	blenderBinarySHA = "bafec183e960819c76fe213c000cdb2284c53a87ef6791812f8728b51f07145b"
	blenderVersion   = "Blender 5.2.2 LTS"
	blenderArtifact  = "a1660c40fcc34e79aaa339f846b95d5973981a23b9e4b87e50ee2c633159762e"
	blenderArchive   = "blender-5.2.2-linux-x64"
)

type blenderManifest struct {
	Schema   int    `json:"schema_version"`
	Profile  string `json:"profile_id"`
	Artifact string `json:"artifact_identity"`
	Tree     string `json:"tree_sha256"`
	Binary   string `json:"binary_sha256"`
	Version  string `json:"blender_version"`
	Adapter  string `json:"adapter_sha256"`
	Backend  string `json:"backend"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Samples  int    `json:"samples"`
}
type blenderReceipt struct {
	Schema     int    `json:"schema_version"`
	Status     string `json:"status"`
	Manifest   string `json:"manifest_sha256"`
	Adapter    string `json:"adapter_sha256"`
	Profile    string `json:"profile_id"`
	Artifact   string `json:"artifact_identity"`
	Tree       string `json:"tree_sha256"`
	Binary     string `json:"binary_sha256"`
	Version    string `json:"blender_version"`
	AppVersion []int  `json:"blender_app_version"`
	BuildHash  string `json:"blender_build_hash"`
	Requested  string `json:"backend_requested"`
	Selected   string `json:"backend_selected"`
	Device     string `json:"device"`
	Output     string `json:"output_name"`
	OutputSHA  string `json:"output_sha256"`
	Error      string `json:"error"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Samples    int    `json:"samples"`
	Size       int64  `json:"output_size_bytes"`
}
type blenderIntent struct {
	Intent          platform.OperationIntent `json:"intent"`
	Spec            platform.OperationSpec   `json:"spec"`
	Purpose         string                   `json:"purpose"`
	RenderProfile   string                   `json:"render_profile,omitempty"`
	ProfileArtifact string                   `json:"profile_artifact_identity"`
	ManifestSHA     string                   `json:"manifest_sha256"`
	AdapterSHA      string                   `json:"adapter_sha256"`
}
type boundedWriter struct {
	b   bytes.Buffer
	max int
}

func (w *boundedWriter) Write(p []byte) (int, error) {
	if len(p) > w.max-w.b.Len() {
		return 0, errors.New("bounded command output exceeded")
	}
	return w.b.Write(p)
}

// Opt-in evidence only: it creates no Device Studio job and never uses PATH.
func TestOptInBlenderBackendQualification(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E3_BLENDER_QUALIFICATION") != "1" {
		t.Skip("opt-in Blender qualification")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("profile supports Linux/amd64 only")
	}
	root := filepath.Clean(os.Getenv("DEVICE_STUDIO_E3_BLENDER_EVIDENCE_DIR"))
	if !filepath.IsAbs(root) || !strings.Contains(filepath.ToSlash(root), "/goal/evidence/E3/tool-profile/backend-qualification/") {
		t.Fatalf("evidence path outside goal evidence: %q", root)
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	profileBytes, profile := resolveBlenderProfile(t)
	if err := writeDurable(root, "resolved-profile.json", profileBytes); err != nil {
		t.Fatal(err)
	}
	bin := profile.GetBlender()
	if profile.GetState() != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT || profile.GetSchemaVersion() != 1 || profile.GetProfileId() != blenderProfileID || profile.GetArtifactIdentity() != blenderArtifact || profile.GetExpectedTreeSha256() != blenderTreeSHA || bin == nil || bin.GetVersion() != blenderVersion || bin.GetSha256() != blenderBinarySHA || bin.GetArchiveMemberPath() != blenderArchive+"/blender" || !filepath.IsAbs(bin.GetPath()) {
		t.Fatalf("profile differs from pinned artifact: %+v", profile)
	}
	if filepath.Base(filepath.Dir(bin.GetPath())) != blenderArchive {
		t.Fatalf("binary outside expected archive root: %q", bin.GetPath())
	}
	for _, backend := range []string{"CPU", "CUDA", "OPTIX"} {
		t.Run(backend, func(t *testing.T) {
			runBlenderAttempt(t, root, bin.GetPath(), profile.GetArtifactIdentity(), backend)
		})
	}
}

func resolveBlenderProfile(t *testing.T) ([]byte, *cliv1.BlenderToolProfile) {
	t.Helper()
	b, p, err := queryBlenderProfile()
	if err != nil {
		t.Fatal(err)
	}
	return b, p
}

func queryBlenderProfile() ([]byte, *cliv1.BlenderToolProfile, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}
	path, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{Explicit: os.Getenv(cliinvoke.BinaryEnvVar), RuntimeHome: home, Getenv: func(string) string { return "" }, LookPath: func(string) (string, error) { return "", exec.ErrNotFound }})
	if err != nil {
		return nil, nil, err
	}
	var out, stderr boundedWriter
	out.max, stderr.max = 64<<10, 8<<10
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	r := cliinvoke.Run(ctx, cliinvoke.Invocation{Binary: path, Args: []string{"host", "renderer-profile", "blender", "--json"}, Timeout: 20 * time.Second, WaitDelay: 2 * time.Second, Stdout: &out, Stderr: &stderr})
	if r.Class != cliinvoke.OK {
		return nil, nil, fmt.Errorf("profile query failed: %s", r.Class)
	}
	p := new(cliv1.BlenderToolProfile)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(out.b.Bytes(), p); err != nil {
		return nil, nil, errors.New("profile response is not typed JSON")
	}
	return bytes.Clone(out.b.Bytes()), p, nil
}

func runBlenderAttempt(t *testing.T, root, blender, artifact, backend string) {
	t.Helper()
	dir := filepath.Join(root, strings.ToLower(backend))
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"home", "tmp", "config", "scripts", "data", "pycache"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	adapter := blenderDigest(blenderBackendProbe)
	m := blenderManifest{1, blenderProfileID, artifact, blenderTreeSHA, blenderBinarySHA, blenderVersion, adapter, backend, 256, 256, 4}
	mb, _ := json.Marshal(m)
	mb = append(mb, '\n')
	if err := writeDurable(dir, "blender_backend_probe.py", blenderBackendProbe); err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(dir, "qualification-manifest.json", mb); err != nil {
		t.Fatal(err)
	}
	manifestsum := blenderDigest(mb)
	runOwnedBlenderOperation(t, dir, blender, artifact, backend, "backend-probe", "", manifestsum, adapter, func() error { return verifyBlenderAttempt(dir, m, mb) })
}

// This private test-only runner accepts only the fixed reviewed purposes below.
func runOwnedBlenderOperation(t *testing.T, dir, blender, artifact, backend, purpose, renderProfile, manifestSHA, adapterSHA string, verify func() error) {
	t.Helper()
	before, err := checkedBlenderProfile(blender)
	if err != nil {
		t.Fatal(err)
	}
	if err = writeDurable(dir, "profile-before.json", before); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"home", "tmp", "config", "scripts", "data", "pycache"} {
		if err = os.MkdirAll(filepath.Join(dir, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	script := "blender_backend_probe.py"
	args := []string{"--background", "--factory-startup", "--disable-autoexec", "--python-use-system-env", "--threads", "2", "--python-exit-code", "17", "--python", filepath.Join(dir, script)}
	wall := 120 * time.Second
	switch purpose {
	case "backend-probe":
		args = append(args, "--", "qualification-manifest.json")
	case "phone-table-scene-export":
		if backend != "CPU" || renderProfile != "" {
			t.Fatal("scene asset export requires the fixed CPU profile")
		}
		args[len(args)-1] = filepath.Join(dir, "export_scene.py")
		args = append(args, "--")
	case "phone-study-environment-export", "phone-starter-derive":
		if backend != "CPU" || renderProfile != "" {
			t.Fatal("asset derivation requires the fixed CPU profile")
		}
		script = "export_environment.py"
		if purpose == "phone-starter-derive" {
			script = "derive_phone_starter.py"
		}
		args[len(args)-1] = filepath.Join(dir, script)
		args = append(args, "--")
	case "ipad-study-environment-export":
		if backend != "CPU" || renderProfile != "" {
			t.Fatal("iPad scene export requires the fixed CPU profile")
		}
		args[len(args)-1] = filepath.Join(dir, "export_ipad_environment.py")
		args = append(args, "--")
		wall = 180 * time.Second
	case "laptop-starter-derive":
		if backend != "CPU" || renderProfile != "" {
			t.Fatal("laptop derivation requires the fixed CPU profile")
		}
		args[len(args)-1] = filepath.Join(dir, "derive_laptop_starter.py")
		args = append(args, "--")
		wall = 180 * time.Second
	case "physical-semantic-probe":
		if backend != "CPU" || renderProfile != "" {
			t.Fatal("adapter semantic probe requires the fixed CPU profile")
		}
		args[len(args)-1] = filepath.Join(dir, "physical_semantic_probe.py")
		args = append(args, "--", filepath.Join(dir, "physical-semantic-probe-manifest.json"))
	case "laptop-partition-study":
		if backend != "CPU" || renderProfile != "" {
			t.Fatal("laptop diagnostic requires fixed CPU profile")
		}
		args[len(args)-1] = filepath.Join(dir, "laptop_partition_diagnostic.py")
		wall = 300 * time.Second
		args = append(args, "--")
	case "phone-scene-study":
		script = "phone_scene_study.py"
		args[len(args)-1] = filepath.Join(dir, script)
		if renderProfile != "study" && renderProfile != "hero" && renderProfile != "posecontactsheet" && renderProfile != "motion-yaw-24" {
			t.Fatalf("unsupported fixed phone render profile %q", renderProfile)
		}
		if renderProfile == "hero" || renderProfile == "motion-yaw-24" {
			wall = 300 * time.Second
		} else {
			wall = 180 * time.Second
		}
		args = append(args, "--")
	default:
		t.Fatalf("unknown fixed Blender operation purpose %q", purpose)
	}
	env := []string{"HOME=" + filepath.Join(dir, "home"), "TMPDIR=" + filepath.Join(dir, "tmp"), "BLENDER_USER_CONFIG=" + filepath.Join(dir, "config"), "BLENDER_USER_SCRIPTS=" + filepath.Join(dir, "scripts"), "BLENDER_USER_DATAFILES=" + filepath.Join(dir, "data"), "PYTHONPYCACHEPREFIX=" + filepath.Join(dir, "pycache"), "PYTHONDONTWRITEBYTECODE=1", "LANG=C", "LC_ALL=C", "OMP_NUM_THREADS=2"}
	spec := platform.OperationSpec{Path: blender, Args: args, Env: env, Dir: dir, Limits: platform.OperationLimits{MemoryMaxBytes: 8 << 30, TasksMax: 128, CPUWeight: 100, WallTime: wall}}
	attempt := fmt.Sprintf("e3-blender-%s-%s-%d", strings.ToLower(purpose), strings.ToLower(backend), time.Now().UnixNano())
	if renderProfile != "" {
		attempt += "-" + renderProfile
	}
	intent, err := platform.NewOperationIntent("device-studio", attempt, spec)
	if err != nil {
		t.Fatal(err)
	}
	ib, _ := json.MarshalIndent(blenderIntent{Intent: intent, Spec: spec, Purpose: purpose, RenderProfile: renderProfile, ProfileArtifact: artifact, ManifestSHA: manifestSHA, AdapterSHA: adapterSHA}, "", "  ")
	if err = writeDurable(dir, "operation-intent.json", append(ib, '\n')); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), wall+10*time.Second)
	defer cancel()
	op, err := platform.StartOwnedOperation(ctx, intent, spec)
	if err != nil {
		ref, e := platform.LookupOwnedOperation(ctx, intent)
		if e != nil {
			b, _ := json.MarshalIndent(map[string]any{"intent": intent, "start_error": err.Error(), "lookup_error": e.Error()}, "", "  ")
			_ = writeDurable(dir, "dispatch-unknown.json", append(b, '\n'))
			t.Fatalf("ambiguous dispatch; exact lookup failed: %v / %v", err, e)
		}
		op = &platform.OwnedOperation{Ref: ref}
	}
	settled := false
	t.Cleanup(func() {
		if settled {
			return
		}
		cleanup, d := context.WithTimeout(context.Background(), 15*time.Second)
		defer d()
		stop, se := platform.StopOwnedOperation(cleanup, op.Ref)
		obs, ie := platform.InspectOwnedOperation(cleanup, op.Ref)
		r := map[string]any{"ref": op.Ref, "stop": stop, "observation": obs, "released": false}
		if se != nil {
			r["stop_error"] = se.Error()
		}
		if ie != nil {
			r["inspect_error"] = ie.Error()
		}
		if se == nil && stop.NativeQuiescent && ie == nil && obs.ExitKnown && !obs.Populated {
			if e := platform.ReleaseOwnedOperation(cleanup, op.Ref); e == nil {
				r["released"] = true
			} else {
				r["release_error"] = e.Error()
			}
		}
		if r["released"] != true {
			r["error"] = "native exit/quiescence/release was not established"
		}
		b, e := json.MarshalIndent(r, "", "  ")
		if e == nil {
			if e = writeDurable(dir, "cleanup-attempt.json", append(b, '\n')); e != nil {
				t.Errorf("persist cleanup evidence: %v", e)
			}
		} else {
			t.Errorf("marshal cleanup evidence: %v", e)
		}
		if r["released"] != true {
			t.Errorf("cleanup could not prove exact release: %v / %v", se, ie)
		}
	})
	rb, _ := json.MarshalIndent(op.Ref, "", "  ")
	if err = writeDurable(dir, "operation-ref.json", append(rb, '\n')); err != nil {
		t.Fatal(err)
	}
	waitErr := op.Wait(ctx)
	obs, err := platform.InspectOwnedOperation(ctx, op.Ref)
	if err != nil {
		t.Fatal(err)
	}
	stop, err := platform.StopOwnedOperation(ctx, op.Ref)
	if err != nil || !stop.NativeQuiescent {
		t.Fatalf("native tree not quiescent: %v %+v", err, stop)
	}
	obs, err = platform.InspectOwnedOperation(ctx, op.Ref)
	if err != nil {
		t.Fatal(err)
	}
	ob, _ := json.MarshalIndent(struct {
		Observation platform.OperationObservation `json:"observation"`
		Stop        platform.OperationStopReceipt `json:"stop"`
	}{obs, stop}, "", "  ")
	if err = writeDurable(dir, "operation-terminal.json", append(ob, '\n')); err != nil {
		t.Fatal(err)
	}
	after, err := checkedBlenderProfile(blender)
	if err != nil {
		t.Fatal(err)
	}
	if err = writeDurable(dir, "profile-after.json", after); err != nil {
		t.Fatal(err)
	}
	if waitErr != nil || !obs.ExitKnown || obs.Populated || obs.Result != "success" || obs.ExecMainStatus != 0 {
		t.Fatalf("operation failed: wait=%v result=%s status=%d", waitErr, obs.Result, obs.ExecMainStatus)
	}
	if err = verify(); err != nil {
		t.Fatal(err)
	}
	if err = platform.ReleaseOwnedOperation(ctx, op.Ref); err != nil {
		t.Fatal(err)
	}
	settled = true
	released, _ := json.MarshalIndent(map[string]any{"ref": op.Ref, "released": true}, "", "  ")
	if err = writeDurable(dir, "operation-released.json", append(released, '\n')); err != nil {
		t.Fatal(err)
	}
}

func verifyBlenderAttempt(dir string, m blenderManifest, mb []byte) error {
	b, err := os.ReadFile(filepath.Join(dir, "qualification-receipt.json"))
	if err != nil || len(b) > 16<<10 {
		return fmt.Errorf("bounded receipt: %v", err)
	}
	var r blenderReceipt
	if err = json.Unmarshal(b, &r); err != nil {
		return err
	}
	if r.Schema != 1 || r.Manifest != blenderDigest(mb) || r.Adapter != m.Adapter || r.Profile != m.Profile || r.Artifact != m.Artifact || r.Tree != m.Tree || r.Binary != m.Binary || r.Version == "" || len(r.AppVersion) != 3 || r.AppVersion[0] != 5 || r.AppVersion[1] != 2 || r.AppVersion[2] != 2 || r.BuildHash == "" || r.Requested != m.Backend || r.Width != 256 || r.Height != 256 || r.Samples != 4 {
		return errors.New("receipt does not match pinned render")
	}
	if r.Status == "backend_unavailable" && (m.Backend == "CPU" || r.Selected != "" || r.Output != "" || r.Size != 0 || r.OutputSHA != "" || r.Error != "") {
		return errors.New("invalid backend-unavailable result")
	}
	if r.Status != "rendered" && r.Status != "backend_unavailable" {
		return errors.New("backend probe or render failed")
	}
	if r.Status == "backend_unavailable" {
		return nil
	}
	if r.Selected != m.Backend || r.Output != "qualification.png" || r.Size <= 0 || r.Size > 8<<20 || r.Error != "" {
		return errors.New("rendered result has invalid output facts")
	}
	f, err := os.Open(filepath.Join(dir, r.Output))
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 8<<20+1))
	if err != nil || len(data) > 8<<20 || int64(len(data)) != r.Size || blenderDigest(data) != r.OutputSHA {
		return errors.New("PNG size or digest mismatch")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width != 256 || cfg.Height != 256 {
		return errors.New("PNG dimensions or header invalid")
	}
	if _, err = png.Decode(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("PNG full decode: %w", err)
	}
	return nil
}

func checkedBlenderProfile(bin string) ([]byte, error) {
	_, p, err := queryBlenderProfile()
	if err != nil {
		return nil, fmt.Errorf("profile owner recheck: %w", err)
	}
	b := p.GetBlender()
	if p.GetSchemaVersion() != 1 || p.GetProfileId() != blenderProfileID || p.GetState() != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT || p.GetArtifactIdentity() != blenderArtifact || p.GetExpectedTreeSha256() != blenderTreeSHA || b == nil || b.GetPath() != bin || b.GetSha256() != blenderBinarySHA || b.GetVersion() != blenderVersion {
		return nil, errors.New("managed Blender tree or executable is no longer verified")
	}
	return protojson.MarshalOptions{UseProtoNames: true}.Marshal(p)
}
func writeDurable(dir, name string, b []byte) error {
	f, e := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if e != nil {
		return e
	}
	if _, e = f.Write(b); e == nil {
		e = f.Sync()
	}
	c := f.Close()
	if e != nil {
		return e
	}
	if c != nil {
		return c
	}
	d, e := os.Open(dir)
	if e != nil {
		return e
	}
	defer d.Close()
	return d.Sync()
}
func blenderDigest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
