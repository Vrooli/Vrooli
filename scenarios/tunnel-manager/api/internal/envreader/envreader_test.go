package envreader

import "testing"

func TestFuncReadsValuesAndTreatsEmptyAsAbsent(t *testing.T) {
	reader := Func(func(key string) string {
		if key == "present" {
			return "value"
		}
		return ""
	})

	if got := reader.Getenv("present"); got != "value" {
		t.Fatalf("Getenv(present) = %q, want value", got)
	}
	if got, ok := reader.LookupEnv("present"); got != "value" || !ok {
		t.Fatalf("LookupEnv(present) = (%q, %t), want (value, true)", got, ok)
	}
	if got, ok := reader.LookupEnv("missing"); got != "" || ok {
		t.Fatalf("LookupEnv(missing) = (%q, %t), want (empty, false)", got, ok)
	}
}

func TestSystemReadsProcessEnvironment(t *testing.T) {
	t.Setenv("TUNNEL_MANAGER_ENVREADER_TEST", "set")
	reader := System{}
	if got := reader.Getenv("TUNNEL_MANAGER_ENVREADER_TEST"); got != "set" {
		t.Fatalf("System.Getenv() = %q, want set", got)
	}
	if got, ok := reader.LookupEnv("TUNNEL_MANAGER_ENVREADER_TEST"); got != "set" || !ok {
		t.Fatalf("System.LookupEnv() = (%q, %t), want (set, true)", got, ok)
	}
}
