package platform

import (
	"fmt"
	"strconv"
	"strings"
)

// NativeWorkloadIdentity is versioned separately from the legacy Linux
// receipt. Its zero value is never authority. RootIncarnation is the exact
// native process identity, not a PID or a cross-platform timestamp guess.
type NativeWorkloadIdentity struct {
	Version           int    `json:"version"`
	Platform          string `json:"platform"`
	Manager           string `json:"manager"`
	Domain            string `json:"domain"`
	Name              string `json:"name"`
	InvocationID      string `json:"invocation_id"`
	Account           string `json:"account"`
	ContainmentID     string `json:"containment_id"`
	BootID            string `json:"boot_id"`
	RootIncarnation   string `json:"root_incarnation,omitempty"`
	ObserverPrincipal string `json:"observer_principal,omitempty"`
	RootUniqueID      string `json:"root_unique_id,omitempty"`
	ProcessOnly       bool   `json:"process_only,omitempty"`
}

func validatePortableNativeIdentity(r OperationRef) error {
	n := r.Native
	if n == nil || n.Version != 2 || r.Scope.Kind != ScopeKindNative || r.Scope.PID != 0 || r.ScopeDevice != 0 || r.ScopeInode != 0 || n.InvocationID != r.Intent.InvocationID || r.NativeInvocationID != n.InvocationID || n.Account == "" || n.ContainmentID == "" || !serviceBootIDPattern.MatchString(strings.ToLower(n.BootID)) || r.Scope.Path != n.Domain+"/"+n.Name {
		return fmt.Errorf("%w: malformed portable native identity", ErrOperationIdentity)
	}
	if !validNativeObserverPrincipal(n.Platform, n.ObserverPrincipal) {
		return ErrOperationIdentity
	}
	if err := validateOperatorNativeIdentity(r); err != nil {
		return err
	}

	for _, value := range []string{n.Account, n.ContainmentID, n.RootIncarnation, n.ObserverPrincipal} {
		if len(value) > 4096 || strings.ContainsAny(value, "\x00\r\n") {
			return ErrOperationIdentity
		}
	}
	if r.RootPID > 0 {
		ticks, err := strconv.ParseUint(r.RootStartTime, 10, 64)
		if err != nil || ticks == 0 || !nativeIncarnationShape(n.Platform, r.RootPID, n.RootIncarnation) || !nativeStartMatches(n.Platform, r.RootStartTime, n.RootIncarnation) {
			return ErrOperationIdentity
		}
	} else if n.RootIncarnation != "" {
		return ErrOperationIdentity
	}
	return nil
}

func validateOperatorNativeIdentity(r OperationRef) error {
	n := r.Native
	if n.Account != r.Intent.NativeAccount || !validNativeObserverPrincipal(n.Platform, n.Account) || n.ObserverPrincipal != "" && n.ObserverPrincipal != n.Account {
		return ErrOperationIdentity
	}
	switch n.Platform {
	case "darwin":
		uid := strings.TrimPrefix(n.Account, "uid:")
		if !strings.HasPrefix(n.Account, "uid:") || n.Manager != "launchd" || n.Domain != "user/"+uid || n.Name != darwinOwnedLabel(r.Intent) || n.ContainmentID != n.Domain+"/"+n.Name || !n.ProcessOnly {
			return ErrOperationIdentity
		}
		if r.RootPID > 0 {
			id, err := strconv.ParseUint(n.RootUniqueID, 10, 64)
			if err != nil || id == 0 || strconv.FormatUint(id, 10) != n.RootUniqueID {
				return ErrOperationIdentity
			}
		} else if n.RootUniqueID != "" {
			return ErrOperationIdentity
		}
	case "windows":
		if n.Manager != "operator-job" || n.Domain != "operator" || n.Name != windowsOwnedServiceName(r.Intent) || n.ContainmentID != windowsOwnedJob(r.Intent) || n.ProcessOnly || n.RootUniqueID != "" {
			return ErrOperationIdentity
		}
	default:
		return ErrOperationIdentity
	}
	return nil
}

func nativeIncarnationShape(platform string, pid int, value string) bool {
	parts := strings.Split(value, ":")
	expected := 3
	if platform == "darwin" {
		expected = 4
	}
	if len(parts) != expected || parts[0] != platform || parts[1] != strconv.Itoa(pid) {
		return false
	}
	for _, s := range parts[2:] {
		if _, err := strconv.ParseUint(s, 10, 64); err != nil || s == "" {
			return false
		}
	}
	return parts[2] != "0"
}

// MatchesIncarnation compares an observed native process identity with the
// exact persisted receipt. It performs no native I/O and asserts no liveness.
func (r ServiceRef) MatchesIncarnation(value string) bool {
	if r.ValidateIdentity() != nil || r.RootPID <= 0 {
		return false
	}
	return OperationRef(r).matchesIncarnation(value)
}
func (r OperationRef) MatchesIncarnation(value string) bool {
	if r.ValidateIdentity() != nil || r.RootPID <= 0 {
		return false
	}
	return r.matchesIncarnation(value)
}
func (r OperationRef) matchesIncarnation(value string) bool {
	if r.Native != nil {
		return value == r.Native.RootIncarnation
	}
	boot, ticks, ok := strings.Cut(value, ":")
	return ok && boot != "" && len(boot) <= 64 && !strings.ContainsAny(boot, "\x00\r\n") && ticks == r.RootStartTime
}

// NativeOwnedIdentityComplete checks durable identity structure. It is not a
// native ownership observation or proof that an operation is still running.
func (r ServiceRef) NativeOwnedIdentityComplete() bool {
	if r.ValidateIdentity() != nil || r.RootPID <= 0 {
		return false
	}
	if r.Native != nil {
		return r.Native.RootIncarnation != ""
	}
	return r.ScopeDevice != 0 && r.ScopeInode != 0
}

// Equal compares receipt values, including separately decoded native identity.
// Pointer addresses are never workload identity.
func (r OperationRef) Equal(other OperationRef) bool {
	rn, on := r.Native, other.Native
	r.Native, other.Native = nil, nil
	if r != other {
		return false
	}
	if rn == nil || on == nil {
		return rn == nil && on == nil
	}
	return *rn == *on
}
func (r ServiceRef) Equal(other ServiceRef) bool { return OperationRef(r).Equal(OperationRef(other)) }

func nativeStartMatches(target, start, incarnation string) bool {
	p := strings.Split(incarnation, ":")
	if target == "darwin" {
		if len(p) != 4 {
			return false
		}
		sec, e := strconv.ParseUint(p[2], 10, 64)
		if e != nil || sec > 18446744073709 {
			return false
		}
		us, e := strconv.ParseUint(p[3], 10, 64)
		if e != nil || us >= 1000000 {
			return false
		}
		return start == strconv.FormatUint(sec*1000000+us, 10)
	}
	if target == "windows" {
		if len(p) != 3 {
			return false
		}
		ticks, e := strconv.ParseUint(start, 10, 64)
		if e != nil || ticks < 116444736000000000 {
			return false
		}
		delta := ticks - 116444736000000000
		if delta > 184467440737095516 {
			return false
		}
		return p[2] == strconv.FormatUint(delta*100, 10)
	}
	return false
}

func validNativeObserverPrincipal(target, value string) bool {
	if value == "" {
		return true
	}
	if target == "darwin" {
		s, ok := strings.CutPrefix(value, "uid:")
		if !ok {
			return false
		}
		n, e := strconv.ParseUint(s, 10, 32)
		return e == nil && n > 0 && s == strconv.FormatUint(n, 10)
	}

	if target == "windows" {
		s, ok := strings.CutPrefix(value, "sid:S-1-")
		if !ok {
			return false
		}
		parts := strings.Split(s, "-")
		if len(parts) < 2 || len(parts) > 16 {
			return false
		}
		authority, e := strconv.ParseUint(parts[0], 10, 48)
		if e != nil || parts[0] != strconv.FormatUint(authority, 10) {
			return false
		}
		for _, part := range parts[1:] {
			n, e := strconv.ParseUint(part, 10, 32)
			if e != nil || part != strconv.FormatUint(n, 10) {
				return false
			}
		}
		return value != "sid:S-1-5-18" && value != "sid:S-1-5-19" && value != "sid:S-1-5-20" && !strings.HasPrefix(value, "sid:S-1-5-32-") && !strings.HasPrefix(value, "sid:S-1-5-80-")
	}
	return false
}
