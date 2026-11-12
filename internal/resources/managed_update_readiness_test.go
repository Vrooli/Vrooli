package resources

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"testing"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

func TestGenericManagedUpdateUsesHTTPReadiness(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	port := listener.Addr().(*net.TCPAddr).Port
	contract := ManagedUpdateContract{Manifest: ResourceManifest{
		Name:         "exclusive-http-fixture",
		HealthChecks: []ResourceHealthCheck{{Kind: manifestpkg.HealthCheckReadiness, Type: "http", Target: "http://127.0.0.1:" + strconv.Itoa(port) + "/ready"}},
	}}
	client, err := observeManagedUpdateReadiness(context.Background(), contract, port, nil, nil)
	if err != nil {
		t.Fatalf("generic exclusive service readiness failed: %v", err)
	}
	client.CloseIdleConnections()
}

func TestManagedUpdateReadinessRequiresOpenRGBSDKCheckKind(t *testing.T) {
	contract := ManagedUpdateContract{Manifest: ResourceManifest{
		Name:         openRGBResourceName,
		HealthChecks: []ResourceHealthCheck{{Kind: manifestpkg.HealthCheckReadiness, Type: "http", Target: "http://127.0.0.1:6742/ready"}},
	}}
	if _, err := observeManagedUpdateReadiness(context.Background(), contract, 6742, nil, nil); err == nil {
		t.Fatal("OpenRGB readiness accepted an ordinary HTTP check in place of SDK v6")
	}
}
