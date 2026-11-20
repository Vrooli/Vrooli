package annotations

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// goldDraft is the blind packet the API returns for one label. It never
// carries the stored answer, so a reviewer cannot anchor on the model.
type goldDraft struct {
	Label          string         `json:"label"`
	DefinitionHash string         `json:"definition_hash"`
	Window         string         `json:"window"`
	MinGold        int            `json:"min_gold"`
	Limit          int            `json:"limit"`
	Question       *goldQuestion  `json:"question,omitempty"`
	Runs           []goldDraftRun `json:"runs"`
}

type goldQuestion struct {
	Type         string   `json:"type"`
	Instructions string   `json:"instructions"`
	Levels       []string `json:"levels,omitempty"`
}

type goldDraftRun struct {
	RunID  string `json:"run_id"`
	Digest string `json:"digest"`
}

// goldRow is one line of the editable gold file. Positive stays null until a
// reviewer decides; the benchmark refuses an unfilled row instead of silently
// dropping it.
type goldRow struct {
	RunID          string `json:"run_id"`
	Label          string `json:"label"`
	Positive       *bool  `json:"positive"`
	DefinitionHash string `json:"definition_hash,omitempty"`
	Digest         string `json:"digest,omitempty"`
}

func goldCommand() cliapp.Command {
	return cliapp.Command{
		Name: "gold", NeedsAPI: true,
		Description: "Write a blind, unlabelled gold-set draft for one label",
		Usage:       "agent-manager annotations gold --label <id> [--window 30d] [--limit 20] [--out <file>] [--json]",
		HelpText: "Fetches the bounded state of runs already observed under the label's current definition, without the stored answer. The packet is wider than the model digest (more messages, a longer last error) so a person can judge the row; blindness protects the stored answer, not the evidence. " +
			"Each JSONL row starts with \"positive\":null; set it to true or false from the evidence, then run " +
			"`agent-manager annotations benchmark --gold <file> --label <id>`. Unfilled rows are rejected so an empty draft cannot produce a gate.",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "label", Required: true, Description: "Label id to build a gold set for"},
			{Name: "window", Default: "30d", Description: "Observation window (7d, 30d, all or RFC3339)"},
			{Name: "limit", Default: "0", Description: "Runs to include; 0 uses the label's gate size (min_gold, capped at 100)"},
			{Name: "out", Description: "Write JSONL rows to this file instead of stdout"},
			{Name: "json", Bool: true, LocalOnly: true, Description: "Print the full draft packet instead of JSONL rows"},
		}},
		RunCtx: func(ctx cliapp.RunContext) error {
			limit := 0
			if raw := strings.TrimSpace(ctx.Flag("limit")); raw != "" && raw != "0" {
				parsed, err := parsePositiveInt(raw, "limit")
				if err != nil {
					return err
				}
				limit = parsed
			}
			query := "label=" + url.QueryEscape(strings.TrimSpace(ctx.Flag("label"))) +
				"&window=" + url.QueryEscape(strings.TrimSpace(ctx.Flag("window"))) +
				"&limit=" + strconv.Itoa(limit)
			response, err := ctx.Core().Request("GET", "annotations/gold?"+query, nil, nil)
			if err != nil {
				return err
			}
			if ctx.JSON() {
				_, err = ctx.Stdout().Write(append(response, '\n'))
				return err
			}
			var draft goldDraft
			if err := json.Unmarshal(response, &draft); err != nil {
				return err
			}
			out := strings.TrimSpace(ctx.Flag("out"))
			var writer io.Writer = ctx.Stdout()
			if out != "" {
				// A gold draft carries transcript excerpts; keep it private.
				file, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
				if err != nil {
					return err
				}
				defer file.Close()
				writer = file
			}
			if err := writeGoldDraft(writer, draft); err != nil {
				return err
			}
			target := out
			if target == "" {
				target = "<stdout>"
			}
			fmt.Fprintf(ctx.Stderr(), "Wrote %d unlabelled row(s) for %s to %s (gate needs %d).\nSet \"positive\" to true or false on every row, then run:\n  agent-manager annotations benchmark --gold %s --label %s\n",
				len(draft.Runs), draft.Label, target, draft.MinGold, target, draft.Label)
			if draft.MinGold > 0 && len(draft.Runs) < draft.MinGold {
				fmt.Fprintf(ctx.Stderr(), "Only %d run(s) carry a current observation in this window; widen --window or annotate more runs before expecting a gate.\n", len(draft.Runs))
			}
			return nil
		},
	}
}

func writeGoldDraft(out io.Writer, draft goldDraft) error {
	encoder := json.NewEncoder(out)
	for _, run := range draft.Runs {
		row := goldRow{
			RunID: run.RunID, Label: draft.Label,
			DefinitionHash: draft.DefinitionHash, Digest: run.Digest,
		}
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}
	return nil
}
