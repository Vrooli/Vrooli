package fixtures

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// OracleAlgorithm names the deterministic expected-state derivation.
//
// fixture-oracle/v1:
//
//	seed checksum   = sha256 over "<path>\n<sha256hex(bytes)>\n" for each seed file in declared order
//	expected state  = {algorithm, files[{path, bytes, records, sha256}], total_records, total_bytes}
//	records         = JSON array length | JSON object key count | CSV rows minus header | 1
//	oracle checksum = sha256 over canonical JSON (sorted keys, no whitespace) of expected state
const OracleAlgorithm = "fixture-oracle/v1"

// FileState is the observed state of one seed file.
type FileState struct {
	Path    string `json:"path"`
	Bytes   int    `json:"bytes"`
	Records int    `json:"records"`
	SHA256  string `json:"sha256"`
}

// ExpectedState is the deterministic expected state of a seeded fixture.
type ExpectedState struct {
	Algorithm    string      `json:"algorithm"`
	Files        []FileState `json:"files"`
	TotalRecords int         `json:"total_records"`
	TotalBytes   int         `json:"total_bytes"`
}

// Computed is the result of deriving seed and oracle checksums from seed bytes.
type Computed struct {
	SeedChecksum   string
	ExpectedState  ExpectedState
	OracleChecksum string
}

// Compute derives the seed checksum and expected state from the fixture's
// seed files. It reads only the declared files, so an undeclared file cannot
// silently change the oracle.
func Compute(w *Workload) (Computed, error) {
	var seedLines strings.Builder
	state := ExpectedState{Algorithm: OracleAlgorithm, Files: make([]FileState, 0, len(w.Seed.Files))}
	for _, rel := range w.Seed.Files {
		if filepath.IsAbs(rel) || strings.Contains(rel, "..") {
			return Computed{}, fmt.Errorf("seed file %q must be a relative path inside the fixture", rel)
		}
		data, err := os.ReadFile(filepath.Join(w.Dir, filepath.FromSlash(rel)))
		if err != nil {
			return Computed{}, fmt.Errorf("read seed %s: %w", rel, err)
		}
		sum := sha256.Sum256(data)
		h := hex.EncodeToString(sum[:])
		fmt.Fprintf(&seedLines, "%s\n%s\n", rel, h)
		records, err := countRecords(rel, data)
		if err != nil {
			return Computed{}, fmt.Errorf("seed %s: %w", rel, err)
		}
		state.Files = append(state.Files, FileState{Path: rel, Bytes: len(data), Records: records, SHA256: h})
		state.TotalRecords += records
		state.TotalBytes += len(data)
	}
	seedSum := sha256.Sum256([]byte(seedLines.String()))
	canon, err := canonicalJSON(state)
	if err != nil {
		return Computed{}, err
	}
	oracleSum := sha256.Sum256(canon)
	return Computed{
		SeedChecksum:   "sha256:" + hex.EncodeToString(seedSum[:]),
		ExpectedState:  state,
		OracleChecksum: "sha256:" + hex.EncodeToString(oracleSum[:]),
	}, nil
}

// VerifyOracle recomputes the fixture's checksums and returns an error naming
// the first mismatch against the declared seed and oracle.
func VerifyOracle(w *Workload) (Computed, error) {
	c, err := Compute(w)
	if err != nil {
		return c, err
	}
	if c.SeedChecksum != w.Seed.Checksum {
		return c, fmt.Errorf("%s: seed checksum drift: declared %s, computed %s", w.ID, w.Seed.Checksum, c.SeedChecksum)
	}
	if c.ExpectedState.TotalRecords != w.Seed.Records {
		return c, fmt.Errorf("%s: seed records drift: declared %d, computed %d", w.ID, w.Seed.Records, c.ExpectedState.TotalRecords)
	}
	if c.OracleChecksum != w.Oracle.Checksum {
		return c, fmt.Errorf("%s: oracle checksum drift: declared %s, computed %s", w.ID, w.Oracle.Checksum, c.OracleChecksum)
	}
	declared, err := canonicalJSON(w.Oracle.ExpectedState)
	if err != nil {
		return c, err
	}
	computed, err := canonicalJSON(c.ExpectedState)
	if err != nil {
		return c, err
	}
	if !bytes.Equal(declared, computed) {
		return c, fmt.Errorf("%s: declared expected_state differs from the state derived from seed bytes", w.ID)
	}
	return c, nil
}

func countRecords(rel string, data []byte) (int, error) {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".json":
		var doc any
		if err := json.Unmarshal(data, &doc); err != nil {
			return 0, fmt.Errorf("invalid JSON: %w", err)
		}
		switch v := doc.(type) {
		case []any:
			return len(v), nil
		case map[string]any:
			return len(v), nil
		default:
			return 1, nil
		}
	case ".csv":
		rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
		if err != nil {
			return 0, fmt.Errorf("invalid CSV: %w", err)
		}
		if len(rows) == 0 {
			return 0, nil
		}
		return len(rows) - 1, nil
	default:
		return 1, nil
	}
}

// canonicalJSON encodes v with sorted keys and no whitespace, matching the
// generator's canonical form regardless of struct field order.
func canonicalJSON(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(generic); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
