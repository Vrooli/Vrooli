package collectors

type platformMemoryReading struct {
	usage      float64
	status     string
	reason     string
	provenance string
	details    map[string]int64
	swap       map[string]interface{}
}
