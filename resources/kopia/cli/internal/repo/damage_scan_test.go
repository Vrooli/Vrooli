package repo

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestClassifyDamagePathUsesKopiaBlobAndTempNames(t *testing.T) {
	cases := []struct {
		path  string
		class DamageClass
		kind  DamageBlobKind
	}{
		{"p/ab/cd/abcdef.f", DamageRepositoryBlob, DamageDataPack},
		{"q/ab/cd/abcdef.f.tmp.0123456789abcdef", DamageLeftoverTemp, DamageMetadataPack},
		{"s/ab/cd/abcdef.f.tmp", DamageLeftoverTemp, DamageSessionMarker},
		{"s/ab/cd/abcdef.f.tmp.not-hex", DamageOther, DamageSessionMarker},
		{"s/ab/cd/abcdef.f", DamageRepositoryBlob, DamageSessionMarker},
		{"x/ab/cd/abcdef.f", DamageRepositoryBlob, DamageIndex},
		{"config.json", DamageOther, DamageBlobOther},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			class, kind := classifyDamagePath(tc.path)
			if class != tc.class || kind != tc.kind {
				t.Fatalf("classifyDamagePath(%q) = %s/%s, want %s/%s", tc.path, class, kind, tc.class, tc.kind)
			}
		})
	}
}

func TestTempFinalNameSupportsSuffixlessKopiaTemp(t *testing.T) {
	if got := tempFinalName("abcdef.f.tmp"); got != "abcdef.f" {
		t.Fatalf("tempFinalName suffixless = %q, want %q", got, "abcdef.f")
	}
	if got := tempFinalName("abcdef.f.tmp.0123456789abcdef"); got != "abcdef.f" {
		t.Fatalf("tempFinalName suffixed = %q, want %q", got, "abcdef.f")
	}
}

type damageInfo struct{ os.FileInfo }

type fakeDamageScanFS struct {
	dirs  map[string][]string
	infos map[string]fs.FileInfo
	errs  map[string]error
}

func (f fakeDamageScanFS) Names(dir string) ([]string, error) {
	if err := f.errs[dir]; err != nil {
		return nil, err
	}
	names, ok := f.dirs[dir]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return names, nil
}

func (f fakeDamageScanFS) Lstat(p string) (fs.FileInfo, error) {
	if err := f.errs[p]; err != nil {
		return nil, err
	}
	info, ok := f.infos[p]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return info, nil
}

func TestScanFilesystemRepositoryReturnsTypedRelativeUnreadableEvidence(t *testing.T) {
	root := filepath.Clean("/fixture/repository")
	rootInfo, err := os.Stat(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(root, "p")
	entry := filepath.Join(dataDir, "abcdef.f.tmp.0123456789abcdef")
	fake := fakeDamageScanFS{
		dirs:  map[string][]string{root: {"p"}, dataDir: {filepath.Base(entry)}},
		infos: map[string]fs.FileInfo{root: rootInfo, dataDir: rootInfo},
		errs:  map[string]error{entry: syscall.EIO},
	}
	result, err := scanFilesystemRepository(context.Background(), root, fake, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Unreadable) != 1 {
		t.Fatalf("unreadable entries = %+v, want one typed entry", result.Unreadable)
	}
	got := result.Unreadable[0]
	if got.RelativePath != filepath.ToSlash(filepath.Join("p", filepath.Base(entry))) || got.Class != DamageLeftoverTemp || got.BlobKind != DamageDataPack || got.Problem != DamageIO {
		t.Fatalf("typed damage result = %+v", got)
	}
	if got.Detail != "input/output error" || filepath.IsAbs(got.RelativePath) || contains(got.Detail, root) {
		t.Fatalf("damage evidence leaked host path or lost typed detail: %+v", got)
	}
	if !got.Orphan {
		t.Fatalf("temp without its final sibling should be typed as orphan: %+v", got)
	}
	finalName := "abcdef.f"
	finalPath := filepath.Join(dataDir, finalName)
	finalFixture := filepath.Join(t.TempDir(), finalName)
	if err := os.WriteFile(finalFixture, []byte("final"), 0o600); err != nil {
		t.Fatal(err)
	}
	finalInfo, err := os.Stat(finalFixture)
	if err != nil {
		t.Fatal(err)
	}
	fake.dirs[dataDir] = append(fake.dirs[dataDir], finalName)
	fake.infos[finalPath] = finalInfo
	paired, err := scanFilesystemRepository(context.Background(), root, fake, nil)
	if err != nil || len(paired.Unreadable) != 1 || paired.Unreadable[0].Orphan {
		t.Fatalf("temp with an existing final sibling must not be marked orphan: scan=%+v err=%v", paired, err)
	}
}

func TestScanFilesystemRepositoryProvesSuffixlessTempOrphanFromMissingFinal(t *testing.T) {
	root := filepath.Clean("/fixture/repository")
	rootInfo, err := os.Stat(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	shard := filepath.Join(root, "s")
	temp := "session.f.tmp"
	fake := fakeDamageScanFS{
		dirs:  map[string][]string{root: {"s"}, shard: {temp}},
		infos: map[string]fs.FileInfo{root: rootInfo, shard: rootInfo},
		errs:  map[string]error{filepath.Join(shard, temp): syscall.EIO},
	}
	result, err := scanFilesystemRepository(context.Background(), root, fake, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Unreadable) != 1 {
		t.Fatalf("unreadable = %+v, want one entry", result.Unreadable)
	}
	got := result.Unreadable[0]
	if got.Class != DamageLeftoverTemp || got.BlobKind != DamageSessionMarker || got.Problem != DamageIO || !got.Orphan {
		t.Fatalf("suffixless unreadable temp should be typed orphan only when final is absent: %+v", got)
	}
	fake.dirs[shard] = append(fake.dirs[shard], "session.f")
	finalFixture := filepath.Join(t.TempDir(), "session.f")
	if err := os.WriteFile(finalFixture, []byte("committed"), 0o600); err != nil {
		t.Fatal(err)
	}
	finalInfo, err := os.Stat(finalFixture)
	if err != nil {
		t.Fatal(err)
	}
	fake.infos[filepath.Join(shard, "session.f")] = finalInfo
	paired, err := scanFilesystemRepository(context.Background(), root, fake, nil)
	if err != nil || len(paired.Unreadable) != 1 || paired.Unreadable[0].Orphan {
		t.Fatalf("temp with its final blob must not be orphan: scan=%+v err=%v", paired, err)
	}
}

func TestScanFilesystemRepositoryFailsClosedWhenRootCannotBeListed(t *testing.T) {
	root := filepath.Clean("/fixture/repository")
	rootInfo, err := os.Stat(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	fake := fakeDamageScanFS{infos: map[string]fs.FileInfo{root: rootInfo}, errs: map[string]error{root: errors.New("denied")}}
	if _, err := scanFilesystemRepository(context.Background(), root, fake, nil); err == nil {
		t.Fatal("root listing failure must fail the scan rather than return a clean repository")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
