package checks

// Registry returns the registered contract checks in run order, each
// mapped one-to-one onto the finding vocabulary frozen in
// .vrooli/maturity.json.
func Registry() []Check {
	return []Check{
		prdPresenceCheck{},
		newTemplateCheck(),
		registryPresenceCheck{},
		registryStructureCheck{},
		registryQualityCheck{},
		readmeCheck{},
		linkageCheck{},
		refExistsCheck{},
		evidenceCheck{},
	}
}
