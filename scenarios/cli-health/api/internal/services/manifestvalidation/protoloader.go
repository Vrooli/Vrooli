package manifestvalidation

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// BufProtoLoader invokes `buf build --path schemas/<scenario>` from
// packages/proto and parses the resulting FileDescriptorSet into a flat
// ProtoSurface. Requires the `buf` binary on PATH; v1's expectation is that
// every Vrooli host already has it (cli-health is an internal tool).
type BufProtoLoader struct {
	RepoRoot string

	// Buf overrides the path to the buf binary; defaults to "buf".
	Buf string
}

// NewBufProtoLoader returns a loader rooted at the given repo dir.
func NewBufProtoLoader(repoRoot string) *BufProtoLoader {
	return &BufProtoLoader{RepoRoot: repoRoot}
}

// sharedControlSubPath is the shared, token-gated search control plane
// (search-hub.v1.control.SearchControlService) — a cross-scenario contract any
// search provider may SERVE. It is loaded alongside every scenario's own proto so
// a provider that adopts it (e.g. cli-health) can bind its reindex verbs from the
// CLI without the validator mistaking a legitimately-served shared service for an
// undeclared one. Files under this path are classified as Shared (bindable, not
// coverage-checked). For search-hub itself the path is part of its own subtree,
// so it stays an own service there (the own-prefix check wins).
var (
	sharedControlSubPath = filepath.Join("schemas", "search-hub", "v1", "control")
	sharedControlPrefix  = "search-hub/v1/control/"
)

// Load shells out to buf to produce a binary FileDescriptorSet for the
// scenario's proto subtree (plus the shared control plane), then walks it to
// extract services and methods.
func (l *BufProtoLoader) Load(ctx context.Context, scenario string) (ProtoSurface, error) {
	bufBin := l.Buf
	if bufBin == "" {
		bufBin = "buf"
	}
	protoDir := filepath.Join(l.RepoRoot, "packages", "proto")
	subPath := filepath.Join("schemas", scenario)

	if _, err := os.Stat(filepath.Join(protoDir, subPath)); err != nil {
		if os.IsNotExist(err) {
			return ProtoSurface{}, fmt.Errorf("no proto schemas at packages/proto/%s for scenario %q", subPath, scenario)
		}
		return ProtoSurface{}, fmt.Errorf("stat proto dir: %w", err)
	}

	tmp, err := os.CreateTemp("", "cli-health-"+scenario+"-*.binpb")
	if err != nil {
		return ProtoSurface{}, fmt.Errorf("create temp: %w", err)
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	args := []string{"build", "--path", subPath}
	// Also load the shared control plane so adopters can bind it. buf dedups when
	// the scenario IS search-hub (the path is already inside its subtree).
	if _, err := os.Stat(filepath.Join(protoDir, sharedControlSubPath)); err == nil {
		args = append(args, "--path", sharedControlSubPath)
	}
	args = append(args, "-o", tmp.Name())

	cmd := exec.CommandContext(ctx, bufBin, args...)
	cmd.Dir = protoDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return ProtoSurface{}, fmt.Errorf("buf build failed for %q: %v\n%s", scenario, err, stderr.String())
	}

	raw, err := os.ReadFile(tmp.Name())
	if err != nil {
		return ProtoSurface{}, fmt.Errorf("read fdset: %w", err)
	}
	var fds descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(raw, &fds); err != nil {
		return ProtoSurface{}, fmt.Errorf("unmarshal fdset: %w", err)
	}

	return surfaceFromFDSet(&fds, scenario), nil
}

// surfaceFromFDSet collects services declared in files that belong to the
// named scenario (own) plus the shared control plane (shared). We filter by file
// path prefix so vendor/transitive deps (googleapis, protovalidate, errors common
// to multiple scenarios) don't leak into the surface and produce spurious orphans.
// A file under the scenario's own prefix is classified as own even if it also
// matches the shared prefix (the search-hub case), so the control service is an
// own service exactly where it is defined.
func surfaceFromFDSet(fds *descriptorpb.FileDescriptorSet, scenario string) ProtoSurface {
	prefix := scenario + "/"
	var own, shared []ProtoService
	for _, f := range fds.File {
		name := f.GetName()
		if name == "" {
			continue
		}
		isOwn := startsWith(name, prefix)
		isShared := startsWith(name, sharedControlPrefix)
		if !isOwn && !isShared {
			continue
		}
		for _, svc := range f.GetService() {
			methods := make([]string, 0, len(svc.GetMethod()))
			for _, m := range svc.GetMethod() {
				methods = append(methods, m.GetName())
			}
			ps := ProtoService{Name: svc.GetName(), Methods: methods}
			if isOwn {
				own = append(own, ps)
			} else {
				shared = append(shared, ps)
			}
		}
	}
	return ProtoSurface{Services: own, Shared: shared}
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
