// Package pkga is one half of the cyclic-import fixture.
package pkga

import (
	"github.com/vrooli/fixtures/go-cycles/pkgb"
)

// TypeA is a small type declared in pkga.
type TypeA struct {
	Name string
}

// CallB references pkgb to create an import edge pkga -> pkgb.
func CallB() string {
	return pkgb.Greet()
}
