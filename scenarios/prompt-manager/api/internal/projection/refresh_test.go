package projection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeFixture(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) (*Service, string, string) {
	t.Helper()
	source, target := t.TempDir(), filepath.Join(t.TempDir(), "native")
	writeFixture(t, filepath.Join(source, "alpha", "SKILL.md"), "---\nname: alpha\ndescription: test skill\n---\nv1\n")
	s := &Service{SourceRoot: source, Targets: []Target{{Runtime: "fixture", Path: target}}, LoadPack: func() (BasePack, error) {
		return BasePack{Skills: []string{"alpha"}, MaxSkills: 1, MaxTokens: 100}, nil
	}}
	return s, filepath.Join(source, "alpha", "SKILL.md"), filepath.Join(target, "alpha", "SKILL.md")
}

func previewApply(t *testing.T, s *Service, req RefreshRequest) RefreshResult {
	t.Helper()
	preview, err := s.Refresh(req)
	if err != nil {
		t.Fatal(err)
	}
	req.Apply, req.ExpectedDigest = true, preview.Digest
	result, err := s.Refresh(req)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestRefreshPreviewDoesNotWriteAndApplyEstablishesFreshness(t *testing.T) {
	s, _, installed := fixture(t)
	preview, err := s.Refresh(RefreshRequest{})
	if err != nil || preview.Rows[0].Status != "missing" {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	if _, err := os.Stat(s.Targets[0].Path); !os.IsNotExist(err) {
		t.Fatal("preview wrote target")
	}
	result := previewApply(t, s, RefreshRequest{})
	if !result.Rows[0].Applied {
		t.Fatalf("not applied: %+v", result)
	}
	next, _ := s.Refresh(RefreshRequest{})
	if next.Rows[0].Status != "current" || next.Rows[0].BaselineHash != next.Rows[0].InstalledHash {
		t.Fatalf("freshness: %+v", next)
	}
	if _, err := os.Stat(installed); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshRejectsStaleSourceAndDestinationPreview(t *testing.T) {
	for _, which := range []string{"source", "destination"} {
		t.Run(which, func(t *testing.T) {
			s, source, dest := fixture(t)
			previewApply(t, s, RefreshRequest{})
			preview, _ := s.Refresh(RefreshRequest{})
			path := dest
			if which == "source" {
				path = source
			}
			body, _ := os.ReadFile(path)
			writeFixture(t, path, string(body)+"concurrent edit\n")
			if _, err := s.Refresh(RefreshRequest{Apply: true, ExpectedDigest: preview.Digest}); err == nil {
				t.Fatal("stale preview accepted")
			}
			if got, _ := os.ReadFile(path); string(got) != string(body)+"concurrent edit\n" {
				t.Fatal("concurrent edit lost")
			}
		})
	}
}

func TestRefreshProtectsLocalEditsAndRecoversInterruptedApply(t *testing.T) {
	s, source, dest := fixture(t)
	previewApply(t, s, RefreshRequest{})
	old, _ := os.ReadFile(dest)
	writeFixture(t, dest, string(old)+"personal edit\n")
	result := previewApply(t, s, RefreshRequest{AdoptLegacy: true})
	if result.Rows[0].Status != "modified" || result.Rows[0].Applied {
		t.Fatalf("local edit protection: %+v", result)
	}
	// Simulate a crash after the receipt is persisted, before file replacement.
	writeFixture(t, dest, string(old))
	body, _ := os.ReadFile(source)
	writeFixture(t, source, string(body)+"v2\n")
	newBody := ensureMarker(string(body) + "v2\n")
	data, _ := json.Marshal(receipt{Before: hash(old), After: hash([]byte(newBody))})
	writeFixture(t, filepath.Join(s.Targets[0].Path, ".prompt-manager", "alpha.json"), string(data))
	preview, _ := s.Refresh(RefreshRequest{})
	if preview.Rows[0].Status != "recoverable" {
		t.Fatalf("recovery not detected: %+v", preview)
	}
	result = previewApply(t, s, RefreshRequest{})
	if !result.Rows[0].Applied {
		t.Fatalf("recovery failed: %+v", result)
	}
	backup, err := os.ReadFile(result.Rows[0].BackupPath)
	if err != nil || string(backup) != string(old) {
		t.Fatalf("backup: %s %v", backup, err)
	}
	if got, _ := os.ReadFile(dest); string(got) != newBody {
		t.Fatal("wrong recovered content")
	}
}

func TestLegacyAdoptionRequiresExactReviewedPolicyAndKeepsBackup(t *testing.T) {
	s, _, dest := fixture(t)
	legacy := "---\nname: alpha\ndescription: previous\n---\n" + generatedMarker + "\nold\n"
	writeFixture(t, dest, legacy)
	result := previewApply(t, s, RefreshRequest{})
	if result.Rows[0].Status != "legacy" || result.Rows[0].Applied {
		t.Fatal("legacy overwritten automatically")
	}
	preview, _ := s.Refresh(RefreshRequest{})
	if _, err := s.Refresh(RefreshRequest{Apply: true, AdoptLegacy: true, ExpectedDigest: preview.Digest}); err == nil {
		t.Fatal("unreviewed policy escalation")
	}
	result = previewApply(t, s, RefreshRequest{AdoptLegacy: true})
	backup, err := os.ReadFile(result.Rows[0].BackupPath)
	if !result.Rows[0].Applied || err != nil || string(backup) != legacy {
		t.Fatalf("adoption: %+v %v", result, err)
	}
}

func TestRefreshRejectsSymlinksAndContinuesOtherTargets(t *testing.T) {
	s, source, dest := fixture(t)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, dest); err != nil {
		t.Fatal(err)
	}
	s.Targets = append(s.Targets, Target{Runtime: "other", Path: filepath.Join(t.TempDir(), "native")})
	result := previewApply(t, s, RefreshRequest{})
	if result.Rows[0].Applied || result.Rows[0].Status != "conflict" || !result.Rows[1].Applied {
		t.Fatalf("isolation: %+v", result)
	}
	body, _ := os.ReadFile(source)
	if string(body) != "---\nname: alpha\ndescription: test skill\n---\nv1\n" {
		t.Fatal("source modified through symlink")
	}
}

func TestRefreshRejectsUnselectedIDsAndSourceOverlap(t *testing.T) {
	s, _, _ := fixture(t)
	for _, id := range []string{"../alpha", "not-in-pack"} {
		if _, err := s.Refresh(RefreshRequest{Skills: []string{id}}); err == nil {
			t.Fatalf("accepted %q", id)
		}
	}
	s.Targets[0].Path = s.SourceRoot
	result := previewApply(t, s, RefreshRequest{})
	if result.Rows[0].Applied || result.Rows[0].Status != "conflict" {
		t.Fatal("source overlap accepted")
	}
}

func TestRefreshUpdatesOnlySelectedSkillAndPreservesSupportingFiles(t *testing.T) {
	s, source, dest := fixture(t)
	previewApply(t, s, RefreshRequest{})
	writeFixture(t, filepath.Join(s.Targets[0].Path, "alpha", "notes.md"), "local notes")
	writeFixture(t, filepath.Join(s.Targets[0].Path, "retired", "SKILL.md"), generatedMarker+"\nretain me")
	body, _ := os.ReadFile(source)
	writeFixture(t, source, string(body)+"updated\n")
	preview, err := s.Refresh(RefreshRequest{Runtime: "fixture", Skills: []string{"alpha"}})
	if err != nil || preview.Rows[0].Status != "outdated" {
		t.Fatalf("outdated: %+v %v", preview, err)
	}
	result := previewApply(t, s, RefreshRequest{Runtime: "fixture", Skills: []string{"alpha"}})
	if !result.Rows[0].Applied {
		t.Fatal("update not applied")
	}
	if got, _ := os.ReadFile(dest); string(got) != ensureMarker(string(body)+"updated\n") {
		t.Fatal("wrong new content")
	}
	for _, item := range []struct{ path, body string }{{"alpha/notes.md", "local notes"}, {"retired/SKILL.md", generatedMarker + "\nretain me"}} {
		if got, _ := os.ReadFile(filepath.Join(s.Targets[0].Path, item.path)); string(got) != item.body {
			t.Fatalf("unselected content lost: %s", item.path)
		}
	}
	// A fresh service instance must derive the same state from disk.
	fresh := *s
	state, err := fresh.Refresh(RefreshRequest{})
	if err != nil || state.Rows[0].Status != "current" {
		t.Fatalf("fresh service: %+v %v", state, err)
	}
}

func TestRefreshRejectsSymlinkedTargetAndReceiptDirectories(t *testing.T) {
	for _, component := range []string{"root", "receipt"} {
		t.Run(component, func(t *testing.T) {
			s, _, _ := fixture(t)
			link := s.Targets[0].Path
			if component == "receipt" {
				if err := os.MkdirAll(link, 0o755); err != nil {
					t.Fatal(err)
				}
				link = filepath.Join(link, ".prompt-manager")
			}
			outside := t.TempDir()
			if err := os.Symlink(outside, link); err != nil {
				t.Fatal(err)
			}
			result := previewApply(t, s, RefreshRequest{})
			if result.Rows[0].Applied || result.Rows[0].Status != "conflict" {
				t.Fatalf("unsafe target accepted: %+v", result)
			}
			entries, err := os.ReadDir(outside)
			if err != nil || len(entries) != 0 {
				t.Fatal("wrote through symlink")
			}
		})
	}
}
