package capacity

import (
	"context"
	"path/filepath"
	"strings"
)

const (
	maxAttributionIDLength = 12
	ownerOllama            = "ollama"
	ownerReranker          = "reranker"
)

// Attribution maps an observed PID to the owner (container/scenario/resource)
// responsible for it. Best-effort: OwnerID is "unknown" when it cannot be
// resolved (e.g. a host process, or a non-linux platform).
type Attribution struct {
	PID           int
	ContainerID   string
	ContainerName string
	OwnerKind     string
	OwnerID       string
}

// NormalizeProcessOwner maps native managed-service worker names to their
// resource owner. Native Ollama intentionally runs llama-server as a child
// process, so cgroup/container attribution is unavailable even though the
// resource has a valid claim. Keep this alias in the control-plane attribution
// seam rather than teaching each resource consumer to recognize executables.
func NormalizeProcessOwner(processName string) string {
	trimmed := strings.TrimSpace(processName)
	base := strings.ToLower(filepath.Base(trimmed))
	switch base {
	case ownerOllama, "llama-server":
		return ownerOllama
	default:
		if strings.HasPrefix(base, ownerReranker+"_") || base == ownerReranker {
			return ownerReranker
		}
		// Native managed-service workers commonly run as `python` from an
		// artifact path. The resource name is the path component immediately
		// below the runtime-home `artifacts` directory; callers still verify it
		// against a declared claim before using it as an owner.
		parts := strings.Split(filepath.ToSlash(trimmed), "/")
		for i := 0; i+1 < len(parts); i++ {
			if strings.EqualFold(parts[i], "artifacts") && strings.TrimSpace(parts[i+1]) != "" {
				return strings.ToLower(parts[i+1])
			}
		}
		return ""
	}
}

// OwnerUnknown is the OwnerID used when attribution fails.
const OwnerUnknown = "unknown"

// Attributor resolves a PID to its owner. Implementations never error — they
// degrade to an unknown Attribution — so reconciliation always produces a
// finding for every observed consumer.
type Attributor interface {
	Attribute(ctx context.Context, pid int) Attribution
}

// CgroupSource resolves a PID to its docker container ID via the OS (the
// build-tagged seam: linux reads /proc/<pid>/cgroup, other platforms return
// false cleanly — cross-platform-readiness).
type CgroupSource interface {
	ContainerID(pid int) (string, bool)
}

// DockerNameSource resolves a container ID to its container name (the docker
// inspect seam; injectable so unit tests never shell out).
type DockerNameSource interface {
	ContainerName(ctx context.Context, containerID string) (string, bool)
}

// DockerAttributor attributes PIDs by chaining the cgroup seam (PID -> container
// ID) and the docker seam (container ID -> name), then normalizing the name to
// an owner id.
type DockerAttributor struct {
	Cgroup CgroupSource
	Docker DockerNameSource
}

// NewDockerAttributor builds the production attributor for the current platform.
// On non-linux hosts the cgroup seam yields no container, so every PID resolves
// to unknown (honest, not a crash).
func NewDockerAttributor() DockerAttributor {
	return DockerAttributor{
		Cgroup: newProcCgroupSource(),
		Docker: dockerCLINameSource{},
	}
}

// Attribute resolves a PID to its owner, degrading to unknown at each missing
// link.
func (a DockerAttributor) Attribute(ctx context.Context, pid int) Attribution {
	out := Attribution{PID: pid, OwnerID: OwnerUnknown}
	if a.Cgroup == nil {
		return out
	}
	containerID, ok := a.Cgroup.ContainerID(pid)
	if !ok || strings.TrimSpace(containerID) == "" {
		return out
	}
	out.ContainerID = containerID
	if a.Docker == nil {
		return out
	}
	name, ok := a.Docker.ContainerName(ctx, containerID)
	if !ok || strings.TrimSpace(name) == "" {
		// We know it's containerized but can't name it; surface the short id.
		out.OwnerID = shortContainerID(containerID)
		return out
	}
	out.ContainerName = name
	out.OwnerID = NormalizeOwnerName(name)
	return out
}

// NormalizeOwnerName cleans a docker container name into an owner id, stripping
// the leading slash, common Vrooli prefixes, and a trailing replica index so a
// container like "/vrooli-whisper-1" maps to "whisper".
func NormalizeOwnerName(name string) string {
	n := strings.TrimSpace(name)
	n = strings.TrimPrefix(n, "/")
	for _, prefix := range []string{"vrooli-", "vrooli_", "resource-", "resource_", "scenario-", "scenario_"} {
		n = strings.TrimPrefix(n, prefix)
	}
	// Strip a trailing compose replica suffix ("-1", "_1").
	for _, sep := range []string{"-", "_"} {
		if i := strings.LastIndex(n, sep); i > 0 && i < len(n)-1 {
			if isAllDigits(n[i+1:]) {
				n = n[:i]
			}
		}
	}
	if n == "" {
		return OwnerUnknown
	}
	return n
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func shortContainerID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) > maxAttributionIDLength {
		return id[:12]
	}
	return id
}
