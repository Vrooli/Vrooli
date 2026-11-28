package support

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const CLIName = "scenario-stack-governor"

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func ParseFlags(fs *flag.FlagSet, args []string) error {
	return cliutil.ParseInterspersed(fs, args)
}

func GetJSON[T any](core *cliapp.ScenarioApp, path string, query url.Values, out *T) error {
	body, err := core.Get(path, query)
	if err != nil {
		return err
	}
	return Decode(body, out)
}

func RequestJSON[T any](core *cliapp.ScenarioApp, method, path string, query url.Values, body any, out *T) error {
	resp, err := core.Request(method, path, query, body)
	if err != nil {
		return err
	}
	return Decode(resp, out)
}

func Decode(body []byte, dest any) error {
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

func PrintList(jsonOutput bool, report any, human cliapp.ListReport) error {
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, human)
}

func PrintOperational(jsonOutput bool, report any, human cliapp.OperationalReport) error {
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, human)
}

func PrintMutation(jsonOutput bool, report any, human cliapp.MutationReport) error {
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, human)
}

func ParseMultiValue(csv string, repeated []string, positional []string) []string {
	combined := append([]string{}, cliutil.ParseCSV(csv)...)
	for _, value := range repeated {
		combined = append(combined, cliutil.ParseCSV(value)...)
	}
	combined = append(combined, positional...)
	seen := make(map[string]struct{}, len(combined))
	out := make([]string, 0, len(combined))
	for _, value := range combined {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func SeverityRank(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 0
	case "high", "error":
		return 1
	case "medium", "warning", "warn":
		return 2
	case "low":
		return 3
	case "info":
		return 4
	default:
		return 5
	}
}

func SortRules(rules []Rule) {
	sort.SliceStable(rules, func(i, j int) bool {
		if SeverityRank(rules[i].Severity) != SeverityRank(rules[j].Severity) {
			return SeverityRank(rules[i].Severity) < SeverityRank(rules[j].Severity)
		}
		if rules[i].Category != rules[j].Category {
			return rules[i].Category < rules[j].Category
		}
		return rules[i].ID < rules[j].ID
	})
}

func SortRuleResults(results []RuleResult) {
	sort.SliceStable(results, func(i, j int) bool {
		leftRank := 0
		if results[i].Passed {
			leftRank = 1
		}
		rightRank := 0
		if results[j].Passed {
			rightRank = 1
		}
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return results[i].RuleID < results[j].RuleID
	})
}

func FindRuleByID(rules []Rule, id string) *Rule {
	id = strings.TrimSpace(id)
	for i := range rules {
		if rules[i].ID == id {
			return &rules[i]
		}
	}
	return nil
}

func EnabledRuleIDs(rules []Rule) []string {
	ids := make([]string, 0, len(rules))
	for _, rule := range rules {
		if rule.Enabled {
			ids = append(ids, rule.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func DefaultEnabledRuleIDs(rules []Rule) []string {
	ids := make([]string, 0, len(rules))
	for _, rule := range rules {
		if rule.DefaultEnabled {
			ids = append(ids, rule.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func StatusWord(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}
