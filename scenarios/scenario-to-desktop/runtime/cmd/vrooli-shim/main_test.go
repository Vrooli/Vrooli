package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	runtimemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"google.golang.org/protobuf/proto"
)

func TestOfflineRendererProfileRunsBeforeRuntimeAppDataLookup(t *testing.T) {
	var output bytes.Buffer
	manifest := &bundleInfo{App: runtimemanifest.App{Name: "demo"}}
	handled, err := runOfflineToolProfile([]string{"host", "renderer-profile", "blender", "--json"}, t.TempDir(), manifest, &output)
	if err != nil || !handled {
		t.Fatalf("offline command = handled %t, err %v", handled, err)
	}
	if !strings.Contains(output.String(), `"profileId":"blender-5.2.2-linux-x64"`) || !strings.Contains(output.String(), `"state":"RENDERER_TOOL_PROFILE_STATE_MISSING"`) {
		t.Fatalf("offline profile JSON differs from generated proto contract: %s", output.String())
	}
}

func TestOfflineRendererProfileRejectsUnboundedArguments(t *testing.T) {
	var output bytes.Buffer
	handled, err := runOfflineToolProfile([]string{"host", "renderer-profile", "blender", "--json", "extra"}, t.TempDir(), &bundleInfo{}, &output)
	if !handled || err == nil || output.Len() != 0 {
		t.Fatalf("unexpected dispatch result handled=%t err=%v output=%q", handled, err, output.String())
	}
}

func TestOfflineRendererProfileResolverHasTwentySecondDeadline(t *testing.T) {
	var output bytes.Buffer
	var deadline time.Time
	resolve := func(ctx context.Context, _ string, _ string, _ []runtimemanifest.ToolProfilePackage) (proto.Message, error) {
		var ok bool
		deadline, ok = ctx.Deadline()
		if !ok {
			t.Fatal("profile resolver has no total deadline")
		}
		return &cliv1.BlenderToolProfile{}, nil
	}
	handled, err := runOfflineToolProfileWithResolver([]string{"host", "renderer-profile", "blender", "--json"}, t.TempDir(), &bundleInfo{}, &output, resolve)
	if err != nil || !handled {
		t.Fatalf("offline profile dispatch handled=%t err=%v", handled, err)
	}
	remaining := deadline.Sub(time.Now())
	if remaining > rendererProfileQueryTimeout || remaining <= 0 {
		t.Fatalf("unexpected total resolver deadline: %s", remaining)
	}
}
