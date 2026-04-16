package appctx

import (
	"encoding/json"
	"net/url"

	"github.com/vrooli/cli-core/cliapp"
)

type scenarioContext struct {
	core *cliapp.ScenarioApp
}

// New returns an app context backed by the shared scenario app.
func New(core *cliapp.ScenarioApp) Context {
	return &scenarioContext{core: core}
}

func (c *scenarioContext) Get(path string, result interface{}) error {
	return c.GetWithQuery(path, nil, result)
}

func (c *scenarioContext) GetWithQuery(path string, query url.Values, result interface{}) error {
	body, err := c.core.Get(path, query)
	if err != nil {
		return err
	}
	return decode(body, result)
}

func (c *scenarioContext) Post(path string, payload interface{}, result interface{}) error {
	body, err := c.core.Request("POST", path, nil, payload)
	if err != nil {
		return err
	}
	return decode(body, result)
}

func (c *scenarioContext) Put(path string, payload interface{}, result interface{}) error {
	body, err := c.core.Request("PUT", path, nil, payload)
	if err != nil {
		return err
	}
	return decode(body, result)
}

func (c *scenarioContext) Delete(path string) error {
	_, err := c.core.Request("DELETE", path, nil, nil)
	return err
}

func (c *scenarioContext) DeleteWithResult(path string, result interface{}) error {
	body, err := c.core.Request("DELETE", path, nil, nil)
	if err != nil {
		return err
	}
	return decode(body, result)
}

func decode(body []byte, result interface{}) error {
	if result == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, result)
}
