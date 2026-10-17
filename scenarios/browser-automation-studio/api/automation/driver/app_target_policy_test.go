package driver

import "testing"

func TestResolveTargetURLPolicyAdmitsAndroidWebViewRoutes(t *testing.T) {
	policy, err := ResolveTargetURLPolicy(TargetKindAndroidWebView)
	if err != nil {
		t.Fatal(err)
	}
	got, err := policy.Resolve("http://127.0.0.1:4173", "/hello-mobile")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://127.0.0.1:4173/hello-mobile" {
		t.Fatalf("resolved URL = %q", got)
	}
}

func TestResolveTargetURLPolicyAdmitsAndroidPackageLocalAsset(t *testing.T) {
	policy, err := ResolveTargetURLPolicy(TargetKindAndroidWebView)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := policy.Resolve("file:///android_asset/index.html", "/"); err != nil || got != "file:///android_asset/index.html" {
		t.Fatalf("package-local asset resolution = %q, error = %v", got, err)
	}
}

func TestResolveTargetURLPolicyDefaultsToElectron(t *testing.T) {
	policy, err := ResolveTargetURLPolicy("")
	if err != nil {
		t.Fatal(err)
	}
	if policy.Kind != TargetKindElectron {
		t.Fatalf("kind = %q", policy.Kind)
	}
	got, err := policy.Resolve("file:///tmp/app.html", "/ignored")
	if err != nil {
		t.Fatal(err)
	}
	if got != "file:///tmp/app.html" {
		t.Fatalf("resolved file URL = %q", got)
	}
}
