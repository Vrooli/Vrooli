// Package chainforensics contains transport-independent evidence contracts and
// deterministic validation for cross-chain case work. It intentionally does
// not perform identity resolution or network access.
package chainforensics

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
)

//go:embed examples/cases/us-v-1hq3/**
var exampleFS embed.FS

// ExampleCase returns the curated public-case fixture used by offline
// acceptance and demonstrations. It does not make a network request.
func ExampleCase() (Case, error) {
	b, err := exampleFS.ReadFile("examples/cases/us-v-1hq3/case.json")
	if err != nil {
		return Case{}, err
	}
	return DecodeCase(bytes.NewReader(b))
}

// DecodeCase decodes a single typed case document and rejects trailing JSON.
func DecodeCase(r io.Reader) (Case, error) {
	var c Case
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return Case{}, fmt.Errorf("decode case: %w", err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return Case{}, errors.New("decode case: expected exactly one JSON document")
	}
	if err := Validate(c); err != nil {
		return Case{}, err
	}
	return c, nil
}

// MarshalPrivatePacket returns a versioned JSON packet only for cases marked
// private. The returned bytes are not transmitted or written anywhere; the
// owning application must make any local export an explicit user action.
func MarshalPrivatePacket(c Case) ([]byte, error) {
	if !c.Private {
		return nil, errors.New("refusing packet export for a case not marked private")
	}
	if err := Validate(c); err != nil {
		return nil, fmt.Errorf("refusing packet export: %w", err)
	}
	return json.MarshalIndent(struct {
		SchemaVersion string `json:"schema_version"`
		Case          Case   `json:"case"`
	}{SchemaVersion: "chain-forensics.packet/v1", Case: c}, "", "  ")
}

type Chain string

const (
	ChainBitcoin  Chain = "bitcoin"
	ChainEthereum Chain = "ethereum"
	ChainBase     Chain = "base"
	ChainSolana   Chain = "solana"
)

type EvidenceKind string

const (
	EvidenceObservation EvidenceKind = "observation"
	EvidenceLegalClaim  EvidenceKind = "legal_claim"
	EvidenceLabel       EvidenceKind = "public_label"
	EvidenceDerivation  EvidenceKind = "derivation"
)

// Grade describes support for an edge, not the likelihood that a person
// controls an address.
type Grade string

const (
	GradeObserved           Grade = "observed"
	GradeProtocolCorrelated Grade = "protocol_correlated"
	GradeSourceCorroborated Grade = "source_corroborated"
	GradeHypothesis         Grade = "hypothesis"
	GradeUnresolved         Grade = "unresolved"
)

type Source struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Retrieved   string `json:"retrieved_at"`
	SHA256      string `json:"sha256"`
	Citation    string `json:"citation,omitempty"`
	Limitations string `json:"limitations,omitempty"`
}

type Occurrence struct {
	ID            string       `json:"id"`
	Chain         Chain        `json:"chain"`
	Transaction   string       `json:"transaction_id"`
	Block         uint64       `json:"block_height"`
	Timestamp     string       `json:"timestamp"`
	EventIndex    uint32       `json:"event_index"`
	Direction     string       `json:"direction,omitempty"`
	PreviousTx    string       `json:"previous_transaction_id,omitempty"`
	PreviousIndex uint32       `json:"previous_output_index,omitempty"`
	From          string       `json:"from,omitempty"`
	To            string       `json:"to,omitempty"`
	Asset         string       `json:"asset"`
	Atomic        string       `json:"atomic_quantity"`
	Scale         uint32       `json:"scale"`
	Kind          EvidenceKind `json:"kind"`
	SourceID      string       `json:"source_id"`
}

type Edge struct {
	ID          string   `json:"id"`
	From        string   `json:"from_occurrence"`
	To          string   `json:"to_occurrence"`
	Grade       Grade    `json:"grade"`
	Rule        string   `json:"rule"`
	EvidenceIDs []string `json:"evidence_ids"`
	Caveats     []string `json:"caveats,omitempty"`
}

// Claim stores an external source's statement independently from chain
// observations. The caller supplies a concise excerpt or paraphrase and a
// page/paragraph locator; it is never upgraded into an observed occurrence.
type Claim struct {
	ID       string       `json:"id"`
	Text     string       `json:"text"`
	SourceID string       `json:"source_id"`
	Kind     EvidenceKind `json:"kind"`
}

type Case struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Summary     string       `json:"summary"`
	Private     bool         `json:"private"`
	Sources     []Source     `json:"sources"`
	Claims      []Claim      `json:"claims,omitempty"`
	Occurrences []Occurrence `json:"occurrences"`
	Edges       []Edge       `json:"edges"`
	Gaps        []string     `json:"gaps"`
}

var (
	sha256Pattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	decimalPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)
)

// Validate rejects incomplete provenance, ambiguous occurrence identities,
// and edges that refer to missing evidence. Amounts remain decimal strings so
// EVM uint256 values cannot overflow machine integers or lose precision.
func Validate(c Case) error {
	if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.Title) == "" {
		return errors.New("case id and title are required")
	}
	sources := make(map[string]Source, len(c.Sources))
	for _, s := range c.Sources {
		if s.ID == "" || s.Name == "" || s.Retrieved == "" || !sha256Pattern.MatchString(s.SHA256) {
			return fmt.Errorf("source %q has incomplete identity or capture provenance", s.ID)
		}
		u, err := url.ParseRequestURI(s.URL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return fmt.Errorf("source %q has invalid URL", s.ID)
		}
		if _, exists := sources[s.ID]; exists {
			return fmt.Errorf("duplicate source id %q", s.ID)
		}
		sources[s.ID] = s
	}
	occurrences := make(map[string]Occurrence, len(c.Occurrences))
	claims := make(map[string]Claim, len(c.Claims))
	for _, claim := range c.Claims {
		if claim.ID == "" || claim.Text == "" || claim.Kind != EvidenceLegalClaim {
			return fmt.Errorf("claim %q must contain text and use legal_claim kind", claim.ID)
		}
		if _, ok := sources[claim.SourceID]; !ok {
			return fmt.Errorf("claim %q references unknown source %q", claim.ID, claim.SourceID)
		}
		if _, exists := claims[claim.ID]; exists {
			return fmt.Errorf("duplicate claim id %q", claim.ID)
		}
		claims[claim.ID] = claim
	}
	for _, o := range c.Occurrences {
		if o.ID == "" || o.Transaction == "" || o.Asset == "" || !decimalPattern.MatchString(o.Atomic) {
			return fmt.Errorf("occurrence %q has incomplete identity or invalid atomic quantity", o.ID)
		}
		if o.Chain != ChainBitcoin && o.Chain != ChainEthereum && o.Chain != ChainBase && o.Chain != ChainSolana {
			return fmt.Errorf("occurrence %q has unsupported chain %q", o.ID, o.Chain)
		}
		if o.Kind != EvidenceObservation && o.Kind != EvidenceLegalClaim && o.Kind != EvidenceLabel && o.Kind != EvidenceDerivation {
			return fmt.Errorf("occurrence %q has unknown evidence kind %q", o.ID, o.Kind)
		}
		if _, ok := sources[o.SourceID]; !ok {
			return fmt.Errorf("occurrence %q references unknown source %q", o.ID, o.SourceID)
		}
		if _, exists := occurrences[o.ID]; exists {
			return fmt.Errorf("duplicate occurrence id %q", o.ID)
		}
		if o.Chain == ChainBitcoin && o.Direction != "input" && o.Direction != "output" {
			return fmt.Errorf("bitcoin occurrence %q must declare input or output direction", o.ID)
		}
		if o.Chain == ChainBitcoin && o.Direction == "input" && o.PreviousTx == "" {
			return fmt.Errorf("bitcoin input occurrence %q must identify its previous outpoint", o.ID)
		}
		occurrences[o.ID] = o
	}
	for _, e := range c.Edges {
		if e.ID == "" || e.Rule == "" || len(e.EvidenceIDs) == 0 {
			return fmt.Errorf("edge %q must name a rule and evidence", e.ID)
		}
		if _, ok := occurrences[e.From]; !ok {
			return fmt.Errorf("edge %q references unknown source occurrence %q", e.ID, e.From)
		}
		if _, ok := occurrences[e.To]; !ok {
			return fmt.Errorf("edge %q references unknown destination occurrence %q", e.ID, e.To)
		}
		switch e.Grade {
		case GradeObserved, GradeProtocolCorrelated, GradeSourceCorroborated, GradeHypothesis, GradeUnresolved:
		default:
			return fmt.Errorf("edge %q has unknown grade %q", e.ID, e.Grade)
		}
		chains := make(map[Chain]struct{})
		for _, id := range e.EvidenceIDs {
			if _, ok := occurrences[id]; !ok {
				if _, claimOK := claims[id]; claimOK {
					if e.Grade == GradeObserved || e.Grade == GradeProtocolCorrelated {
						return fmt.Errorf("edge %q cannot use a legal claim as %s evidence", e.ID, e.Grade)
					}
					continue
				}
				return fmt.Errorf("edge %q references unknown evidence %q", e.ID, id)
			}
			chains[occurrences[id].Chain] = struct{}{}
		}
		if e.Grade == GradeProtocolCorrelated && len(chains) < 2 {
			return fmt.Errorf("edge %q is protocol_correlated but has no evidence on two distinct chains", e.ID)
		}
	}
	return nil
}
