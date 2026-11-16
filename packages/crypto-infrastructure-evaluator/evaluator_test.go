package evaluator

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

func TestEvaluatePacketAcceptsLargeRetainedEvidenceEnvelope(t *testing.T) {
	c, err := chainforensics.ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	// Several per-source-bounded history pages can make the base64-wrapped
	// owner response exceed the former 16 MiB ceiling.
	body := bytes.Repeat([]byte{'x'}, 4<<20)
	digest := sha256.Sum256(body)
	for i := 0; i < 4; i++ {
		c.Sources = append(c.Sources, chainforensics.Source{
			ID: fmt.Sprintf("large-evidence-%d", i), Name: "Bounded history fixture",
			URL: fmt.Sprintf("https://example.invalid/history/%d", i), Retrieved: c.Sources[0].Retrieved,
			SHA256: hex.EncodeToString(digest[:]), ResultBody: body,
		})
	}
	p, err := chainforensics.BuildPrivatePacket(c)
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		PacketJSON string `json:"packet_json"`
		SHA256     string `json:"sha256"`
	}
	response.PacketJSON = base64.StdEncoding.EncodeToString(p.Bytes)
	response.SHA256 = p.SHA256
	b, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) <= 16<<20 {
		t.Fatalf("fixture envelope did not exceed the former cap: %d bytes", len(b))
	}
	r, err := EvaluatePacket(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("evaluate large retained evidence packet: %v", err)
	}
	if !r.Valid || r.PacketSHA256 != p.SHA256 || r.SourceCount != len(c.Sources) {
		t.Fatalf("large packet digest or evidence inventory changed: %+v", r)
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

func TestSeededDeploymentSuiteRunsReproducibleLinkageAttacks(t *testing.T) {
	c, err := chainforensics.ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	deployments, err := SeededDeployments(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(deployments) != 3 {
		t.Fatalf("expected three deployment targets, got %d", len(deployments))
	}
	var reports []LinkabilityReport
	for _, d := range deployments {
		r, err := EvaluateDeployment(c, d)
		if err != nil {
			t.Fatalf("evaluate %s: %v", d.ID, err)
		}
		if !r.InventoryComplete || r.LayerCount != 7 || r.CoverageScore != 1 {
			t.Fatalf("incomplete layer inventory: %+v", r)
		}
		reports = append(reports, r)
	}
	if reports[0].MatchedAttackCount == 0 || reports[1].MatchedAttackCount == 0 || reports[2].MatchedAttackCount != 0 {
		t.Fatalf("seeded exact/coarse joins and negative control did not separate as expected: %+v", reports)
	}
	negative := deployments[2].Attacks[0]
	negativeAddress, ok := negative.ObservedFields["address"]
	if !ok || len(negative.ObservedFields) != 1 {
		t.Fatalf("negative control must use one supported address field, got %+v", negative.ObservedFields)
	}
	start, err := parseUTC(negative.TimeWindow.StartUTC)
	if err != nil {
		t.Fatal(err)
	}
	end, err := parseUTC(negative.TimeWindow.EndUTC)
	if err != nil {
		t.Fatal(err)
	}
	for _, occurrence := range c.Occurrences {
		observedAt, err := parseUTC(occurrence.Timestamp)
		if err != nil {
			t.Fatal(err)
		}
		if !observedAt.Before(start) && !observedAt.After(end) && fieldsMatch(map[string]string{"address": negativeAddress}, occurrence) {
			t.Fatalf("negative-control address unexpectedly matches in-window occurrence %q", occurrence.ID)
		}
	}
	if reports[2].Attacks[0].Outcome != "no_exact_field_link_found" {
		t.Fatalf("expected absent supported address to produce bounded negative result: %+v", reports[2].Attacks[0])
	}
	for i, d := range deployments {
		r2, err := EvaluateDeployment(c, d)
		if err != nil {
			t.Fatal(err)
		}
		b1, _ := json.Marshal(reports[i])
		b2, _ := json.Marshal(r2)
		if !bytes.Equal(b1, b2) {
			t.Fatalf("evaluation of %s was not byte deterministic", d.ID)
		}
	}
	if !strings.Contains(reports[0].Attacks[0].ResidualUncertainty, "does not establish person") {
		t.Fatalf("report overstates the field join: %+v", reports[0].Attacks[0])
	}
}

func TestEvaluateDeploymentFailsClosedOnMissingLayerAndAttackContext(t *testing.T) {
	c, err := chainforensics.ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	deployments, err := SeededDeployments(c)
	if err != nil {
		t.Fatal(err)
	}
	d := deployments[0]
	d.Layers = d.Layers[:len(d.Layers)-1]
	if _, err := EvaluateDeployment(c, d); err == nil || !strings.Contains(err.Error(), "missing required layer") {
		t.Fatalf("expected missing layer rejection, got %v", err)
	}
	d = deployments[0]
	d.Attacks[0].TimeWindow.StartUTC = "bad-time"
	if _, err := EvaluateDeployment(c, d); err == nil || !strings.Contains(err.Error(), "invalid start_utc") {
		t.Fatalf("expected malformed time window rejection, got %v", err)
	}
	d = deployments[0]
	d.Layers[0].Observers = nil
	if _, err := EvaluateDeployment(c, d); err == nil || !strings.Contains(err.Error(), "explicitly inventory") {
		t.Fatalf("expected unspecified observer inventory rejection, got %v", err)
	}
	deployments, err = SeededDeployments(c)
	if err != nil {
		t.Fatal(err)
	}
	d = deployments[0]
	d.Attacks[0].ObservedFields["unmodeled_field"] = "value"
	if _, err := EvaluateDeployment(c, d); err == nil || !strings.Contains(err.Error(), "unsupported observed field") {
		t.Fatalf("expected unsupported attack field rejection, got %v", err)
	}
}

func TestBoundedAttackDoesNotMatchOutOfWindowOccurrence(t *testing.T) {
	c, err := chainforensics.ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	deployments, err := SeededDeployments(c)
	if err != nil {
		t.Fatal(err)
	}
	d := deployments[0]
	d.Attacks[0].TimeWindow = TimeInterval{StartUTC: "2000-01-01T00:00:00Z", EndUTC: "2000-01-02T00:00:00Z"}
	r, err := EvaluateDeployment(c, d)
	if err != nil {
		t.Fatal(err)
	}
	if r.MatchedAttackCount != 0 || r.Attacks[0].Outcome != "no_exact_field_link_found" {
		t.Fatalf("out-of-window record matched bounded attack: %+v", r.Attacks[0])
	}
}

func TestAttackRequiresDeclaredLayerVantage(t *testing.T) {
	c, err := chainforensics.ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	deployments, err := SeededDeployments(c)
	if err != nil {
		t.Fatal(err)
	}
	d := deployments[0]
	d.Attacks[0].Vantage = "undeclared payment observer"
	if _, err := EvaluateDeployment(c, d); err == nil || !strings.Contains(err.Error(), "not declared as an observer") {
		t.Fatalf("expected undeclared vantage rejection, got %v", err)
	}
}

func TestPacketSeededSuiteConsumesCanonicalPrivatePacket(t *testing.T) {
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
	suite, err := EvaluatePacketSeededSuite(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if suite.CaseEvidence.CaseID != c.ID || len(suite.Deployments) != 3 {
		t.Fatalf("expected packet-backed case evidence and three deployment reports, got %+v", suite)
	}
}
