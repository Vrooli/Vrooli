package sketch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestSketchRoundTripPreservesUnknownKeys(t *testing.T) {
	// [REQ:EPD-001]
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "switchboard", "experience", "pages", "conversations.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	before := "{\n  \"kind\": \"experience-page\",\n  \"claims\": [{\"id\":\"claim-a\"}],\n  \"bindings\": {\"elements\": {}},\n  \"states\": [{\"id\":\"ready\"}],\n  \"priorities\": [{\"statement\":\"Primary conversation surface\"}]\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewStore(root)
	doc := Document{Viewport: "desktop", Template: &AssetRef{Asset: "templates.conversation-page", Version: "1.0.0"}}
	if err := saveCurrent(store, "switchboard", "conversations", doc); err != nil {
		t.Fatal(err)
	}
	afterBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	after := string(afterBytes)
	for _, original := range []string{
		"\"claims\": [{\"id\":\"claim-a\"}]",
		"\"bindings\": {\"elements\": {}}",
		"\"states\": [{\"id\":\"ready\"}]",
		"\"priorities\": [{\"statement\":\"Primary conversation surface\"}]",
	} {
		if !strings.Contains(after, original) {
			t.Fatalf("non-sketch content changed or disappeared: %s\n%s", original, after)
		}
	}
	order := []string{"\"kind\"", "\"claims\"", "\"bindings\"", "\"states\"", "\"priorities\"", "\"sketch\""}
	last := -1
	for _, key := range order {
		index := strings.Index(after, key)
		if index <= last {
			t.Fatalf("top-level key order changed at %s: %s", key, after)
		}
		last = index
	}
	loaded, err := store.Load("switchboard", "conversations")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Template == nil || loaded.Template.Asset != doc.Template.Asset {
		t.Fatalf("round trip template = %#v", loaded.Template)
	}
}

func TestSketchWriteRefusesPathOutsideExperience(t *testing.T) {
	// [REQ:EPD-001]
	store := NewStore(t.TempDir())
	for _, target := range [][2]string{{"../switchboard", "conversations"}, {"switchboard", "../service"}, {"switchboard/../../outside", "page"}} {
		err := saveCurrent(store, target[0], target[1], Document{})
		if err == nil || !strings.Contains(err.Error(), "rejected sketch path") {
			t.Fatalf("Save(%q, %q) error = %v", target[0], target[1], err)
		}
	}
}

func TestSketchSaveReplacesOnlyExistingSketch(t *testing.T) {
	// [REQ:EPD-001]
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "switchboard", "experience", "pages", "page.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	before := "{\n  \"page\": {\"id\": \"page\"},\n  \"sketch\": {\"viewport\": \"mobile\"},\n  \"claims\": []\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := saveCurrent(NewStore(root), "switchboard", "page", Document{Viewport: "wide"}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(after), "\"sketch\"") != 1 || !strings.Contains(string(after), "\"viewport\": \"wide\"") {
		t.Fatalf("existing sketch was not replaced cleanly: %s", after)
	}
}

func saveCurrent(store *Store, scenario, page string, doc Document) error {
	snapshot, err := store.Read(scenario, page)
	if err != nil {
		return err
	}
	_, err = store.Save(scenario, page, snapshot.ContentHash, doc)
	return err
}

func revisionFixture(t *testing.T, content string) (*Store, string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return NewStore(root), path
}

func TestConcurrentStoresRejectLostUpdate(t *testing.T) {
	// [REQ:EPD-001]
	store, path := revisionFixture(t, `{"claims":[],"sketch":{"viewport":"desktop"}}`)
	before, err := store.Read("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for _, viewport := range []string{"phone", "tablet"} {
		group.Add(1)
		go func(viewport string) {
			defer group.Done()
			<-start
			_, err := NewStore(store.repoRoot).Save("demo", "home", before.ContentHash, Document{Viewport: viewport})
			results <- err
		}(viewport)
	}
	close(start)
	group.Wait()
	close(results)
	passed, conflicted := 0, 0
	for err := range results {
		var conflict *ConflictError
		if err == nil {
			passed++
		} else if errors.As(err, &conflict) {
			conflicted++
			if conflict.Current == before.ContentHash {
				t.Fatal("conflict did not return current hash")
			}
		} else {
			t.Fatal(err)
		}
	}
	if passed != 1 || conflicted != 1 {
		t.Fatalf("writes=%d conflicts=%d", passed, conflicted)
	}
	after, err := store.Read("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save("demo", "home", "", Document{}); !errors.Is(err, ErrExpectedRevision) {
		t.Fatalf("missing revision: %v", err)
	}
	unchanged, err := store.Save("demo", "home", after.ContentHash, after.Document)
	if err != nil || unchanged.Changed {
		t.Fatalf("no-op changed state: %+v %v", unchanged, err)
	}
	final, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(final) {
		t.Fatal("no-op rewrote source formatting")
	}
}

func TestSketchUpdatePreservesNestedExtensions(t *testing.T) {
	store, path := revisionFixture(t, `{"page":{"route":"/","unknown":{"x":1}},"sketch":{"x-future":{"enabled":true},"template":{"asset":"templates.page","x-layout":"dense"},"placements":[{"region":"main","fills":{"asset":"controls.button","x-fixture":"keep"},"state":"declared","x-selection":true}],"regions":[{"id":"main","grid":{"x":0,"y":0,"w":1,"h":1,"x-grid":"keep"},"x-region":true}]},"claims":[{"id":"keep"}],"bindings":{"elements":{}},"states":[]}`)
	current, err := store.Read("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	current.Document.Viewport = "phone"
	if _, err := store.Save("demo", "home", current.ContentHash, current.Document); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"x-future", "x-layout", "x-fixture", "x-selection", "x-region", "x-grid", "unknown", "claims", "bindings", "states"} {
		if !strings.Contains(string(raw), `"`+key+`"`) {
			t.Fatalf("lost %s: %s", key, raw)
		}
	}
	var page map[string]json.RawMessage
	if err := json.Unmarshal(raw, &page); err != nil {
		t.Fatal(err)
	}
	if string(page["page"]) != `{"route":"/","unknown":{"x":1}}` {
		t.Fatalf("unrelated formatting changed: %s", page["page"])
	}
}

func TestRevisionHashIgnoresFormattingAndPreservesLargeNumbers(t *testing.T) {
	a, err := decodeSnapshot([]byte(`{"count":9007199254740992,"sketch":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := decodeSnapshot([]byte("{\n \"sketch\": {}, \"count\": 9007199254740992 }"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := decodeSnapshot([]byte(`{"count":9007199254740993,"sketch":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	if a.ContentHash != b.ContentHash {
		t.Fatal("formatting changed content identity")
	}
	if a.ContentHash == c.ContentHash {
		t.Fatal("large integer change collapsed to same identity")
	}
}

func TestSymlinkPageAndAncestorAreRejected(t *testing.T) {
	for _, ancestor := range []bool{false, true} {
		t.Run(fmt.Sprint(ancestor), func(t *testing.T) {
			store, path := revisionFixture(t, `{"sketch":{}}`)
			outside := t.TempDir()
			target := filepath.Join(outside, "home.json")
			original := []byte(`{"external":true}`)
			if err := os.WriteFile(target, original, 0600); err != nil {
				t.Fatal(err)
			}
			link := path
			destination := target
			if ancestor {
				link = filepath.Dir(path)
				destination = outside
			}
			if err := os.RemoveAll(link); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(destination, link); err != nil {
				t.Skip(err)
			}
			if _, err := store.Read("demo", "home"); err == nil {
				t.Fatal("read followed symlink")
			}
			if _, err := store.Save("demo", "home", "expected", Document{}); err == nil {
				t.Fatal("write followed symlink")
			}
			actual, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if string(actual) != string(original) {
				t.Fatal("outside bytes changed")
			}
		})
	}
}

func TestInterruptedPublicationRecoversWithoutLosingHistory(t *testing.T) {
	for _, step := range []string{"revisions", "manifest", "page", "complete"} {
		t.Run(step, func(t *testing.T) {
			store, _ := revisionFixture(t, "{\n  \"claims\": [{\"id\":\"keep\"}],\n  \"sketch\": {\"viewport\":\"desktop\"}\n}\n")
			initial, err := store.Read("demo", "home")
			if err != nil {
				t.Fatal(err)
			}
			injected := errors.New("interrupted publication")
			store.afterPublish = func(at string) error {
				if at == step {
					return injected
				}
				return nil
			}
			_, err = store.Save("demo", "home", initial.ContentHash, Document{Viewport: "phone"})
			if !errors.Is(err, injected) {
				t.Fatalf("fault at %s: %v", step, err)
			}
			restarted := NewStore(store.repoRoot)
			recovered, err := restarted.Recover("demo", "home")
			if err != nil {
				t.Fatal(err)
			}
			if step == "revisions" {
				if recovered.Document.Viewport != "desktop" {
					t.Fatal("uncommitted revision applied without manifest")
				}
				recovered, err = restarted.Save("demo", "home", initial.ContentHash, Document{Viewport: "phone"})
				if err != nil {
					t.Fatal(err)
				}
			}
			if recovered.Document.Viewport != "phone" {
				t.Fatalf("recovery: %+v", recovered)
			}
			history, err := restarted.History("demo", "home")
			if err != nil {
				t.Fatal(err)
			}
			if len(history) != 2 {
				t.Fatalf("history lost or duplicated: %+v", history)
			}
			if history[0].ContentHash != recovered.ContentHash || history[1].ContentHash != initial.ContentHash {
				t.Fatal("revision identities do not match source")
			}
			raw, err := os.ReadFile(filepath.Join(store.repoRoot, "scenarios", "demo", "experience", "pages", "home.json"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(raw), `"claims": [{"id":"keep"}]`) {
				t.Fatalf("recovery changed authored formatting: %s", raw)
			}
			again, err := restarted.Recover("demo", "home")
			if err != nil || again.Changed {
				t.Fatalf("recovery retry not idempotent: %+v %v", again, err)
			}
		})
	}
}

func TestRecoveryDoesNotOverwriteInterveningManualEdit(t *testing.T) {
	store, path := revisionFixture(t, `{"sketch":{"viewport":"desktop"}}`)
	initial, err := store.Read("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	store.afterPublish = func(step string) error {
		if step == "manifest" {
			return errors.New("stop")
		}
		return nil
	}
	if _, err := store.Save("demo", "home", initial.ContentHash, Document{Viewport: "phone"}); err == nil {
		t.Fatal("fault not triggered")
	}
	manual := []byte(`{"sketch":{"viewport":"manual"},"claims":[{"id":"keep"}]}`)
	if err := os.WriteFile(path, manual, 0600); err != nil {
		t.Fatal(err)
	}
	_, err = NewStore(store.repoRoot).Recover("demo", "home")
	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("recovery did not conflict: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(manual) {
		t.Fatalf("manual edit overwritten: %s %v", after, err)
	}
}

func TestReapplyOldContentPreservesImmutableRevision(t *testing.T) {
	store, _ := revisionFixture(t, `{"sketch":{"viewport":"desktop"}}`)
	original, err := store.Read("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	phone, err := store.Save("demo", "home", original.ContentHash, Document{Viewport: "phone"})
	if err != nil {
		t.Fatal(err)
	}
	history, err := store.History("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	previousBytes := append([]byte(nil), history[1].Page...)
	restored, err := store.Save("demo", "home", phone.ContentHash, original.Document)
	if err != nil {
		t.Fatal(err)
	}
	if restored.ContentHash != original.ContentHash {
		t.Fatal("same content acquired a new identity")
	}
	history, err = store.History("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || string(history[1].Page) != string(previousBytes) {
		t.Fatal("old revision was rewritten")
	}
}

func TestAmbiguousPageJSONIsRejected(t *testing.T) {
	for _, raw := range []string{`{"sketch":{},"sketch":{"viewport":"phone"}}`, `{"sketch":{},"claims":[{"id":"first","id":"second"}]}`, `null`, `[]`, `{"sketch":null}`} {
		store, path := revisionFixture(t, raw)
		if _, err := store.Read("demo", "home"); err == nil {
			t.Fatalf("accepted ambiguous or invalid JSON: %s", raw)
		}
		if _, err := store.Save("demo", "home", "expected", Document{}); err == nil {
			t.Fatalf("wrote ambiguous JSON: %s", raw)
		}
		after, err := os.ReadFile(path)
		if err != nil || string(after) != raw {
			t.Fatalf("invalid input changed: %s %v", after, err)
		}
	}
}

func TestCrossProcessRevisionConflict(t *testing.T) {
	if root := os.Getenv("RCL_SKETCH_CAS_TEST_ROOT"); root != "" {
		_, err := NewStore(root).Save("demo", "home", os.Getenv("RCL_SKETCH_CAS_TEST_HASH"), Document{Viewport: os.Getenv("RCL_SKETCH_CAS_TEST_VIEWPORT")})
		if err == nil {
			return
		}
		var conflict *ConflictError
		if errors.As(err, &conflict) {
			os.Exit(3)
		}
		t.Fatal(err)
	}
	store, _ := revisionFixture(t, `{"sketch":{"viewport":"desktop"}}`)
	original, err := store.Read("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	commands := make([]*exec.Cmd, 0, 2)
	for _, viewport := range []string{"phone", "tablet"} {
		command := exec.Command(executable, "-test.run=^TestCrossProcessRevisionConflict$")
		command.Env = append(os.Environ(), "RCL_SKETCH_CAS_TEST_ROOT="+store.repoRoot, "RCL_SKETCH_CAS_TEST_HASH="+original.ContentHash, "RCL_SKETCH_CAS_TEST_VIEWPORT="+viewport)
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, command)
	}
	passed, conflicts := 0, 0
	for _, command := range commands {
		err := command.Wait()
		if err == nil {
			passed++
			continue
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 3 {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if passed != 1 || conflicts != 1 {
		t.Fatalf("cross-process writers: passed=%d conflicts=%d", passed, conflicts)
	}
	history, err := store.History("demo", "home")
	if err != nil || len(history) != 2 {
		t.Fatalf("cross-process history: %d %v", len(history), err)
	}
}
