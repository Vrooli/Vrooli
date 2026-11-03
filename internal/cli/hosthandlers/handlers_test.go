package hosthandlers

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	blenderrenderer "github.com/vrooli/vrooli/internal/tools/blender-renderer"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

type testContext struct {
	out bytes.Buffer
}

func TestRendererCacheRepairRequiresExactTargetAndReportsDryRun(t *testing.T) {
	for _, tc := range []struct {
		name      string
		target    string
		wantCalls int
		wantErr   bool
	}{
		{name: "wrong target", target: "blender", wantErr: true},
		{name: "fixed cache target", target: "blender-python-cache", wantCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			calls := 0
			runCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
				Schema:      cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "target"}}, Flags: []cliapp.Flag{{Name: "dry-run", Bool: true}}},
				Positionals: map[string]string{"target": tc.target}, BoolFlags: map[string]bool{"dry-run": true}, JSON: true, Stdout: &output,
			})
			parent := &rootcli.CommandContext{Context: context.Background(), Globals: rootcli.GlobalOptions{JSON: true}}
			err := runRendererCacheRepairWith(runCtx, parent, func(_ context.Context, dry bool) (blenderrenderer.PythonCacheRepairReceipt, error) {
				calls++
				if !dry {
					t.Fatal("handler did not preserve explicit dry-run")
				}
				return blenderrenderer.PythonCacheRepairReceipt{SchemaVersion: 1, ProfileID: blenderrenderer.ProfileID, DryRun: true, Detail: "verified"}, nil
			})
			if (err != nil) != tc.wantErr || calls != tc.wantCalls {
				t.Fatalf("err=%v calls=%d; want err=%t calls=%d", err, calls, tc.wantErr, tc.wantCalls)
			}
			if !tc.wantErr && !strings.Contains(output.String(), `"dry_run": true`) {
				t.Fatalf("JSON receipt missing dry-run state: %q", output.String())
			}
		})
	}
}

func TestRendererProfileDispatchesTypedFFmpegAndBlenderProfiles(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   string
	}{
		{name: "ffmpeg-video", id: "ffmpeg-test"},
		{name: "blender", id: "blender-test"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			ffmpegCalls, blenderCalls := 0, 0
			runCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
				Schema: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "profile"}}},
				JSON:   true, Positionals: map[string]string{"profile": tc.name}, Stdout: &output,
			})
			parent := &rootcli.CommandContext{Context: context.Background(), Globals: rootcli.GlobalOptions{JSON: true}}
			err := runRendererProfileWith(
				runCtx, parent,
				func(context.Context) (*cliv1.RendererToolProfile, error) {
					ffmpegCalls++
					return &cliv1.RendererToolProfile{ProfileId: "ffmpeg-test"}, nil
				},
				func(context.Context) (*cliv1.BlenderToolProfile, error) {
					blenderCalls++
					return &cliv1.BlenderToolProfile{ProfileId: "blender-test"}, nil
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			want := `"profile_id": "` + tc.id + `"`
			if !strings.Contains(output.String(), want) {
				t.Fatalf("typed profile output %q does not contain %q", output.String(), want)
			}
			if tc.name == "blender" && (blenderCalls != 1 || ffmpegCalls != 0) {
				t.Fatalf("Blender dispatch calls = ffmpeg:%d blender:%d", ffmpegCalls, blenderCalls)
			}
			if tc.name == "ffmpeg-video" && (ffmpegCalls != 1 || blenderCalls != 0) {
				t.Fatalf("FFmpeg dispatch calls = ffmpeg:%d blender:%d", ffmpegCalls, blenderCalls)
			}
		})
	}
}

func TestRootHandlerDelegatesHelp(t *testing.T) {
	ctx := &testContext{}
	handler := RootHandler(rootcli.HandlerDeps[*testContext]{
		Root:    func(*testContext) string { return "." },
		Globals: func(*testContext) rootcli.GlobalOptions { return rootcli.GlobalOptions{} },
		Stdout:  func(ctx *testContext) io.Writer { return &ctx.out },
		Stderr:  func(ctx *testContext) io.Writer { return &ctx.out },
	})
	if err := handler(ctx, []string{"--help"}); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if ctx.out.Len() == 0 {
		t.Fatal("host handler should write help")
	}
}
