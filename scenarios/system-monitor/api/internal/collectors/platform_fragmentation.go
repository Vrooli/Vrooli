package collectors

type fragmentationReading struct {
	values     map[string]interface{}
	status     string
	reason     string
	provenance string
}

func (r fragmentationReading) payload() map[string]interface{} {
	values := map[string]interface{}{
		"fragmentation_status":     r.status,
		"fragmentation_provenance": r.provenance,
	}
	if r.reason != "" {
		values["fragmentation_reason"] = r.reason
	}
	for key, value := range r.values {
		values[key] = value
	}
	return values
}
