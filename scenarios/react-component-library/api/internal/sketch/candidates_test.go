package sketch

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCandidatesPreservePageAndRemainReadableAfterEdits(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"claims":[{"id":"keep"}],"regions":[{"id":"main"}],"sketch":{"x-extension":true}}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	store := NewStore(root)
	base, err := store.Read("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	doc := Document{Template: &AssetRef{Asset: "templates.collection-page", Version: "1.0.7"}}
	saved, err := store.SaveCandidate("demo", "home", "design-one", base.ContentHash, doc)
	if err != nil {
		t.Fatal(err)
	}
	again, err := store.SaveCandidate("demo", "home", "design-one", base.ContentHash, doc)
	if err != nil || again.Hash != saved.Hash {
		t.Fatal("candidate retry changed identity", err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(after, original) {
		t.Fatal("candidate creation changed current page")
	}
	if !bytes.Contains(saved.PageBytes, []byte(`"x-extension"`)) || !bytes.Contains(saved.PageBytes, []byte(`"claims"`)) {
		t.Fatal("candidate lost authored extensions")
	}
	doc.Viewport = "phone"
	if _, err := store.Save("demo", "home", base.ContentHash, doc); err != nil {
		t.Fatal(err)
	}
	child, err := store.DeriveCandidate("demo", "design-one", saved.Hash, Document{Viewport: "tablet"})
	if err != nil || child.ParentHash != saved.Hash || child.BaseHash != base.ContentHash {
		t.Fatal("immutable branch lineage missing", err)
	}
	retry, err := store.DeriveCandidate("demo", "design-one", saved.Hash, Document{Viewport: "tablet"})
	if err != nil || retry.Hash != child.Hash {
		t.Fatal("branch retry not idempotent", err)
	}
	listed, err := store.ListCandidates("demo", "home")
	if err != nil || len(listed) != 2 {
		t.Fatalf("saved revisions not discoverable: %d %v", len(listed), err)
	}
	other, err := store.ListCandidates("demo", "other")
	if err != nil || len(other) != 0 {
		t.Fatal("page filter leaked candidates", err)
	}
	current, err := store.Read("demo", "home")
	if err != nil || current.Document.Viewport != "phone" {
		t.Fatal("candidate branch changed current page", err)
	}
	read, err := store.ReadCandidate("demo", "design-one", saved.Hash)
	if err != nil || read.BaseHash != base.ContentHash {
		t.Fatal("immutable candidate tied to mutable current page", err)
	}
	snapshot, err := read.Snapshot()
	if err != nil || snapshot.ContentHash != saved.Hash || len(snapshot.DeclaredRegions) != 1 {
		t.Fatal("candidate render identity incorrect", err)
	}
	if _, err := store.SaveCandidate("demo", "home", "design-one", base.ContentHash, doc); err == nil {
		t.Fatal("stale candidate publication accepted")
	}
	relative, _ := candidatePath("design-one", saved.Hash)
	candidatePath := filepath.Join(root, "scenarios", "demo", "experience", relative)
	var altered Candidate
	if err := json.Unmarshal(mustCandidateBytes(t, candidatePath), &altered); err != nil {
		t.Fatal(err)
	}
	altered.PageBytes = bytes.ReplaceAll(altered.PageBytes, []byte("templates.collection-page"), []byte("templates.other"))
	damaged, err := json.Marshal(altered)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidatePath, damaged, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ListCandidates("demo", "home"); err == nil {
		t.Fatal("inventory hid a corrupt candidate")
	}
	if _, err := store.ReadCandidate("demo", "design-one", saved.Hash); err == nil {
		t.Fatal("tampered immutable candidate accepted")
	}
}

func TestNestedRegionLocksProtectDescendantsAndContainingLayout(t *testing.T) {
	base := Document{Regions: []Region{{ID: "header", Locked: true}, {ID: "start"}}, Render: &RenderSettings{Regions: []RenderRegion{{ID: "header", Slot: []string{"header"}}, {ID: "start", Parent: "header", Slot: []string{"actions"}}}, Bindings: map[string]any{"header": map[string]any{"title": "Conversations"}, "start": map[string]any{"children": "Start conversation"}}}}
	clone := func() Document { var d Document; raw, _ := json.Marshal(base); _ = json.Unmarshal(raw, &d); return d }
	changed := clone()
	changed.Render.Bindings["start"] = map[string]any{"children": "Changed"}
	if validateRegionLocks(base, changed) == nil {
		t.Fatal("nested control bypassed locked parent")
	}
	if refinementContent(base, "header") == refinementContent(changed, "header") {
		t.Fatal("nested change was omitted from parent refinement scope")
	}
	changed = clone()
	changed.Render.Regions[1].Parent = ""
	if validateRegionLocks(base, changed) == nil {
		t.Fatal("moving a child out bypassed locked parent")
	}
	base.Regions[0].Locked = false
	base.Regions[1].Locked = true
	changed = clone()
	changed.Render.Bindings["header"] = map[string]any{"title": "Changed layout"}
	if validateRegionLocks(base, changed) == nil {
		t.Fatal("containing layout change bypassed locked child")
	}
}
func mustCandidateBytes(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestCandidateIdentityRejectsTraversal(t *testing.T) {
	for _, id := range []string{"../outside", "a/b", ""} {
		if _, err := candidatePath(id, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"); err == nil {
			t.Fatal("unsafe design identity accepted")
		}
	}
}

func TestRegionLocksProtectPageWritesAndCandidateBranches(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	doc := renderSnapshot().Document
	doc.Regions = []Region{{ID: "inspector", Locked: true}, {ID: "other"}}
	raw, _ := json.Marshal(map[string]any{"sketch": doc, "claims": []any{map[string]any{"id": "preserve"}}})
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	store := NewStore(root)
	base, err := store.Read("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	parent, err := store.SaveCandidate("demo", "home", "locked", base.ContentHash, doc)
	if err != nil {
		t.Fatal(err)
	}
	clone := func() Document {
		var result Document
		encoded, _ := json.Marshal(doc)
		_ = json.Unmarshal(encoded, &result)
		return result
	}
	cases := []struct {
		name   string
		change func(*Document)
	}{
		{"asset", func(d *Document) { d.Placements[0].Fills.Version = "9.0.0" }},
		{"remove region", func(d *Document) { d.Regions = d.Regions[1:] }},
		{"bindings", func(d *Document) { d.Render.Bindings["inspector"] = map[string]any{"children": "Changed"} }},
		{"template", func(d *Document) { d.Template.Version = "9.0.0" }},
		{"slot", func(d *Document) { d.Render.Regions[0].Slot = []string{"changed"} }},
		{"shared interaction", func(d *Document) { d.Render.Bindings["$preview"] = map[string]any{"initial": "changed"} }},
		{"unlock and edit", func(d *Document) { d.Regions[0].Locked = false; d.Regions[0].Note = "Changed" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			next := clone()
			c.change(&next)
			if _, err := store.Save("demo", "home", base.ContentHash, next); err == nil {
				t.Fatal("locked page changed")
			}
			if _, err := store.DeriveCandidate("demo", "locked", parent.Hash, next); err == nil {
				t.Fatal("locked candidate changed")
			}
		})
	}
	other := clone()
	other.Regions[1].Note = "Independent refinement"
	child, err := store.DeriveCandidate("demo", "locked", parent.Hash, other)
	if err != nil || child.ParentHash != parent.Hash {
		t.Fatal("unlocked region could not change", err)
	}
	unlocked := clone()
	unlocked.Regions[0].Locked = false
	next, err := store.Save("demo", "home", base.ContentHash, unlocked)
	if err != nil {
		t.Fatal(err)
	}
	if next.Document.Regions[0].Locked {
		t.Fatal("unlock flag was retained by extension merge")
	}
	unlocked.Placements[0].Fills.Version = "9.0.0"
	if _, err := store.Save("demo", "home", next.ContentHash, unlocked); err != nil {
		t.Fatal("separately unlocked region stayed frozen", err)
	}
}
