package example
import (
 "testing"
 check "github.com/stretchr/testify/require"
)
func TestAlias(t *testing.T) {
 check.Equal(t, 5, 2+3)
}
