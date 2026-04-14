package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/shell"
)

const (
	vrooliVersion = "2.0.0"
	cliVersion    = "1.0.0"
)

var (
	resolveSourceRootFn = buildinfo.ResolveSourceRoot
	checkStalenessFn    = buildinfo.CheckStaleness
	rebuildAndReexecFn  = buildinfo.RebuildAndReexec
	lookPathFn          = shell.LookPath
	newLoggerFn         = createCommandLogger
)

type globalOptions = rootcli.GlobalOptions
type parsedArgs = rootcli.ParsedArgs

type versionOutput struct {
	CLIVersion      string `json:"cli_version"`
	PlatformVersion string `json:"platform_version"`
	Root            string `json:"root"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return configuredRunner().Run(args, stdout, stderr)
}

func primeRootEnv(root string) {
	_ = os.Setenv("VROOLI_ROOT", root)
	if strings.TrimSpace(os.Getenv(buildinfo.SourceRootEnvVar)) == "" {
		_ = os.Setenv(buildinfo.SourceRootEnvVar, root)
	}
}

func passthroughFlags(globals globalOptions, existing []string) []string {
	return rootcli.PassthroughFlags(globals, existing)
}

func containsArg(args []string, target string) bool {
	return rootcli.ContainsArg(args, target)
}

func setEnvValue(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			updated := append([]string(nil), env...)
			updated[i] = prefix + value
			return updated
		}
	}
	return append(append([]string(nil), env...), prefix+value)
}

func showVersion(w io.Writer, root string, globals globalOptions) error {
	format, err := formatFromJSON(globals.JSON)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, versionOutput{
			CLIVersion:      cliVersion,
			PlatformVersion: vrooliVersion,
			Root:            root,
		})
	}
	_, _ = fmt.Fprintf(w, "Vrooli CLI v%s\n", cliVersion)
	_, _ = fmt.Fprintf(w, "Vrooli Platform v%s\n", vrooliVersion)
	_, _ = fmt.Fprintf(w, "Root: %s\n", root)
	return nil
}
