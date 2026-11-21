package projectcli

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/project"
)

func TestDoctorHandlerPassesResultAndErrorThroughCommandPipeline(t *testing.T) {
	var stdout bytes.Buffer
	format := func(struct{}) (cliout.Format, error) { return cliout.FormatHuman, nil }

	want := project.DoctorReport{Checks: []project.DoctorCheck{{Name: "go", Status: "ok"}}}
	handler := DoctorHandler(func(struct{}) io.Writer { return &stdout }, format,
		func(_ struct{}, req DoctorRequest) (project.DoctorReport, error) {
			if req.RepairFilePermissions {
				t.Fatal("unexpected repair request")
			}
			return want, nil
		})
	if err := handler(struct{}{}, nil); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if !strings.Contains(stdout.String(), "go") || !strings.Contains(stdout.String(), "ok") {
		t.Fatalf("rendered result = %q", stdout.String())
	}

	stdout.Reset()
	serviceErr := errors.New("doctor failed")
	handler = DoctorHandler(func(struct{}) io.Writer { return &stdout }, format,
		func(_ struct{}, _ DoctorRequest) (project.DoctorReport, error) {
			return want, serviceErr
		})
	if err := handler(struct{}{}, nil); err != serviceErr {
		t.Fatalf("handler error = %v, want identical service error", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("rendered response despite service error: %q", stdout.String())
	}
}
