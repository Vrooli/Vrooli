package hostinventory

// HostMemory is the portable memory signal used by lifecycle concurrency
// budgets. AvailableBytes is conservative and platform-specific; Trustworthy
// is explicit so callers never interpret a failed zero reading as unlimited
// capacity.
type HostMemory struct {
	TotalBytes     uint64
	AvailableBytes uint64
	Trustworthy    bool
}

// HostMemoryFacts returns host total and available memory through the
// platform-specific implementation. Unsupported platforms return an explicit
// untrustworthy result.
func HostMemoryFacts() (HostMemory, error) {
	return hostMemoryFacts()
}
