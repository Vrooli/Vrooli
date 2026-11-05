package b
import "testing"
var check func(*testing.T)
func TestSeparatePackage(t *testing.T) { check(t) }
