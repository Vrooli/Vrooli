package example
import "testing"
func checkSum(t *testing.T) {
 t.Helper()
 if 2+3 != 5 { t.Fatal("sum") }
}
func bridge(t *testing.T) { checkSum(t) }
func TestDirectHelper(t *testing.T) { checkSum(t) }
func TestTwoLevels(t *testing.T) { bridge(t) }
