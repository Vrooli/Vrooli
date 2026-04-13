package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/topcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/maintenance"
)

func TestExecuteTopLevelCommandRendersHelpOnlyErrors(t *testing.T) {
	var stdout bytes.Buffer
	ctx := &commandContext{Stdout: &stdout}

	err := executeCommandAction(&App{}, ctx, nil, commandAction[struct{}, struct{}]{
		parse: func(ctx *commandContext, args []string) (struct{}, error) {
			return struct{}{}, commandHelpOnly("Usage: vrooli status")
		},
		run: func(app *App, ctx *commandContext, req struct{}) (cliout.Format, struct{}, error) {
			t.Fatal("run should not be called")
			return "", struct{}{}, nil
		},
		render: func(w io.Writer, format cliout.Format, resp struct{}) error {
			t.Fatal("render should not be called")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("executeCommandAction returned error: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Usage: vrooli status") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestParseTopLevelDiagnosePortRequestRejectsInvalidPort(t *testing.T) {
	if _, err := topcli.ParseDiagnosePortRequest([]string{"bogus"}); err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestRenderTopLevelLocksResponseHumanIncludesStaleStatus(t *testing.T) {
	var stdout bytes.Buffer
	err := topcli.RenderLocksResponse(&stdout, cliout.FormatHuman, topcli.LocksResponse{
		List: []maintenance.LockInfo{{
			Port:     21234,
			Scenario: "alpha",
			PID:      999,
			Stale:    true,
		}},
	})
	if err != nil {
		t.Fatalf("renderTopLevelLocksResponse returned error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "21234") || !strings.Contains(output, "stale") {
		t.Fatalf("stdout = %q", output)
	}
}

func TestRenderTopLevelOrphansResponseHumanHandlesKillReport(t *testing.T) {
	var stdout bytes.Buffer
	err := topcli.RenderOrphansResponse(&stdout, cliout.FormatHuman, topcli.OrphansResponse{
		KillReport: &control.StopReport{
			Stopped: []control.ResultItem{control.Stopped("123", "sleep 30")},
		},
	})
	if err != nil {
		t.Fatalf("renderTopLevelOrphansResponse returned error: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Stopped orphan PID 123") {
		t.Fatalf("stdout = %q", got)
	}
}
