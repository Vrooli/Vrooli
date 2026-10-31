package example
import (
 "sync"
 "testing"
)
type Handler interface { Handle() }
type implementation struct{}
func (implementation) Handle() {}
var _ Handler = implementation{}
func TestCompileContract(t *testing.T) {}
func TestConcurrentExercise(t *testing.T) {
 var mu sync.Mutex
 var wg sync.WaitGroup
 value := 0
 for i := 0; i < 2; i++ {
  wg.Add(1)
  go func() { defer wg.Done(); mu.Lock(); value++; mu.Unlock() }()
 }
 wg.Wait()
}
