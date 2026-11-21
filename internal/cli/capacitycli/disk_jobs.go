package capacitycli

import (
	"fmt"
	"strconv"
	"strings"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	engine "github.com/vrooli/vrooli/internal/capacity"
)

type DiskReserveArgs struct{ RequestFile, LeaseFile string }
type DiskLeaseArgs struct{ LeaseFile string }
type DiskExtendArgs struct{ LeaseFile, PathsFile string }
type DiskCompleteArgs struct {
	LeaseFile       string
	CleanupVerified bool
}
type DiskListArgs struct{ Active, Observe bool }
type DiskReservationsArgs struct {
	Active            bool
	Owner, Filesystem string
}
type DiskReconcileArgs struct {
	LeaseID                         string
	ProcessStopped, CleanupVerified bool
}
type DiskObserveArgs struct{ LeaseFile string }
type DiskSampleArgs struct {
	LeaseFile, Path                    string
	DockerContainer, DockerDestination string
	DockerUpperDir, DockerRootDir      bool
	Bytes, ExtendPeak                  int64
}

func ParseDiskReserveRequest(args []string) (DiskReserveArgs, error) {
	p, err := parse(CommandDiskReserve, "vrooli capacity disk-reserve", args)
	if err != nil {
		return DiskReserveArgs{}, err
	}
	out := DiskReserveArgs{RequestFile: p.FlagValue("--request-file"), LeaseFile: p.FlagValue("--lease-file")}
	if out.RequestFile == "" || out.LeaseFile == "" {
		return out, fmt.Errorf("disk-reserve requires --request-file and --lease-file")
	}
	return out, nil
}
func ParseDiskLeaseRequest(id CommandID, args []string) (DiskLeaseArgs, error) {
	p, err := parse(id, "vrooli capacity "+string(id), args)
	if err != nil {
		return DiskLeaseArgs{}, err
	}
	out := DiskLeaseArgs{LeaseFile: p.FlagValue("--lease-file")}
	if out.LeaseFile == "" {
		return out, fmt.Errorf("%s requires --lease-file", id)
	}
	return out, nil
}
func ParseDiskExtendRequest(args []string) (DiskExtendArgs, error) {
	p, err := parse(CommandDiskExtend, "vrooli capacity disk-extend", args)
	if err != nil {
		return DiskExtendArgs{}, err
	}
	out := DiskExtendArgs{LeaseFile: p.FlagValue("--lease-file"), PathsFile: p.FlagValue("--paths-file")}
	if out.LeaseFile == "" || out.PathsFile == "" {
		return out, fmt.Errorf("disk-extend requires --lease-file and --paths-file")
	}
	return out, nil
}
func ParseDiskCompleteRequest(args []string) (DiskCompleteArgs, error) {
	p, err := parse(CommandDiskComplete, "vrooli capacity disk-complete", args)
	if err != nil {
		return DiskCompleteArgs{}, err
	}
	out := DiskCompleteArgs{LeaseFile: p.FlagValue("--lease-file"), CleanupVerified: p.HasFlag("--cleanup-verified")}
	if out.LeaseFile == "" || !out.CleanupVerified {
		return out, fmt.Errorf("disk-complete requires --lease-file and --cleanup-verified")
	}
	return out, nil
}
func ParseDiskListRequest(args []string) (DiskListArgs, error) {
	p, err := parse(CommandDiskList, "vrooli capacity disk-list", args)
	if err != nil {
		return DiskListArgs{}, err
	}
	return DiskListArgs{Active: p.HasFlag("--active"), Observe: p.HasFlag("--observe")}, nil
}
func ParseDiskReservationsRequest(args []string) (DiskReservationsArgs, error) {
	p, err := parse(CommandDiskReservations, "vrooli capacity disk-reservations", args)
	if err != nil {
		return DiskReservationsArgs{}, err
	}
	return DiskReservationsArgs{Active: p.HasFlag("--active"), Owner: strings.TrimSpace(p.FlagValue("--owner")), Filesystem: strings.TrimSpace(p.FlagValue("--filesystem"))}, nil
}
func ParseDiskReconcileRequest(args []string) (DiskReconcileArgs, error) {
	p, err := parse(CommandDiskReconcile, "vrooli capacity disk-reconcile", args)
	if err != nil {
		return DiskReconcileArgs{}, err
	}
	out := DiskReconcileArgs{LeaseID: strings.TrimSpace(p.FlagValue("--lease-id")), ProcessStopped: p.HasFlag("--process-stopped"), CleanupVerified: p.HasFlag("--cleanup-verified")}
	if out.LeaseID == "" || !out.ProcessStopped || !out.CleanupVerified {
		return out, fmt.Errorf("disk-reconcile requires --lease-id, --process-stopped, and --cleanup-verified")
	}
	return out, nil
}

func ParseDiskObserveRequest(args []string) (DiskObserveArgs, error) {
	p, err := parse(CommandDiskObserve, "vrooli capacity disk-observe", args)
	if err != nil {
		return DiskObserveArgs{}, err
	}
	out := DiskObserveArgs{LeaseFile: p.FlagValue("--lease-file")}
	if out.LeaseFile == "" {
		return out, fmt.Errorf("disk-observe requires --lease-file")
	}
	return out, nil
}

func ParseDiskSampleRequest(args []string) (DiskSampleArgs, error) {
	p, err := parse(CommandDiskSample, "vrooli capacity disk-sample", args)
	if err != nil {
		return DiskSampleArgs{}, err
	}
	out := DiskSampleArgs{LeaseFile: p.FlagValue("--lease-file"), Path: p.FlagValue("--path"), DockerContainer: p.FlagValue("--docker-container"), DockerDestination: p.FlagValue("--docker-destination"), DockerUpperDir: p.HasFlag("--docker-upper-dir"), DockerRootDir: p.HasFlag("--docker-root-dir")}
	bytes, err := strconv.ParseInt(p.FlagValue("--bytes"), 10, 64)
	if err != nil || bytes < 0 {
		return out, fmt.Errorf("disk-sample requires nonnegative --bytes")
	}
	out.Bytes = bytes
	if raw := p.FlagValue("--extend-peak"); raw != "" {
		out.ExtendPeak, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || out.ExtendPeak <= 0 {
			return out, fmt.Errorf("--extend-peak must be positive")
		}
	}
	if out.LeaseFile == "" || out.Path == "" {
		return out, fmt.Errorf("disk-sample requires --lease-file, --path, and --bytes")
	}
	return out, nil
}

type DiskJobRequest = engine.DiskJobRequest
type DiskJobPath = engine.DiskJobPath
type DiskJobCredential = capacityapp.DiskJobCredential
