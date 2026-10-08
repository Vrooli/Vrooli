package scenarioruntime

import "testing"

// TestInstanceKeyNormalize pins the backward-compatibility invariant: an empty
// variant collapses to "live", so every legacy call site that passes only a
// scenario name keeps addressing the live instance.
func TestInstanceKeyNormalize(t *testing.T) {
	cases := []struct {
		name        string
		in          InstanceKey
		wantScen    string
		wantVariant string
	}{
		{"empty variant becomes live", InstanceKey{Scenario: "swarm-manager"}, "swarm-manager", "live"},
		{"explicit live preserved", InstanceKey{Scenario: "swarm-manager", Variant: "live"}, "swarm-manager", "live"},
		{"variant lower-cased", InstanceKey{Scenario: "swarm-manager", Variant: "Shadow"}, "swarm-manager", "shadow"},
		{"whitespace trimmed", InstanceKey{Scenario: "  swarm-manager  ", Variant: "  shadow  "}, "swarm-manager", "shadow"},
		{"non-default variant kept", InstanceKey{Scenario: "test-genie", Variant: "shadow"}, "test-genie", "shadow"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.Normalize()
			if got.Scenario != tc.wantScen || got.Variant != tc.wantVariant {
				t.Fatalf("Normalize() = {%q,%q}, want {%q,%q}", got.Scenario, got.Variant, tc.wantScen, tc.wantVariant)
			}
			// Normalize is idempotent.
			if again := got.Normalize(); again != got {
				t.Fatalf("Normalize not idempotent: %+v vs %+v", again, got)
			}
		})
	}
}

func TestInstanceKeyIsLive(t *testing.T) {
	if !(InstanceKey{Scenario: "x"}).IsLive() {
		t.Fatal("empty variant should be live")
	}
	if !(InstanceKey{Scenario: "x", Variant: "LIVE"}).IsLive() {
		t.Fatal("explicit LIVE should be live")
	}
	if (InstanceKey{Scenario: "x", Variant: "shadow"}).IsLive() {
		t.Fatal("shadow should not be live")
	}
}

func TestInstanceKeySlug(t *testing.T) {
	cases := []struct {
		in   InstanceKey
		want string
	}{
		{InstanceKey{Scenario: "swarm-manager"}, "swarm-manager"},
		{InstanceKey{Scenario: "swarm-manager", Variant: "live"}, "swarm-manager"},
		{InstanceKey{Scenario: "swarm-manager", Variant: "shadow"}, "swarm-manager@shadow"},
		{InstanceKey{Scenario: "test-genie", Variant: "Shadow"}, "test-genie@shadow"},
	}
	for _, tc := range cases {
		if got := tc.in.Slug(); got != tc.want {
			t.Errorf("Slug(%+v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestInstanceKeyNamespace pins every derived storage/addressing name. The live
// forms MUST be byte-identical to the legacy slug-keyed forms so no existing
// reader changes behavior.
func TestInstanceKeyNamespace(t *testing.T) {
	cases := []struct {
		name string
		in   InstanceKey
		want Namespace
	}{
		{
			name: "live keeps legacy forms",
			in:   InstanceKey{Scenario: "swarm-manager"},
			want: Namespace{
				Variant:          "live",
				RecordSlug:       "swarm-manager",
				PostgresDB:       "vrooli_swarm_manager",
				DataDirName:      "swarm-manager",
				PortSeed:         "swarm-manager",
				StorageNamespace: "swarm-manager",
				EnvVars: map[string]string{
					EnvVariant:          "live",
					EnvStorageNamespace: "swarm-manager",
				},
			},
		},
		{
			name: "shadow isolates every engine",
			in:   InstanceKey{Scenario: "swarm-manager", Variant: "shadow"},
			want: Namespace{
				Variant:          "shadow",
				RecordSlug:       "swarm-manager@shadow",
				PostgresDB:       "vrooli_swarm_manager_shadow",
				DataDirName:      "swarm-manager@shadow",
				PortSeed:         "swarm-manager@shadow",
				StorageNamespace: "swarm-manager_shadow",
				EnvVars: map[string]string{
					EnvVariant:          "shadow",
					EnvStorageNamespace: "swarm-manager_shadow",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.Namespace()
			if got.Variant != tc.want.Variant ||
				got.RecordSlug != tc.want.RecordSlug ||
				got.PostgresDB != tc.want.PostgresDB ||
				got.DataDirName != tc.want.DataDirName ||
				got.PortSeed != tc.want.PortSeed ||
				got.StorageNamespace != tc.want.StorageNamespace {
				t.Fatalf("Namespace() = %+v, want %+v", got, tc.want)
			}
			for k, v := range tc.want.EnvVars {
				if got.EnvVars[k] != v {
					t.Errorf("EnvVars[%q] = %q, want %q", k, got.EnvVars[k], v)
				}
			}
		})
	}
}

// TestNamespaceLivePortSeedUnchanged guards the most dangerous regression: the
// live port seed feeds CRC32 exactly as the legacy code did
// (`scenarioName+"_"+portName`), so live ports must not shift.
func TestNamespaceLivePortSeedUnchanged(t *testing.T) {
	const scenario = "landing-page-business-suite"
	if got := (InstanceKey{Scenario: scenario}).Namespace().PortSeed; got != scenario {
		t.Fatalf("live PortSeed = %q, want bare scenario %q (else live ports move)", got, scenario)
	}
}

// TestParseInstanceKey covers the §1a resolution rules: suffix and flag are
// equivalent, disagreement is a hard error, empty resolves to live.
func TestParseInstanceKey(t *testing.T) {
	cases := []struct {
		name        string
		nameArg     string
		flagVariant string
		wantScen    string
		wantVariant string
		wantErr     bool
	}{
		{"bare name is live", "swarm-manager", "", "swarm-manager", "live", false},
		{"flag only", "swarm-manager", "shadow", "swarm-manager", "shadow", false},
		{"suffix only", "swarm-manager@shadow", "", "swarm-manager", "shadow", false},
		{"flag and suffix agree", "swarm-manager@shadow", "shadow", "swarm-manager", "shadow", false},
		{"flag and suffix agree case-insensitive", "swarm-manager@shadow", "Shadow", "swarm-manager", "shadow", false},
		{"flag and suffix disagree", "swarm-manager@shadow", "live", "", "", true},
		{"explicit live flag", "swarm-manager", "live", "swarm-manager", "live", false},
		{"empty scenario errors", "@shadow", "", "", "", true},
		{"whitespace handled", "  test-genie@shadow  ", "", "test-genie", "shadow", false},
		{"trailing at is live", "swarm-manager@", "", "swarm-manager", "live", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseInstanceKey(tc.nameArg, tc.flagVariant)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseInstanceKey(%q,%q) = %+v, want error", tc.nameArg, tc.flagVariant, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseInstanceKey(%q,%q) unexpected error: %v", tc.nameArg, tc.flagVariant, err)
			}
			if got.Scenario != tc.wantScen || got.Variant != tc.wantVariant {
				t.Fatalf("ParseInstanceKey(%q,%q) = {%q,%q}, want {%q,%q}",
					tc.nameArg, tc.flagVariant, got.Scenario, got.Variant, tc.wantScen, tc.wantVariant)
			}
		})
	}
}

// TestParseInstanceKeyRoundTrip pins Slug ⇄ ParseInstanceKey symmetry: the
// rendered Slug parses back to the same key. This is what makes the
// disambiguated `<s>@variant` status row double as a copy-pasteable recovery
// hint (§1a "round-trip symmetric").
func TestParseInstanceKeyRoundTrip(t *testing.T) {
	for _, k := range []InstanceKey{
		{Scenario: "swarm-manager"},
		{Scenario: "swarm-manager", Variant: "shadow"},
		{Scenario: "test-genie", Variant: "live"},
		{Scenario: "agent-manager", Variant: "shadow"},
	} {
		want := k.Normalize()
		got, err := ParseInstanceKey(want.Slug(), "")
		if err != nil {
			t.Fatalf("round-trip parse of %q failed: %v", want.Slug(), err)
		}
		if got != want {
			t.Fatalf("round-trip: ParseInstanceKey(%q) = %+v, want %+v", want.Slug(), got, want)
		}
	}
}

// TestParseInstanceKeyFlagSuffixEquivalence pins that the two spellings of the
// same instance are exactly equal — `--instance shadow` ≡ `@shadow`.
func TestParseInstanceKeyFlagSuffixEquivalence(t *testing.T) {
	viaFlag, err1 := ParseInstanceKey("swarm-manager", "shadow")
	viaSuffix, err2 := ParseInstanceKey("swarm-manager@shadow", "")
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v %v", err1, err2)
	}
	if viaFlag != viaSuffix {
		t.Fatalf("flag %+v != suffix %+v", viaFlag, viaSuffix)
	}
}
