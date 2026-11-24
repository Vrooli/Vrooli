// Command boundary-inventory prints the admitted Program Runtime capability
// inventory as a deterministic, versioned JSON document.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	repocontract "github.com/vrooli/repo-contract-go"
	"program-runtime/internal/bindings"
)

type document struct {
	Version        int                       `json:"version"`
	Provenance     string                    `json:"provenance"`
	InputDigest    string                    `json:"input_digest"`
	RowsDigest     string                    `json:"rows_digest"`
	SourceDigests  map[string]string         `json:"source_digests"`
	Targets        []bindings.BoundaryTarget `json:"targets"`
	ContractDigest string                    `json:"contract_digest"`
	TargetDigest   string                    `json:"target_digest"`
	Rows           []bindings.Boundary       `json:"rows"`
}

func main() {
	root := flag.String("root", "", "repository root (defaults to current directory or its parent)")
	output := flag.String("output", "", "write JSON to this path (defaults to stdout)")
	check := flag.Bool("check", false, "verify the checked-in owner export matches current admitted registry inputs without writing")
	flag.Parse()
	base := *root
	if base == "" {
		base, _ = os.Getwd()
	}
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(base, "packages", "proto", "gen", "descriptor", "image.binpb")); err == nil {
			break
		}
		base = filepath.Dir(base)
	}
	r, err := bindings.Load(base)
	if err != nil {
		fatal(err)
	}
	rows, sources, targets, contractDigest, err := r.BoundaryExportSnapshot(base)
	if err != nil {
		fatal(err)
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		fatal(err)
	}
	digest := sha256.Sum256(encoded)
	sourceBytes, err := json.Marshal(sources)
	if err != nil {
		fatal(err)
	}
	sourceDigest := sha256.Sum256(sourceBytes)
	targetBytes, _ := json.Marshal(targets)
	targetHash := sha256.Sum256(targetBytes)
	doc := document{Version: 1, Provenance: "program-runtime-admitted-capability", InputDigest: "sha256:" + hex.EncodeToString(sourceDigest[:]), RowsDigest: "sha256:" + hex.EncodeToString(digest[:]), SourceDigests: sources, Rows: rows, Targets: targets, ContractDigest: contractDigest, TargetDigest: "sha256:" + hex.EncodeToString(targetHash[:])}
	var buf bytes.Buffer
	out := json.NewEncoder(&buf)
	out.SetIndent("", "  ")
	if err := out.Encode(doc); err != nil {
		fatal(err)
	}
	verify := func() error {
		confirmedRows, confirmedSources, confirmedTargets, confirmedContract, verifyErr := r.BoundaryExportSnapshot(base)
		if verifyErr != nil {
			return verifyErr
		}
		if !reflect.DeepEqual(rows, confirmedRows) || !reflect.DeepEqual(sources, confirmedSources) || !reflect.DeepEqual(targets, confirmedTargets) || contractDigest != confirmedContract {
			return fmt.Errorf("Program Runtime inputs changed before export commit")
		}
		currentTargets, currentDigest, verifyErr := loadTargetInventory(base)
		if verifyErr != nil {
			return verifyErr
		}
		if !reflect.DeepEqual(targets, currentTargets) || contractDigest != currentDigest {
			return fmt.Errorf("Program Runtime target inventory changed before export commit")
		}
		return nil
	}
	if *check {
		if *output != "" {
			fatal(fmt.Errorf("--check cannot be combined with --output"))
		}
		path := filepath.Join(base, "scenarios", "program-runtime", ".vrooli", "generated", "interoperability-boundaries.json")
		if err := checkExport(path, buf.Bytes(), verify); err != nil {
			fatal(err)
		}
		fmt.Fprintln(os.Stdout, "Program Runtime admitted boundary export is current")
		return
	}
	if *output == "" {
		if err := verify(); err != nil {
			fatal(err)
		}
		if _, err := os.Stdout.Write(buf.Bytes()); err != nil {
			fatal(err)
		}
		return
	}
	if err := atomicWrite(*output, buf.Bytes(), verify); err != nil {
		fatal(err)
	}
}

func checkExportBytes(path string, expected []byte) error {
	actual, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(actual, expected) {
		return fmt.Errorf("Program Runtime admitted boundary export is stale; regenerate with `go run ./cmd/boundary-inventory --output ../.vrooli/generated/interoperability-boundaries.json`")
	}
	return nil
}

func checkExport(path string, expected []byte, verify func() error) error {
	if verify != nil {
		if err := verify(); err != nil {
			return err
		}
	}
	return checkExportBytes(path, expected)
}

func loadTargetInventory(root string) ([]bindings.BoundaryTarget, string, error) {
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return nil, "", err
	}
	targets, err := contract.EnumerateTargets(root)
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(filepath.Join(root, ".vrooli", "repo-contract.json"))
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(data)
	out := make([]bindings.BoundaryTarget, 0, len(targets))
	for _, target := range targets {
		runtimeName := target.ID
		if target.Kind == repocontract.TargetKindProject {
			runtimeName = "vrooli"
		}
		out = append(out, bindings.BoundaryTarget{Kind: string(target.Kind), ID: target.ID, Root: target.Root, RuntimeName: runtimeName})
	}
	return out, "sha256:" + hex.EncodeToString(sum[:]), nil
}

func atomicWrite(path string, data []byte, verify func() error) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".boundary-inventory-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if verify != nil {
		if err = verify(); err != nil {
			return err
		}
	}
	// Go uses the platform replacement primitive (including Windows
	// MOVEFILE_REPLACE_EXISTING); both paths are in this directory.
	return os.Rename(tmpPath, path)
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
