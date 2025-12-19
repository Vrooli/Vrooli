package database

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

// StringSlice is a []string that can be scanned from/written to JSONB.
type StringSlice []string

// Scan implements sql.Scanner for StringSlice.
func (s *StringSlice) Scan(src interface{}) error {
	if src == nil {
		*s = nil
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into StringSlice", src)
	}

	return json.Unmarshal(data, s)
}

// Value implements driver.Valuer for StringSlice.
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

// UUIDSlice is a []uuid.UUID that can be scanned from/written to JSONB.
type UUIDSlice []uuid.UUID

// Scan implements sql.Scanner for UUIDSlice.
func (u *UUIDSlice) Scan(src interface{}) error {
	if src == nil {
		*u = nil
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into UUIDSlice", src)
	}

	return json.Unmarshal(data, u)
}

// Value implements driver.Valuer for UUIDSlice.
func (u UUIDSlice) Value() (driver.Value, error) {
	if u == nil {
		return "[]", nil
	}
	return json.Marshal(u)
}

// ContextAttachmentSlice is a []domain.ContextAttachment for JSONB.
type ContextAttachmentSlice []domain.ContextAttachment

// Scan implements sql.Scanner for ContextAttachmentSlice.
func (c *ContextAttachmentSlice) Scan(src interface{}) error {
	if src == nil {
		*c = nil
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into ContextAttachmentSlice", src)
	}

	return json.Unmarshal(data, c)
}

// Value implements driver.Valuer for ContextAttachmentSlice.
func (c ContextAttachmentSlice) Value() (driver.Value, error) {
	if c == nil {
		return "[]", nil
	}
	return json.Marshal(c)
}

// NullableRunSummary wraps *domain.RunSummary for JSONB scanning.
type NullableRunSummary struct {
	V *domain.RunSummary
}

// Scan implements sql.Scanner for NullableRunSummary.
func (n *NullableRunSummary) Scan(src interface{}) error {
	if src == nil {
		n.V = nil
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into NullableRunSummary", src)
	}

	n.V = &domain.RunSummary{}
	return json.Unmarshal(data, n.V)
}

// Value implements driver.Valuer for NullableRunSummary.
func (n NullableRunSummary) Value() (driver.Value, error) {
	if n.V == nil {
		return nil, nil
	}
	return json.Marshal(n.V)
}

// NullableRunConfig wraps *domain.RunConfig for JSONB scanning.
type NullableRunConfig struct {
	V *domain.RunConfig
}

// Scan implements sql.Scanner for NullableRunConfig.
func (n *NullableRunConfig) Scan(src interface{}) error {
	if src == nil {
		n.V = nil
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into NullableRunConfig", src)
	}

	n.V = &domain.RunConfig{}
	return json.Unmarshal(data, n.V)
}

// Value implements driver.Valuer for NullableRunConfig.
func (n NullableRunConfig) Value() (driver.Value, error) {
	if n.V == nil {
		return nil, nil
	}
	return json.Marshal(n.V)
}

// PolicyRulesJSON wraps domain.PolicyRules for JSONB scanning.
type PolicyRulesJSON struct {
	V domain.PolicyRules
}

// Scan implements sql.Scanner for PolicyRulesJSON.
func (p *PolicyRulesJSON) Scan(src interface{}) error {
	if src == nil {
		p.V = domain.PolicyRules{}
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into PolicyRulesJSON", src)
	}

	return json.Unmarshal(data, &p.V)
}

// Value implements driver.Valuer for PolicyRulesJSON.
func (p PolicyRulesJSON) Value() (driver.Value, error) {
	return json.Marshal(p.V)
}

// EventDataJSON wraps domain.EventPayload for JSONB scanning.
type EventDataJSON struct {
	V         domain.EventPayload
	EventType domain.RunEventType
}

// Scan implements sql.Scanner for EventDataJSON.
func (e *EventDataJSON) Scan(src interface{}) error {
	if src == nil {
		e.V = nil
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into EventDataJSON", src)
	}

	// Parse into legacy RunEventData first, then convert
	var legacy domain.RunEventData
	if err := json.Unmarshal(data, &legacy); err != nil {
		return err
	}
	e.V = legacy.ToTypedPayload()
	return nil
}

// Value implements driver.Valuer for EventDataJSON.
func (e EventDataJSON) Value() (driver.Value, error) {
	if e.V == nil {
		return "{}", nil
	}
	return json.Marshal(e.V)
}

// MetadataMap wraps map[string]string for JSONB scanning.
type MetadataMap map[string]string

// Scan implements sql.Scanner for MetadataMap.
func (m *MetadataMap) Scan(src interface{}) error {
	if src == nil {
		*m = nil
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into MetadataMap", src)
	}

	return json.Unmarshal(data, m)
}

// Value implements driver.Valuer for MetadataMap.
func (m MetadataMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	return json.Marshal(m)
}

// NullableTime wraps *time.Time for nullable timestamp handling.
type NullableTime struct {
	Time  time.Time
	Valid bool
}

// Common SQLite/PostgreSQL time formats to try when parsing strings
var timeFormats = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999-07:00",
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05.999999999Z",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	time.RFC3339Nano,
	time.RFC3339,
}

// parseTimeString attempts to parse a time string using multiple formats.
func parseTimeString(s string) (time.Time, error) {
	for _, format := range timeFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time string %q", s)
}

// SQLiteTime is a time.Time that can be scanned from SQLite TEXT columns.
// SQLite stores timestamps as strings, so we need custom scanning logic.
type SQLiteTime time.Time

// Scan implements sql.Scanner for SQLiteTime.
func (t *SQLiteTime) Scan(src interface{}) error {
	if src == nil {
		*t = SQLiteTime(time.Time{})
		return nil
	}

	switch v := src.(type) {
	case time.Time:
		*t = SQLiteTime(v)
		return nil
	case string:
		parsed, err := parseTimeString(v)
		if err != nil {
			return err
		}
		*t = SQLiteTime(parsed)
		return nil
	case []byte:
		parsed, err := parseTimeString(string(v))
		if err != nil {
			return err
		}
		*t = SQLiteTime(parsed)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into SQLiteTime", src)
	}
}

// Value implements driver.Valuer for SQLiteTime.
// Returns a formatted string for SQLite compatibility.
func (t SQLiteTime) Value() (driver.Value, error) {
	tt := time.Time(t)
	if tt.IsZero() {
		return nil, nil
	}
	// Format in ISO 8601 / RFC 3339 for consistent parsing
	return tt.Format("2006-01-02 15:04:05.999999999"), nil
}

// Time returns the underlying time.Time.
func (t SQLiteTime) Time() time.Time {
	return time.Time(t)
}

// Scan implements sql.Scanner for NullableTime.
func (n *NullableTime) Scan(src interface{}) error {
	if src == nil {
		n.Valid = false
		return nil
	}

	switch v := src.(type) {
	case time.Time:
		n.Time = v
		n.Valid = true
		return nil
	case string:
		// SQLite stores timestamps as strings - try common formats
		for _, format := range timeFormats {
			if t, err := time.Parse(format, v); err == nil {
				n.Time = t
				n.Valid = true
				return nil
			}
		}
		return fmt.Errorf("cannot parse time string %q into NullableTime", v)
	case []byte:
		// Some drivers return []byte for strings
		s := string(v)
		for _, format := range timeFormats {
			if t, err := time.Parse(format, s); err == nil {
				n.Time = t
				n.Valid = true
				return nil
			}
		}
		return fmt.Errorf("cannot parse time bytes %q into NullableTime", s)
	default:
		return fmt.Errorf("cannot scan %T into NullableTime", src)
	}
}

// Value implements driver.Valuer for NullableTime.
// Returns a formatted string for SQLite compatibility.
func (n NullableTime) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	// Format in ISO 8601 / RFC 3339 for consistent parsing
	return n.Time.Format("2006-01-02 15:04:05.999999999"), nil
}

// ToPtr converts NullableTime to *time.Time.
func (n NullableTime) ToPtr() *time.Time {
	if !n.Valid {
		return nil
	}
	return &n.Time
}

// NewNullableTime creates a NullableTime from a *time.Time.
func NewNullableTime(t *time.Time) NullableTime {
	if t == nil {
		return NullableTime{Valid: false}
	}
	return NullableTime{Time: *t, Valid: true}
}

// NullableUUID wraps *uuid.UUID for nullable UUID handling.
type NullableUUID struct {
	UUID  uuid.UUID
	Valid bool
}

// Scan implements sql.Scanner for NullableUUID.
func (n *NullableUUID) Scan(src interface{}) error {
	if src == nil {
		n.Valid = false
		return nil
	}

	switch v := src.(type) {
	case []byte:
		parsed, err := uuid.ParseBytes(v)
		if err != nil {
			return err
		}
		n.UUID = parsed
		n.Valid = true
		return nil
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			return err
		}
		n.UUID = parsed
		n.Valid = true
		return nil
	default:
		return fmt.Errorf("cannot scan %T into NullableUUID", src)
	}
}

// Value implements driver.Valuer for NullableUUID.
func (n NullableUUID) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.UUID.String(), nil
}

// ToPtr converts NullableUUID to *uuid.UUID.
func (n NullableUUID) ToPtr() *uuid.UUID {
	if !n.Valid {
		return nil
	}
	return &n.UUID
}

// NewNullableUUID creates a NullableUUID from a *uuid.UUID.
func NewNullableUUID(u *uuid.UUID) NullableUUID {
	if u == nil {
		return NullableUUID{Valid: false}
	}
	return NullableUUID{UUID: *u, Valid: true}
}
