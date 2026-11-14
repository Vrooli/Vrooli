package glossary

import (
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	glossaryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/glossary"
	"google.golang.org/protobuf/proto"
	"vrooli-onboarding/cli/internal/support"
)

const searchProcedure = "/vrooli.vrooli_onboarding.v1.glossary.GlossaryService/SearchGlossary"

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{Title: "Glossary", Commands: []cliapp.Command{{Name: "glossary", Description: "Look up Vrooli glossary terms", NeedsAPI: true, Run: func(args []string) error { return run(core, args) }}}}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("glossary")
	query := fs.String("query", "", "Optional search term")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	response := new(glossaryv1.SearchGlossaryResponse)
	if err := support.RequestProto(core, searchProcedure, &glossaryv1.SearchGlossaryRequest{Query: *query}, response, "glossary"); err != nil {
		return err
	}
	if *jsonOutput {
		return printJSON(response)
	}
	summary := []string{fmt.Sprintf("Glossary entries: %d", response.GetCount())}
	if response.GetQuery() != "" {
		summary = append(summary, fmt.Sprintf("Query: %q", response.GetQuery()))
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{Summary: summary, ResultsHeading: "Terms", Results: rows(response.GetEntries()), RetrievalHints: []string{fmt.Sprintf("%s glossary --query postgres", support.CLIName), fmt.Sprintf("%s resources list", support.CLIName)}})
}

func printJSON(message proto.Message) error {
	return support.PrintProto(os.Stdout, message, "glossary")
}

func rows(entries []*glossaryv1.GlossaryEntry) []string {
	if len(entries) == 0 {
		return []string{"(no matching glossary entries)"}
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, fmt.Sprintf("%s [%s] -> %s", entry.GetTerm(), entry.GetCategory(), entry.GetDescription()))
	}
	return out
}
