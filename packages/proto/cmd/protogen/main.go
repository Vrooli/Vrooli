package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/packages/proto/protogen"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: protogen <generate|verify|descriptor|lint|format|breaking|clean|refresh-vendor>")
	}
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}
	protoRoot, err := resolveProtoRoot(workingDir)
	if err != nil {
		return err
	}
	repoRoot := filepath.Clean(filepath.Join(protoRoot, "..", ".."))
	command := args[0]
	switch command {
	case "generate", "gen-code", "gen-manifests":
		fs := flag.NewFlagSet(command, flag.ContinueOnError)
		fs.SetOutput(stderr)
		var scenarios scenarioFlags
		changed := fs.Bool("changed", false, "derive scope from per-scenario generation locks")
		fs.Var(&scenarios, "scenario", "scenario to regenerate (repeatable)")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		config := protogen.DefaultConfig(repoRoot)
		config.ProtoRoot = protoRoot
		config.Scenarios = scenarios
		config.Changed = *changed
		config.Logger = stdout
		generator, err := protogen.New(config)
		if err != nil {
			return err
		}
		return generator.Generate(context.Background())
	case "verify":
		config := protogen.DefaultConfig(repoRoot)
		config.ProtoRoot = protoRoot
		config.Logger = stdout
		generator, err := protogen.New(config)
		if err != nil {
			return err
		}
		return generator.Verify(context.Background())
	case "descriptor":
		config := protogen.DefaultConfig(repoRoot)
		config.ProtoRoot = protoRoot
		generator, err := protogen.New(config)
		if err != nil {
			return err
		}
		return generator.Descriptor(context.Background())
	case "lint":
		return runBuf(context.Background(), protoRoot, "lint")
	case "format":
		return runBuf(context.Background(), protoRoot, "format", "-w")
	case "breaking":
		fs := flag.NewFlagSet(command, flag.ContinueOnError)
		fs.SetOutput(stderr)
		scenario := fs.String("scenario", "", "scenario to inspect")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if strings.TrimSpace(*scenario) == "" {
			return fmt.Errorf("breaking requires --scenario")
		}
		return runExternal(context.Background(), "proto-health", "impact", "scenario", *scenario)
	case "clean":
		// Staging trees are protogen's own output too. Leaving them behind made
		// the one command named "clean" the one command that did not remove the
		// largest thing protogen writes.
		if err := protogen.CleanStages(protogen.DefaultStageParent(protoRoot)); err != nil {
			return err
		}
		return protogen.Clean(filepath.Join(protoRoot, "gen"))
	case "refresh-vendor":
		if err := runBuf(context.Background(), protoRoot, "export", "buf.build/googleapis/googleapis", "-o", filepath.Join(protoRoot, "vendor", "googleapis")); err != nil {
			return err
		}
		return runBuf(context.Background(), protoRoot, "export", "buf.build/bufbuild/protovalidate", "-o", filepath.Join(protoRoot, "vendor", "protovalidate"))
	default:
		return fmt.Errorf("unknown protogen command %q", command)
	}
}

// resolveProtoRoot accepts either the repository root, packages/proto, or a
// descendant of packages/proto. The old basename-only fallback appended
// packages/proto to a packages/ directory and could create the nested
// packages/proto/packages/proto output tree after a caller changed directory.
func resolveProtoRoot(start string) (string, error) {
	current := filepath.Clean(start)
	for {
		if filepath.Base(current) == "proto" && isProtoRoot(current) {
			return current, nil
		}
		candidate := filepath.Join(current, "packages", "proto")
		if isProtoRoot(candidate) {
			return candidate, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", fmt.Errorf("locate packages/proto from %s", start)
}

func isProtoRoot(path string) bool {
	for _, marker := range []string{"buf.yaml", "schemas"} {
		if _, err := os.Stat(filepath.Join(path, marker)); err != nil {
			return false
		}
	}
	return true
}

type scenarioFlags []string

func (f *scenarioFlags) String() string { return strings.Join(*f, ",") }

func (f *scenarioFlags) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("scenario cannot be empty")
	}
	*f = append(*f, value)
	return nil
}

func runBuf(ctx context.Context, dir string, args ...string) error {
	return runExternalInDir(ctx, dir, "buf", args...)
}

func runExternal(ctx context.Context, name string, args ...string) error {
	return runExternalInDir(ctx, "", name, args...)
}

func runExternalInDir(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
