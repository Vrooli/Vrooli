package capabilities_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/capabilities/mocks"
)

func TestResourcesFSUsesExplicitLifecycleDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "whisper"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "whisper", "resource.json"), []byte(`{"platforms":{"linux":"supported"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_RESOURCES_DIR", root)
	t.Setenv("VROOLI_SCENARIO_DIR", "")
	fsys := capabilities.ResourcesFS()
	if fsys == nil {
		t.Fatal("ResourcesFS returned nil for explicit directory")
	}
	if _, err := fs.ReadFile(fsys, "whisper/resource.json"); err != nil {
		t.Fatalf("resource file unavailable: %v", err)
	}
}

const kyutaiManifest = `{
  "platforms":{"linux":"supported","macos":"unsupported","windows":"unsupported"},
  "deployment":{"profiles":{"desktop":{
    "linux":{"support":"conditional","reason":"Docker engine required"},
    "macos":{"support":"unsupported","reason":"Real-time streaming STT requires an NVIDIA CUDA GPU; Docker Desktop on macOS cannot pass through NVIDIA GPUs, and CPU decode is not real-time."},
    "windows":{"support":"unsupported","reason":"Real-time streaming STT requires an NVIDIA CUDA GPU with linux/amd64 container support; CPU decode is not real-time."}
  }}}
}`

func TestResourcePlatformResolver_ForcedGOOSUsesManifest(t *testing.T) {
	fsys := fstest.MapFS{"kyutai-stt/resource.json": {Data: []byte(kyutaiManifest)}}
	cases := []struct {
		goos string
		want capabilities.PlatformSupport
	}{
		{"linux", capabilities.PlatformDegraded},
		{"darwin", capabilities.PlatformUnsupported},
		{"windows", capabilities.PlatformUnsupported},
	}
	for _, tc := range cases {
		t.Run(tc.goos, func(t *testing.T) {
			got := capabilities.NewResourcePlatformResolver(fsys, tc.goos).Resolve("kyutai-stt")
			if got.Support != tc.want {
				t.Fatalf("support = %q, want %q", got.Support, tc.want)
			}
			if tc.want == capabilities.PlatformUnsupported && got.Reason == "" {
				t.Fatal("unsupported verdict must retain the resource manifest reason")
			}
		})
	}
}

func TestRegistry_UnsupportedPlatformDoesNotProbe(t *testing.T) {
	checker := mocks.NewFakeChecker(capabilities.StatusAvailable, "must not run")
	reg := capabilities.NewRegistry([]capabilities.Def{{
		ID: "kyutai-stt", Platform: capabilities.PlatformVerdict{
			Support: capabilities.PlatformUnsupported, Reason: "CUDA is unavailable on macOS",
		},
	}}, map[string]capabilities.Checker{"kyutai-stt": checker}, time.Minute)
	states := reg.Resolve(context.Background())
	if checker.CallCount() != 0 {
		t.Fatalf("unsupported resource was probed %d times", checker.CallCount())
	}
	if len(states) != 1 || states[0].Status != capabilities.StatusUnavailable {
		t.Fatalf("state = %+v, want unavailable", states)
	}
	if states[0].Message != "unavailable by design: CUDA is unavailable on macOS" {
		t.Fatalf("message = %q", states[0].Message)
	}
}

func TestKnownForPlatformDarwinMarksLocalSpeechProvidersUnsupported(t *testing.T) {
	root := t.TempDir()
	for _, slug := range []string{"whisper", "kyutai-stt", "kokoro", "sherpa-onnx"} {
		if err := os.Mkdir(filepath.Join(root, slug), 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := `{"platforms":{"macos":"unsupported"},"deployment":{"profiles":{"desktop":{"macos":{"support":"unsupported","reason":"native artifact is not qualified"}}}}}`
		if err := os.WriteFile(filepath.Join(root, slug, "resource.json"), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("VROOLI_RESOURCES_DIR", root)
	defs := capabilities.KnownForPlatform("darwin")
	want := map[string]bool{"whisper-stt": true, "kyutai-stt": true, "kokoro-tts": true, "speaker-verification": true}
	for _, def := range defs {
		if !want[def.ID] {
			continue
		}
		if def.Platform.Support != capabilities.PlatformUnsupported || def.Platform.Reason == "" {
			t.Errorf("%s platform = %+v, want unsupported with reason", def.ID, def.Platform)
		}
		delete(want, def.ID)
	}
	for id := range want {
		t.Errorf("catalogue did not include %s", id)
	}
}

func TestDarwinBYOKProvidersKeepSpeechCapabilitiesServiceable(t *testing.T) {
	defs := []capabilities.Def{
		{ID: "whisper-stt", Features: []string{"voice-input"}, Platform: capabilities.PlatformVerdict{Support: capabilities.PlatformUnsupported, Reason: "no native macOS artifact"}},
		{ID: "kokoro-tts", Features: []string{"voice-output"}, Platform: capabilities.PlatformVerdict{Support: capabilities.PlatformUnsupported, Reason: "no native macOS artifact"}},
		{ID: "openai-whisper", Features: []string{"voice-input"}},
		{ID: "openai-tts", Features: []string{"voice-output"}},
	}
	reg := capabilities.NewRegistry(defs, map[string]capabilities.Checker{
		"openai-whisper": mocks.NewFakeChecker(capabilities.StatusAvailable, "BYOK credential configured"),
		"openai-tts":     mocks.NewFakeChecker(capabilities.StatusAvailable, "BYOK credential configured"),
	}, time.Minute)
	groups := capabilities.Serviceability(reg.Resolve(context.Background()))
	seen := map[string]bool{}
	for _, group := range groups {
		if !group.Serviceable {
			t.Errorf("%s is not serviceable: %+v", group.Capability, group)
		}
		seen[group.Capability.String()] = true
	}
	for _, want := range []string{"CAPABILITY_STT", "CAPABILITY_TTS"} {
		if !seen[want] {
			t.Errorf("missing serviceability group %s", want)
		}
	}
}
