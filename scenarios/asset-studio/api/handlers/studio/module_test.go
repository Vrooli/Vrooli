package studio

import (
	"testing"

	core "asset-studio/internal/studio"
)

func TestImageToolsDispatcherFromEnvironmentValidatesOrigin(t *testing.T) {
	lookup := func(string) (string, bool) { return "https://image-tools.example.test", true }
	if _, ok := imageToolsDispatcherFromEnvironment(lookup).(*core.ImageToolsDispatcher); !ok {
		t.Fatal("valid Image Tools origin did not configure dispatcher")
	}
	bad := func(string) (string, bool) { return "https://user:secret@image-tools.example.test/?x=1", true }
	if _, ok := imageToolsDispatcherFromEnvironment(bad).(core.UnavailableRenderDispatcher); !ok {
		t.Fatal("unsafe Image Tools URL must be rejected")
	}
}

func TestGatewayVideoDispatcherFromEnvironmentValidatesOrigin(t *testing.T) {
	lookup := func(string) (string, bool) { return "http://ai-gateway.example.test", true }
	if _, ok := gatewayVideoDispatcherFromEnvironment(lookup).(*core.GatewayVideoDispatcher); !ok {
		t.Fatal("valid Gateway origin did not configure video dispatcher")
	}
}

func TestBrowserCaptureDispatcherFromEnvironmentValidatesOrigin(t *testing.T) {
	lookup := func(string) (string, bool) { return "http://browser-automation-studio.example.test", true }
	if _, ok := browserCaptureDispatcherFromEnvironment(lookup).(*core.BrowserCaptureDispatcher); !ok {
		t.Fatal("valid Browser Automation Studio origin did not configure capture dispatcher")
	}
}
