package prototime

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// OptionalTimestamp leaves zero times unset and converts populated times to UTC.
func OptionalTimestamp(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value.UTC())
}
