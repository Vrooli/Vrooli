package baselinefloor

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration is a time.Duration that marshals to/from a human-friendly Go duration
// string ("3h", "90m") in JSON, so the engagement manifest is readable by an
// operator inspecting it by hand. It tolerates a numeric value (nanoseconds) on
// read for forward-compatibility, but always writes the string form.
type Duration time.Duration

// AsDuration returns the underlying time.Duration.
func (d Duration) AsDuration() time.Duration {
	return time.Duration(d)
}

// String renders the Go duration form ("0s" for zero).
func (d Duration) String() string {
	return time.Duration(d).String()
}

// MarshalJSON writes the human-friendly string form.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON accepts either a duration string ("3h") or a raw nanosecond
// number.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("baselinefloor: parse duration %q: %w", value, err)
		}
		*d = Duration(parsed)
		return nil
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case nil:
		*d = 0
		return nil
	default:
		return fmt.Errorf("baselinefloor: invalid duration %v (%T)", v, v)
	}
}
