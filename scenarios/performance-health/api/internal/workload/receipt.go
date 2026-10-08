package workload

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	sweepv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/sweep"
)

// producerReceipt is the declared native producer protocol. The workload owner
// computes percentiles from these observations; no aggregate/pass input exists.
type producerReceipt struct {
	SchemaVersion   int       `json:"schema_version"`
	OperationID     string    `json:"operation_id"`
	StartedAt       time.Time `json:"started_at"`
	FinishedAt      time.Time `json:"finished_at"`
	ProducerDigest  string    `json:"producer_digest"`
	ContractDigest  string    `json:"contract_digest"`
	FixtureRevision string    `json:"fixture_revision"`
	SampleCount     int       `json:"sample_count"`
	DeclaredWarmups int       `json:"declared_warmups"`
	Before          candidate `json:"before"`
	After           candidate `json:"after"`
	Attempts        []sample  `json:"attempts"`
	Errors          []string  `json:"errors"`
}

type candidate struct {
	BuildIdentity  string `json:"build_identity"`
	DriverVersion  string `json:"driver_version"`
	BrowserVersion string `json:"browser_version"`
}

type sample struct {
	Index        int        `json:"index"`
	Warmup       bool       `json:"warmup"`
	Status       string     `json:"status"`
	Error        string     `json:"error"`
	OperationID  string     `json:"operation_id"`
	OwnerMS      float64    `json:"owner_duration_ms"`
	WallMS       float64    `json:"wall_ms"`
	ResponseFile string     `json:"response_file"`
	ResponseSHA  string     `json:"response_sha256"`
	Artifacts    []artifact `json:"artifacts"`
}

type artifact struct {
	Type   string `json:"type"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

func evaluate(raw []byte, d declaration, expected candidate, started, finished time.Time, r *sweepv1.WorkloadReading) error {
	var receipt producerReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return fmt.Errorf("malformed producer receipt: %w", err)
	}
	r.AttemptCount = int32(len(receipt.Attempts))
	if receipt.SchemaVersion != 1 || receipt.OperationID == "" || len(receipt.Errors) > 0 {
		return errors.New("producer receipt failed or lacks its protocol/operation identity")
	}
	if receipt.StartedAt.Before(started.Add(-time.Second)) || receipt.FinishedAt.After(finished.Add(time.Second)) || receipt.FinishedAt.Before(receipt.StartedAt) {
		return errors.New("producer receipt is outside the owned invocation")
	}
	if receipt.ProducerDigest != d.producerDigest || receipt.ContractDigest != d.contractDigest || receipt.Before != expected || receipt.After != expected || !validDigest(receipt.FixtureRevision) {
		return errors.New("producer, contract, fixture or deployed candidate identity mismatch")
	}
	if receipt.SampleCount != d.Samples || receipt.DeclaredWarmups != d.Warmups || len(receipt.Attempts) != d.Samples+d.Warmups {
		return errors.New("producer changed the declared first-attempt denominator")
	}
	seen := make(map[string]bool)
	var owner, wall []float64
	for i, s := range receipt.Attempts {
		if s.Index != i-d.Warmups || s.Warmup != (i < d.Warmups) || s.Status != "verified" || s.Error != "" || s.OperationID == "" || seen[s.OperationID] || !positive(s.OwnerMS) || !positive(s.WallMS) || s.WallMS < s.OwnerMS {
			return fmt.Errorf("attempt %d is failed, repeated, missing, reordered or malformed", i)
		}
		seen[s.OperationID] = true
		if s.ResponseFile == "" || len(s.ResponseSHA) != 64 || len(s.Artifacts) == 0 {
			return fmt.Errorf("attempt %d lacks raw response or artifact evidence", i)
		}
		if !s.Warmup {
			owner = append(owner, s.OwnerMS)
			wall = append(wall, s.WallMS)
		}
	}
	r.FixtureRevision = receipt.FixtureRevision
	r.SampleCount, r.DeclaredWarmups = int32(d.Samples), int32(d.Warmups)
	r.P95Ms, r.WallP95Ms = percentile95(owner), percentile95(wall)
	r.WithinBudget = r.P95Ms <= d.P95MaxMS && r.WallP95Ms <= d.P95MaxMS
	r.Outcome = sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_MEASURED
	return nil
}

func positive(n float64) bool { return n > 0 && !math.IsNaN(n) && !math.IsInf(n, 0) }

func validDigest(value string) bool {
	b, err := hex.DecodeString(value)
	return err == nil && len(b) == 32
}

func percentile95(values []float64) float64 {
	sort.Float64s(values)
	return values[int(math.Ceil(float64(len(values))*0.95))-1]
}

// retainArtifacts copies verified content into this owner's receipt directory,
// deduplicated by hash. BAS retention can then remove its copy without erasing
// qualification evidence. Reads check these retained bytes, not mutable originals.
func retainArtifacts(raw []byte, output string) error {
	var receipt producerReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return err
	}
	for _, s := range receipt.Attempts {
		if err := verifyResponse(output, s); err != nil {
			return err
		}
		for _, a := range s.Artifacts {
			if len(a.SHA256) != 64 || strings.ContainsAny(a.SHA256, "/\\.") {
				return errors.New("invalid artifact digest")
			}
			b, err := readBounded(a.Path, 16<<20)
			if err != nil {
				return err
			}
			if len(b) != a.Bytes || digest(b) != a.SHA256 {
				return errors.New("artifact changed before owner retention")
			}
			if err := os.WriteFile(filepath.Join(output, "artifacts", a.SHA256), b, 0600); err != nil {
				return err
			}
		}
	}
	return nil
}

func verifyRetained(raw []byte, output string) error {
	var receipt producerReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, s := range receipt.Attempts {
		if err := verifyResponse(output, s); err != nil {
			return err
		}
		for _, a := range s.Artifacts {
			if seen[a.SHA256] {
				continue
			}
			seen[a.SHA256] = true
			if len(a.SHA256) != 64 || strings.ContainsAny(a.SHA256, "/\\.") {
				return errors.New("invalid artifact digest")
			}
			b, err := readBounded(filepath.Join(output, "artifacts", a.SHA256), 16<<20)
			if err != nil || len(b) != a.Bytes || digest(b) != a.SHA256 {
				return errors.New("retained artifact missing or changed")
			}
		}
	}
	return nil
}

func verifyResponse(output string, s sample) error {
	if s.ResponseFile == "" || filepath.Base(s.ResponseFile) != s.ResponseFile {
		return errors.New("invalid raw response reference")
	}
	b, err := readBounded(filepath.Join(output, s.ResponseFile), 16<<20)
	if err != nil || digest(b) != s.ResponseSHA {
		return errors.New("raw response missing or changed")
	}
	return nil
}

func readBounded(path string, limit int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("receipt artifact exceeds its file bound")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if int64(len(b)) > limit {
		return nil, errors.New("receipt artifact grew beyond its file bound")
	}
	return b, err
}
