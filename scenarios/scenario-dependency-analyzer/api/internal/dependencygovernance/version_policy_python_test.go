package dependencygovernance

import "testing"

func TestPythonStableNumericReleaseRanges(t *testing.T) {
	for _, tc := range []struct {
		observed, approved string
		allowed            bool
	}{
		{"5.10.0", ">=4.41", true}, {"2.2.6", ">=1.26,<3", true},
		{"0.7.0", ">=0.4", true}, {"2.7.1+cu126", ">=2.7.1,<=2.8.0", true},
		{"2.7.1+cu126", "==2.7.1", true}, {"2.7.1+cu126", ">2.7.1", false},
		{"2.7.1+cu126", "<2.7.1", false}, {"2.8.0", ">=2.7.1,<2.8.0", false},
		{"1.0.0", "==1", true}, {"5.8.0", ">=5.10.0", false},
		{"2.8.3.post2+cu12torch2.7cxx11abiTRUE", "2.8.3.post2+cu12torch2.7cxx11abiTRUE", true},
	} {
		t.Run(tc.observed+" in "+tc.approved, func(t *testing.T) {
			decision := evaluateVersionPolicy("pip", tc.observed, tc.approved, "")
			if decision.Allowed != tc.allowed {
				t.Fatalf("decision=%+v", decision)
			}
		})
	}
}

func TestPythonUnsupportedRangeSyntaxRemainsUnqualified(t *testing.T) {
	for _, tc := range []struct{ observed, approved string }{
		{"2.8.3.post2", ">=2.8"}, {"1.0rc1", ">=1"}, {"1.0.dev2", ">=1"},
		{"1.0", ">=1.0.post1"}, {"1.0", "==1.*"}, {"1.0", "~=1.0"},
		{"1.0", ">=1 || <2"}, {"1.0", ">=1,"}, {"2.7.1+cu126", ">=2.7.1+cu126"},
	} {
		t.Run(tc.observed+" in "+tc.approved, func(t *testing.T) {
			d := evaluateVersionPolicy("pip", tc.observed, tc.approved, "")
			if d.Allowed || d.FindingClass != "VERSION_RANGE_UNPARSEABLE" {
				t.Fatalf("unsupported range not explicit: %+v", d)
			}
		})
	}
}
