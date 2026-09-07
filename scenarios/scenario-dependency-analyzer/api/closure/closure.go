// Package closure owns source-distribution closure analysis for consumers
// such as scenario-to-repository. It does not assemble archives or publish;
// it produces the authoritative inventory and explicit unresolved obligations.
package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type File struct {
	SourcePath string `json:"sourcePath"`
	ExportPath string `json:"exportPath"`
	SHA256 string `json:"sha256"`
	Mode uint32 `json:"mode"`
	SizeBytes int64 `json:"sizeBytes"`
	Reasons []string `json:"reasonRefs"`
}
type Node struct { ID string `json:"id"`; Kind string `json:"kind"` }
type Rewrite struct { Kind string `json:"kind"`; ProposalRef string `json:"proposalRef"` }
type SourceClosure struct {
	SourceDigest string `json:"sourceDigest"`
	Scenario string `json:"scenario"`
	Nodes []Node `json:"nodes"`
	Files []File `json:"files"`
	Rewrites []Rewrite `json:"rewrites"`
	Unresolved []string `json:"unresolved"`
	ClosureDigest string `json:"closureDigest"`
}

type Resolver struct{}
func NewResolver() Resolver { return Resolver{} }

func (Resolver) Resolve(root, scenario, sourceDigest string) (SourceClosure, error) {
	root, err := filepath.Abs(root); if err != nil { return SourceClosure{}, fmt.Errorf("resolve source root: %w", err) }
	info, err := os.Stat(root); if err != nil { return SourceClosure{}, fmt.Errorf("source root: %w", err) }; if !info.IsDir() { return SourceClosure{}, fmt.Errorf("source root is not a directory") }
	result := SourceClosure{SourceDigest:sourceDigest, Scenario:scenario, Nodes:[]Node{{ID:"scenario:"+scenario, Kind:"scenario_source"}}}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil { return walkErr }; rel, err := filepath.Rel(root, path); if err != nil { return err }; if rel == "." { return nil }
		parts := strings.Split(filepath.ToSlash(rel), "/"); for _, p := range parts { if p == ".git" || p == "node_modules" || p == "dist" || p == "build" { if entry.IsDir() { return fs.SkipDir }; return nil } }
		if entry.Type()&os.ModeSymlink != 0 { target, err := filepath.EvalSymlinks(path); if err != nil { return err }; inside, err := filepath.Rel(root, target); if err != nil || inside == ".." || strings.HasPrefix(filepath.ToSlash(inside), "../") { return fmt.Errorf("symlink escapes source root: %s", rel) }; return nil }
		if entry.IsDir() { return nil }; if !entry.Type().IsRegular() { return fmt.Errorf("unsupported source entry: %s", rel) }
		if privatePath(rel) { return nil }; data, err := os.ReadFile(path); if err != nil { return err }; hash := sha256.Sum256(data)
		result.Files = append(result.Files, File{SourcePath:filepath.ToSlash(rel), ExportPath:filepath.ToSlash(rel), SHA256:hex.EncodeToString(hash[:]), Mode:uint32(entry.Type().Perm()), SizeBytes:int64(len(data)), Reasons:[]string{"edge:root"}}); return nil
	})
	if err != nil { return SourceClosure{}, err }; sort.Slice(result.Files, func(i,j int) bool { return result.Files[i].ExportPath < result.Files[j].ExportPath })
	result.Nodes = append(result.Nodes, Node{ID:"runtime:declared", Kind:"runtime_requirement"}); canonical, err := json.Marshal(struct{Scenario string `json:"scenario"`; Files []File `json:"files"`; Unresolved []string `json:"unresolved"`}{result.Scenario,result.Files,result.Unresolved}); if err != nil { return SourceClosure{}, err }; digest := sha256.Sum256(canonical); result.ClosureDigest="sha256:"+hex.EncodeToString(digest[:]); return result, nil
}

func privatePath(path string) bool { base:=strings.ToLower(filepath.Base(path)); return strings.HasPrefix(base,".env") || strings.HasSuffix(base,".pem") || strings.HasSuffix(base,".key") || strings.HasSuffix(base,".p12") || strings.Contains(filepath.ToSlash(path), ".vrooli/plan-artifacts/") }
