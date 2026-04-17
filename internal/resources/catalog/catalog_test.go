package catalog

import (
	"path/filepath"
	"testing"

	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

func TestDiscoverReportContinuesAfterInvalidResourceManifest(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testscenario.WriteProjectResourceConfig(t, fixture.Root, "redis", true)
	testscenario.WriteProjectResourceConfig(t, fixture.Root, "broken", true)
	testresource.WriteExternalCLIResourceFixture(t, fixture.Root, "redis", "#!/usr/bin/env bash\nexit 0\n")
	testresource.WriteMalformedResourceManifest(t, fixture.Root, "broken", `{"name":"broken","driver":`)

	report, err := New(fixture.Root).DiscoverReport(DiscoverOptions{})
	if err != nil {
		t.Fatalf("DiscoverReport: %v", err)
	}
	if len(report.Items) != 1 || report.Items[0].Name != "redis" {
		t.Fatalf("items = %#v", report.Items)
	}
	if len(report.Failures) != 1 {
		t.Fatalf("failures = %#v", report.Failures)
	}
	if report.Failures[0].Name != "broken" {
		t.Fatalf("failure = %#v", report.Failures[0])
	}
	if report.Failures[0].Path != filepath.Join(fixture.Root, "resources", "broken", "resource.json") {
		t.Fatalf("path = %q", report.Failures[0].Path)
	}
}

func TestDiscoverOneDoesNotDependOnGlobalResourceDiscovery(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testscenario.WriteProjectResourceConfig(t, fixture.Root, "redis", true)
	testscenario.WriteProjectResourceConfig(t, fixture.Root, "broken", true)
	testresource.WriteExternalCLIResourceFixture(t, fixture.Root, "redis", "#!/usr/bin/env bash\nexit 0\n")
	testresource.WriteMalformedResourceManifest(t, fixture.Root, "broken", `{"name":"broken","driver":`)

	item, err := New(fixture.Root).DiscoverOne("redis", DiscoverOptions{})
	if err != nil {
		t.Fatalf("DiscoverOne(redis): %v", err)
	}
	if item == nil || item.Name != "redis" {
		t.Fatalf("item = %#v", item)
	}
}
