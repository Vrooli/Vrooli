package hostinventory

import (
	"errors"
	"math"
)

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
	facts, err := hostMemoryFacts()
	if err != nil {
		return HostMemory{}, err
	}
	return validateHostMemoryFacts(facts)
}

func validateHostMemoryFacts(facts HostMemory) (HostMemory, error) {
	if facts.TotalBytes > math.MaxInt64 || facts.AvailableBytes > facts.TotalBytes || facts.Trustworthy && (facts.TotalBytes == 0 || facts.AvailableBytes == 0) {
		return HostMemory{}, errors.New("host memory counters are inconsistent or exceed supported capacity bounds")
	}
	return facts, nil
}
func hostMemoryFromDarwinPages(total, pageSize, free, inactive uint64) (HostMemory, error) {
	if pageSize == 0 || free > math.MaxUint64-inactive || free+inactive > math.MaxUint64/pageSize {
		return HostMemory{}, errors.New("native memory page counter overflow or invalid page size")
	}
	available := (free + inactive) * pageSize
	return validateHostMemoryFacts(HostMemory{TotalBytes: total, AvailableBytes: available, Trustworthy: total > 0 && available > 0})
}
