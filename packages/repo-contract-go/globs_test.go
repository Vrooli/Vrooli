package repocontract

import "testing"

func TestValidateRepoGlob(t *testing.T) {
	if err := ValidateRepoGlob("scenarios/test-genie/**"); err != nil {
		t.Fatalf("ValidateRepoGlob() error = %v", err)
	}
	assertErrorKind(t, ValidateRepoGlob(" "), ErrInvalidInput)
	assertErrorKind(t, ValidateRepoGlob("/tmp/**"), ErrInvalidInput)
	assertErrorKind(t, ValidateRepoGlob("../**"), ErrInvalidInput)
	assertErrorKind(t, ValidateRepoGlob("["), ErrInvalidInput)
}

func TestMatchRepoGlob(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
		wantErr ErrorKind
	}{
		{name: "doublestar match", pattern: "scenarios/test-genie/**", path: "scenarios/test-genie/api/main.go", want: true},
		{name: "native separators normalize", pattern: `scenarios\test-genie\**`, path: `scenarios\test-genie\api\main.go`, want: true},
		{name: "non-match", pattern: "scenarios/test-genie/**", path: "packages/api-core/server.go", want: false},
		{name: "absolute rejected", pattern: "/tmp/**", path: "scenarios/test-genie/api/main.go", wantErr: ErrInvalidInput},
		{name: "traversal rejected", pattern: "../**", path: "scenarios/test-genie/api/main.go", wantErr: ErrInvalidInput},
		{name: "bad rel path", pattern: "scenarios/**", path: "/tmp/file.go", wantErr: ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MatchRepoGlob(tt.pattern, tt.path)
			if tt.wantErr != "" {
				assertErrorKind(t, err, tt.wantErr)
				return
			}
			if err != nil {
				t.Fatalf("MatchRepoGlob() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("MatchRepoGlob() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContractMatchRepoGlobDelegatesToSharedBehavior(t *testing.T) {
	contract := validContract(t)
	got, err := contract.MatchRepoGlob("scenarios/demo/**", "scenarios/demo/api/main.go")
	if err != nil || !got {
		t.Fatalf("Contract.MatchRepoGlob() = %v, %v", got, err)
	}
}

func TestAffectedScenarios(t *testing.T) {
	contract := validContract(t)
	got := contract.AffectedScenarios([]string{
		"scenarios/test-genie/**",
		"scenarios/test-genie/api/*.go",
		"scenarios/swarm-manager/ui/**",
		"packages/api-core/**",
		"scenarios/*/docs/**",
		"./scenarios/scenario-to-cloud/**",
		`scenarios\git-control-tower\api\**`,
		"",
		"../bad",
	})

	want := []string{"git-control-tower", "scenario-to-cloud", "swarm-manager", "test-genie"}
	if len(got) != len(want) {
		t.Fatalf("AffectedScenarios() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("AffectedScenarios()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNormalizeRepoRelative(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		forbidEmpty bool
		want        string
		wantErr     ErrorKind
	}{
		{name: "trim dot prefix", value: "./scenarios/demo", forbidEmpty: true, want: "scenarios/demo"},
		{name: "windows style", value: `scenarios\demo\api`, forbidEmpty: true, want: "scenarios/demo/api"},
		{name: "empty allowed", value: " . ", forbidEmpty: false, want: ""},
		{name: "empty rejected", value: " . ", forbidEmpty: true, wantErr: ErrInvalidInput},
		{name: "absolute rejected", value: "/tmp/demo", forbidEmpty: true, wantErr: ErrInvalidInput},
		{name: "traversal rejected", value: "../demo", forbidEmpty: true, wantErr: ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeRepoRelative(tt.value, tt.forbidEmpty)
			if tt.wantErr != "" {
				assertErrorKind(t, err, tt.wantErr)
				return
			}
			if err != nil {
				t.Fatalf("normalizeRepoRelative() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeRepoRelative() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContainsGlobMeta(t *testing.T) {
	if !containsGlobMeta("scenarios/*/api") {
		t.Fatal("containsGlobMeta() = false, want true")
	}
	if containsGlobMeta("scenarios/demo/api") {
		t.Fatal("containsGlobMeta() = true, want false")
	}
}
