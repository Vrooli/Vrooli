package recovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

const (
	codingAgentMaxLedgerBytes = 8 << 20
	codingAgentMaxRecords     = 4096
)

var codingAgentRecordFields = func() map[string]bool {
	fields := map[string]bool{}
	t := reflect.TypeOf(CodingAgentRecord{})
	for i := 0; i < t.NumField(); i++ {
		name := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			fields[name] = true
		}
	}
	return fields
}()

func readCodingAgentRecords(path string) ([]CodingAgentRecord, error) {
	// Reject special files and aliases before Open: a FIFO must not hang the
	// API's initialization before it can report degraded escalation.
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() > codingAgentMaxLedgerBytes {
		return nil, fmt.Errorf("coding-agent ledger is not a bounded regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > codingAgentMaxLedgerBytes {
		return nil, fmt.Errorf("coding-agent ledger is not a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(f, codingAgentMaxLedgerBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > codingAgentMaxLedgerBytes {
		return nil, fmt.Errorf("coding-agent ledger exceeds byte limit")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	first, err := d.Token()
	if err != nil {
		return nil, err
	}
	// Preserve the historical empty-ledger representation.
	if first == nil {
		if _, err := d.Token(); err != io.EOF {
			return nil, fmt.Errorf("ledger has trailing content")
		}
		return nil, nil
	}
	if first != json.Delim('[') {
		return nil, fmt.Errorf("coding-agent ledger must be an array")
	}
	var records []CodingAgentRecord
	seenIDs := map[string]bool{}
	for d.More() {
		if len(records) == codingAgentMaxRecords {
			return nil, fmt.Errorf("coding-agent ledger exceeds record limit")
		}
		var raw json.RawMessage
		if err := d.Decode(&raw); err != nil {
			return nil, err
		}
		r, err := decodeCodingAgentRecord(raw)
		if err != nil {
			return nil, err
		}
		if r.ID == "" || len(r.ID) > 256 || r.Scenario == "" || len(r.Scenario) > 256 || seenIDs[r.ID] {
			return nil, fmt.Errorf("coding-agent ledger has invalid or duplicate receipt identity")
		}
		seenIDs[r.ID] = true
		switch r.Outcome {
		case "scheduled", "running", "unconfirmed", "succeeded", "failed":
		default:
			return nil, fmt.Errorf("coding-agent ledger has an unsupported outcome")
		}
		records = append(records, r)
	}
	if last, err := d.Token(); err != nil || last != json.Delim(']') {
		return nil, fmt.Errorf("coding-agent ledger is incomplete")
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, fmt.Errorf("coding-agent ledger has trailing content")
	}
	return records, nil
}

func decodeCodingAgentRecord(data []byte) (CodingAgentRecord, error) {
	var r CodingAgentRecord
	d := json.NewDecoder(bytes.NewReader(data))
	if first, err := d.Token(); err != nil || first != json.Delim('{') {
		return r, fmt.Errorf("invalid coding-agent receipt object")
	}
	seen := map[string]bool{}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return r, err
		}
		name, ok := token.(string)
		if !ok || !codingAgentRecordFields[name] || seen[name] {
			return r, fmt.Errorf("coding-agent receipt has unknown, noncanonical or duplicate fields")
		}
		seen[name] = true
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return r, err
		}
	}
	if last, err := d.Token(); err != nil || last != json.Delim('}') {
		return r, fmt.Errorf("coding-agent receipt is incomplete")
	}
	if err := json.Unmarshal(data, &r); err != nil {
		// Timestamp decoders can quote the entire invalid value. Ledger data
		// must not become an unbounded or sensitive API health/log message.
		return r, fmt.Errorf("coding-agent receipt has invalid typed values")
	}
	value := reflect.ValueOf(r)
	for i := 0; i < value.NumField(); i++ {
		if field := value.Field(i); field.Kind() == reflect.String && field.Len() > 4096 {
			return r, fmt.Errorf("coding-agent receipt string exceeds its bound")
		}
	}
	return r, nil
}
