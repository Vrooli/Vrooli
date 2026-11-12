package platform

import "fmt"

// ProcessSecurityContext records only native facts, not an authorization
// decision. Principal is a UID on Unix and a SID on Windows. Elevated is a
// native root/UAC elevation fact, not a proof that the account is least privilege.
type ProcessSecurityContext struct {
	Principal    string
	EffectiveUID int
	Elevated     bool
}

func ProcessSecurity(pid int) (ProcessSecurityContext, error) {
	if pid <= 0 {
		return ProcessSecurityContext{}, fmt.Errorf("platform: invalid pid %d", pid)
	}
	return nativeProcessSecurity(pid)
}
