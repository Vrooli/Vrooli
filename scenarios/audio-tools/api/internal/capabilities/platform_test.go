package capabilities_test

import (
	"context"
	"testing"
	"testing/fstest"
	"time"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/capabilities/mocks"
)

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
