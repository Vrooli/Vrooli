package manifestvalidation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
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

// sharedProtoContracts are cross-scenario contracts a scenario may SERVE from
// its CLI/API without owning every RPC. They are loaded alongside every
// scenario's own proto and classified as Shared (bindable, not coverage-checked).
// For a scenario whose own subtree contains one of these paths, the own-prefix
// check wins so the service remains coverage-checked there.
type sharedProtoContract struct {
	subPath string
	prefix  string
}

type sharedProtoContractsFile struct {
	Contracts []struct {
		Prefix string `json:"prefix"`
	} `json:"contracts"`
}

func loadSharedProtoContracts(repoRoot string) []sharedProtoContract {
	data, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "schemas", "shared-proto-contracts.json"))
	if err != nil {
		return nil
	}
	var declaration sharedProtoContractsFile
	if json.Unmarshal(data, &declaration) != nil {
		return nil
	}
	contracts := make([]sharedProtoContract, 0, len(declaration.Contracts))
	for _, contract := range declaration.Contracts {
		prefix := filepath.ToSlash(strings.TrimSuffix(contract.Prefix, "/"))
		if prefix == "" {
			continue
		}
		contracts = append(contracts, sharedProtoContract{subPath: filepath.FromSlash(filepath.Join("schemas", prefix)), prefix: prefix + "/"})
	}
	return contracts
}

// Load shells out to buf to produce a binary FileDescriptorSet for the
// scenario's proto subtree (plus the shared control plane), then walks it to
// extract services and methods.
func (l *BufProtoLoader) Load(ctx context.Context, scenario string) (ProtoSurface, error) {
	bufBin := l.Buf
	if bufBin == "" {
		bufBin = "buf"
	}
	protoDir := filepath.Join(l.RepoRoot, "packages", "proto")
	sharedContracts := loadSharedProtoContracts(l.RepoRoot)
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
	for _, contract := range sharedContracts {
		if _, err := os.Stat(filepath.Join(protoDir, contract.subPath)); err == nil {
			args = append(args, "--path", contract.subPath)
		}
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

	return surfaceFromFDSet(&fds, scenario, sharedContracts), nil
}

// surfaceFromFDSet collects services declared in files that belong to the named
// scenario (own) plus shared proto contracts (shared). We filter by file path
// prefix so vendor/transitive deps (googleapis, protovalidate, errors common to
// multiple scenarios) don't leak into the surface and produce spurious orphans.
// A file under the scenario's own prefix is classified as own even if it also
// matches a shared prefix, so shared services are own where they are defined.
func surfaceFromFDSet(fds *descriptorpb.FileDescriptorSet, scenario string, sharedContracts []sharedProtoContract) ProtoSurface {
	prefix := scenario + "/"
	var own, shared []ProtoService
	requests := make(map[string]protoreflect.MessageDescriptor)
	requestCandidates := make(map[string][]ProtoRequestCandidate)
	if files, err := protodesc.NewFiles(fds); err == nil {
		files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
			isOwn := startsWith(fd.Path(), prefix)
			isShared := !isOwn
			if isShared {
				isShared = false
				for _, contract := range sharedContracts {
					if startsWith(fd.Path(), contract.prefix) {
						isShared = true
						break
					}
				}
			}
			if !isOwn && !isShared {
				return true
			}
			for i := 0; i < fd.Services().Len(); i++ {
				svc := fd.Services().Get(i)
				for j := 0; j < svc.Methods().Len(); j++ {
					method := svc.Methods().Get(j)
					key := string(svc.Name()) + "." + string(method.Name())
					candidate := ProtoRequestCandidate{Request: method.Input(), Source: fd.Path(), Shared: isShared}
					requestCandidates[key] = append(requestCandidates[key], candidate)
					if _, exists := requests[key]; !exists {
						requests[key] = method.Input()
					}
				}
			}
			return true
		})
	}
	for _, f := range fds.File {
		name := f.GetName()
		if name == "" {
			continue
		}
		isOwn := startsWith(name, prefix)
		isShared := false
		for _, contract := range sharedContracts {
			if startsWith(name, contract.prefix) {
				isShared = true
				break
			}
		}
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
	return ProtoSurface{Services: own, Shared: shared, Requests: requests, RequestCandidates: requestCandidates}
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
