package example
import "testing"
func cycleA(t *testing.T) { cycleB(t) }
func cycleB(t *testing.T) { cycleA(t) }
func TestCycle(t *testing.T) { cycleA(t) }
