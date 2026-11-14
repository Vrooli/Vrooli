package engine

import (
	"context"
	"net/http"
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/driver"
)

func TestAndroidWebViewEngineRequiresTargetAndLease(t *testing.T) {
	pw, err := NewPlaywrightEngineWithHTTPClient("http://127.0.0.1:1", http.DefaultClient, nil)
	if err != nil {
		t.Fatal(err)
	}
	webview, err := NewAndroidWebViewEngine(pw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := webview.StartSession(context.Background(), SessionSpec{}); err == nil {
		t.Fatal("expected missing AppTarget error")
	}
	if _, err := webview.StartSession(context.Background(), SessionSpec{AppTarget: &driver.AppTarget{TargetKind: driver.TargetKindAndroidWebView}}); err == nil {
		t.Fatal("expected missing lease error")
	}
}

func TestDefaultFactoryRegistersAndroidWebViewEngine(t *testing.T) {
	factory, err := DefaultFactory(nil)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := factory.Resolve(context.Background(), "android-webview")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Name() != "android-webview" {
		t.Fatalf("engine name = %q", resolved.Name())
	}
}
