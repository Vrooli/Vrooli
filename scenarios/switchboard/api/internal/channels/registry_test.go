package channels

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type lifecycleAdapter struct {
	started chan struct{}
	stopped chan struct{}
}

func (a *lifecycleAdapter) ID() string                        { return "x" }
func (a *lifecycleAdapter) Probe(context.Context) ProbeResult { return ProbeResult{Available: true} }
func (a *lifecycleAdapter) Connect(ctx context.Context, _ func(Envelope) error) error {
	close(a.started)
	<-ctx.Done()
	close(a.stopped)
	return ctx.Err()
}
func (a *lifecycleAdapter) Send(context.Context, Outbound) error { return nil }

func writeDescriptor(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "x.json"), []byte(body), 0o600))
	return dir
}

// [REQ:SWBD-P0-001] [REQ:SWBD-P0-002] [REQ:SWBD-P1-012] [REQ:SWBD-P1-013]
func TestLoadAndAvailabilityFailClosed(t *testing.T) {
	dir := writeDescriptor(t, `{"kind":"channel","schemaVersion":1,"id":"x","displayName":"X","transport":"fixture","supports":{},"limits":{"maxTextBytes":1,"maxMediaBytes":2},"setup":{"friction":3},"cost":"free","requires":[{"key":"mac","description":"connect a Mac"}]}`)
	r, err := Load(dir)
	require.NoError(t, err)
	got := r.List(nil, HostFacts{})
	require.Len(t, got, 1)
	require.Equal(t, Unknown, got[0].Availability)
	got = r.List(nil, HostFacts{"mac": false})
	require.Equal(t, Unavailable, got[0].Availability)
	got = r.List(nil, HostFacts{"mac": true})
	require.Equal(t, Available, got[0].Availability)
}

func TestLoadNamesFileAndFieldOnInvalidDescriptor(t *testing.T) {
	dir := writeDescriptor(t, `{"kind":"channel"}`)
	_, err := Load(dir)
	require.ErrorContains(t, err, "x.json")
	require.ErrorContains(t, err, "schemaVersion")
}

func TestStartRunsAvailableExternalAdapterAndStopsIt(t *testing.T) {
	adapter := &lifecycleAdapter{started: make(chan struct{}), stopped: make(chan struct{})}
	r, err := Load(writeDescriptor(t, `{"kind":"channel","schemaVersion":1,"id":"x","displayName":"X","transport":"fixture","supports":{},"limits":{"maxTextBytes":1,"maxMediaBytes":2},"setup":{"friction":3},"cost":"free"}`), adapter)
	require.NoError(t, err)
	stop := r.Start(context.Background(), HostFacts{}, nil, nil)
	t.Cleanup(stop)
	select {
	case <-adapter.started:
	case <-time.After(time.Second):
		t.Fatal("adapter did not start")
	}
	stop()
	select {
	case <-adapter.stopped:
	case <-time.After(time.Second):
		t.Fatal("adapter did not stop")
	}
}
