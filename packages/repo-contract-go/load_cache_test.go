package repocontract

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestLoadAlwaysObservesCurrentBytesEvenWithUnchangedMetadata(t *testing.T) {
	root := fixtureRoot(t)
	path := filepath.Join(root, ".vrooli", "repo-contract.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	first, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(original, []byte(`"version": "1.2.0"`), []byte(`"version": "1.2.1"`), 1)
	if bytes.Equal(original, changed) {
		t.Fatal("fixture version not found")
	}
	if err := os.WriteFile(path, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	second, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if second.Version() != "1.2.1" || first.Version() != "1.2.0" {
		t.Fatalf("versions=%s,%s", first.Version(), second.Version())
	}
	if err := os.WriteFile(path, []byte("invalid JSON"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("invalid replacement returned a previous valid contract")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("deleted contract returned a previous valid contract")
	}
}

func TestConcurrentLoadsAndReturnedValuesRemainIndependent(t *testing.T) {
	root := fixtureRoot(t)
	path := filepath.Join(root, ".vrooli", "repo-contract.json")
	initial, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	spec := initial.RuntimeHomeSpec()
	delete(spec.Entries, HomeKeyState)
	markers := initial.RootMarkers()
	markers.RequiredDirs[0] = "mutated"
	var group sync.WaitGroup
	for range 20 {
		group.Add(1)
		go func() {
			defer group.Done()
			c, err := Load(path)
			if err != nil {
				t.Error(err)
				return
			}
			if _, ok := c.RuntimeHomeSpec().Entries[HomeKeyState]; !ok {
				t.Error("caller mutation leaked into another load")
			}
			if c.RootMarkers().RequiredDirs[0] == "mutated" {
				t.Error("caller mutation leaked into root markers")
			}
		}()
	}
	group.Wait()
}

func BenchmarkLoadCanonicalContract(b *testing.B) {
	path := filepath.Join("..", "..", ".vrooli", "repo-contract.json")
	if _, err := Load(path); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Load(path); err != nil {
			b.Fatal(err)
		}
	}
}

func TestParsedContractCacheBoundsRetentionAndDoesNotCacheErrors(t *testing.T) {
	cache := newContractCache(2, 10)
	loads := 0
	load := func(key string) {
		t.Helper()
		_, err := cache.load([]byte(key), func() (*Contract, error) { loads++; return &Contract{}, nil })
		if err != nil {
			t.Fatal(err)
		}
	}
	load("a")
	load("b")
	load("a")
	load("c")
	load("b")
	if loads != 4 {
		t.Fatalf("bounded LRU decode calls=%d, want 4", loads)
	}
	load("oversized-document")
	load("oversized-document")
	if loads != 6 {
		t.Fatal("oversized documents were retained")
	}
	failures := 0
	for range 2 {
		if _, err := cache.load([]byte("bad"), func() (*Contract, error) { failures++; return nil, errors.New("invalid") }); err == nil {
			t.Fatal("invalid contract accepted")
		}
	}
	if failures != 2 {
		t.Fatal("decode failures were cached")
	}
}
