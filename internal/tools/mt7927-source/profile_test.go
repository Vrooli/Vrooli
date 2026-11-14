package mt7927source

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestStagePublishesOnlyCompleteVerifiedInputClosure(t *testing.T) {
	profile, _ := Lookup(Name)
	body := "source fixture"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, body)
	}))
	defer server.Close()
	items := []input{
		{Name: "backport", URL: server.URL, Bytes: int64(len(body)), SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(body)))},
		{Name: "kernel", URL: server.URL, Bytes: int64(len(body)), SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("unexpected")))},
	}
	ctx := context.Background()
	parent := t.TempDir()
	if path, err := profile.stage(ctx, parent, items); err == nil || path != "" {
		t.Fatalf("mismatched stage = %q, %v", path, err)
	}
	if entries, err := os.ReadDir(parent); err != nil || len(entries) != 0 {
		t.Fatalf("failed stage left files: %v, %v", entries, err)
	}
	items[1].SHA256 = items[0].SHA256
	path, err := profile.stage(ctx, parent, items)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(parent, Version) {
		t.Fatalf("stage path = %s", path)
	}
	server.Close()
	if reused, err := profile.stage(ctx, parent, items); err != nil || reused != path {
		t.Fatalf("offline reuse = %q, %v", reused, err)
	}
	if err := os.WriteFile(filepath.Join(path, "kernel"), []byte("edited fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if reused, err := profile.stage(ctx, parent, items); err == nil || reused != "" {
		t.Fatalf("tampered closure selected: %q, %v", reused, err)
	}
}

func TestNobleProfilePinsOnlyExactDistroSourceAndPCIeSeries(t *testing.T) {
	profile, ok := Lookup(NobleName)
	if !ok || profile.Version() != NobleVersion {
		t.Fatal("exact Noble profile unavailable")
	}
	items, err := profile.inputs()
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]int64{
		"linux-hwe-7.0_7.0.0-30.30~24.04.1.dsc":     8259,
		"linux-hwe-7.0_7.0.0.orig.tar.gz":           254937830,
		"linux-hwe-7.0_7.0.0-30.30~24.04.1.diff.gz": 1988323,
		"mt7927-pcie-series.patch":                  74504,
	}
	if len(items) != len(expected) {
		t.Fatalf("unexpected number of Noble inputs: %v", items)
	}
	for _, item := range items {
		if expected[item.Name] != item.Bytes {
			t.Fatalf("unexpected Noble input: %+v", item)
		}
		if item.Name == "linux-hwe-7.0_7.0.0.orig.tar.gz" &&
			(item.URL != "https://cdn.kernel.org/pub/linux/kernel/v7.x/linux-7.0.tar.gz" ||
				item.SHA256 != "c6343795c8c7e4feff55f5b309365c8e0eaf49cd6abcb412e32315313f149536") {
			t.Fatalf("orig mirror no longer uses the authenticated Ubuntu content pin: %+v", item)
		}
	}
	if profile.ManifestPath(".") == filepath.Join("internal", "tools", Name, Manifest) {
		t.Fatal("Noble profile overlaps the community backport descriptor")
	}
}

func TestASUSBluetoothProfilePinsOnlyTheOfficialArchive(t *testing.T) {
	profile, ok := Lookup(ASUSBluetoothName)
	if !ok || profile.Version() != ASUSBluetoothVersion {
		t.Fatal("ASUS local archive profile unavailable")
	}
	items, err := profile.inputs()
	if err != nil || len(items) != 1 {
		t.Fatalf("ASUS source closure %+v %v", items, err)
	}
	item := items[0]
	if item.Name != ASUSBluetoothArchive || item.Bytes != 7142068 || item.SHA256 != ASUSBluetoothVersion ||
		item.URL != "https://dlcdnets.asus.com/pub/ASUS/mb/02BT/"+ASUSBluetoothArchive {
		t.Fatalf("ASUS archive identity changed %+v", item)
	}
}
