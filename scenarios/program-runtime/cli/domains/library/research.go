package library

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/researchhome"
	"github.com/vrooli/cli-core/cliutil"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shared"
)

// Research programs (research/<topic>/<program>, research/<program>) are
// files in the research home. Program Runtime runs them like any declared
// program; when it cannot serve one — down, an older binary, another research
// home — the CLI reads or runs the files so research never depends on it.

// researchFallback reports whether a failed call for name should be served
// from the research files. A run falls back only when nothing was submitted
// (the API was unreachable, or this session's credential could not be
// resolved) or the API did not know the program; a lost response is never
// retried locally. A read may also fall back when the API answered
// unavailable.
func researchFallback(name string, err error, read bool) bool {
	if !researchhome.IsID(name) {
		return false
	}
	code := connect.CodeOf(err)
	return unreachable(err) || cliutil.CredentialNotResolved(err) || code == connect.CodeNotFound || (read && code == connect.CodeUnavailable)
}

// unreachable reports that the request never reached the API: the dial
// failed, the host did not resolve, or no API base was discovered.
func unreachable(err error) bool {
	var op *net.OpError
	var dns *net.DNSError
	return (errors.As(err, &op) && op.Op == "dial") || errors.As(err, &dns) || strings.Contains(err.Error(), "unsupported protocol scheme")
}

// standaloneRunner is the kernel's own outage runner, beside program_helper.py.
var standaloneRunner = func() string {
	return filepath.Join(cliutil.ResolveScenarioPath("program-runtime"), "kernel", "host", "standalone.py")
}

// runResearchStandalone runs a research program with the kernel's helper but
// without Program Runtime. The result has the shape of a declared run, so the
// report and outcome are the same; a program that needs governed bindings is
// refused with an envelope naming the file to follow by hand.
func runResearchStandalone(name string, inputs map[string]any, why error, stderr io.Writer) (*libraryv1.RunDeclaredProgramResponse, error) {
	program, err := researchhome.ReadProgram("", name)
	if err != nil {
		return nil, fmt.Errorf("Program Runtime could not run %s (%v) and its files cannot be read: %w", name, why, err)
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		return nil, fmt.Errorf("Program Runtime could not run %s (%v) and python3 is not installed; read %s and do its steps by hand", name, why, program.Source.Path)
	}
	encoded, err := json.Marshal(inputs)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(stderr, "program-runtime: running %s from %s without Program Runtime (%v)\n", name, program.Contract.Path, why)
	var stdout, errOut bytes.Buffer
	cmd := exec.Command(python, "-I", standaloneRunner(), program.Contract.Path, "--inputs-json", string(encoded))
	cmd.Stdout, cmd.Stderr = &stdout, &errOut
	runErr := cmd.Run()
	var exit *exec.ExitError
	if runErr != nil && (!errors.As(runErr, &exit) || exit.ExitCode() == 2) {
		return nil, fmt.Errorf("standalone run of %s failed: %v %s", name, runErr, bytes.TrimSpace(errOut.Bytes()))
	}
	return &libraryv1.RunDeclaredProgramResponse{Terminal: true, Program: &programsv1.Program{
		Id: "standalone:" + name, ProgramName: name, Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, Stdout: stdout.String(),
	}}, nil
}

// note is where fallback notes go; never nil.
func (h *handlers) note() io.Writer {
	if h.progress == nil {
		return io.Discard
	}
	return h.progress
}

// researchLibraryProgram is `library get` read from the files.
func researchLibraryProgram(name string, why error, stderr io.Writer) (*libraryv1.GetLibraryResponse, error) {
	program, err := researchhome.ReadProgram("", name)
	if err != nil {
		return nil, fmt.Errorf("Program Runtime could not read %s (%v) and its files cannot be read: %w", name, why, err)
	}
	var contract struct {
		Version string                     `json:"version"`
		Purpose string                     `json:"purpose"`
		Inputs  map[string]json.RawMessage `json:"inputs"`
		Binds   []struct {
			ID string `json:"id"`
		} `json:"bindings"`
	}
	if err := json.Unmarshal(program.Contract.Content, &contract); err != nil {
		return nil, fmt.Errorf("%s: %w", program.Contract.Path, err)
	}
	fmt.Fprintf(stderr, "program-runtime: read %s from %s (%v)\n", name, program.Contract.Path, why)
	version, _ := strconv.ParseInt(contract.Version, 10, 64)
	out := &sharedv1.LibraryProgram{Name: name, Id: name, Version: version, Kind: "contract", Description: contract.Purpose, Purpose: contract.Purpose, Path: program.Contract.Path, Source: string(program.Source.Content), Origin: "research files"}
	for input := range contract.Inputs {
		out.DeclaredInputs = append(out.DeclaredInputs, input)
	}
	sort.Strings(out.DeclaredInputs)
	for _, binding := range contract.Binds {
		out.CalledBindingIds = append(out.CalledBindingIds, binding.ID)
	}
	return &libraryv1.GetLibraryResponse{Program: out}, nil
}
