package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// NativeProcessSecurityProof observes the actual native user and elevation for
// comparison with a persisted child claim. It asserts neither vendor sandboxing
// nor least privilege. Exact incarnation/image/Job ownership remain separate.
func NativeProcessSecurityProof(pid int) (string, error) {
	if pid <= 0 {
		return "", ErrOperationIdentity
	}
	return nativeProcessSecurityProof(pid)
}

func windowsOperatorSecurityFactsProof(principal string, elevated bool) (string, error) {
	if principal == "" || !validNativeObserverPrincipal("windows", principal) || elevated {
		return "", ErrOperationIdentity
	}
	raw, _ := json.Marshal(struct {
		Version   int
		Principal string
		Elevated  bool
	}{1, principal, false})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
