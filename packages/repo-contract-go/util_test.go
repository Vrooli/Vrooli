package repocontract

import "testing"

func TestUtilityHelpers(t *testing.T) {
	if !isAbsolutePathLike("/tmp/repo") {
		t.Fatal("isAbsolutePathLike(/tmp/repo) = false, want true")
	}
	if !isAbsolutePathLike(`C:\repo`) {
		t.Fatal(`isAbsolutePathLike(C:\repo) = false, want true`)
	}
	if isAbsolutePathLike("scenarios/demo") {
		t.Fatal("isAbsolutePathLike(scenarios/demo) = true, want false")
	}

	if got := filepathToSlashTrimmed(` scenarios\demo\api `); got != "scenarios/demo/api" {
		t.Fatalf("filepathToSlashTrimmed() = %q", got)
	}

	if got := cleanSlashPath("/scenarios/demo/../demo/api"); got != "scenarios/demo/api" {
		t.Fatalf("cleanSlashPath() = %q", got)
	}

	values := []string{"b", "a", "c"}
	sortStrings(values)
	if values[0] != "a" || values[1] != "b" || values[2] != "c" {
		t.Fatalf("sortStrings() = %v", values)
	}
}
