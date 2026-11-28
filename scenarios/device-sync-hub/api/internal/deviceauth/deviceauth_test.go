package deviceauth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"device-sync-hub/internal/deviceauth"
	"device-sync-hub/internal/devices"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAuthn resolves one known token to one TRUSTED device; everything else is
// untrusted.
type fakeAuthn struct {
	token  string
	device devices.Device
}

func (f fakeAuthn) Authenticate(_ context.Context, raw string) (devices.Device, error) {
	if raw == f.token {
		return f.device, nil
	}
	return devices.Device{}, devices.ErrUntrustedDevice
}

func TestMiddlewareInjectsTrustedDevice(t *testing.T) {
	dev := devices.Device{ID: "dev-1", OwnerID: "owner-1", TrustState: devices.TrustTrusted}
	mw := deviceauth.Middleware(fakeAuthn{token: "good", device: dev}, nil)

	var gotID string
	var present bool
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		d, ok := deviceauth.RequireDevice(r.Context())
		present = ok == nil
		gotID = d.ID
	})

	// Header path.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/realtime/events", nil)
	req.Header.Set(deviceauth.HeaderName, "good")
	mw(next).ServeHTTP(httptest.NewRecorder(), req)
	assert.True(t, present)
	assert.Equal(t, "dev-1", gotID)

	// Query-param fallback (EventSource cannot set headers).
	present, gotID = false, ""
	req = httptest.NewRequest(http.MethodGet, "/api/v1/realtime/events?token=good", nil)
	mw(next).ServeHTTP(httptest.NewRecorder(), req)
	assert.True(t, present)
	assert.Equal(t, "dev-1", gotID)
}

func TestMiddlewareLeavesContextCleanForBadToken(t *testing.T) {
	mw := deviceauth.Middleware(fakeAuthn{token: "good"}, nil)
	var err error
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, err = deviceauth.RequireDevice(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(deviceauth.HeaderName, "wrong")
	mw(next).ServeHTTP(httptest.NewRecorder(), req)
	assert.ErrorIs(t, err, devices.ErrUntrustedDevice)

	// No token at all → RequireDevice still fails closed.
	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	mw(next).ServeHTTP(httptest.NewRecorder(), req)
	assert.ErrorIs(t, err, devices.ErrUntrustedDevice)
}

func TestRequireDeviceDirect(t *testing.T) {
	dev := devices.Device{ID: "d"}
	ctx := deviceauth.WithDevice(context.Background(), dev)
	got, err := deviceauth.RequireDevice(ctx)
	require.NoError(t, err)
	assert.Equal(t, "d", got.ID)
}
