package example
import (
 "os/exec"
 "testing"
)
func TestConditionalSkip(t *testing.T) {
 if _, err := exec.LookPath("go"); err != nil { t.Skip("tool unavailable") }
 if 2+3 != 5 { t.Fatal("sum") }
}
func TestPlaceholder(t *testing.T) {
 t.Skip("TODO implement behavior")
}
