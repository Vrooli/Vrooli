package invokers

// renderedInvokers lists the argv of every core native unit as platform-go
// renders it. It is populated by the service-definition seam; until that
// seam exists this returns nothing so the catalog entries still run.
func renderedInvokers() ([]Invoker, error) {
	return nil, nil
}
