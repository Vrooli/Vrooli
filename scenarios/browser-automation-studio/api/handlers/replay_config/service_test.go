package replay_config

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/vrooli/browser-automation-studio/database"
	replayconfigv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/replay_config"
	replayconfigconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/replay_config/replay_configconnect"
)

type fakeStore struct {
	value     string
	getErr    error
	setErr    error
	deleteErr error

	lastSetKey, lastSetValue string
	lastDeleteKey            string
}

func (f *fakeStore) GetSetting(_ context.Context, _ string) (string, error) {
	return f.value, f.getErr
}

func (f *fakeStore) SetSetting(_ context.Context, key, value string) error {
	f.lastSetKey = key
	f.lastSetValue = value
	if f.setErr != nil {
		return f.setErr
	}
	f.value = value
	return nil
}

func (f *fakeStore) DeleteSetting(_ context.Context, key string) error {
	f.lastDeleteKey = key
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.value = ""
	return nil
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { w.t.Log(string(p)); return len(p), nil }

func newTestClient(t *testing.T, store SettingsStore) replayconfigconnect.ReplayConfigServiceClient {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(testWriter{t})
	mount := Module(Deps{Store: store, Logger: logger})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return replayconfigconnect.NewReplayConfigServiceClient(srv.Client(), srv.URL)
}

func TestModulePanicsWithoutLogger(t *testing.T) {
	require.Panics(t, func() { Module(Deps{Store: &fakeStore{}}) })
}

func TestModulePanicsWithoutStore(t *testing.T) {
	require.Panics(t, func() { Module(Deps{Logger: logrus.New()}) })
}

func TestGetReturnsEmptyWhenUnset(t *testing.T) {
	client := newTestClient(t, &fakeStore{getErr: database.ErrNotFound})
	res, err := client.Get(context.Background(), connect.NewRequest(&replayconfigv1.GetReplayConfigRequest{}))
	require.NoError(t, err)
	require.NotNil(t, res.Msg.GetConfig())
	require.Empty(t, res.Msg.GetConfig().GetFields())
}

func TestGetReturnsStoredConfig(t *testing.T) {
	store := &fakeStore{value: `{"chromeTheme":"midnight","cursorScale":1.5}`}
	client := newTestClient(t, store)
	res, err := client.Get(context.Background(), connect.NewRequest(&replayconfigv1.GetReplayConfigRequest{}))
	require.NoError(t, err)
	fields := res.Msg.GetConfig().GetFields()
	require.Equal(t, "midnight", fields["chromeTheme"].GetStringValue())
	require.InDelta(t, 1.5, fields["cursorScale"].GetNumberValue(), 0.0001)
}

func TestPutPersistsConfig(t *testing.T) {
	store := &fakeStore{}
	client := newTestClient(t, store)
	cfg, err := structpb.NewStruct(map[string]any{"chromeTheme": "aurora"})
	require.NoError(t, err)
	res, err := client.Put(context.Background(), connect.NewRequest(&replayconfigv1.PutReplayConfigRequest{Config: cfg}))
	require.NoError(t, err)
	require.Equal(t, "aurora", res.Msg.GetConfig().GetFields()["chromeTheme"].GetStringValue())
	require.Equal(t, settingsKey, store.lastSetKey)
	require.Contains(t, store.lastSetValue, "aurora")
}

func TestPutRejectsMissingConfig(t *testing.T) {
	client := newTestClient(t, &fakeStore{})
	_, err := client.Put(context.Background(), connect.NewRequest(&replayconfigv1.PutReplayConfigRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestPutSurfacesPersistFailure(t *testing.T) {
	store := &fakeStore{setErr: errors.New("boom")}
	client := newTestClient(t, store)
	cfg, _ := structpb.NewStruct(map[string]any{"a": "b"})
	_, err := client.Put(context.Background(), connect.NewRequest(&replayconfigv1.PutReplayConfigRequest{Config: cfg}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestResetDeletes(t *testing.T) {
	store := &fakeStore{value: `{"a":1}`}
	client := newTestClient(t, store)
	res, err := client.Reset(context.Background(), connect.NewRequest(&replayconfigv1.ResetReplayConfigRequest{}))
	require.NoError(t, err)
	require.Empty(t, res.Msg.GetConfig().GetFields())
	require.Equal(t, settingsKey, store.lastDeleteKey)
}

func TestResetSurfacesError(t *testing.T) {
	store := &fakeStore{deleteErr: errors.New("nope")}
	client := newTestClient(t, store)
	_, err := client.Reset(context.Background(), connect.NewRequest(&replayconfigv1.ResetReplayConfigRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}
