package support

import (
	"io"
	"os"

	"github.com/vrooli/cli-core/cliapp"
)

type Dependencies struct {
	Core  func() *cliapp.ScenarioApp
	Stdin func() io.Reader
}

func (d Dependencies) ScenarioApp() *cliapp.ScenarioApp {
	if d.Core == nil {
		return nil
	}
	return d.Core()
}

func (d Dependencies) Input() io.Reader {
	if d.Stdin == nil {
		return os.Stdin
	}
	if r := d.Stdin(); r != nil {
		return r
	}
	return os.Stdin
}
