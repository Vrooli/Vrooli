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
	node := firstNonEmptyEnv("VROOLI_BRIDGE_NODE_ID", "VROOLI_NODE_ID", "VROOLI_HOST_NODE")
	values := []string{
		runtime.GOOS,
		runtime.GOARCH,
		node,
		os.Getenv("VROOLI_INIT_SYSTEM"),
		os.Getenv("XDG_SESSION_TYPE"),
		os.Getenv("VROOLI_DISPLAY_ATTACHED"),
		os.Getenv("VROOLI_HEADLESS"),
		os.Getenv("VROOLI_WSL"),
	}
	hash := sha256.Sum256([]byte(strings.Join(values, "\x00")))
	return hostEvidence{OS: runtime.GOOS, Arch: runtime.GOARCH, Node: node, FactDigest: hex.EncodeToString(hash[:])}
}

func firstNonEmptyEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func (h hostEvidence) String() string {
	return fmt.Sprintf("%s/%s node=%s digest=%s", h.OS, h.Arch, h.Node, h.FactDigest)
}
