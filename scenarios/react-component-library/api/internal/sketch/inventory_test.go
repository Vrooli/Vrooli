package sketch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInventoryIncludesMissingInvalidAndUnregisteredPages(t *testing.T) {
	root := t.TempDir()
	experience := filepath.Join(root, "scenarios", "demo", "experience")
	if err := os.MkdirAll(filepath.Join(experience, "pages"), 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"index.json":        `{"pages":[{"id":"good","path":"pages/good.json","title":"Good"},{"id":"missing","path":"pages/missing.json"},{"id":"bad","path":"pages/bad.json"}]}`,
		"pages/good.json":   `{"page":{"title":"A good page","routes":["/good","/good/:id"]},"regions":[{"id":"body"}],"sketch":{"regions":[{"id":"body"},{"id":"footer"}]}}`,
		"pages/bad.json":    `{"sketch":null}`,
		"pages/orphan.json": `{"page":{"title":"Orphan"}}`,
	} {
		if err := os.WriteFile(filepath.Join(experience, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := NewStore(root).ListDesignPages(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Pages) != 4 {
		t.Fatalf("page lost: %+v", got)
	}
	pages := map[string]DesignPage{}
	for _, page := range got.Pages {
		pages[page.Page] = page
	}
	if pages["missing"].Status != "missing" || pages["bad"].Status != "invalid" || pages["orphan"].Registered {
		t.Fatalf("gaps concealed: %+v", got)
	}
	if good := pages["good"]; good.Title != "A good page" || good.RegionCount != 2 || good.Route != "/good" || len(good.Routes)!=2 || good.ContentHash == "" {
		t.Fatalf("wrong page projection: %+v", good)
	}
}

func TestInventoryRejectsTraversalAndSymlinkPage(t *testing.T) {
	root := t.TempDir()
	pages := filepath.Join(root, "scenarios", "demo", "experience", "pages")
	if err := os.MkdirAll(pages, 0755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "secret.json")
	if err := os.WriteFile(outside, []byte(`{"page":{"title":"secret"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(pages, "linked.json")); err != nil {
		t.Fatal(err)
	}
	store := NewStore(root)
	if _, err := store.ListDesignPages(context.Background(), "../demo"); err == nil {
		t.Fatal("traversal accepted")
	}
	got, err := store.ListDesignPages(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Pages) != 1 || got.Pages[0].Issue == "" || got.Pages[0].ContentHash != "" || got.Pages[0].Title == "secret" {
		t.Fatalf("symlink followed: %+v", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.ListDesignPages(ctx, ""); err == nil {
		t.Fatal("cancelled discovery succeeded")
	}
}
