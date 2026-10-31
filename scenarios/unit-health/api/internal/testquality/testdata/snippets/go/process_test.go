package example
import (
 "os"
 "testing"
)
func TestRegisteredHelper(t *testing.T) {
 if os.Getenv("UNIT_CHILD") != "1" { return }
 os.Exit(0)
}
func TestLooksLikeHelper(t *testing.T) {
 if os.Getenv("UNIT_CHILD") != "1" { return }
 os.Exit(0)
}
