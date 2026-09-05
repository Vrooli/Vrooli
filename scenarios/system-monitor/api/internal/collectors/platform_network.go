package collectors

type platformNetworkReading struct {
	values     map[string]interface{}
	status     string
	reason     string
	provenance string
}

func networkReading(values map[string]interface{}, source string) platformNetworkReading {
	return platformNetworkReading{values: values, status: "measured", provenance: source}
}
