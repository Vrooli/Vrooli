package support

import (
	"fmt"
	"scenario-to-desktop/cli/internal/cmdutil"

	"github.com/vrooli/cli-core/cliapp"
)

type Dependencies struct {
	Core func() *cliapp.ScenarioApp
}

func (d Dependencies) ScenarioApp() *cliapp.ScenarioApp {
	if d.Core == nil {
		return nil
	}
	return d.Core()
}

func (d Dependencies) Get(path string, query map[string]string) ([]byte, error) {
	core := d.ScenarioApp()
	if core == nil {
		return nil, fmt.Errorf("scenario app not configured")
	}
	return core.Get(path, cmdutil.MapToValues(query))
}

func (d Dependencies) Request(method, path string, query map[string]string, body interface{}) ([]byte, error) {
	core := d.ScenarioApp()
	if core == nil {
		return nil, fmt.Errorf("scenario app not configured")
	}
	return core.Request(method, path, cmdutil.MapToValues(query), body)
}

func (d Dependencies) GetRoot(path string, query map[string]string) ([]byte, error) {
	core := d.ScenarioApp()
	if core == nil {
		return nil, fmt.Errorf("scenario app not configured")
	}
	return core.GetRoot(path, cmdutil.MapToValues(query))
}

func (d Dependencies) RequestRoot(method, path string, query map[string]string, body interface{}) ([]byte, error) {
	core := d.ScenarioApp()
	if core == nil {
		return nil, fmt.Errorf("scenario app not configured")
	}
	return core.RequestRoot(method, path, cmdutil.MapToValues(query), body)
}
