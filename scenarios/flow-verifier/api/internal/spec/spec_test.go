package spec

import "testing"

func TestSchemaConstants(t *testing.T) {
	if SchemaVersion <= 0 {
		t.Fatalf("SchemaVersion must be positive, got %d", SchemaVersion)
	}
	if GeneratorVersion <= 0 {
		t.Fatalf("GeneratorVersion must be positive, got %d", GeneratorVersion)
	}
	if GeneratorPath == "" {
		t.Fatal("GeneratorPath must not be empty")
	}
}

func TestQuintReservedIdentifiersContainsCoreSymbols(t *testing.T) {
	for _, name := range []string{"Status", "Event", "init", "step", "apply", GeneratedCheckTransitionTable} {
		if !QuintReservedIdentifiers[name] {
			t.Fatalf("expected %q in QuintReservedIdentifiers", name)
		}
	}
}

func TestCommandIdentifiers(t *testing.T) {
	commands := []string{CommandTypecheck, CommandTest, CommandVerify, CommandRun}
	seen := map[string]struct{}{}
	for _, c := range commands {
		if c == "" {
			t.Fatal("command identifier must not be empty")
		}
		if _, ok := seen[c]; ok {
			t.Fatalf("duplicate command identifier %q", c)
		}
		seen[c] = struct{}{}
	}
}
