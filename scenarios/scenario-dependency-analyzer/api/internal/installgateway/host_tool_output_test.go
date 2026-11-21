package installgateway

import (
	"bytes"
	"strings"
	"testing"
)

func TestHostToolOutputPreservesSmallCommandDiagnostics(t *testing.T) {
	var output hostToolOutput
	for _, text := range []string{"normal output\n", "failure details\n"} {
		n, err := output.Write([]byte(text))
		if err != nil || n != len(text) {
			t.Fatal(n, err)
		}
	}
	if output.String() != "normal output\nfailure details\n" {
		t.Fatal(output.String())
	}
}

func TestHostToolOutputKeepsBoundedTailAndExplicitTruncation(t *testing.T) {
	for _, chunkBytes := range []int{137, 3 * hostToolOutputBytes} {
		var output hostToolOutput
		chunk := bytes.Repeat([]byte{'x'}, chunkBytes)
		for total := 0; total < 4*hostToolOutputBytes; total += len(chunk) {
			n, err := output.Write(chunk)
			if err != nil || n != len(chunk) {
				t.Fatal(n, err)
			}
			if len(output.tail) > hostToolOutputBytes || cap(output.tail) > hostToolOutputBytes {
				t.Fatal("retained output grew beyond budget")
			}
		}
		_, _ = output.Write([]byte("\nresolver failure witness"))
		got := output.String()
		if !strings.HasPrefix(got, "[earlier package-manager output truncated;") || !strings.HasSuffix(got, "resolver failure witness") || len(output.tail) != hostToolOutputBytes || len(got) > hostToolOutputBytes+100 {
			t.Fatalf("bounded tail lost failure or truncation: bytes=%d retained=%d", len(got), len(output.tail))
		}
	}
}

func TestHostToolOutputPreservesByteOrderAcrossRingWrapAndExactBudget(t *testing.T) {
	var output hostToolOutput
	exact := bytes.Repeat([]byte("abcd"), hostToolOutputBytes/4)
	_, _ = output.Write(exact)
	if output.String() != string(exact) {
		t.Fatal("exact budget was truncated or reordered")
	}
	var expected bytes.Buffer
	_, _ = expected.Write(exact)
	for i := 0; i < 100; i++ {
		chunk := bytes.Repeat([]byte{byte(i)}, 13013)
		_, _ = output.Write(chunk)
		_, _ = expected.Write(chunk)
	}
	want := expected.Bytes()[expected.Len()-hostToolOutputBytes:]
	if !strings.HasSuffix(output.String(), string(want)) {
		t.Fatal("wrapped tail changed byte order")
	}
}
