package manifest

import (
	"encoding/json"

	component "github.com/vrooli/vrooli/packages/desktop-component"
)

// ProjectComponent delegates to the projection shared with the bundle producer.
func ProjectComponent(owner string, services []Service) ([]Service, error) {
	raw, err := json.Marshal(services)
	if err != nil {
		return nil, err
	}
	raw, err = component.ProjectServices(owner, raw)
	if err != nil {
		return nil, err
	}
	var result []Service
	err = json.Unmarshal(raw, &result)
	return result, err
}
