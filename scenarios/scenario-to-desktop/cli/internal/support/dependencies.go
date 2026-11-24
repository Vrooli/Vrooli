package support

import (
	"fmt"
	"os"
	"time"

	"scenario-to-desktop/cli/internal/cmdutil"

	"github.com/vrooli/cli-core/cliapp"
)

type Dependencies struct {
	Core      func() *cliapp.ScenarioApp
	Now       func() time.Time
	LookupEnv func(string) (string, bool)
	UserHome  func() (string, error)
}

var processEnvironment = struct {
	now       func() time.Time
	lookupEnv func(string) (string, bool)
	userHome  func() (string, error)
}{now: time.Now, lookupEnv: os.LookupEnv, userHome: os.UserHomeDir}

// WithProcessEnvironment fills unset process-boundary functions with the real
// process adapters. Callers and tests can supply narrow replacements per app.
func (d Dependencies) WithProcessEnvironment() Dependencies {
	if d.Now == nil {
		d.Now = processEnvironment.now
	}
	if d.LookupEnv == nil {
		d.LookupEnv = processEnvironment.lookupEnv
	}
	if d.UserHome == nil {
		d.UserHome = processEnvironment.userHome
	}
	return d
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
