package execution

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// hostEvidence is the small, stable provenance snapshot attached to every
// execution. It deliberately contains only outcome-relevant facts and a
// digest, rather than a dump of the host environment.
type hostEvidence struct {
	OS         string
	Arch       string
	Node       string
	FactDigest string
}

func currentHostEvidence() hostEvidence {
	return hostEvidenceFromEnvironment(runtime.GOOS, runtime.GOARCH, os.Getenv)
}

func hostEvidenceFromEnvironment(goos, goarch string, getenv func(string) string) hostEvidence {
	node := firstNonEmptyEnv(getenv, "VROOLI_BRIDGE_NODE_ID", "VROOLI_NODE_ID", "VROOLI_HOST_NODE")
	values := []string{
		goos,
		goarch,
		node,
		getenv("VROOLI_INIT_SYSTEM"),
		getenv("XDG_SESSION_TYPE"),
		getenv("VROOLI_DISPLAY_ATTACHED"),
		getenv("VROOLI_HEADLESS"),
		getenv("VROOLI_WSL"),
	}
	hash := sha256.Sum256([]byte(strings.Join(values, "\x00")))
	return hostEvidence{OS: goos, Arch: goarch, Node: node, FactDigest: hex.EncodeToString(hash[:])}
}

func firstNonEmptyEnv(getenv func(string) string, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func (h hostEvidence) String() string {
	return fmt.Sprintf("%s/%s node=%s digest=%s", h.OS, h.Arch, h.Node, h.FactDigest)
}
