package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"react-vite-temporal-model/internal/artifact"
	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/discovery"
	"react-vite-temporal-model/internal/filesystem"
	"react-vite-temporal-model/internal/quint"
)

func Run(ctx context.Context, args []string, stdout io.Writer, _ io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp(stdout)
		return nil
	}
	command := args[0]
	flags, err := parseFlags(args[1:])
	if err != nil {
		return err
	}
	root, err := filepath.Abs(flags.root)
	if err != nil {
		return err
	}
	contracts, err := discovery.FindContracts(root)
	if err != nil {
		return err
	}
	selected := discovery.Filter(contracts, flags.flow)
	if flags.flow != "" && len(selected) == 0 {
		return fmt.Errorf("unknown flow id %s", flags.flow)
	}

	switch command {
	case "list":
		for _, c := range selected {
			fmt.Fprintln(stdout, c.FlowID)
		}
		return nil
	case "validate":
		for _, c := range selected {
			fmt.Fprintf(stdout, "valid %s\n", c.FlowID)
		}
		return nil
	case "generate":
		return generate(ctx, stdout, root, selected, false)
	case "check":
		return generate(ctx, stdout, root, selected, true)
	case "explain":
		if flags.flow == "" {
			return fmt.Errorf("explain requires --flow <flow-id>")
		}
		return explain(stdout, selected[0])
	default:
		return fmt.Errorf("unknown command %s", command)
	}
}

type flags struct {
	root string
	flow string
}

func parseFlags(args []string) (flags, error) {
	out := flags{root: "../.."}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--root":
			if i+1 >= len(args) || args[i+1] == "" {
				return out, fmt.Errorf("--root requires a path")
			}
			out.root = args[i+1]
			i++
		case "--flow":
			if i+1 >= len(args) || args[i+1] == "" {
				return out, fmt.Errorf("--flow requires a flow id")
			}
			out.flow = args[i+1]
			i++
		default:
			return out, fmt.Errorf("unknown argument %s", args[i])
		}
	}
	return out, nil
}

func generate(ctx context.Context, stdout io.Writer, root string, contracts []contract.Contract, check bool) error {
	runner := quint.ExecRunner{}
	version, err := runner.Run(ctx, quint.Command{Args: []string{"quint", "--version"}, Dir: root})
	if err != nil {
		return err
	}
	quintVersion := trim(version.Stdout)
	if quintVersion == "" {
		return fmt.Errorf("quint --version returned an empty version")
	}
	wrote := 0
	for _, c := range contracts {
		rendered := quint.Render(c)
		modelPath := filesystem.Abs(root, c.Outputs.ModelPath)
		if check {
			if err := artifact.AssertFresh(modelPath, []byte(rendered), c.FlowID); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(modelPath), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(modelPath, []byte(rendered), 0o644); err != nil {
				return err
			}
		}
		built, err := artifact.Build(ctx, c, artifact.BuildOptions{
			Root:         root,
			Rendered:     rendered,
			QuintVersion: quintVersion,
			RunQuint:     true,
			Runner:       runner,
		})
		if err != nil {
			return err
		}
		data, err := artifact.CanonicalJSON(built)
		if err != nil {
			return err
		}
		artifactPath := filesystem.Abs(root, c.Outputs.ArtifactPath)
		if check {
			if err := artifact.AssertFresh(artifactPath, data, c.FlowID); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "fresh %s\n", c.FlowID)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(artifactPath, data, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "wrote %s\n", c.Outputs.ModelPath)
		fmt.Fprintf(stdout, "wrote %s\n", c.Outputs.ArtifactPath)
		wrote++
	}
	if !check {
		fmt.Fprintf(stdout, "generated %d temporal flow(s)\n", wrote)
	}
	return nil
}

func explain(stdout io.Writer, flow contract.Contract) error {
	fmt.Fprintf(stdout, "flow: %s\n", flow.FlowID)
	fmt.Fprintf(stdout, "contract: %s\n", flow.ContractPath)
	fmt.Fprintf(stdout, "model: %s\n", flow.Outputs.ModelPath)
	fmt.Fprintf(stdout, "artifact: %s\n", flow.Outputs.ArtifactPath)
	fmt.Fprintf(stdout, "states: %d\n", len(flow.States))
	fmt.Fprintf(stdout, "events: %d\n", len(flow.Events))
	fmt.Fprintf(stdout, "expanded transitions: %d\n", len(flow.ExpandedTransitions))
	fmt.Fprintf(stdout, "named traces: %d\n", len(flow.Traces))
	return nil
}

func trim(value string) string {
	for len(value) > 0 && (value[len(value)-1] == '\n' || value[len(value)-1] == '\r' || value[len(value)-1] == ' ' || value[len(value)-1] == '\t') {
		value = value[:len(value)-1]
	}
	return value
}

func printHelp(stdout io.Writer) {
	fmt.Fprintln(stdout, "Usage: go run . <list|validate|generate|check|explain> [--root <path>] [--flow <flow-id>]")
}
