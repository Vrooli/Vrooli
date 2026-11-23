package dependencies

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	depdomain "security-health/internal/dependencies"
	"security-health/internal/dependencies/aisearch"
	apiserver "security-health/internal/server"

	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"

	dependenciesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/dependencies"
	dependenciesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/dependencies/dependencies_v1connect"
)

type stubSearcher struct {
	gotReq     depdomain.SearchRequest
	gotVulnReq depdomain.VulnerabilityQuery
	resp       depdomain.SearchResponse
	status     depdomain.Status
	vulns      depdomain.VulnerabilityList
	vuln       depdomain.VulnerabilityRecord
	found      bool
}

type recoverySemanticIndex struct {
	available bool
	queryErr  error
}

func (i *recoverySemanticIndex) EnsureCollection(context.Context) error { return nil }
func (i *recoverySemanticIndex) Sync(context.Context, []aisearch.Item) (int, int, error) {
	return 0, 0, nil
}
func (i *recoverySemanticIndex) Query(context.Context, string, int) ([]aisearch.KeyScore, error) {
	if i.queryErr != nil {
		return nil, i.queryErr
	}
	return []aisearch.KeyScore{{Key: "npm|react|18.0.0", Score: .93}}, nil
}
func (i *recoverySemanticIndex) CountPoints(context.Context) (int, error) { return 1, nil }
func (i *recoverySemanticIndex) Available(context.Context) (bool, bool) {
	return i.available, i.available
}

func TestSearchRoute_ProviderLossKeepsCorpusAndRecoversSameQuery(t *testing.T) {
	ctx := context.Background()
	storeDB := db.NewSQLite(t)
	if _, err := storeDB.ExecContext(ctx, depdomain.Schema()); err != nil {
		t.Fatal(err)
	}
	store := depdomain.NewStore(storeDB)
	if err := store.Apply(ctx, "", []depdomain.DependencyRecord{{
		Scenario: "alpha", Ecosystem: depdomain.EcosystemNPM, Name: "react", Version: "18.0.0", SourceFile: "ui/pnpm-lock.yaml",
	}}, "2026-11-21T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	index := &recoverySemanticIndex{available: true}
	service := depdomain.NewService(depdomain.Deps{Store: store, RepoRoot: t.TempDir(), Clock: schedule.System(), Index: index})
	logger := log.New(io.Discard, "", 0)
	router := apiserver.New(apiserver.Deps{Clock: schedule.System(), Logger: logger}, Module(logger, service))
	httpServer := httptest.NewServer(router.Handler())
	t.Cleanup(httpServer.Close)
	client := dependenciesconnect.NewDependencyServiceClient(httpServer.Client(), httpServer.URL)
	req := connect.NewRequest(&dependenciesv1.SearchRequest{Query: "react", Mode: dependenciesv1.Mode_MODE_AI})

	initial, err := client.Search(ctx, req)
	if err != nil || initial.Msg.GetModeUsed() != dependenciesv1.Mode_MODE_AI || len(initial.Msg.GetResults()) != 1 {
		t.Fatalf("initial route search = (%+v, %v)", initial, err)
	}

	index.available = false
	index.queryErr = errors.New("semantic backend unavailable")
	degraded, err := client.Search(ctx, req)
	if err != nil || degraded.Msg.GetModeUsed() != dependenciesv1.Mode_MODE_TEXT || len(degraded.Msg.GetResults()) != 1 || degraded.Msg.GetResults()[0].GetRecord().GetName() != "react" {
		t.Fatalf("provider-loss route search = (%+v, %v)", degraded, err)
	}
	status, err := client.Status(ctx, connect.NewRequest(&dependenciesv1.StatusRequest{}))
	if err != nil || status.Msg.GetIndexReady() || !status.Msg.GetAvailable() || status.Msg.GetIndexedCount() != 1 {
		t.Fatalf("provider-loss route status = (%+v, %v)", status, err)
	}

	index.available = true
	index.queryErr = nil
	recovered, err := client.Search(ctx, req)
	if err != nil || recovered.Msg.GetModeUsed() != dependenciesv1.Mode_MODE_AI || len(recovered.Msg.GetResults()) != 1 || recovered.Msg.GetResults()[0].GetRecord().GetName() != "react" {
		t.Fatalf("same-query route recovery = (%+v, %v)", recovered, err)
	}
}

func (s *stubSearcher) Search(_ context.Context, req depdomain.SearchRequest) (depdomain.SearchResponse, error) {
	s.gotReq = req
	return s.resp, nil
}
func (s *stubSearcher) Status(context.Context) (depdomain.Status, error) { return s.status, nil }
func (s *stubSearcher) ListVulnerabilities(_ context.Context, req depdomain.VulnerabilityQuery) (depdomain.VulnerabilityList, error) {
	s.gotVulnReq = req
	return s.vulns, nil
}

func (s *stubSearcher) ExplainVulnerability(_ context.Context, req depdomain.VulnerabilityQuery) (depdomain.VulnerabilityRecord, bool, error) {
	s.gotVulnReq = req
	return s.vuln, s.found, nil
}

func TestSearch_MapsFiltersAndRecords(t *testing.T) {
	stub := &stubSearcher{resp: depdomain.SearchResponse{
		ModeUsed: depdomain.ModeText,
		Results: []depdomain.SearchResult{{
			Record: depdomain.DependencyRecord{Scenario: "a", Ecosystem: depdomain.EcosystemGo, Name: "golang.org/x/net", Version: "v0.17.0", VulnIDs: []string{"GO-1"}, MaxSeverity: "high"},
			Score:  0.9,
		}},
	}}
	h := NewConnectHandler(Deps{Service: stub})
	resp, err := h.Search(context.Background(), connect.NewRequest(&dependenciesv1.SearchRequest{
		Query:          "net",
		Ecosystem:      dependenciesv1.Ecosystem_ECOSYSTEM_GO,
		VulnerableOnly: true,
		NameGlob:       "golang.org/x/*",
		Limit:          5,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if stub.gotReq.Ecosystem != depdomain.EcosystemGo || !stub.gotReq.VulnerableOnly || stub.gotReq.NameGlob != "golang.org/x/*" || stub.gotReq.Limit != 5 {
		t.Fatalf("request not mapped: %+v", stub.gotReq)
	}
	if len(resp.Msg.Results) != 1 {
		t.Fatalf("want 1 result, got %d", len(resp.Msg.Results))
	}
	rec := resp.Msg.Results[0].GetRecord()
	if rec.GetEcosystem() != dependenciesv1.Ecosystem_ECOSYSTEM_GO || rec.GetName() != "golang.org/x/net" || rec.GetMaxSeverity() != "high" {
		t.Errorf("record mapping wrong: %+v", rec)
	}
	if resp.Msg.GetModeUsed() != dependenciesv1.Mode_MODE_TEXT {
		t.Errorf("mode_used = %v, want TEXT", resp.Msg.GetModeUsed())
	}
}

func TestStatus_Maps(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &stubSearcher{status: depdomain.Status{
		Available: true, IndexedCount: 7, VulnerableCount: 2, LastReconcileAt: "t",
		IndexedVectors: 4123, ExpectedVectors: 4390, IndexReady: false,
	}}})
	resp, err := h.Status(context.Background(), connect.NewRequest(&dependenciesv1.StatusRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Msg.GetAvailable() || resp.Msg.GetIndexedCount() != 7 || resp.Msg.GetVulnerableCount() != 2 {
		t.Errorf("status mapping wrong: %+v", resp.Msg)
	}
	if resp.Msg.GetIndexedVectors() != 4123 || resp.Msg.GetExpectedVectors() != 4390 || resp.Msg.GetIndexReady() {
		t.Errorf("coverage fields not mapped: %+v", resp.Msg)
	}
}

func TestListVulnerabilities_MapsEvidence(t *testing.T) {
	stub := &stubSearcher{vulns: depdomain.VulnerabilityList{
		Total: 1,
		Vulnerabilities: []depdomain.VulnerabilityRecord{{
			VulnerabilityID:    "GHSA-1",
			Ecosystem:          depdomain.EcosystemNPM,
			Name:               "vite",
			Version:            "5.0.0",
			NormalizedSeverity: "high",
			Source:             depdomain.VulnerabilitySourceOSV,
			Reachability:       depdomain.ReachabilityLockfileAffected,
			Confidence:         depdomain.EvidenceConfidenceAdvisory,
			AffectedRanges:     []depdomain.AffectedVersionRange{{Range: "<5.1.0", Fixed: "5.1.0"}},
			FixedRanges:        []depdomain.FixedVersionRange{{Range: ">= 5.1.0", Version: "5.1.0"}},
			Scenarios:          []string{"demo"},
			SourceFiles:        []string{"ui/pnpm-lock.yaml"},
		}},
	}}
	h := NewConnectHandler(Deps{Service: stub})
	resp, err := h.ListVulnerabilities(context.Background(), connect.NewRequest(&dependenciesv1.ListVulnerabilitiesRequest{
		Ecosystem:         dependenciesv1.Ecosystem_ECOSYSTEM_NPM,
		PackageName:       "vite",
		MinimumConfidence: dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_ADVISORY,
		Limit:             5,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if stub.gotVulnReq.Ecosystem != depdomain.EcosystemNPM || stub.gotVulnReq.PackageName != "vite" || stub.gotVulnReq.MinimumConfidence != depdomain.EvidenceConfidenceAdvisory || stub.gotVulnReq.Limit != 5 {
		t.Fatalf("request not mapped: %+v", stub.gotVulnReq)
	}
	if resp.Msg.GetTotal() != 1 || len(resp.Msg.GetVulnerabilities()) != 1 {
		t.Fatalf("response count wrong: %+v", resp.Msg)
	}
	got := resp.Msg.GetVulnerabilities()[0]
	if got.GetSource() != dependenciesv1.VulnerabilitySource_VULNERABILITY_SOURCE_OSV || got.GetConfidence() != dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_ADVISORY {
		t.Fatalf("evidence enums not mapped: %+v", got)
	}
	if len(got.GetAffectedRanges()) != 1 || got.GetAffectedRanges()[0].GetFixed() != "5.1.0" {
		t.Fatalf("ranges not mapped: %+v", got.GetAffectedRanges())
	}
}

func TestDependencyService_RouterMountsAllProcedures(t *testing.T) {
	stub := &stubSearcher{
		resp: depdomain.SearchResponse{
			ModeUsed: depdomain.ModeText,
			Results: []depdomain.SearchResult{{
				Record: depdomain.DependencyRecord{Scenario: "demo", Ecosystem: depdomain.EcosystemNPM, Name: "vite", Version: "5.0.0"},
				Score:  1,
			}},
		},
		status: depdomain.Status{Available: true, IndexedCount: 1},
		vulns: depdomain.VulnerabilityList{
			Total: 1,
			Vulnerabilities: []depdomain.VulnerabilityRecord{{
				VulnerabilityID:    "GHSA-1234",
				Ecosystem:          depdomain.EcosystemNPM,
				Name:               "vite",
				Version:            "5.0.0",
				NormalizedSeverity: "high",
				Confidence:         depdomain.EvidenceConfidenceDegraded,
			}},
		},
		vuln: depdomain.VulnerabilityRecord{
			VulnerabilityID:    "GHSA-1234",
			Ecosystem:          depdomain.EcosystemNPM,
			Name:               "vite",
			Version:            "5.0.0",
			NormalizedSeverity: "high",
			Confidence:         depdomain.EvidenceConfidenceDegraded,
		},
		found: true,
	}
	srv := apiserver.New(
		apiserver.Deps{Clock: schedule.System(), Logger: log.New(io.Discard, "", 0)},
		Module(log.New(io.Discard, "", 0), stub),
	)
	httpSrv := httptest.NewServer(srv.Handler())
	t.Cleanup(httpSrv.Close)

	client := dependenciesconnect.NewDependencyServiceClient(httpSrv.Client(), httpSrv.URL)
	if resp, err := client.Search(context.Background(), connect.NewRequest(&dependenciesv1.SearchRequest{})); err != nil || len(resp.Msg.GetResults()) != 1 {
		t.Fatalf("Search through router = (%+v, %v), want one result", resp, err)
	}
	if resp, err := client.Status(context.Background(), connect.NewRequest(&dependenciesv1.StatusRequest{})); err != nil || !resp.Msg.GetAvailable() {
		t.Fatalf("Status through router = (%+v, %v), want available", resp, err)
	}
	if resp, err := client.ListVulnerabilities(context.Background(), connect.NewRequest(&dependenciesv1.ListVulnerabilitiesRequest{})); err != nil || resp.Msg.GetTotal() != 1 {
		t.Fatalf("ListVulnerabilities through router = (%+v, %v), want total=1", resp, err)
	}
	if resp, err := client.ExplainVulnerability(context.Background(), connect.NewRequest(&dependenciesv1.ExplainVulnerabilityRequest{VulnerabilityId: "GHSA-1234"})); err != nil || !resp.Msg.GetFound() {
		t.Fatalf("ExplainVulnerability through router = (%+v, %v), want found", resp, err)
	}
}
