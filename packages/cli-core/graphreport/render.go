// Package graphreport shares human-readable rendering for language graph CLIs.
package graphreport

import (
	"fmt"
	"sort"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"
)

type Extraction struct {
	Nodes, Edges, Warnings, ElapsedMS int
	GraphHash, Path, RewriteFlag      string
	RewriteTarget                     string
	Packages                          []string
	ShowWarningsAlways                bool
}

func RenderExtraction(ctx cliapp.RunContext, payload proto.Message, result Extraction) error {
	packages := append([]string(nil), result.Packages...)
	sort.Strings(packages)
	if len(packages) > 10 {
		packages = packages[:10]
	}
	summary := []string{}
	if result.ShowWarningsAlways {
		summary = append(summary, fmt.Sprintf("Extracted %d node(s), %d edge(s), %d warning(s) in %dms (hash=%s).", result.Nodes, result.Edges, result.Warnings, result.ElapsedMS, result.GraphHash))
	} else {
		summary = append(summary, fmt.Sprintf("Extracted %d node(s), %d edge(s) in %dms (hash=%s).", result.Nodes, result.Edges, result.ElapsedMS, result.GraphHash))
		if result.Warnings > 0 {
			summary = append(summary, fmt.Sprintf("%d warning(s) returned.", result.Warnings))
		}
	}
	return cliapp.RenderProtoList(ctx, payload, cliapp.ListReport{
		Summary: summary, ResultsHeading: "Top-level packages", Results: packages,
		RetrievalHints: []string{fmt.Sprintf("`rewrite plan <ops.json> --%s %s` — plan a rewrite for this %s", result.RewriteFlag, result.Path, result.RewriteTarget)},
	})
}

type Fixture struct {
	Name, Path  string
	HasExpected bool
}

func RenderFixtures(ctx cliapp.RunContext, payload proto.Message, fixtures []Fixture) error {
	results := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		status := "no expected graph"
		if fixture.HasExpected {
			status = "expected graph present"
		}
		results = append(results, fmt.Sprintf("%s (%s) — %s", fixture.Name, fixture.Path, status))
	}
	sort.Strings(results)
	return cliapp.RenderProtoList(ctx, payload, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d fixture(s) available.", len(fixtures))},
		ResultsHeading: "Fixtures", Results: results,
		RetrievalHints: []string{"`graph validate-fixture <name>` — re-extract and byte-compare a fixture"},
	})
}

type Validation struct {
	Name, Diff, GraphHash      string
	ExpectedBytes, ActualBytes int64
	Passed                     bool
}

func RenderValidation(ctx cliapp.RunContext, payload proto.Message, result Validation) error {
	verdict := "PASS"
	if !result.Passed {
		verdict = "FAIL"
	}
	summary := []string{fmt.Sprintf("%s — fixture %q (expected %d bytes, actual %d bytes, hash=%s)", verdict, result.Name, result.ExpectedBytes, result.ActualBytes, result.GraphHash)}
	var details []string
	if !result.Passed && result.Diff != "" {
		details = append(details, result.Diff)
	}
	if err := cliapp.RenderProtoList(ctx, payload, cliapp.ListReport{Summary: summary, ResultsHeading: "Diff (expected vs actual)", Results: details}); err != nil {
		return err
	}
	if !result.Passed {
		return fmt.Errorf("fixture %q does not match its expected graph", result.Name)
	}
	return nil
}
