// Package protoint contains checked conversions at protobuf boundaries.
package protoint

import "math"

// FromInt clamps a host-sized integer to protobuf int32's representable
// range. Protobuf mappers must not silently wrap counters or user-controlled
// dimensions when a host grows beyond int32.
func FromInt(value int) int32 {
	return clampInt32(int64(value))
}

// FromInt64 clamps a wider signed counter before projecting it to protobuf int32.
func FromInt64(value int64) int32 {
	return clampInt32(value)
}

// FromUint64 clamps an unsigned sequence before projecting it to a signed ledger value.
func FromUint64(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value) // #nosec G115 -- bounds are checked immediately above.
}

// ToUint64 maps a signed ledger sequence without allowing negative wraparound.
func ToUint64(value int64) uint64 {
	if value < 0 {
		return 0
	}
	return uint64(value) // #nosec G115 -- the negative case is handled above.
}

// PCMInt16 decodes a signed 16-bit PCM sample from its wire representation.
func PCMInt16(value uint16) int16 {
	return int16(value) // #nosec G115 -- this is an intentional two's-complement PCM reinterpretation.
}

// PCMUint16 encodes a signed 16-bit PCM sample without changing its bit pattern.
func PCMUint16(value int16) uint16 {
	return uint16(value) // #nosec G115 -- this is an intentional two's-complement PCM reinterpretation.
}

// FromInt32ToInt16 clamps an intermediate sample calculation to PCM range.
func FromInt32ToInt16(value int32) int16 {
	return clampInt16(int64(value))
}

// FromIntToInt16 clamps a host-sized sample calculation to PCM range.
func FromIntToInt16(value int) int16 {
	return clampInt16(int64(value))
}

// FromIntToUint32 clamps a host-sized buffer length to a wire uint32.
func FromIntToUint32(value int) uint32 {
	if value < 0 {
		return 0
	}
	if uint64(value) > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(value) // #nosec G115 -- bounds are checked immediately above.
}

func clampInt32(value int64) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}
	return int32(value) // #nosec G115 -- bounds are checked immediately above.
}

func clampInt16(value int64) int16 {
	if value > math.MaxInt16 {
		return math.MaxInt16
	}
	if value < math.MinInt16 {
		return math.MinInt16
	}
	return int16(value) // #nosec G115 -- bounds are checked immediately above.
}
