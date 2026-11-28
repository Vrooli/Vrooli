package sessionprofile

import (
	"context"
	"io/fs"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	platform "github.com/vrooli/platform-go"
)

// interleavedProfileFiles holds one stale read until a competing writer either
// commits or reaches the native serialization boundary. No sleep decides order.
type interleavedProfileFiles struct {
	persistence.OSFileSystem
	profilePath string
	armed       atomic.Bool
	reading     chan struct{}
	release     chan struct{}
	competing   chan struct{}
	once        sync.Once
}

func (f *interleavedProfileFiles) ReadFile(path string) ([]byte, error) {
	data, err := f.OSFileSystem.ReadFile(path)
	if path == f.profilePath && f.armed.CompareAndSwap(true, false) {
		close(f.reading)
		<-f.release
	}
	return data, err
}

func (f *interleavedProfileFiles) observeCompetitor() {
	select {
	case <-f.reading:
		f.once.Do(func() { close(f.competing) })
	default:
	}
}

func (f *interleavedProfileFiles) WriteFileAtomic(path string, data []byte, mode fs.FileMode) error {
	err := f.OSFileSystem.WriteFileAtomic(path, data, mode)
	if path == f.profilePath {
		f.observeCompetitor()
	}
	return err
}

func (f *interleavedProfileFiles) Remove(path string) error {
	err := f.OSFileSystem.Remove(path)
	if path == f.profilePath {
		f.observeCompetitor()
	}
	return err
}

func (f *interleavedProfileFiles) Lock(path string) (func(), error) {
	f.observeCompetitor()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return platform.AcquireFileLockContext(ctx, path)
}

// [REQ:BAS-RH-J06] Successful independent changes must survive an overlapping save.
func TestProfileConcurrentMutationsPreserveAcknowledgedChanges(t *testing.T) {
	for _, competing := range []string{"storage", "delete"} {
		t.Run(competing, func(t *testing.T) {
			root := t.TempDir()
			files := &interleavedProfileFiles{profilePath: filepath.Join(root, "identity.json"), reading: make(chan struct{}), release: make(chan struct{}), competing: make(chan struct{})}
			var release sync.Once
			unblock := func() { release.Do(func() { close(files.release) }) }
			t.Cleanup(unblock)
			config := persistence.FileRepositoryConfig{FileSystem: files, Authority: testutil.ProfileCredentialAuthority}
			firstRepo := persistence.NewFileRepositoryWithConfig(root, nil, config)
			secondRepo := persistence.NewFileRepositoryWithConfig(root, nil, config)
			require.NoError(t, firstRepo.Create(&persistence.SessionProfile{ID: "identity", Name: "Original", StorageState: []byte(`{"cookies":[{"value":"old"}]}`)}))
			first, second := NewService(firstRepo, nil), NewService(secondRepo, nil)
			files.armed.Store(true)
			firstDone, secondDone := make(chan error, 1), make(chan error, 1)
			go func() {
				_, err := first.SaveOpenTabs("identity", []persistence.TabState{{URL: "https://fixture.invalid/new-tab"}})
				firstDone <- err
			}()
			select {
			case <-files.reading:
			case <-time.After(5 * time.Second):
				t.Fatal("first mutation did not read its snapshot")
			}
			newState := []byte(`{"cookies":[{"value":"new"}]}`)
			go func() {
				if competing == "delete" {
					secondDone <- second.DeleteProfile("identity")
					return
				}
				_, err := second.SaveStorageState("identity", newState)
				secondDone <- err
			}()
			select {
			case <-files.competing:
			case <-time.After(5 * time.Second):
				t.Fatal("competing mutation did not reach its boundary")
			}
			unblock()
			require.NoError(t, <-firstDone)
			require.NoError(t, <-secondDone)
			got, err := persistence.NewFileRepositoryWithConfig(root, nil, config).Get("identity")
			require.NoError(t, err)
			if competing == "delete" {
				require.Nil(t, got, "an acknowledged delete must not be resurrected by a stale save")
				return
			}
			require.NotNil(t, got)
			require.Equal(t, newState, []byte(got.StorageState), "a successful storage update must survive a concurrent tab update")
			require.Equal(t, []persistence.TabState{{URL: "https://fixture.invalid/new-tab"}}, got.OpenTabs)
		})
	}
}
