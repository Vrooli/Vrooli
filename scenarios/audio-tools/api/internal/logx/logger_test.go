package logx

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStdPrintf_WritesToConfiguredLogger(t *testing.T) {
	var buf bytes.Buffer
	s := Std{L: log.New(&buf, "", 0)}
	s.Printf("hello %s", "world")
	require.Contains(t, buf.String(), "hello world")
}

func TestStdPrintf_NilFallsBackToDefault(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Default().Writer()
	log.Default().SetOutput(&buf)
	t.Cleanup(func() { log.Default().SetOutput(prev) })

	Std{}.Printf("fallback %d", 42)
	require.True(t, strings.Contains(buf.String(), "fallback 42"), "got %q", buf.String())
}
