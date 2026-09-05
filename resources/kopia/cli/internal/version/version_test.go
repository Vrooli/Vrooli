package version

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    Semver
		wantErr bool
	}{
		{name: "kopia --version output", in: "kopia version 0.23.0 build: abc123 from: source", want: Semver{0, 23, 0}},
		{name: "bare semver", in: "0.18.2", want: Semver{0, 18, 2}},
		{name: "with v prefix", in: "v1.2.3", want: Semver{1, 2, 3}},
		{name: "no version", in: "no numbers here", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) expected error", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("Parse(%q) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}

func TestAtLeast(t *testing.T) {
	pinned := PinnedSemver()
	cases := []struct {
		in   Semver
		want bool
	}{
		{Semver{0, 23, 0}, true},
		{Semver{0, 24, 0}, true},
		{Semver{1, 0, 0}, true},
		{Semver{0, 23, 1}, true},
		{Semver{0, 22, 9}, false},
		{Semver{0, 18, 0}, false},
	}
	for _, tc := range cases {
		if got := tc.in.AtLeast(pinned); got != tc.want {
			t.Errorf("%s.AtLeast(%s) = %v, want %v", tc.in, pinned, got, tc.want)
		}
	}
}

func TestCheckFailsBelowPin(t *testing.T) {
	if _, ok, err := Check("kopia version 0.18.0"); err != nil || ok {
		t.Fatalf("Check(0.18.0) = ok %v err %v, want ok=false", ok, err)
	}
	if installed, ok, err := Check("kopia version 0.23.0 build: x"); err != nil || !ok {
		t.Fatalf("Check(0.23.0) = installed %s ok %v err %v, want ok=true", installed, ok, err)
	}
}

func TestPinnedConstant(t *testing.T) {
	if Pinned == "" {
		t.Fatal("Pinned must be set")
	}
	if Tag != "v"+Pinned {
		t.Fatalf("Tag = %q, want %q", Tag, "v"+Pinned)
	}
}
