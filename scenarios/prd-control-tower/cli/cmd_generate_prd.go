package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

type GeneratePRDOutput struct {
	Generation AIGenerateDraftResponse `json:"generation"`
	Publish    *PublishResponse        `json:"publish,omitempty"`
}

func (a *App) cmdGeneratePRD(args []string) error {
	fs := flag.NewFlagSet("generate-prd", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	entityType := fs.String("type", "scenario", "Entity type: scenario or resource")
	context := fs.String("context", "", "Context for AI generation")
	contextFile := fs.String("context-file", "", "Path to a file containing context for AI generation")
	model := fs.String("model", "", "Override OpenRouter model (e.g. openrouter/x-ai/grok-code-fast-1)")
	owner := fs.String("owner", "", "Owner metadata for the created/updated draft")

	templateName := fs.String("template", "", "If set, publish the generated PRD into a scenario created from this template")
	force := fs.Bool("force", false, "Allow overwriting an existing scenario when publishing with --template")
	runHooks := fs.Bool("run-hooks", false, "Run template hooks when publishing with --template")
	noSaveDraft := fs.Bool("no-save-draft", false, "Do not persist generated content into the draft (default: persist)")

	remaining, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return fmt.Errorf("usage: generate-prd <name> [--type scenario|resource] [--context ...|--context-file ...] [--template name] [--json]")
	}

	name := strings.TrimSpace(remaining[0])
	if name == "" {
		return fmt.Errorf("usage: generate-prd <name> [--type scenario|resource] [--context ...|--context-file ...] [--template name] [--json]")
	}

	resolvedType := strings.ToLower(strings.TrimSpace(*entityType))
	if resolvedType != "scenario" && resolvedType != "resource" {
		return fmt.Errorf("invalid --type %q (must be scenario or resource)", *entityType)
	}

	finalContext := strings.TrimSpace(*context)
	if *contextFile != "" {
		data, err := os.ReadFile(*contextFile)
		if err != nil {
			return fmt.Errorf("read --context-file: %w", err)
		}
		fileContext := strings.TrimSpace(string(data))
		if finalContext == "" {
			finalContext = fileContext
		} else if fileContext != "" {
			finalContext = finalContext + "\n\n" + fileContext
		}
	}

	save := true
	if *noSaveDraft {
		save = false
	}

	req := AIGenerateDraftRequest{
		EntityType: resolvedType,
		EntityName: name,
		Owner:      strings.TrimSpace(*owner),
		Section:    "🎯 Full PRD",
		Context:    finalContext,
		Action:     "",
		Model:      strings.TrimSpace(*model),
	}
	req.SaveGeneratedToDraft = &save

	genBody, genResp, err := a.services.AI.GenerateDraft(req)
	if err != nil {
		return err
	}
	if !genResp.Success {
		if *jsonOutput {
			cliutil.PrintJSON(genBody)
			return nil
		}
		if strings.TrimSpace(genResp.Message) != "" {
			return fmt.Errorf("AI generation failed: %s", genResp.Message)
		}
		return fmt.Errorf("AI generation failed")
	}

	var publishParsed *PublishResponse
	if strings.TrimSpace(*templateName) != "" {
		pubReq := PublishRequest{
			CreateBackup: true,
			DeleteDraft:  true,
			Template: &PublishTemplateRequest{
				Name: strings.TrimSpace(*templateName),
				Variables: map[string]string{
					"SCENARIO_ID": name,
				},
				Force:    *force,
				RunHooks: *runHooks,
			},
		}
		pubBody, pubResp, err := a.services.Drafts.Publish(genResp.DraftID, pubReq)
		if err != nil {
			return err
		}
		if !pubResp.Success {
			if *jsonOutput {
				out := GeneratePRDOutput{Generation: genResp, Publish: &pubResp}
				data, _ := json.MarshalIndent(out, "", "  ")
				fmt.Println(string(data))
				return nil
			}
			if strings.TrimSpace(pubResp.Message) != "" {
				return fmt.Errorf("publish failed: %s", pubResp.Message)
			}
			return fmt.Errorf("publish failed")
		}
		_ = pubBody
		publishParsed = &pubResp
	}

	if *jsonOutput {
		out := GeneratePRDOutput{Generation: genResp, Publish: publishParsed}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return fmt.Errorf("encode output: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Generated draft %s for %s/%s\n", genResp.DraftID, genResp.EntityType, genResp.EntityName)
	if genResp.DraftFilePath != "" {
		fmt.Printf("Draft file: %s\n", genResp.DraftFilePath)
	}
	if publishParsed != nil {
		fmt.Printf("Published to: %s\n", publishParsed.PublishedTo)
		if publishParsed.CreatedScenario && publishParsed.ScenarioPath != "" {
			fmt.Printf("Scenario created: %s\n", publishParsed.ScenarioPath)
		}
	}
	return nil
}
