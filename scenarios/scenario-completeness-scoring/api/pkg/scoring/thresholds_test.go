package scoring

import (
	"testing"
)

// [REQ:SCS-CFG-002] Test category thresholds
func TestGetThresholds(t *testing.T) {
	// Known category
	utility := GetThresholds("utility")
	if utility.Requirements.Good != 15 {
		t.Errorf("Expected utility requirements.good = 15, got %d", utility.Requirements.Good)
	}

	// Unknown category should return default
	unknown := GetThresholds("unknown-category")
	if unknown.Requirements.Good != 15 {
		t.Errorf("Expected default requirements.good = 15, got %d", unknown.Requirements.Good)
	}

	// Developer tools
	devtools := GetThresholds("developer_tools")
	if devtools.Requirements.Good != 30 {
		t.Errorf("Expected developer_tools requirements.good = 30, got %d", devtools.Requirements.Good)
	}
}

// [REQ:SCS-CFG-002] Test all category threshold configs exist
func TestCategoryThresholdsExist(t *testing.T) {
	expectedCategories := []string{
		"utility",
		"business-application",
		"automation",
		"platform",
		"developer_tools",
	}

	for _, cat := range expectedCategories {
		thresholds := GetThresholds(cat)
		if thresholds.Requirements.OK == 0 && thresholds.Requirements.Good == 0 {
			t.Errorf("Category %s should have non-zero thresholds", cat)
		}
	}
}

// [REQ:SCS-CFG-002] Test default thresholds
func TestDefaultThresholds(t *testing.T) {
	defaults := DefaultThresholds()

	// Verify requirements thresholds
	if defaults.Requirements.OK != 10 {
		t.Errorf("Expected requirements.ok = 10, got %d", defaults.Requirements.OK)
	}
	if defaults.Requirements.Good != 15 {
		t.Errorf("Expected requirements.good = 15, got %d", defaults.Requirements.Good)
	}
	if defaults.Requirements.Excellent != 25 {
		t.Errorf("Expected requirements.excellent = 25, got %d", defaults.Requirements.Excellent)
	}

	// Verify tests thresholds
	if defaults.Tests.OK != 15 {
		t.Errorf("Expected tests.ok = 15, got %d", defaults.Tests.OK)
	}
	if defaults.Tests.Good != 25 {
		t.Errorf("Expected tests.good = 25, got %d", defaults.Tests.Good)
	}

	// Verify UI thresholds
	if defaults.UI.FileCount.Good != 25 {
		t.Errorf("Expected ui.file_count.good = 25, got %d", defaults.UI.FileCount.Good)
	}
	if defaults.UI.TotalLOC.OK != 300 {
		t.Errorf("Expected ui.total_loc.ok = 300, got %d", defaults.UI.TotalLOC.OK)
	}
}

// [REQ:SCS-CFG-002] Test category threshold hierarchy
func TestCategoryThresholdHierarchy(t *testing.T) {
	// Platform should have higher thresholds than utility
	platform := GetThresholds("platform")
	utility := GetThresholds("utility")

	if platform.Requirements.Excellent <= utility.Requirements.Excellent {
		t.Errorf("Platform should have higher requirements than utility")
	}

	if platform.Tests.Excellent <= utility.Tests.Excellent {
		t.Errorf("Platform should require more tests than utility")
	}

	// Business-application should have higher thresholds than automation
	business := GetThresholds("business-application")
	automation := GetThresholds("automation")

	if business.UI.TotalLOC.Good <= automation.UI.TotalLOC.Good {
		t.Errorf("Business application should require more UI LOC than automation")
	}
}
