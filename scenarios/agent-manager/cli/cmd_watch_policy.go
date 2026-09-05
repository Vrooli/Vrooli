package main

import (
	"flag"
	"fmt"
	"github.com/vrooli/cli-core/cliutil"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"os"
)

func (a *App) watchPolicyCandidate(args []string) error {
	fs := flag.NewFlagSet("watch policy-candidate", flag.ContinueOnError)
	file := fs.String("policy-file", "", "SupervisionPolicyDefinition JSON file")
	supersedes := fs.String("supersedes", "", "Predecessor policy version")

	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *file == "" {
		return fmt.Errorf("--policy-file is required")
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	value := &domainpb.SupervisionPolicyDefinition{}
	if err := protojson.Unmarshal(raw, value); err != nil {
		return err
	}

	body, response, err := a.services.Watches.PolicyCandidate(&domainpb.CreateSupervisionPolicyCandidateRequest{Policy: value, Supersedes: *supersedes})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Policy %s: %s\n", response.GetPolicy().GetVersion(), response.GetState())
	return nil
}

func (a *App) watchPolicyEvaluate(args []string) error {
	fs := flag.NewFlagSet("watch policy-evaluate", flag.ContinueOnError)
	version := fs.String("version", "", "Policy version")

	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *version == "" {
		return fmt.Errorf("policy version is required")
	}

	body, response, err := a.services.Watches.PolicyEvaluate(&domainpb.EvaluateSupervisionPolicyRequest{Version: *version})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Policy %s: replay=%t (%d samples, %d false positives, %d false negatives); rollout=%t (%d samples); safety violations=%d\n", response.GetVersion(), response.GetReplayPassed(), response.GetSampleCount(), response.GetFalsePositives(), response.GetFalseNegatives(), response.GetRolloutPassed(), response.GetRolloutSamples(), response.GetSafetyViolations())
	return nil
}

func (a *App) watchPolicyAssess(args []string) error {
	fs := flag.NewFlagSet("watch policy-assess", flag.ContinueOnError)
	file := fs.String("outcome-file", "", "SupervisionOutcomeRecord JSON file")

	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *file == "" {
		return fmt.Errorf("--outcome-file is required")
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	value := &domainpb.SupervisionOutcomeRecord{}
	if err := protojson.Unmarshal(raw, value); err != nil {
		return err
	}

	body, response, err := a.services.Watches.PolicyAssess(&domainpb.RecordSupervisionOutcomeRequest{Outcome: value})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Outcome %s recorded; ledger synced=%t\n", response.GetOutcome().GetOutcomeId(), response.GetSourceLedgerSynced())
	if response.GetDegradationReason() != "" {
		fmt.Println(response.GetDegradationReason())
	}
	return nil
}

func (a *App) watchPolicyPromote(args []string) error {
	fs := flag.NewFlagSet("watch policy-promote", flag.ContinueOnError)
	version := fs.String("version", "", "Policy version")

	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *version == "" {
		return fmt.Errorf("policy version is required")
	}

	body, response, err := a.services.Watches.PolicyPromote(&domainpb.PromoteSupervisionPolicyRequest{Version: *version})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Policy %s: %s\n", response.GetPolicy().GetVersion(), response.GetState())
	return nil
}

func (a *App) watchPolicyReject(args []string) error {
	fs := flag.NewFlagSet("watch policy-reject", flag.ContinueOnError)
	version := fs.String("version", "", "Policy version")
	reason := fs.String("reason", "", "Rejection reason")

	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *version == "" {
		return fmt.Errorf("policy version is required")
	}

	body, response, err := a.services.Watches.PolicyReject(&domainpb.RejectSupervisionPolicyRequest{Version: *version, Reason: *reason})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Policy %s: %s\n", response.GetPolicy().GetVersion(), response.GetState())
	return nil
}

func (a *App) watchPolicyRollback(args []string) error {
	fs := flag.NewFlagSet("watch policy-rollback", flag.ContinueOnError)
	version := fs.String("active-version", "", "Policy version")

	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *version == "" {
		return fmt.Errorf("policy version is required")
	}

	body, response, err := a.services.Watches.PolicyRollback(&domainpb.RollbackSupervisionPolicyRequest{ActiveVersion: *version})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Policy %s: %s\n", response.GetPolicy().GetVersion(), response.GetState())
	return nil
}

func (a *App) watchPolicyDisable(args []string) error {
	fs := flag.NewFlagSet("watch policy-disable", flag.ContinueOnError)
	disabled := fs.Bool("disabled", true, "Disable automatic supervision")
	reason := fs.String("reason", "", "Reason")

	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, response, err := a.services.Watches.PolicyDisable(&domainpb.SetSupervisionPolicyDisabledRequest{Disabled: *disabled, Reason: *reason})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Supervision disabled=%t: %s\n", response.GetDisabled(), response.GetReason())
	return nil
}
