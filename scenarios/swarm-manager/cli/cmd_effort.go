package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// effortControl is the CLI projection of the versioned owner effort-control
// aggregate. Field tags mirror swarm-manager's identity.EffortControl so the
// authored authority is rendered without copying it into a second authority.
type effortControl struct {
	EffortID         string   `json:"effort_id"`
	Slug             string   `json:"slug,omitempty"`
	Revision         int64    `json:"revision"`
	FamilyID         string   `json:"family_id,omitempty"`
	DelegatedActions []string `json:"delegated_actions,omitempty"`
	Scope            struct {
		Allow []string `json:"allow,omitempty"`
		Deny  []string `json:"deny,omitempty"`
	} `json:"scope"`
	PolicyBinding struct {
		Source string `json:"source"`
		Digest string `json:"digest"`
	} `json:"policy_binding"`
	WorkReferences []struct {
		Owner string `json:"owner"`
		Kind  string `json:"kind"`
		ID    string `json:"id"`
		Role  string `json:"role,omitempty"`
	} `json:"work_references,omitempty"`
	Completion struct {
		EvidenceComplete bool     `json:"evidence_complete"`
		EvidenceRefs     []string `json:"evidence_refs,omitempty"`
		HumanAccepted    bool     `json:"human_accepted"`
		HumanActor       string   `json:"human_actor,omitempty"`
		AcceptedAt       string   `json:"accepted_at,omitempty"`
	} `json:"completion"`
}

type effortControlResponse struct {
	Effort          effortControl `json:"effort"`
	AuthorityDigest string        `json:"authority_digest"`
}

func effortIDFlagArg(effortID string) string {
	return url.PathEscape(strings.TrimSpace(effortID))
}

// cmdEffortGet returns the current admitted effort revision.
func (a *App) cmdEffortGet(args []string) error {
	fs := flag.NewFlagSet("effort get", flag.ContinueOnError)
	effortIDFlag := fs.String("effort-id", "", "Effort identity (single path segment)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("effort-id", *effortIDFlag); err != nil {
		return fmt.Errorf("usage: effort get --effort-id ID [--json]\n\n%s", err)
	}
	body, err := a.core.Get("/efforts/"+effortIDFlagArg(*effortIDFlag), nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	response, err := decodeResponse[effortControlResponse](body)
	if err != nil {
		return err
	}
	printEffortControl(response)
	return nil
}

// cmdEffortAdmit creates the first revision of an effort. It refuses to replace
// an existing effort; authority changes only through amend.
func (a *App) cmdEffortAdmit(args []string) error {
	return a.mutateEffort("admit", "POST", args)
}

// cmdEffortAmend changes reviewed authority at the exact next revision.
func (a *App) cmdEffortAmend(args []string) error {
	return a.mutateEffort("amend", "PUT", args)
}

func (a *App) mutateEffort(verb, method string, args []string) error {
	fs := flag.NewFlagSet("effort "+verb, flag.ContinueOnError)
	effortIDFlag := fs.String("effort-id", "", "Effort identity (single path segment)")
	dataFlag := fs.String("data", "", "Reviewed effort-control JSON document")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("effort-id", *effortIDFlag, "data", *dataFlag); err != nil {
		return fmt.Errorf("usage: effort %s --effort-id ID --data JSON [--json]\n\n%s", verb, err)
	}
	payload := json.RawMessage(strings.TrimSpace(*dataFlag))
	if !json.Valid(payload) {
		return fmt.Errorf("--data must be valid JSON")
	}
	body, err := a.core.Request(method, "/efforts/"+effortIDFlagArg(*effortIDFlag), nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	response, err := decodeResponse[effortControlResponse](body)
	if err != nil {
		return err
	}
	printEffortControl(response)
	return nil
}

// cmdEffortEvidenceComplete records owner evidence completion. It can never set
// human acceptance: the owner keeps those standings separate.
func (a *App) cmdEffortEvidenceComplete(args []string) error {
	fs := flag.NewFlagSet("effort evidence-complete", flag.ContinueOnError)
	effortIDFlag := fs.String("effort-id", "", "Effort identity (single path segment)")
	refsFlag := fs.String("refs", "", "Comma-separated evidence references")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("effort-id", *effortIDFlag); err != nil {
		return fmt.Errorf("usage: effort evidence-complete --effort-id ID [--refs a,b] [--json]\n\n%s", err)
	}
	var refs []string
	for _, ref := range strings.Split(*refsFlag, ",") {
		if trimmed := strings.TrimSpace(ref); trimmed != "" {
			refs = append(refs, trimmed)
		}
	}
	payload, err := json.Marshal(map[string]any{"refs": refs})
	if err != nil {
		return err
	}
	body, err := a.core.Request("POST", "/efforts/"+effortIDFlagArg(*effortIDFlag)+"/completion/evidence", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	response, err := decodeResponse[effortControlResponse](body)
	if err != nil {
		return err
	}
	printEffortControl(response)
	return nil
}

// cmdEffortAccept records the authenticated human product disposition. The
// owner refuses a blank actor; evidence completion is not acceptance.
func (a *App) cmdEffortAccept(args []string) error {
	fs := flag.NewFlagSet("effort accept", flag.ContinueOnError)
	effortIDFlag := fs.String("effort-id", "", "Effort identity (single path segment)")
	actorFlag := fs.String("actor", "", "Authenticated human actor")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("effort-id", *effortIDFlag, "actor", *actorFlag); err != nil {
		return fmt.Errorf("usage: effort accept --effort-id ID --actor ACTOR [--json]\n\n%s", err)
	}
	payload, err := json.Marshal(map[string]string{"actor": strings.TrimSpace(*actorFlag)})
	if err != nil {
		return err
	}
	body, err := a.core.Request("POST", "/efforts/"+effortIDFlagArg(*effortIDFlag)+"/completion/accept", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	response, err := decodeResponse[effortControlResponse](body)
	if err != nil {
		return err
	}
	printEffortControl(response)
	return nil
}

func printEffortControl(response effortControlResponse) {
	effort := response.Effort
	printSection("Effort Control")
	fmt.Printf("  Effort:    %s\n", effort.EffortID)
	if effort.Slug != "" {
		fmt.Printf("  Slug:      %s\n", effort.Slug)
	}
	fmt.Printf("  Revision:  %d\n", effort.Revision)
	if effort.FamilyID != "" {
		fmt.Printf("  Family:    %s\n", effort.FamilyID)
	}
	fmt.Printf("  Authority: %s\n", response.AuthorityDigest)
	if len(effort.DelegatedActions) > 0 {
		fmt.Printf("  Actions:   %s\n", strings.Join(effort.DelegatedActions, ", "))
	}
	if len(effort.Scope.Allow) > 0 {
		fmt.Printf("  Allow:     %s\n", strings.Join(effort.Scope.Allow, ", "))
	}
	if len(effort.Scope.Deny) > 0 {
		fmt.Printf("  Deny:      %s\n", strings.Join(effort.Scope.Deny, ", "))
	}
	if effort.PolicyBinding.Source != "" {
		fmt.Printf("  Policy:    %s (%s)\n", effort.PolicyBinding.Source, effort.PolicyBinding.Digest)
	}
	for _, reference := range effort.WorkReferences {
		fmt.Printf("  Work:      %s/%s %s [%s]\n", reference.Owner, reference.Kind, reference.ID, reference.Role)
	}
	fmt.Printf("  Evidence:  complete=%t refs=%s\n", effort.Completion.EvidenceComplete, strings.Join(effort.Completion.EvidenceRefs, ","))
	fmt.Printf("  Accepted:  %t actor=%s at=%s\n", effort.Completion.HumanAccepted, effort.Completion.HumanActor, effort.Completion.AcceptedAt)
}
