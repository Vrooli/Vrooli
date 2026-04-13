package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/resources"
)

func TestParseResourceStatusRequestSupportsFastFlagsAndOptionalName(t *testing.T) {
	req, err := parseResourceStatusRequest(globalOptions{}, []string{"--no-fast", "redis"})
	if err != nil {
		t.Fatalf("parseResourceStatusRequest: %v", err)
	}
	if req.Fast {
		t.Fatalf("expected fast=false")
	}
	if req.Name != "redis" {
		t.Fatalf("name = %q", req.Name)
	}
}

func TestExecuteResourceCommandRendersHelpOnlyErrors(t *testing.T) {
	var stdout bytes.Buffer
	err := executeResourceCommand(nil, globalOptions{}, nil, &stdout, io.Discard, resourceCommandAction[struct{}, struct{}]{
		parse: func(globals globalOptions, args []string) (struct{}, error) {
			return struct{}{}, commandHelpOnly("Usage: vrooli resource fake")
		},
		run: func(controller *resources.Controller, ctx *commandContext, req struct{}) (cliout.Format, struct{}, error) {
			t.Fatal("run should not be called for help-only command")
			return "", struct{}{}, nil
		},
		render: func(w io.Writer, format cliout.Format, resp struct{}) error {
			t.Fatal("render should not be called for help-only command")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("executeResourceCommand: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Usage: vrooli resource fake") {
		t.Fatalf("help output missing usage text: %q", got)
	}
}

func TestParseResourceValidateRequestAcceptsOptionalName(t *testing.T) {
	req, err := parseResourceValidateRequest(globalOptions{}, []string{"redis"})
	if err != nil {
		t.Fatalf("parseResourceValidateRequest: %v", err)
	}
	if req.Name != "redis" {
		t.Fatalf("name = %q, want redis", req.Name)
	}
}
