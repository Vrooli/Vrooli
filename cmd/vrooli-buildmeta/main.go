package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
)

type fingerprintFunc func(root string, options buildinfo.FingerprintOptions, relPaths ...string) (buildinfo.FingerprintReport, error)

type command struct {
	computeFingerprint fingerprintFunc
}

type request struct {
	root    string
	targets []string
	json    bool
	verbose bool
}

type result struct {
	report buildinfo.FingerprintReport
}

type errorKind string

const (
	exitCodeSuccess    = 0
	exitCodeInternal   = 1
	exitCodeUsage      = 2
	exitCodeValidation = 3
)

const (
	errorKindMissingTargets errorKind = "missing_targets"
	errorKindNoGoFiles      errorKind = "no_go_files"
	errorKindInvalidTarget  errorKind = "invalid_target"
	errorKindInternal       errorKind = "internal"
)

type errorResponse struct {
	Error   string    `json:"error"`
	Kind    errorKind `json:"kind"`
	Root    string    `json:"root,omitempty"`
	Targets []string  `json:"targets,omitempty"`
}

func main() {
	cmd := command{
		computeFingerprint: buildinfo.ComputeSourceFingerprintReport,
	}
	os.Exit(cmd.run(os.Args[1:], os.Stdout, os.Stderr))
}

func (c command) run(args []string, stdout, stderr io.Writer) int {
	req, code, done, err := c.parse(args, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "vrooli-buildmeta: %v\n", err)
		return exitCodeInternal
	}
	if done {
		return code
	}

	res, err := c.execute(req)
	if err != nil {
		return renderError(stdout, stderr, err, req)
	}

	return renderSuccess(stdout, stderr, req, res)
}

func (c command) parse(args []string, stderr io.Writer) (request, int, bool, error) {
	flags := flag.NewFlagSet("vrooli-buildmeta", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("root", ".", "repository root used for fingerprinting")
	jsonOutput := flags.Bool("json", false, "emit the full fingerprint report as JSON")
	verbose := flags.Bool("verbose", false, "print a short summary to stderr on success")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return request{}, exitCodeSuccess, true, writeUsage(stderr, flags)
		}
		if writeErr := writeLine(stderr, fmt.Sprintf("vrooli-buildmeta: %v", err)); writeErr != nil {
			return request{}, exitCodeInternal, true, writeErr
		}
		return request{}, exitCodeUsage, true, writeUsage(stderr, flags)
	}
	if len(flags.Args()) == 0 {
		if err := writeLine(stderr, "vrooli-buildmeta: at least one relative path is required"); err != nil {
			return request{}, exitCodeInternal, true, err
		}
		return request{}, exitCodeUsage, true, writeUsage(stderr, flags)
	}

	return request{
		root:    *root,
		targets: append([]string(nil), flags.Args()...),
		json:    *jsonOutput,
		verbose: *verbose,
	}, exitCodeSuccess, false, nil
}

func (c command) execute(req request) (result, error) {
	computeFingerprint := c.computeFingerprint
	if computeFingerprint == nil {
		computeFingerprint = buildinfo.ComputeSourceFingerprintReport
	}

	report, err := computeFingerprint(req.root, buildinfo.FingerprintOptions{
		RequireExistingTargets: true,
		RequireGoFiles:         true,
	}, req.targets...)
	if err != nil {
		return result{}, err
	}
	return result{report: report}, nil
}

func renderSuccess(stdout, stderr io.Writer, req request, res result) int {
	if req.json {
		if err := writeJSON(stdout, res.report); err != nil {
			if writeErr := writeLine(stderr, fmt.Sprintf("vrooli-buildmeta: encode JSON output: %v", err)); writeErr != nil {
				return exitCodeInternal
			}
			return exitCodeInternal
		}
	} else {
		if err := writeLine(stdout, res.report.Fingerprint); err != nil {
			if writeErr := writeLine(stderr, fmt.Sprintf("vrooli-buildmeta: write fingerprint: %v", err)); writeErr != nil {
				return exitCodeInternal
			}
			return exitCodeInternal
		}
	}

	if req.verbose {
		if err := writeLine(stderr, fmt.Sprintf("vrooli-buildmeta: fingerprinted %d Go files under %q for targets %s",
			res.report.MatchedFiles,
			res.report.Root,
			strings.Join(res.report.Targets, ","))); err != nil {
			return exitCodeInternal
		}
	}
	return exitCodeSuccess
}

func renderError(stdout, stderr io.Writer, err error, req request) int {
	var missing buildinfo.MissingTargetsError
	var noGoFiles buildinfo.NoGoFilesMatchedError
	var targetPath buildinfo.TargetPathError

	response := errorResponse{
		Error:   err.Error(),
		Kind:    errorKindInternal,
		Root:    req.root,
		Targets: append([]string(nil), req.targets...),
	}
	message := fmt.Sprintf("vrooli-buildmeta: %v (root=%q targets=%s)", err, req.root, strings.Join(req.targets, ","))
	exitCode := exitCodeInternal

	switch {
	case errors.As(err, &missing):
		response.Kind = errorKindMissingTargets
		response.Error = fmt.Sprintf("requested targets do not exist: %s", strings.Join(missing.Targets, ","))
		message = fmt.Sprintf("vrooli-buildmeta: requested targets do not exist: %s (root=%q targets=%s)",
			strings.Join(missing.Targets, ","),
			req.root,
			strings.Join(req.targets, ","),
		)
		exitCode = exitCodeValidation
	case errors.As(err, &noGoFiles):
		response.Kind = errorKindNoGoFiles
		response.Error = fmt.Sprintf("requested targets do not contain any Go files: %s", strings.Join(noGoFiles.Targets, ","))
		message = fmt.Sprintf("vrooli-buildmeta: requested targets do not contain any Go files: %s (root=%q targets=%s)",
			strings.Join(noGoFiles.Targets, ","),
			req.root,
			strings.Join(req.targets, ","),
		)
		exitCode = exitCodeValidation
	case errors.As(err, &targetPath):
		response.Kind = errorKindInvalidTarget
		response.Error = fmt.Sprintf("invalid target %q: %v", targetPath.Target, err)
		message = fmt.Sprintf("vrooli-buildmeta: invalid target %q: %v (root=%q targets=%s)",
			targetPath.Target,
			err,
			req.root,
			strings.Join(req.targets, ","),
		)
		exitCode = exitCodeValidation
	}

	if req.json {
		if encodeErr := writeJSON(stdout, response); encodeErr != nil {
			if writeErr := writeLine(stderr, fmt.Sprintf("vrooli-buildmeta: encode JSON error output: %v", encodeErr)); writeErr != nil {
				return exitCodeInternal
			}
			return exitCodeInternal
		}
		return exitCode
	}

	if writeErr := writeLine(stderr, message); writeErr != nil {
		return exitCodeInternal
	}
	return exitCode
}

func writeUsage(stderr io.Writer, flags *flag.FlagSet) error {
	if err := writeLine(stderr, "Usage: vrooli-buildmeta --root <repo-root> <relative-path> [<relative-path>...]"); err != nil {
		return err
	}
	if err := writeLine(stderr, "Computes deterministic Go-source build metadata for Vrooli binaries."); err != nil {
		return err
	}
	if err := writeLine(stderr, "All requested paths must exist and must include at least one Go source file."); err != nil {
		return err
	}

	var usageErr error
	flags.VisitAll(func(f *flag.Flag) {
		if usageErr != nil {
			return
		}
		defValue := fmt.Sprintf(" (default %q)", f.DefValue)
		if f.DefValue == "" {
			defValue = ""
		}
		usageErr = writeLine(stderr, fmt.Sprintf("  -%s: %s%s", f.Name, f.Usage, defValue))
	})
	return usageErr
}

func writeLine(w io.Writer, line string) error {
	_, err := io.WriteString(w, line+"\n")
	return err
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
