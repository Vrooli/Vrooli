package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

func (a *App) autoFilerClient() apiconnect.AutoFilerServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(a.core)
	return apiconnect.NewAutoFilerServiceClient(httpClient, baseURL)
}

func (a *App) cmdAutoFilerStatus(args []string) error {
	fs := flag.NewFlagSet("autofiler status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	resp, err := a.autoFilerClient().GetStatus(context.Background(), connect.NewRequest(&apipb.AutoFilerStatusRequest{}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	printAutoFilerStatus(resp.Msg)
	return nil
}

func (a *App) cmdAutoFilerRunNow(args []string) error {
	fs := flag.NewFlagSet("autofiler run-now", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	resp, err := a.autoFilerClient().RunNow(context.Background(), connect.NewRequest(&apipb.AutoFilerRunNowRequest{}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	printAutoFilerStatus(resp.Msg)
	return nil
}

func (a *App) cmdBacklogDismiss(args []string) error {
	fs := flag.NewFlagSet("backlog dismiss", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	reasonFlag := fs.String("reason", "", "Dismissal reason")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog dismiss --kind KIND --name NAME [--reason MSG] [--json]\n\n%s", err)
	}
	resp, err := a.autoFilerClient().DismissSuggestion(context.Background(), connect.NewRequest(&apipb.DismissAutoFilerSuggestionRequest{
		Kind:   strings.TrimSpace(*kindFlag),
		Name:   strings.TrimSpace(*nameFlag),
		Reason: optionalString(strings.TrimSpace(*reasonFlag)),
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	item := resp.Msg.GetItem()
	printSection("Result")
	fmt.Printf("  Dismissed %s/%s\n", item.GetKind(), item.GetName())
	if item.GetArchivedAt() != "" {
		fmt.Printf("  Archived at: %s\n", item.GetArchivedAt())
	}
	if item.GetFindingRef() != "" {
		fmt.Printf("  Finding: %s\n", item.GetFindingRef())
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("autofiler", "status"),
		cliCommand("backlog", "list", "--status", "suggested"),
	})
	return nil
}

func printAutoFilerStatus(status *apipb.AutoFilerStatusResponse) {
	printSection("Status")
	state := "disabled"
	if status.GetEnabled() {
		state = "enabled"
	}
	fmt.Printf("  Auto-filer: %s\n", state)
	fmt.Printf("  Mode: %s\n", status.GetMode())
	fmt.Printf("  Strategy: %s\n", status.GetStrategy())
	if status.GetLastCycleTime() != "" {
		fmt.Printf("  Last cycle: %s\n", status.GetLastCycleTime())
	}
	if status.GetLastError() != "" {
		fmt.Printf("  Last error: %s\n", status.GetLastError())
	}

	printSection("Policy")
	fmt.Printf("  Open auto-filed: %d / %d\n", status.GetOpenAutoFiled(), status.GetMaxOpenAutoFiled())
	fmt.Printf("  Remaining budget: %d\n", status.GetRemainingBudget())
	fmt.Printf("  Remembered dismissals: %d\n", status.GetDismissalCount())
	brake := status.GetBrake()
	if brake != nil {
		brakeState := "open"
		if brake.GetBraked() {
			brakeState = "braked"
		}
		fmt.Printf("  Velocity brake: %s (%d/%d transitions in %d day(s))\n",
			brakeState, brake.GetObserved(), brake.GetMinimum(), brake.GetWindowDays())
	}

	printSection("Latest Cycle")
	fmt.Printf("  Candidates: %d\n", status.GetCandidates())
	fmt.Printf("  Findings: %d\n", status.GetFindings())
	fmt.Printf("  Created: %d\n", status.GetCreated())
	fmt.Printf("  Skipped dismissed: %d\n", status.GetSkippedDismissed())
	fmt.Printf("  Reconciled closed: %d\n", status.GetReconciledClosed())
	fmt.Printf("  Reconciled noted: %d\n", status.GetReconciledNoted())
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
