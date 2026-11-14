package evaluator

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	chainforensics "github.com/vrooli/chain-forensics"
)

func TestEvaluateUsesSharedCaseValidationAndPacketSerializer(t *testing.T) {
	c, err := chainforensics.ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	r, err := Evaluate(c)
	if err != nil {
		t.Fatal(err)
	}
	if r.Version != Version || !r.Valid || !r.PrivatePacket || r.PacketSHA256 == "" {
		t.Fatalf("expected valid private packet report, got %+v", r)
	}
	if r.CaseID != c.ID || r.SourceCount != len(c.Sources) || r.OccurrenceCount != len(c.Occurrences) || r.EdgeCount != len(c.Edges) {
		t.Fatalf("report inventory diverged from shared case: %+v", r)
	}
	if r.GradeCounts[string(chainforensics.GradeObserved)] == 0 || r.LegalClaimCount == 0 {
		t.Fatalf("expected observed flows and separately counted claim, got %+v", r)
	}
	if len(r.ResidualRisks) == 0 || !strings.Contains(strings.Join(r.ResidualRisks, " "), "claims") {
		t.Fatalf("expected claim boundary to remain explicit, got %+v", r.ResidualRisks)
	}
}

func TestEvaluatePacketChecksSharedCanonicalDigest(t *testing.T) {
	c, err := chainforensics.ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	p, err := chainforensics.BuildPrivatePacket(c)
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		PacketJSON string `json:"packet_json"`
		SHA256     string `json:"sha256"`
	}
	response.PacketJSON, response.SHA256 = base64.StdEncoding.EncodeToString(p.Bytes), p.SHA256
	b, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	r, err := EvaluatePacket(bytes.NewReader(b))
	if err != nil || r.PacketSHA256 != p.SHA256 {
		t.Fatalf("expected package-backed packet check, report=%+v err=%v", r, err)
	}
	response.SHA256 = strings.Repeat("0", 64)
	b, _ = json.Marshal(response)
	if _, err := EvaluatePacket(bytes.NewReader(b)); err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("expected tamper rejection, got %v", err)
	}
}

func TestEvaluateFailsClosedOnTamperedSourceBytes(t *testing.T) {
	c, err := chainforensics.ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	c.Sources[0].ResultBody = append(c.Sources[0].ResultBody, 'x')
	if _, err := Evaluate(c); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("expected shared validator to reject tampered source, got %v", err)
	}
}
