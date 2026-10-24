package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
)

func (a *App) cmdInvestigation(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: agent-manager investigation <start|get|list|wait|cancel>")
	}
	switch args[0] {
	case "start":
		return a.investigationStart(args[1:])
	case "get":
		return a.investigationGet(args[1:])
	case "list":
		return a.investigationList(args[1:])
	case "wait":
		return a.investigationWait(args[1:])
	case "cancel":
		return a.investigationCancel(args[1:])
	default:
		return fmt.Errorf("unknown investigation subcommand: %s", args[0])
	}
}

func (a *App) investigationStart(args []string) error {
	fs := flag.NewFlagSet("investigation start", flag.ContinueOnError)
	file := fs.String("file", "", "InvestigationRequest JSON file")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*file) == "" {
		return fmt.Errorf("--file is required")
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		return fmt.Errorf("read investigation request: %w", err)
	}
	request := &apipb.InvestigationRequest{}
	decoder := protojson.UnmarshalOptions{DiscardUnknown: false}
	if err := decoder.Unmarshal(raw, request); err != nil {
		return fmt.Errorf("decode investigation request: %w", err)
	}
	body, response, err := a.services.Investigations.Start(&apipb.StartInvestigationRequest{Request: request})
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	item := response.GetInvestigation()
	if item == nil {
		return fmt.Errorf("investigation start returned no record")
	}
	fmt.Printf("%s status=%s reused=%t workflow=%s\n", item.GetInvestigationId(), item.GetOperationStatus(), response.GetReused(), item.GetWorkflowRef())
	return nil
}

func (a *App) investigationGet(args []string) error {
	fs := flag.NewFlagSet("investigation get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: agent-manager investigation get <investigation-id> [--json]")
	}
	body, response, err := a.services.Investigations.Get(&apipb.GetInvestigationRequest{InvestigationId: fs.Args()[0]})
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("%s status=%s workflow=%s\n", response.GetInvestigationId(), response.GetOperationStatus(), response.GetWorkflowRef())
	return nil
}

func (a *App) investigationList(args []string) error {
	fs := flag.NewFlagSet("investigation list", flag.ContinueOnError)
	status := fs.String("operation-status", "", "Operation status filter")
	fs.StringVar(status, "status", "", "Compatibility alias for --operation-status")
	limit := fs.Int("limit", 50, "Maximum investigations")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, response, err := a.services.Investigations.List(&apipb.ListInvestigationsRequest{OperationStatus: *status, Limit: int32(*limit)})
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	for _, item := range response.GetInvestigations() {
		fmt.Printf("%s status=%s workflow=%s\n", item.GetInvestigationId(), item.GetOperationStatus(), item.GetWorkflowRef())
	}
	return nil
}

func (a *App) investigationWait(args []string) error {
	fs := flag.NewFlagSet("investigation wait", flag.ContinueOnError)
	timeout := fs.Int("timeout-seconds", 30, "One bounded wait duration")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: agent-manager investigation wait <investigation-id> [--timeout-seconds n] [--json]")
	}
	body, response, err := a.services.Investigations.Wait(&apipb.WaitInvestigationRequest{InvestigationId: fs.Args()[0], TimeoutSeconds: int32(*timeout)})
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	item := response.GetInvestigation()
	fmt.Printf("%s status=%s terminal=%t\n", item.GetInvestigationId(), item.GetOperationStatus(), response.GetTerminal())
	return nil
}

func (a *App) investigationCancel(args []string) error {
	fs := flag.NewFlagSet("investigation cancel", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: agent-manager investigation cancel <investigation-id> [--json]")
	}
	body, response, err := a.services.Investigations.Cancel(&apipb.CancelInvestigationRequest{InvestigationId: fs.Args()[0]})
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("%s status=%s cancel_requested=%t\n", response.GetInvestigationId(), response.GetOperationStatus(), response.GetCancelRequested())
	return nil
}
