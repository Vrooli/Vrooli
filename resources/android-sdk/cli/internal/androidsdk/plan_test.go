package androidsdk

import "testing"

func TestInstallPlanUsesPlayTargetDefaults(t *testing.T) {
	t.Setenv("ANDROID_API_LEVEL", "")
	t.Setenv("ANDROID_PLATFORM_TOOLS_URL", "")
	plan, err := installPlan()
	if err != nil {
		t.Fatalf("installPlan() error = %v", err)
	}
	if plan.PlatformLevel != defaultAPI {
		t.Fatalf("PlatformLevel = %q, want %q", plan.PlatformLevel, defaultAPI)
	}
	if plan.PlatformPackage != "platforms;android-"+defaultAPI {
		t.Fatalf("PlatformPackage = %q", plan.PlatformPackage)
	}
}
