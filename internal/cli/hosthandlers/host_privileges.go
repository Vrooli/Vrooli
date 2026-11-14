package hosthandlers

import (
	"fmt"
	"runtime"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/privilegebroker"
)

func runHostKernelMemory(runCtx cliapp.RunContext, parent *rootcli.CommandContext) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("kernel slab diagnostics are not supported on %s", runtime.GOOS)
	}
	report, err := hostinventory.ReadKernelMemory()
	if err != nil {
		result, brokerErr := privilegebroker.NewClient().Do(parent.OperationContext(), privilegebroker.Request{
			Version: privilegebroker.ProtocolVersion, RequestID: uuid.NewString(), Action: privilegebroker.ActionKernelMemoryInspect,
		})
		if brokerErr != nil || result.Status != "completed" || result.KernelMemory == nil {
			return fmt.Errorf("kernel memory diagnostics unavailable (%v; %s): upgrade through `vrooli setup broker --sudo-mode=ask`", brokerErr, result.Code)
		}
		report = *result.KernelMemory
	}
	if parent.Globals.JSON || runCtx.JSON() {
		return cliout.WriteJSONValue(parent.Stdout, report)
	}
	rows := make([][]string, 0, len(report.Caches)+1)
	rows = append(rows, []string{"Cache", "Allocated bytes", "Reclaimable"})
	for _, cache := range report.Caches {
		reclaimable := "unknown"
		if cache.Reclaimable != nil {
			reclaimable = fmt.Sprint(*cache.Reclaimable)
		}
		rows = append(rows, []string{cache.Name, fmt.Sprint(cache.AllocatedBytes), reclaimable})
	}
	fmt.Fprintf(parent.Stdout, "Kernel slab allocation: %d bytes across %d caches\n", report.AllocatedBytes, report.CacheCount)
	return cliout.WriteSection(parent.Stdout, cliout.Section{Rows: rows})
}

func runHostServiceRecovery(runCtx cliapp.RunContext, parent *rootcli.CommandContext) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("broker service recovery is not supported on %s", runtime.GOOS)
	}
	result, err := privilegebroker.NewClient().Do(parent.OperationContext(), privilegebroker.Request{
		Version: privilegebroker.ProtocolVersion, RequestID: uuid.NewString(), Action: privilegebroker.ActionServiceRecover,
		Service: &privilegebroker.ServiceSubject{Unit: runCtx.Positional("unit"), Operation: runCtx.Positional("operation")},
	})
	if err != nil {
		return fmt.Errorf("service recovery unavailable: %w; install through `vrooli setup broker --sudo-mode=ask`", err)
	}
	if parent.Globals.JSON || runCtx.JSON() {
		if err := cliout.WriteJSONValue(parent.Stdout, result); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(parent.Stdout, "Service recovery: %s (%s)\n", result.Status, result.Code)
	}
	if result.Status != "verified" {
		return fmt.Errorf("service recovery failed: %s; inspect `vrooli setup broker --dry-run --json`", result.Code)
	}
	return nil
}
