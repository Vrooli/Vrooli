package markedrefs

import "testing"

func TestKnownMarkersAreDefensiveCopy(t *testing.T) {
	got := KnownMarkers()
	if len(got) == 0 {
		t.Fatal("expected markers")
	}
	got[0].Name = "mutated"
	if IsKnownMarker("mutated") {
		t.Fatal("KnownMarkers returned mutable registry")
	}
	if !IsKnownMarker(MarkerTopic) {
		t.Fatalf("expected %q marker to be known", MarkerTopic)
	}
}

func TestKnownQualifiersAreDefensiveCopy(t *testing.T) {
	got := KnownQualifiers()
	if len(got) == 0 {
		t.Fatal("expected qualifiers")
	}
	got[0].Name = "mutated"
	if IsKnownQualifier("mutated") {
		t.Fatal("KnownQualifiers returned mutable registry")
	}
	if !IsKnownQualifier(QualifierFuture) {
		t.Fatalf("expected %q qualifier to be known", QualifierFuture)
	}
}

func TestUnknownHelpers(t *testing.T) {
	ref := Reference{Marker: "unknown", Qualifiers: []string{"future", "mystery"}}
	if !UnknownMarker(ref) {
		t.Fatal("expected unknown marker")
	}
	got := UnknownQualifiers(ref)
	if len(got) != 1 || got[0] != "mystery" {
		t.Fatalf("UnknownQualifiers() = %#v, want [mystery]", got)
	}
}

func TestRequiresExistence(t *testing.T) {
	tests := []struct {
		name string
		ref  Reference
		want bool
	}{
		{name: "plain path", ref: Reference{Marker: MarkerPath}, want: true},
		{name: "example path", ref: Reference{Marker: MarkerPath, Qualifiers: []string{QualifierExample}}, want: false},
		{name: "old topic", ref: Reference{Marker: MarkerTopic, Qualifiers: []string{QualifierOld}}, want: false},
		{name: "literal marker", ref: Reference{Marker: MarkerLiteral}, want: false},
		{name: "literal qualifier", ref: Reference{Marker: MarkerPath, Qualifiers: []string{QualifierLiteral}}, want: false},
		{name: "unknown marker", ref: Reference{Marker: "unknown"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RequiresExistence(tt.ref); got != tt.want {
				t.Fatalf("RequiresExistence() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsLiteral(t *testing.T) {
	if !IsLiteral(Reference{Marker: MarkerLiteral}) {
		t.Fatal("literal marker should be literal")
	}
	if !IsLiteral(Reference{Marker: MarkerPath, Qualifiers: []string{QualifierLiteral}}) {
		t.Fatal("literal qualifier should be literal")
	}
	if IsLiteral(Reference{Marker: MarkerPath}) {
		t.Fatal("plain path should not be literal")
	}
}
