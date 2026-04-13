package resources

import resourceenv "github.com/vrooli/vrooli/internal/resources/env"

type PortRegistry = resourceenv.PortRegistry

func LoadPortRegistry(root string) (PortRegistry, error) {
	return resourceenv.LoadPortRegistry(root)
}

func LoadResourceEnvironment(root, home, resourceName string) (map[string]string, error) {
	report, err := resourceenv.ResolveResource(root, home, resourceName, resourceenv.ResolveOptions{})
	if err != nil {
		return nil, err
	}
	return report.Values, nil
}
