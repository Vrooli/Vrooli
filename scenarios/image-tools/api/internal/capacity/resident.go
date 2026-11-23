package capacity

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// ResidentOwnerPrefix starts the ledger owner id of every claim a warm worker
// holds for a resident model (its RAM claim and its GPU-UUID VRAM claim).
const ResidentOwnerPrefix = "image-tools:resident:"

// residentTokenMarker separates the model from the owning process's token.
const residentTokenMarker = "@"

// ResidentOwnerID names a resident claim's ledger owner:
// "image-tools:resident:<model>@<process token>[:<suffix>]". The token ties the
// claim to the process that holds it, so a later process can tell a live
// resident from one whose worker died with its owner (a crash, kill -9, or a
// restart whose release timed out) and give the orphan back instead of leaving
// it to the heartbeat expiry, which keeps the memory refused for minutes.
func ResidentOwnerID(model string, suffix ...string) string {
	id := ResidentOwnerPrefix + model + residentTokenMarker + ProcessToken()
	for _, s := range suffix {
		id += ":" + s
	}
	return id
}

var (
	tokenOnce sync.Once
	token     string
)

// ProcessToken identifies this process for its lifetime: pid plus the kernel's
// start time where it is available, so a recycled pid is not mistaken for it.
func ProcessToken() string {
	tokenOnce.Do(func() { token = processToken(os.Getpid()) })
	return token
}

func processToken(pid int) string {
	if start, ok := processStartTicks(pid); ok {
		return fmt.Sprintf("p%d-%d", pid, start)
	}
	return fmt.Sprintf("p%d", pid)
}

// processStartTicks reads field 22 (starttime) of /proc/<pid>/stat. The command
// name is parenthesised and may hold spaces, so fields are counted after the
// last closing parenthesis.
func processStartTicks(pid int) (uint64, bool) {
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0, false
	}
	text := string(raw)
	end := strings.LastIndex(text, ")")
	if end < 0 {
		return 0, false
	}
	fields := strings.Fields(text[end+1:])
	// fields[0] is field 3 (state); starttime is field 22.
	if len(fields) < 20 {
		return 0, false
	}
	start, err := strconv.ParseUint(fields[19], 10, 64)
	return start, err == nil
}

// residentTokenOf extracts the process token from a resident owner id. A claim
// without one was made by a build that did not record it.
func residentTokenOf(ownerID string) (string, bool) {
	rest, ok := strings.CutPrefix(ownerID, ResidentOwnerPrefix)
	if !ok {
		return "", false
	}
	_, after, found := strings.Cut(rest, residentTokenMarker)
	if !found {
		return "", true
	}
	token, _, _ := strings.Cut(after, ":")
	return token, true
}

// IsOrphanedResident reports whether the owner id names a resident claim whose
// holder is gone: it carries no token (a build before tokens existed; at boot
// every such holder has exited with the previous process) or the token names a
// process that is no longer running. Another live image-tools process's
// residents are never orphans.
func IsOrphanedResident(ownerID string) bool {
	tok, resident := residentTokenOf(ownerID)
	if !resident {
		return false
	}
	if tok == "" {
		return true
	}
	return !tokenAlive(tok)
}

func tokenAlive(tok string) bool {
	if tok == ProcessToken() {
		return true
	}
	pidPart, _, hasStart := strings.Cut(strings.TrimPrefix(tok, "p"), "-")
	pid, err := strconv.Atoi(pidPart)
	if err != nil || pid <= 0 {
		return false
	}
	if hasStart {
		// The start time is the identity; any other value is a recycled pid.
		if now, ok := processStartTicks(pid); ok {
			return tok == fmt.Sprintf("p%d-%d", pid, now)
		}
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// ledger is the part of the shared client that reconciliation needs.
type residentLedger interface {
	ListActiveCapacityClaims(ctx context.Context, owns func(ownerID string) bool) ([]vroolicli.CapacityClaimInfo, error)
	ReleaseCapacityClaim(ctx context.Context, claimID string) error
}

// ReleaseOrphanedResidents gives back every active resident claim whose holder
// is gone and returns how many it released. A claim that cannot be released is
// reported but does not stop the rest; the heartbeat expiry still bounds it.
func (b *CLIBroker) ReleaseOrphanedResidents(ctx context.Context) (int, error) {
	client := b.Client
	if client == nil {
		client = vroolicli.New()
	}
	return releaseOrphanedResidents(ctx, client)
}

func releaseOrphanedResidents(ctx context.Context, ledger residentLedger) (int, error) {
	orphans, err := ledger.ListActiveCapacityClaims(ctx, IsOrphanedResident)
	if err != nil {
		return 0, err
	}
	released := 0
	var errs []error
	for _, claim := range orphans {
		if err := ledger.ReleaseCapacityClaim(ctx, claim.ClaimID); err != nil {
			errs = append(errs, err)
			continue
		}
		released++
	}
	return released, errors.Join(errs...)
}
