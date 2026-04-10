package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
)

func TestDeploymentVPSBundles_ListAndGCContract(t *testing.T) {
	t.Setenv("API_PORT", "0")

	manifest := domain.CloudManifest{
		Version: "1.0.0",
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS: &domain.ManifestVPS{
				Host:    "203.0.113.10",
				Port:    22,
				User:    "root",
				KeyPath: "/tmp/fake-key",
				Workdir: "/root/Vrooli",
			},
		},
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Dependencies: domain.ManifestDependencies{
			Scenarios: []string{"landing-page-business-suite"},
			Resources: []string{},
		},
		Bundle: domain.ManifestBundle{IncludePackages: true, IncludeAutoheal: true},
		Ports:  domain.ManifestPorts{},
		Edge:   domain.ManifestEdge{Domain: "example.com", Caddy: domain.ManifestCaddy{Enabled: true}},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	deploymentID := "dep-123"
	shaOld := strings.Repeat("a", 64)
	shaMid := strings.Repeat("b", 64)
	shaNew := strings.Repeat("c", 64)

	// Simulated remote bundles directory state. Times are unix seconds.
	remote := map[string]struct {
		size int64
		mt   int64
	}{
		"mini-vrooli_landing-page-business-suite_" + shaOld + ".tar.gz": {size: 100, mt: 1000},
		"mini-vrooli_landing-page-business-suite_" + shaMid + ".tar.gz": {size: 200, mt: 2000},
		"mini-vrooli_landing-page-business-suite_" + shaNew + ".tar.gz": {size: 300, mt: 3000},
		"mini-vrooli_other-scenario_" + strings.Repeat("d", 64) + ".tar.gz": {size: 400, mt: 4000},
	}

	fakeSSH := &FakeSSHRunner{
		Handler: func(cmd string) (ssh.Result, error, bool) {
			// List command: return stat lines for current remote map.
			if strings.Contains(cmd, "stat --printf") && strings.Contains(cmd, "mini-vrooli_") {
				var b strings.Builder
				for name, meta := range remote {
					b.WriteString(strconv.FormatInt(meta.size, 10))
					b.WriteByte('\t')
					b.WriteString(name)
					b.WriteByte('\t')
					b.WriteString(strconv.FormatInt(meta.mt, 10))
					b.WriteByte('\n')
				}
				return ssh.Result{Stdout: b.String(), ExitCode: 0}, nil, true
			}

			// Delete command: rm -f -- 'file' 'file' ...
			if strings.Contains(cmd, " rm -f -- ") {
				fields := strings.Fields(cmd)
				// Find the "--" token and delete every subsequent quoted filename.
				i := 0
				for ; i < len(fields); i++ {
					if fields[i] == "--" {
						i++
						break
					}
				}
				for ; i < len(fields); i++ {
					fn := strings.Trim(fields[i], "'")
					delete(remote, fn)
				}
				return ssh.Result{Stdout: "", ExitCode: 0}, nil, true
			}

			// Allow "echo ok" or other probes in unrelated handlers, but keep strict by default.
			return ssh.Result{}, nil, false
		},
		DefaultErr: nil,
	}

	srv := newTestServer()
	srv.sshRunner = fakeSSH
	srv.deploymentRepo = &FakeDeploymentRepo{
		Deployment: &domain.Deployment{
			ID:          deploymentID,
			Name:        "lpbs @ example.com",
			ScenarioID:   "landing-page-business-suite",
			Status:       domain.StatusDeployed,
			Manifest:     manifestJSON,
			BundleSHA256: ptr(shaNew), // protect newest, but it's already within keep_latest=2
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	// 1) List endpoint should return all remote bundles (no filtering at this layer).
	resp, err := http.Get(ts.URL + "/api/v1/deployments/" + deploymentID + "/bundles/vps")
	if err != nil {
		t.Fatalf("get list: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d", resp.StatusCode)
	}

	var listOut domain.VPSBundleListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listOut); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if !listOut.OK {
		t.Fatalf("expected ok list, got error=%q", listOut.Error)
	}
	if len(listOut.Bundles) != 4 {
		t.Fatalf("expected 4 bundles, got %d", len(listOut.Bundles))
	}
	if listOut.TotalSizeBytes != 1000 {
		t.Fatalf("expected total_size_bytes=1000, got %d", listOut.TotalSizeBytes)
	}

	// 2) GC dry-run should return a plan and not delete anything.
	gcReq := domain.VPSBundleGCRequest{KeepLatest: 2, DryRun: true}
	gcBody, _ := json.Marshal(gcReq)
	gcResp, err := http.Post(ts.URL+"/api/v1/deployments/"+deploymentID+"/bundles/vps/gc", "application/json", bytes.NewReader(gcBody))
	if err != nil {
		t.Fatalf("post gc dry-run: %v", err)
	}
	defer gcResp.Body.Close()
	if gcResp.StatusCode != http.StatusOK {
		t.Fatalf("gc dry-run status=%d", gcResp.StatusCode)
	}
	var gcOut domain.VPSBundleGCResponse
	if err := json.NewDecoder(gcResp.Body).Decode(&gcOut); err != nil {
		t.Fatalf("decode gc dry-run: %v", err)
	}
	if !gcOut.OK || !gcOut.DryRun {
		t.Fatalf("expected ok dry-run, got ok=%v dry_run=%v err=%q", gcOut.OK, gcOut.DryRun, gcOut.Error)
	}
	if gcOut.DeletedCount != 1 {
		t.Fatalf("expected 1 planned deletion (oldest), got %d", gcOut.DeletedCount)
	}
	if len(remote) != 4 {
		t.Fatalf("expected no deletion in dry-run; remote size=%d", len(remote))
	}

	// 3) GC execute should delete the oldest scenario bundle only (keep_latest=2 for that scenario).
	gcReq = domain.VPSBundleGCRequest{KeepLatest: 2, DryRun: false}
	gcBody, _ = json.Marshal(gcReq)
	gcResp2, err := http.Post(ts.URL+"/api/v1/deployments/"+deploymentID+"/bundles/vps/gc", "application/json", bytes.NewReader(gcBody))
	if err != nil {
		t.Fatalf("post gc execute: %v", err)
	}
	defer gcResp2.Body.Close()
	if gcResp2.StatusCode != http.StatusOK {
		t.Fatalf("gc execute status=%d", gcResp2.StatusCode)
	}
	var gcOut2 domain.VPSBundleGCResponse
	if err := json.NewDecoder(gcResp2.Body).Decode(&gcOut2); err != nil {
		t.Fatalf("decode gc execute: %v", err)
	}
	if !gcOut2.OK || gcOut2.DryRun {
		t.Fatalf("expected ok execute, got ok=%v dry_run=%v err=%q", gcOut2.OK, gcOut2.DryRun, gcOut2.Error)
	}
	if gcOut2.DeletedCount != 1 {
		t.Fatalf("expected 1 deletion, got %d", gcOut2.DeletedCount)
	}
	if _, ok := remote["mini-vrooli_landing-page-business-suite_"+shaOld+".tar.gz"]; ok {
		t.Fatalf("expected oldest bundle deleted")
	}
	if len(remote) != 3 {
		t.Fatalf("expected remote size=3 after deletion, got %d", len(remote))
	}
}

func TestDeploymentVPSBundles_GCProtectsRecordedBundleSHA(t *testing.T) {
	t.Setenv("API_PORT", "0")

	shaOld := strings.Repeat("a", 64)
	shaMid := strings.Repeat("b", 64)
	shaNew := strings.Repeat("c", 64)

	manifest := domain.CloudManifest{
		Version: "1.0.0",
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS: &domain.ManifestVPS{
				Host:    "203.0.113.10",
				KeyPath: "/tmp/fake-key",
				Workdir: "/root/Vrooli",
			},
		},
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Dependencies: domain.ManifestDependencies{
			Scenarios: []string{"landing-page-business-suite"},
		},
		Bundle: domain.ManifestBundle{IncludePackages: true, IncludeAutoheal: true},
		Ports:  domain.ManifestPorts{},
		Edge:   domain.ManifestEdge{Domain: "example.com", Caddy: domain.ManifestCaddy{Enabled: true}},
	}
	manifestJSON, _ := json.Marshal(manifest)

	remote := map[string]struct {
		size int64
		mt   int64
	}{
		"mini-vrooli_landing-page-business-suite_" + shaOld + ".tar.gz": {size: 100, mt: 1000},
		"mini-vrooli_landing-page-business-suite_" + shaMid + ".tar.gz": {size: 200, mt: 2000},
		"mini-vrooli_landing-page-business-suite_" + shaNew + ".tar.gz": {size: 300, mt: 3000},
	}

	fakeSSH := &FakeSSHRunner{
		Handler: func(cmd string) (ssh.Result, error, bool) {
			if strings.Contains(cmd, "stat --printf") {
				var b strings.Builder
				for name, meta := range remote {
					b.WriteString(strconv.FormatInt(meta.size, 10))
					b.WriteByte('\t')
					b.WriteString(name)
					b.WriteByte('\t')
					b.WriteString(strconv.FormatInt(meta.mt, 10))
					b.WriteByte('\n')
				}
				return ssh.Result{Stdout: b.String(), ExitCode: 0}, nil, true
			}
			if strings.Contains(cmd, " rm -f -- ") {
				fields := strings.Fields(cmd)
				for i := 0; i < len(fields); i++ {
					if fields[i] == "--" {
						for j := i + 1; j < len(fields); j++ {
							fn := strings.Trim(fields[j], "'")
							delete(remote, fn)
						}
						break
					}
				}
				return ssh.Result{Stdout: "", ExitCode: 0}, nil, true
			}
			return ssh.Result{}, nil, false
		},
	}

	srv := newTestServer()
	srv.sshRunner = fakeSSH
	// Protect the oldest bundle via recorded deployment bundle SHA. With keep_latest=2,
	// this would normally be deleted unless protected.
	srv.deploymentRepo = &FakeDeploymentRepo{
		Deployment: &domain.Deployment{
			ID:          "dep-1",
			Name:        "lpbs",
			ScenarioID:   "landing-page-business-suite",
			Status:       domain.StatusDeployed,
			Manifest:     manifestJSON,
			BundleSHA256: ptr(shaOld),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	req := domain.VPSBundleGCRequest{KeepLatest: 2, DryRun: false}
	body, _ := json.Marshal(req)
	resp, err := http.Post(ts.URL+"/api/v1/deployments/dep-1/bundles/vps/gc", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	var out domain.VPSBundleGCResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !out.OK {
		t.Fatalf("expected ok, got %q", out.Error)
	}
	if out.DeletedCount != 0 {
		t.Fatalf("expected 0 deletions (oldest protected), got %d", out.DeletedCount)
	}
	if len(remote) != 3 {
		t.Fatalf("expected all bundles kept, remote=%d", len(remote))
	}
}

func ptr[T any](v T) *T { return &v }

