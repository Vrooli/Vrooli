package platform

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// ParseJournalJSON parses newline-delimited journalctl JSON records. It keeps
// the parser independent from the Linux command runner so captured records
// can be verified on every build target.
func ParseJournalJSON(raw []byte) []HostLogEntry {
	var entries []HostLogEntry
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var record map[string]any
		if json.Unmarshal([]byte(line), &record) != nil {
			continue
		}
		entry := HostLogEntry{
			Timestamp: parseRecordTime(record["__REALTIME_TIMESTAMP"]),
			Message:   stringValue(record["MESSAGE"]),
			Process:   firstString(record, "_COMM", "_EXE"),
			Provider:  firstString(record, "SYSLOG_IDENTIFIER", "_TRANSPORT"),
			EventID:   stringValue(record["SYSLOG_PID"]),
			Unit:      stringValue(record["_SYSTEMD_UNIT"]),
			UserUnit:  stringValue(record["_SYSTEMD_USER_UNIT"]),
			Hostname:  stringValue(record["_HOSTNAME"]),
			BootID:    stringValue(record["_BOOT_ID"]),
			Cursor:    stringValue(record["__CURSOR"]),
			Priority:  intValue(record["PRIORITY"]),
			PID:       intValue(record["_PID"]),
			Raw:       line,
		}
		if entry.Timestamp.IsZero() {
			entry.Timestamp = parseRecordTime(record["timestamp"])
		}
		if entry.Message != "" {
			entries = append(entries, entry)
		}
	}
	return entries
}

// ParseMacOSNDJSON parses `log show --style ndjson` records.
func ParseMacOSNDJSON(raw []byte) []HostLogEntry {
	return parseJSONRecords(raw, func(record map[string]any, raw string) HostLogEntry {
		return HostLogEntry{
			Timestamp: parseRecordTime(record["timestamp"]),
			Message:   firstString(record, "eventMessage", "message"),
			Process:   firstString(record, "processImagePath", "process"),
			Provider:  firstString(record, "senderImagePath", "subsystem"),
			EventID:   firstString(record, "eventType"),
			Hostname:  firstString(record, "host"),
			Raw:       raw,
		}
	})
}

// ParseWindowsEventJSON parses objects emitted by Get-WinEvent after
// ConvertTo-Json. It accepts both a single object and an array.
func ParseWindowsEventJSON(raw []byte) []HostLogEntry {
	return parseJSONRecords(raw, func(record map[string]any, raw string) HostLogEntry {
		return HostLogEntry{
			Timestamp: parseRecordTime(record["TimeCreated"]),
			Message:   firstString(record, "Message", "message"),
			Process:   firstString(record, "ProcessId", "process"),
			Provider:  firstString(record, "ProviderName", "provider"),
			EventID:   firstString(record, "Id", "EventId", "event_id"),
			Hostname:  firstString(record, "MachineName", "machine"),
			Priority:  intValue(record["Level"]),
			Raw:       raw,
		}
	})
}

func parseJSONRecords(raw []byte, convert func(map[string]any, string) HostLogEntry) []HostLogEntry {
	trimmed := bytes.TrimSpace(stripProvenanceComments(raw))
	if len(trimmed) == 0 {
		return nil
	}
	if trimmed[0] == '[' {
		var records []map[string]any
		if json.Unmarshal(trimmed, &records) != nil {
			return nil
		}
		entries := make([]HostLogEntry, 0, len(records))
		for _, record := range records {
			entry := convert(record, marshalRecord(record))
			if entry.Message != "" {
				entries = append(entries, entry)
			}
		}
		return entries
	}

	var entries []HostLogEntry
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var record map[string]any
		if json.Unmarshal([]byte(line), &record) != nil {
			continue
		}
		entry := convert(record, line)
		if entry.Message != "" {
			entries = append(entries, entry)
		}
	}
	return entries
}

func stripProvenanceComments(raw []byte) []byte {
	lines := bytes.Split(raw, []byte("\n"))
	filtered := make([][]byte, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || bytes.HasPrefix(trimmed, []byte("#")) {
			continue
		}
		filtered = append(filtered, line)
	}
	return bytes.Join(filtered, []byte("\n"))
}

func marshalRecord(record map[string]any) string {
	data, _ := json.Marshal(record)
	return string(data)
}

func firstString(record map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(record[key]); value != "" {
			return value
		}
	}
	return ""
}

func stringValue(value any) string {
	switch value := value.(type) {
	case string:
		return strings.TrimSpace(value)
	case json.Number:
		return value.String()
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	default:
		return ""
	}
}

func intValue(value any) int {
	parsed, err := strconv.Atoi(stringValue(value))
	if err == nil {
		return parsed
	}
	if number, ok := value.(float64); ok {
		return int(number)
	}
	return 0
}

func parseRecordTime(value any) time.Time {
	if raw := strings.TrimSpace(stringValue(value)); raw != "" {
		if micros, err := strconv.ParseInt(raw, 10, 64); err == nil && micros > 1000000000000 {
			return time.UnixMicro(micros).UTC()
		}
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.0000000 -0700 MST"} {
			if parsed, err := time.Parse(layout, raw); err == nil {
				return parsed.UTC()
			}
		}
	}
	return time.Time{}
}
