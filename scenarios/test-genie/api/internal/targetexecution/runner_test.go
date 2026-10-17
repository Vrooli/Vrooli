package targetexecution

import "testing"

func TestForLanguage(t *testing.T) {
	goRunner, err := ForLanguage("go")
	if err != nil || len(goRunner.Command) == 0 {
		t.Fatalf("Go runner = %#v, err=%v", goRunner, err)
	}
	tsRunner, err := ForLanguage("typescript")
	if err != nil || len(tsRunner.Command) == 0 {
		t.Fatalf("TypeScript runner = %#v, err=%v", tsRunner, err)
	}
	if _, err := ForLanguage("python"); err == nil {
		t.Fatal("unsupported language unexpectedly received a runner")
	}
}
