// Package viewport owns BAS's named CSS-pixel viewport presets.
package viewport

import "fmt"

type Dimensions struct{ Width, Height int32 }

var presets = map[string]Dimensions{
	"mobile":  {Width: 390, Height: 844},
	"tablet":  {Width: 768, Height: 1024},
	"desktop": {Width: 1440, Height: 900},
}

func Resolve(name string) (Dimensions, error) {
	dimensions, ok := presets[name]
	if !ok {
		return Dimensions{}, fmt.Errorf("unknown viewport preset %q (want mobile|tablet|desktop)", name)
	}
	return dimensions, nil
}

func Default() Dimensions { return presets["desktop"] }
