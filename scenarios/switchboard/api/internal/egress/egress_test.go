package egress

// [REQ:SWBD-P0-013]
import (
	"context"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"switchboard/internal/channels"
	"switchboard/internal/channels/adapters/fake"
	"testing"
)

func TestRoutesSameChannelAndEnforcesDescriptorLimit(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "x.json"), []byte(`{"kind":"channel","schemaVersion":1,"id":"x","displayName":"X","transport":"fixture","supports":{},"limits":{"maxTextBytes":2,"maxMediaBytes":2},"setup":{"friction":1},"cost":"free"}`), 0600))
	a := fake.New("x")
	reg, err := channels.Load(dir, a)
	require.NoError(t, err)
	r := Router{Registry: reg}
	require.NoError(t, r.Send(context.Background(), channels.Outbound{ChannelID: "x", Text: "ok"}))
	require.Equal(t, 1, a.SentCount())
	require.ErrorContains(t, r.Send(context.Background(), channels.Outbound{ChannelID: "x", Text: "too long"}), "maxTextBytes")
}
