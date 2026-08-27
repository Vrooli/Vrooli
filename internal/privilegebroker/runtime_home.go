package privilegebroker

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

const (
	runtimeHomeParameterA = 100_000
)

func executeRuntimeHomeRepair(ctx context.Context, subject RuntimeHomeSubject) Result {
	root, err := runtimeHomeRootForSubject(subject)
	if err != nil {
		return NewFailure("", ActionRuntimeHomeOwnershipRepair, "runtime_home_contract_unavailable")
	}
	return executeRuntimeHomeRepairAt(ctx, subject, root)
}

func executeRuntimeHomeRepairAt(ctx context.Context, subject RuntimeHomeSubject, root string) Result {
	classRoot, err := approvedRuntimeHomeClassRoot(root, subject.Class)
	if err != nil {
		return NewFailure("", ActionRuntimeHomeOwnershipRepair, "runtime_home_class_unavailable")
	}
	service := config.RepairService{ResolveRoot: func(class string) (string, error) {
		if class != subject.Class {
			return "", fmt.Errorf("repair class changed during execution")
		}
		return classRoot, nil
	}}
	result, err := service.Repair(ctx, config.RepairRequest{
		Scope: config.RepairScope{RootClass: subject.Class}, ExpectedUID: subject.ExpectedUID,
		ExpectedGID: subject.ExpectedGID, Apply: true, MaxEntries: runtimeHomeParameterA,
	})
	if err != nil {
		return NewFailure("", ActionRuntimeHomeOwnershipRepair, "runtime_home_repair_failed")
	}
	out := Result{Version: ProtocolVersion, Action: ActionRuntimeHomeOwnershipRepair, Status: "unchanged", Evidence: Evidence{Scanned: result.Scanned, Repaired: result.Repaired, Failed: result.Failed}}
	if result.Status != config.RepairComplete || result.Failed > 0 {
		out.Status = "failed"
		out.Code = "runtime_home_repair_partial"
	} else if result.Repaired > 0 {
		out.Status = "changed"
		out.Changed = true
	}
	return out
}

func runtimeHomeRootForSubject(subject RuntimeHomeSubject) (string, error) {
	home, err := userHomeForUID(subject.ExpectedUID)
	if err != nil {
		return "", fmt.Errorf("lookup runtime-home owner: %w", err)
	}
	return runtimeHomeRoot(home)
}

func approvedRuntimeHomeClassRoot(root, class string) (string, error) {
	switch class {
	case "bin", "cache", "logs", "metrics", "processes", "build", "test_runs", "backups", "artifacts":
		return filepath.Join(root, class), nil
	case repocontract.HomeKeySecretsEnc:
		// secrets.enc.json is a single runtime-home entry, not a directory
		// class. Resolve it through the repository contract so the broker never
		// accepts a caller-supplied path while still repairing the exact file.
		return repocontract.RuntimeHomeEntryPath(filepath.Dir(root), repocontract.HomeKeySecretsEnc)
	default:
		return "", fmt.Errorf("runtime-home class %q is not approved", class)
	}
}

func installedRuntimeHomeRoot(repoRoot string, uid uint32) (string, error) {
	if filepath.Clean(repoRoot) == "." || repoRoot == "" {
		repoRoot = os.Getenv("VROOLI_REPO_ROOT")
	}
	if repoRoot == "" {
		var err error
		repoRoot, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return "", err
	}
	home, err := userHomeForUID(uid)
	if err != nil {
		return "", err
	}
	return contract.RuntimeHome(home)
}

func userHomeForUID(uid uint32) (string, error) {
	entry, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil || entry.HomeDir == "" {
		return "", fmt.Errorf("lookup uid %d: %w", uid, err)
	}
	return filepath.Clean(entry.HomeDir), nil
}

func runtimeHomeRoot(home string) (string, error) {
	root := os.Getenv("VROOLI_REPO_ROOT")
	if root == "" {
		root, _ = os.Getwd()
	}
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return "", err
	}
	return contract.RuntimeHome(home)
}
