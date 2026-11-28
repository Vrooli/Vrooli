package state

import "testing"

func TestMergeArtifactsReplacesByPlatformAndAddsNewPlatforms(t *testing.T) {
	existing := []BuildArtifact{
		{Platform: "linux", Status: "ready", FilePath: "/old/linux.zip"},
		{Platform: "windows", Status: "ready", FilePath: "/old/windows.zip"},
	}
	updates := []BuildArtifact{
		{Platform: "windows", Status: "failed", ErrorMessage: "signing failed"},
		{Platform: "mac", Status: "ready", FilePath: "/new/mac.zip"},
	}

	merged := mergeArtifacts(existing, updates)
	if len(merged) != 3 {
		t.Fatalf("merged artifacts = %d, want three platforms", len(merged))
	}
	byPlatform := make(map[string]BuildArtifact, len(merged))
	for _, artifact := range merged {
		byPlatform[artifact.Platform] = artifact
	}

	windows, ok := byPlatform["windows"]
	if !ok || windows.Status != "failed" || windows.ErrorMessage != "signing failed" {
		t.Errorf("windows artifact = %+v, want updated signing failure", windows)
	}
	linux, ok := byPlatform["linux"]
	if !ok || linux.Status != "ready" || linux.FilePath != "/old/linux.zip" {
		t.Errorf("linux artifact = %+v, want preserved existing artifact", linux)
	}
	mac, ok := byPlatform["mac"]
	if !ok || mac.Status != "ready" || mac.FilePath != "/new/mac.zip" {
		t.Errorf("mac artifact = %+v, want newly added artifact", mac)
	}
}
