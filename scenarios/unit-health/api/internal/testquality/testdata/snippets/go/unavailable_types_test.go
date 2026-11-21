package example
import (
 "testing"
 external "example.invalid/unavailable/assertions"
)
func TestUnavailableTypes(t *testing.T) { external.Check(t) }
