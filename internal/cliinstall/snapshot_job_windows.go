//go:build windows

package cliinstall

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func snapshotJobObjectName(prefix, lockPath string) (string, error) {
	identity, err := snapshotJobLockIdentity(lockPath)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(identity))
	return prefix + hex.EncodeToString(digest[:]), nil
}

func snapshotJobLockIdentity(lockPath string) (string, error) {
	file, err := os.Open(lockPath)
	if err != nil {
		return "", fmt.Errorf("open snapshot lock for Windows Job Object identity %q: %w", lockPath, err)
	}
	defer file.Close()

	var identity windowsFileIDInfo
	err = windows.GetFileInformationByHandleEx(windows.Handle(file.Fd()), windows.FileIdInfo, (*byte)(unsafe.Pointer(&identity)), uint32(unsafe.Sizeof(identity)))
	if err == nil {
		return fmt.Sprintf("%016x:%032x", identity.VolumeSerialNumber, identity.FileID), nil
	}

	// Some filesystems do not implement FileIdInfo. The older volume serial and
	// 64-bit file index are still stable across DOS, UNC, and mapped-volume names.
	var legacy windows.ByHandleFileInformation
	if legacyErr := windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &legacy); legacyErr != nil {
		return "", fmt.Errorf("read snapshot lock file identity: FileIdInfo: %v; legacy file ID: %w", err, legacyErr)
	}
	return fmt.Sprintf("%08x:%08x%08x", legacy.VolumeSerialNumber, legacy.FileIndexHigh, legacy.FileIndexLow), nil
}

type windowsFileIDInfo struct {
	VolumeSerialNumber uint64
	FileID             [16]byte
}

func inspectSnapshotLeaseJob(root, snapshot string) (snapshotJobState, error) {
	names, err := snapshotLeaseJobNames(root, snapshot)
	if err != nil {
		return snapshotJobAbsent, err
	}
	return inspectSnapshotJobNames(names)
}

func inspectSnapshotRootJob(root string) (snapshotJobState, error) {
	names, err := snapshotRootJobNames(root)
	if err != nil {
		return snapshotJobAbsent, err
	}
	return inspectSnapshotJobNames(names)
}

func snapshotLeaseJobNames(root, snapshot string) ([]string, error) {
	path, err := snapshotLeasePath(root, snapshot)
	if err != nil {
		return nil, err
	}
	return snapshotWitnessNames(`Global\VrooliSnapshotLease-`, path)
}

func snapshotRootJobNames(root string) ([]string, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute snapshot root: %w", err)
	}
	return snapshotWitnessNames(`Global\VrooliSnapshotRoot-`, filepath.Join(absoluteRoot, "snapshots.lock"))
}

func snapshotWitnessNames(prefix, lockPath string) ([]string, error) {
	names := make([]string, 0, 2)
	name, identityErr := snapshotJobObjectName(prefix, lockPath)
	if identityErr == nil {
		names = append(names, name)
	} else if !errors.Is(identityErr, os.ErrNotExist) {
		return nil, identityErr
	}
	legacyName, legacyErr := snapshotLegacyPathJobObjectName(prefix, lockPath)
	if legacyErr != nil {
		return nil, legacyErr
	}
	if len(names) == 0 || names[0] != legacyName {
		names = append(names, legacyName)
	}
	return names, nil
}

func snapshotLegacyPathJobObjectName(prefix, lockPath string) (string, error) {
	canonicalDir, err := filepath.EvalSymlinks(filepath.Dir(lockPath))
	if err != nil {
		return "", fmt.Errorf("resolve legacy snapshot lock directory: %w", err)
	}
	canonical := strings.ToLower(filepath.ToSlash(filepath.Join(canonicalDir, filepath.Base(lockPath))))
	digest := sha256.Sum256([]byte(canonical))
	return prefix + hex.EncodeToString(digest[:]), nil
}

func inspectSnapshotJobNames(names []string) (snapshotJobState, error) {
	return inspectSnapshotJobNamesUsing(names, inspectNamedSnapshotJob)
}

func inspectNamedSnapshotJob(name string) (snapshotJobState, error) {
	nameUTF16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return snapshotJobAbsent, err
	}
	job, err := openSnapshotJobForQuery(nameUTF16)
	if err != nil {
		if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
			return snapshotJobAbsent, nil
		}
		return snapshotJobAbsent, fmt.Errorf("open named snapshot Job Object: %w", err)
	}
	var accounting snapshotJobAccountingInformation
	queryErr := windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&accounting)), uint32(unsafe.Sizeof(accounting)), nil)
	closeErr := windows.CloseHandle(job)
	if queryErr != nil {
		return snapshotJobAbsent, fmt.Errorf("query named snapshot Job Object: %w", queryErr)
	}
	if closeErr != nil {
		return snapshotJobAbsent, fmt.Errorf("close named snapshot Job Object probe: %w", closeErr)
	}
	if accounting.ActiveProcesses > 0 {
		return snapshotJobActive, nil
	}
	return snapshotJobIdle, nil
}

func openSnapshotJobForQuery(name *uint16) (windows.Handle, error) {
	const jobObjectQuery = 0x0004
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("OpenJobObjectW")
	handle, _, callErr := proc.Call(jobObjectQuery, 0, uintptr(unsafe.Pointer(name)))
	if handle == 0 {
		if callErr == windows.ERROR_SUCCESS {
			callErr = windows.ERROR_INVALID_HANDLE
		}
		return 0, callErr
	}
	return windows.Handle(handle), nil
}

type snapshotJobAccountingInformation struct {
	TotalUserTime, TotalKernelTime                     int64
	ThisPeriodTotalUserTime, ThisPeriodTotalKernelTime int64
	TotalPageFaults, TotalProcesses, ActiveProcesses   uint32
	TotalTerminatedProcesses                           uint32
}
