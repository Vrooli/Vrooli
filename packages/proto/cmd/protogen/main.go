package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/vrooli/packages/proto/protogen"
)

func main() {
	ctx, stop := interruptContext()
	err := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// interruptContext turns the first SIGINT or SIGTERM into cancellation, so an
// interrupted run unwinds through its deferred cleanup (stopping buf and
// removing its staging tree) instead of dying mid-write. A second signal gets
// the default behaviour, so an operator can still force an exit.
func interruptContext() (context.Context, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		stop()
	}()
	return ctx, stop
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: protogen <generate|verify|descriptor|artifact|lint|format|breaking|clean|cleanup|refresh-vendor>")
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
		config.PublishArtifacts = true
		generator, err := protogen.New(config)
		if err != nil {
			return err
		}
		return generator.Generate(ctx)
	case "verify":
		config := protogen.DefaultConfig(repoRoot)
		config.ProtoRoot = protoRoot
		config.Logger = stdout
		generator, err := protogen.New(config)
		if err != nil {
			return err
		}
		return generator.Verify(ctx)
	case "descriptor":
		config := protogen.DefaultConfig(repoRoot)
		config.ProtoRoot = protoRoot
		generator, err := protogen.New(config)
		if err != nil {
			return err
		}
		return generator.Descriptor(ctx)
	case "artifact":
		return runArtifactCommand(ctx, args[1:], stdout, stderr)
	case "lint":
		return runBuf(ctx, protoRoot, "lint")
	case "format":
		return runBuf(ctx, protoRoot, "format", "-w")
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
		return runExternal(ctx, "proto-health", "impact", "scenario", *scenario)
	case "clean":
		// Staging trees are protogen's own output too. Leaving them behind made
		// the one command named "clean" the one command that did not remove the
		// largest thing protogen writes. Live trees of a concurrent run survive.
		parents := append([]string{protogen.StagingRoot(protogen.ResolveArtifactRoot())}, protogen.InTreeStageParents(protoRoot)...)
		if err := protogen.CleanStages(parents...); err != nil {
			return err
		}
		return protogen.Clean(filepath.Join(protoRoot, "gen"))
	case "cleanup":
		fs := flag.NewFlagSet(command, flag.ContinueOnError)
		fs.SetOutput(stderr)
		var scenarios scenarioFlags
		fs.Var(&scenarios, "scenario", "schema owner to narrow cleanup (repeatable)")
		apply := fs.Bool("apply", false, "apply deterministic cleanup; default is dry-run")
		dryRun := fs.Bool("dry-run", false, "preview cleanup without writing (the default)")
		jsonOutput := fs.Bool("json", false, "write machine-readable output")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *apply && *dryRun {
			return fmt.Errorf("cleanup: --apply and --dry-run are mutually exclusive")
		}
		plan, err := protogen.PlanCleanup(protogen.CleanupOptions{
			RepoRoot: repoRoot, ProtoRoot: protoRoot, Scenarios: scenarios, DryRun: !*apply,
		})
		if err != nil {
			return err
		}
		result := protogen.CleanupResult{Plan: plan}
		if *apply {
			result, err = protogen.ApplyCleanup(ctx, protogen.CleanupOptions{
				RepoRoot: repoRoot, ProtoRoot: protoRoot, Scenarios: scenarios, DryRun: false,
			}, plan, stdout)
			if err != nil {
				return err
			}
		}
		if *jsonOutput {
			return json.NewEncoder(stdout).Encode(result)
		}
		return writeCleanupReport(stdout, result)
	case "refresh-vendor":
		if err := runBuf(ctx, protoRoot, "export", "buf.build/googleapis/googleapis", "-o", filepath.Join(protoRoot, "vendor", "googleapis")); err != nil {
			return err
		}
		return runBuf(ctx, protoRoot, "export", "buf.build/bufbuild/protovalidate", "-o", filepath.Join(protoRoot, "vendor", "protovalidate"))
	default:
		return fmt.Errorf("unknown protogen command %q", command)
	}
}

func writeCleanupReport(stdout io.Writer, result protogen.CleanupResult) error {
	verb := "would"
	if result.Applied {
		verb = "did"
	}
	fmt.Fprintf(stdout, "protogen cleanup: %s apply %d deterministic action(s)\n", verb, len(result.Plan.Actions))
	for _, action := range result.Plan.Actions {
		if action.Kind == "regenerate_generated_tree" {
			fmt.Fprintf(stdout, "  %s regenerate %s (%s)\n", verb, action.Path, action.Reason)
			continue
		}
		operation := "remove"
		if action.Kind == "empty_schema_owner" {
			operation = "remove empty owner"
		}
		fmt.Fprintf(stdout, "  %s %s %s (%s)\n", verb, operation, action.Path, action.Reason)
	}
	for _, note := range result.Plan.Notes {
		fmt.Fprintf(stdout, "  note: %s\n", note)
	}
	if result.GenerationRan {
		fmt.Fprintln(stdout, "  generation: completed")
	}
	return nil
}

func runArtifactCommand(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: protogen artifact <status|promote|rollback|cleanup>")
	}
	fs := flag.NewFlagSet("artifact "+args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "runtime artifact root (defaults to $"+protogen.ArtifactRootEnv+", then ~/.vrooli/artifacts/proto)")
	artifactID := fs.String("artifact", "", "validated artifact id")
	jsonOutput := fs.Bool("json", false, "write machine-readable JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	artifactRoot := *root
	if strings.TrimSpace(artifactRoot) == "" {
		artifactRoot = protogen.ResolveArtifactRoot()
	}
	store, err := protogen.NewArtifactStore(artifactRoot)
	if err != nil {
		return err
	}
	switch args[0] {
	case "status":
		status, err := store.Status(ctx)
		if err != nil {
			return err
		}
		if *jsonOutput {
			return json.NewEncoder(stdout).Encode(status)
		}
		fmt.Fprintf(stdout, "root=%s\nactive=%s valid=%t\nlast_known_good=%s valid=%t\nsnapshots=%d leases=%d\n", status.Root, status.Active.Artifact, status.Active.Valid, status.LastKnownGood.Artifact, status.LastKnownGood.Valid, len(status.Snapshots), len(status.Leases))
		if status.Active.Error != "" {
			fmt.Fprintf(stdout, "active_error=%s\n", status.Active.Error)
		}
		if status.LastKnownGood.Error != "" {
			fmt.Fprintf(stdout, "last_known_good_error=%s\n", status.LastKnownGood.Error)
		}
		return nil
	case "promote", "rollback":
		if strings.TrimSpace(*artifactID) == "" {
			return fmt.Errorf("artifact %s requires --artifact", args[0])
		}
		if err := store.Promote(ctx, *artifactID); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "artifact=%s operation=%s status=promoted\n", *artifactID, args[0])
		return nil
	case "cleanup":
		removed, err := store.Collect(ctx, time.Now())
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "removed=%d\n", removed)
		return nil
	default:
		return fmt.Errorf("unknown artifact command %q", args[0])
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
