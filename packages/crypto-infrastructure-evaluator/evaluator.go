// Package evaluator scores deployment evidence using the shared chain-forensics
// evidence contract. It deliberately does not implement a second case model.
package evaluator

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	chainforensics "github.com/vrooli/chain-forensics"
)

const Version = "ci2-linkability/v1"

// Report is a deterministic inventory of evidence and residual attribution risk.
// Scores are evidence coverage indicators, not anonymity or safety guarantees.
type Report struct {
	Version             string         `json:"version"`
	CaseID              string         `json:"case_id"`
	Valid               bool           `json:"valid"`
	PrivatePacket       bool           `json:"private_packet"`
	PacketSHA256        string         `json:"packet_sha256,omitempty"`
	Chains              []string       `json:"chains"`
	SourceCount         int            `json:"source_count"`
	OccurrenceCount     int            `json:"occurrence_count"`
	EdgeCount           int            `json:"edge_count"`
	GradeCounts         map[string]int `json:"grade_counts"`
	UnresolvedEdges     int            `json:"unresolved_edges"`
	HypothesisEdges     int            `json:"hypothesis_edges"`
	LegalClaimCount     int            `json:"legal_claim_count"`
	CounterexampleCount int            `json:"counterexample_count"`
	ResidualRisks       []string       `json:"residual_risks"`
}

// Evaluate validates the consumer input using the shared package, then derives
// a report without promoting legal claims or weak correlations into findings.
func Evaluate(c chainforensics.Case) (Report, error) {
	if err := chainforensics.Validate(c); err != nil {
		return Report{}, fmt.Errorf("validate chain-forensics case: %w", err)
	}
	r := Report{
		Version: Version, CaseID: c.ID, Valid: true, PrivatePacket: c.Private,
		SourceCount: len(c.Sources), OccurrenceCount: len(c.Occurrences),
		EdgeCount: len(c.Edges), LegalClaimCount: len(c.Claims),
		CounterexampleCount: len(c.Counterexamples) + len(c.Disagreements),
		GradeCounts:         map[string]int{},
	}
	chains := map[string]struct{}{}
	for _, occurrence := range c.Occurrences {
		chains[string(occurrence.Chain)] = struct{}{}
	}
	for chain := range chains {
		r.Chains = append(r.Chains, chain)
	}
	sort.Strings(r.Chains)
	for _, edge := range c.Edges {
		r.GradeCounts[string(edge.Grade)]++
		switch edge.Grade {
		case chainforensics.GradeUnresolved:
			r.UnresolvedEdges++
		case chainforensics.GradeHypothesis:
			r.HypothesisEdges++
		}
	}
	if c.Private {
		packet, err := chainforensics.BuildPrivatePacket(c)
		if err != nil {
			return Report{}, fmt.Errorf("build shared private packet: %w", err)
		}
		r.PacketSHA256 = packet.SHA256
	} else {
		r.ResidualRisks = append(r.ResidualRisks, "case is not marked private; no exportable private packet was produced")
	}
	if r.HypothesisEdges > 0 {
		r.ResidualRisks = append(r.ResidualRisks, "hypothesis-grade links remain and do not establish identity or control")
	}
	if r.UnresolvedEdges > 0 {
		r.ResidualRisks = append(r.ResidualRisks, "unresolved flow edges remain")
	}
	if r.LegalClaimCount > 0 {
		r.ResidualRisks = append(r.ResidualRisks, "legal claims are counted separately from observed chain activity")
	}
	if r.CounterexampleCount > 0 {
		r.ResidualRisks = append(r.ResidualRisks, "counterexamples or source disagreements are present and require analyst review")
	}
	return r, nil
}

// EvaluatePacket accepts the persisted chain-forensics packet wire format and
// checks the supplied digest against the canonical serializer from the shared
// owner package before evaluating the embedded case.
func EvaluatePacket(r io.Reader) (Report, error) {
	var response struct {
		PacketJSON string `json:"packet_json"`
		SHA256     string `json:"sha256"`
	}
	b, err := io.ReadAll(io.LimitReader(r, 16<<20+1))
	if err != nil {
		return Report{}, fmt.Errorf("read packet response: %w", err)
	}
	if len(b) > 16<<20 {
		return Report{}, fmt.Errorf("packet response exceeds 16 MiB")
	}
	if err := json.Unmarshal(b, &response); err != nil {
		return Report{}, fmt.Errorf("decode packet response: %w", err)
	}
	if response.PacketJSON == "" || response.SHA256 == "" {
		return Report{}, fmt.Errorf("packet response must include packet_json and sha256")
	}
	packetBytes, err := base64.StdEncoding.DecodeString(response.PacketJSON)
	if err != nil {
		return Report{}, fmt.Errorf("decode packet_json bytes: %w", err)
	}
	var wire struct {
		SchemaVersion string          `json:"schema_version"`
		Case          json.RawMessage `json:"case"`
	}
	if err := json.Unmarshal(packetBytes, &wire); err != nil {
		return Report{}, fmt.Errorf("decode packet envelope: %w", err)
	}
	if wire.SchemaVersion != "chain-forensics.packet/v1" {
		return Report{}, fmt.Errorf("unsupported packet schema %q", wire.SchemaVersion)
	}
	c, err := chainforensics.DecodeCase(bytes.NewReader(wire.Case))
	if err != nil {
		return Report{}, fmt.Errorf("decode shared case: %w", err)
	}
	packet, err := chainforensics.BuildPrivatePacket(c)
	if err != nil {
		return Report{}, fmt.Errorf("rebuild shared packet: %w", err)
	}
	if response.SHA256 != packet.SHA256 {
		return Report{}, fmt.Errorf("packet SHA-256 mismatch: supplied %q calculated %q", response.SHA256, packet.SHA256)
	}
	report, err := Evaluate(c)
	if err != nil {
		return Report{}, err
	}
	if report.PacketSHA256 != response.SHA256 {
		return Report{}, fmt.Errorf("shared packet digest changed during evaluation")
	}
	return report, nil
}
