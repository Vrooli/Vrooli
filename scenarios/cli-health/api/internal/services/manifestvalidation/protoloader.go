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

// Load shells out to buf to produce a binary FileDescriptorSet for the
// scenario's proto subtree, then walks it to extract services and methods.
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

	cmd := exec.CommandContext(ctx, bufBin, "build", "--path", subPath, "-o", tmp.Name())
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
// named scenario. We filter by file path prefix so vendor/transitive deps
// (googleapis, protovalidate, errors common to multiple scenarios) don't
// leak into the surface and produce spurious orphans.
func surfaceFromFDSet(fds *descriptorpb.FileDescriptorSet, scenario string) ProtoSurface {
	prefix := scenario + "/"
	var services []ProtoService
	for _, f := range fds.File {
		if f.GetName() == "" {
			continue
		}
		if !startsWith(f.GetName(), prefix) {
			continue
		}
		for _, svc := range f.GetService() {
			methods := make([]string, 0, len(svc.GetMethod()))
			for _, m := range svc.GetMethod() {
				methods = append(methods, m.GetName())
			}
			services = append(services, ProtoService{
				Name:    svc.GetName(),
				Methods: methods,
			})
		}
	}
	return ProtoSurface{Services: services}
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
