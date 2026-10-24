package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"agent-manager/internal/conversationpolicy"

	"github.com/vrooli/cli-core/cliutil"
)

// conversationPolicy is `agent-manager conversation policy {list,add,remove,set,test}`.
// It is a local, file-backed command: the declarative visibility/exclusion
// policy lives in the scenario's conversation-policy.json, which both this CLI
// and the API server read (the server at startup through config.LoadLevers).
// Keeping it local means no server round-trip and no proto surface — editing a
// JSON file of rules is the whole operation — while the server enforces it at
// the import and search seams.
func (a *App) conversationPolicy(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Println("usage: agent-manager conversation policy {list|add|remove|set|test}")
		fmt.Println("  list                 Show the active rules and where each is enforced")
		fmt.Println("  add <flags>          Append a rule (--action --enforce-at --harness --model --project --cwd --tag --name)")
		fmt.Println("  remove <index>       Remove the rule at the given zero-based index")
		fmt.Println("  set <file>           Replace all rules from a JSON document ({\"rules\":[...]})")
		fmt.Println("  test <flags>         Evaluate a sample conversation and print the matching rule + action")
		return nil
	}
	switch args[0] {
	case "list":
		return a.conversationPolicyList(args[1:])
	case "add":
		return a.conversationPolicyAdd(args[1:])
	case "remove":
		return a.conversationPolicyRemove(args[1:])
	case "set":
		return a.conversationPolicySet(args[1:])
	case "test":
		return a.conversationPolicyTest(args[1:])
	default:
		return fmt.Errorf("unknown conversation policy subcommand %q; expected list, add, remove, set, or test", args[0])
	}
}

// conversationPolicyPath resolves the policy file via cliutil.ResolveRepoRoot
// (repo-contract-backed and sandbox-aware), matching every other
// agent-manager CLI command rather than config.RepoRoot's naive "../../.."
// walk. The AGENT_MANAGER_CONVERSATION_POLICY_FILE override still wins inside
// ResolveFilePath, which is how the server and CLI stay pointed at the same
// file in tests and under an explicit override.
func conversationPolicyPath() string {
	return conversationpolicy.ResolveFilePath(cliutil.ResolveRepoRoot())
}

func (a *App) conversationPolicyList(args []string) error {
	fs := flag.NewFlagSet("conversation policy list", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	path := conversationPolicyPath()
	doc, err := conversationpolicy.LoadDocument(path)
	if err != nil {
		return err
	}
	if *jsonOutput {
		payload, marshalErr := json.MarshalIndent(doc, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		cliutil.PrintJSON(payload)
		return nil
	}
	fmt.Printf("Conversation visibility policy: %s\n", path)
	if len(doc.Rules) == 0 {
		fmt.Println("  (no rules — every conversation is imported and searchable)")
		return nil
	}
	policy, err := conversationpolicy.New(doc.Rules)
	if err != nil {
		return err
	}
	fmt.Println("  Rules (first match wins, per enforcement point):")
	for _, line := range policy.DescribeRules() {
		fmt.Printf("    %s\n", line)
	}
	fmt.Println("  Enforcement: import rules drop runs before any events are written; " +
		"search rules hide indexed runs from results. Exclusion counts are logged by the running server.")
	return nil
}

func (a *App) conversationPolicyAdd(args []string) error {
	fs := flag.NewFlagSet("conversation policy add", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	rule := conversationpolicy.Rule{}
	action := fs.String("action", "", "Action: exclude, redact, or index")
	enforceAt := fs.String("enforce-at", "", "Enforcement point: import or search")
	fs.StringVar(&rule.Harness, "harness", "", "Match harness (empty = any)")
	fs.StringVar(&rule.Model, "model", "", "Match model (empty = any)")
	fs.StringVar(&rule.ProjectScope, "project", "", "Match project scope (empty = any)")
	fs.StringVar(&rule.CwdScope, "cwd", "", "Match working-directory scope (empty = any)")
	fs.StringVar(&rule.Tag, "tag", "", "Match tag (empty = any)")
	fs.StringVar(&rule.Name, "name", "", "Optional rule name for observability")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	rule.Action = conversationpolicy.Action(strings.TrimSpace(*action))
	rule.EnforceAt = conversationpolicy.EnforcePoint(strings.TrimSpace(*enforceAt))
	if err := rule.Validate(); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}
	// An import-enforced rule can only observe harness and tag. A model, project,
	// or cwd matcher on an import rule can never fire at import and would
	// silently match nothing, so surface it at add time instead of letting the
	// operator believe the rule is in force. (The rule is still accepted — the
	// operator may want it staged, and the no-silent-failure invariant is met by
	// the warning.)
	if unobservable := rule.ImportUnobservableFields(); len(unobservable) > 0 {
		fmt.Printf("warning: import rules cannot match %s (only harness and tag are known at import); "+
			"this rule will never fire at import — enforce it at search instead\n", strings.Join(unobservable, ", "))
	}
	path := conversationPolicyPath()
	var saved conversationpolicy.Document
	if err := conversationpolicy.MutateDocument(path, func(doc conversationpolicy.Document) (conversationpolicy.Document, error) {
		doc.Rules = append(doc.Rules, rule)
		saved = doc
		return doc, nil
	}); err != nil {
		return err
	}
	return conversationPolicyReport(saved, *jsonOutput, fmt.Sprintf("Added rule %d: %s", len(saved.Rules)-1, conversationpolicy.DescribeRule(rule)))
}

func (a *App) conversationPolicyRemove(args []string) error {
	fs := flag.NewFlagSet("conversation policy remove", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errorsUsage("usage: agent-manager conversation policy remove <index>")
	}
	index, err := strconv.Atoi(strings.TrimSpace(fs.Arg(0)))
	if err != nil || index < 0 {
		return errorsUsage("index must be a non-negative integer")
	}
	path := conversationPolicyPath()
	var saved conversationpolicy.Document
	var removed conversationpolicy.Rule
	if err := conversationpolicy.MutateDocument(path, func(doc conversationpolicy.Document) (conversationpolicy.Document, error) {
		if index >= len(doc.Rules) {
			return doc, fmt.Errorf("index %d out of range: policy has %d rule(s)", index, len(doc.Rules))
		}
		removed = doc.Rules[index]
		doc.Rules = append(doc.Rules[:index], doc.Rules[index+1:]...)
		saved = doc
		return doc, nil
	}); err != nil {
		return err
	}
	return conversationPolicyReport(saved, *jsonOutput, fmt.Sprintf("Removed rule %d: %s", index, conversationpolicy.DescribeRule(removed)))
}

func (a *App) conversationPolicySet(args []string) error {
	fs := flag.NewFlagSet("conversation policy set", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errorsUsage("usage: agent-manager conversation policy set <rules.json>")
	}
	doc, err := conversationpolicy.LoadDocument(strings.TrimSpace(fs.Arg(0)))
	if err != nil {
		return err
	}
	path := conversationPolicyPath()
	if err := conversationpolicy.SaveDocument(path, doc); err != nil {
		return err
	}
	return conversationPolicyReport(doc, *jsonOutput, fmt.Sprintf("Replaced policy with %d rule(s)", len(doc.Rules)))
}

func (a *App) conversationPolicyTest(args []string) error {
	fs := flag.NewFlagSet("conversation policy test", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	harness := fs.String("harness", "", "Sample harness (e.g. ollama, codex, claude-code)")
	model := fs.String("model", "", "Sample model")
	project := fs.String("project", "", "Sample project scope")
	cwd := fs.String("cwd", "", "Sample working-directory scope")
	tag := fs.String("tag", "", "Sample tag")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	path := conversationPolicyPath()
	policy, err := conversationpolicy.LoadPolicy(cliutil.ResolveRepoRoot())
	if err != nil {
		return err
	}
	fields := conversationpolicy.Fields{Harness: *harness, Model: *model, ProjectScope: *project, CwdScope: *cwd, Tag: *tag}
	importDecision := policy.Evaluate(conversationpolicy.EnforceImport, fields)
	searchDecision := policy.Evaluate(conversationpolicy.EnforceSearch, fields)
	if *jsonOutput {
		payload, marshalErr := json.MarshalIndent(map[string]any{
			"path":       path,
			"sample":     fields,
			"import":     conversationPolicyDecisionJSON(importDecision),
			"search":     conversationPolicyDecisionJSON(searchDecision),
			"imported":   !importDecision.Hidden(),
			"searchable": !importDecision.Hidden() && !searchDecision.Hidden(),
		}, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		cliutil.PrintJSON(payload)
		return nil
	}
	fmt.Printf("Sample: harness=%q model=%q project=%q cwd=%q tag=%q\n", *harness, *model, *project, *cwd, *tag)
	// At a real import only harness and tag are observed, so model/project/cwd
	// matchers on import rules never fire there even though this sample lets you
	// supply them. Flag any such rule so the `test` verdict is not mistaken for
	// the import seam's actual behavior.
	for i, rule := range policy.Rules() {
		if unobservable := rule.ImportUnobservableFields(); len(unobservable) > 0 {
			fmt.Printf("  note: import rule %d matches on %s, which is not evaluated at the import seam "+
				"(only harness and tag are); it can only fire at search\n", i, strings.Join(unobservable, ", "))
		}
	}
	fmt.Printf("  import: %s\n", conversationPolicyDecisionLine(importDecision))
	fmt.Printf("  search: %s\n", conversationPolicyDecisionLine(searchDecision))
	switch {
	case importDecision.Hidden():
		fmt.Println("  => EXCLUDED at import: this conversation would never be written (no run_events).")
	case searchDecision.Hidden():
		fmt.Println("  => IMPORTED but HIDDEN from search results.")
	default:
		fmt.Println("  => IMPORTED and SEARCHABLE.")
	}
	return nil
}

func conversationPolicyReport(doc conversationpolicy.Document, jsonOutput bool, message string) error {
	if jsonOutput {
		payload, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return err
		}
		cliutil.PrintJSON(payload)
		return nil
	}
	fmt.Println(message)
	fmt.Printf("Policy now has %d rule(s) at %s\n", len(doc.Rules), conversationPolicyPath())
	return nil
}

func conversationPolicyDecisionLine(decision conversationpolicy.Decision) string {
	if !decision.Matched {
		return "no rule matched (default: index/keep)"
	}
	return fmt.Sprintf("rule %d %s -> action=%s", decision.RuleIndex, conversationpolicy.DescribeRule(decision.Rule), decision.Action)
}

func conversationPolicyDecisionJSON(decision conversationpolicy.Decision) map[string]any {
	out := map[string]any{"matched": decision.Matched, "action": string(decision.Action), "hidden": decision.Hidden()}
	if decision.Matched {
		out["ruleIndex"] = decision.RuleIndex
		out["rule"] = decision.Rule
	}
	return out
}
