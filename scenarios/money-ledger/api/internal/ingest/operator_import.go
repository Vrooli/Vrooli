package ingest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OperatorFieldReport struct {
	Path       string
	Status     string
	Written    bool
	Kind       string
	Unit       string
	WindowDays int
	ObservedAt *time.Time
	Reason     string
}

type OperatorImportReport struct {
	Fields   []OperatorFieldReport
	Read     int
	Written  int
	Findings int
	Applied  bool
}

func (s *Store) OperatorInputStatus(ctx context.Context, bookID string) ([]OperatorFieldReport, error) {
	fields := make([]OperatorFieldReport, 0, len(operatorPaths))
	for _, path := range operatorPaths {
		kind, unit, window := operatorFieldClass(path)
		field := OperatorFieldReport{Path: path, Status: "absent", Kind: kind, Unit: unit, WindowDays: window}
		if kind == "derived-rate" {
			field.Status = "rejected"
			field.Reason = "derived rate is computed from journal telemetry and cannot be imported as a posting"
			fields = append(fields, field)
			continue
		}
		var observed string
		var err error
		if kind == "measure" {
			err = s.db.QueryRowContext(ctx, `SELECT observed_at FROM operator_measures WHERE path=? ORDER BY observed_at DESC,id DESC LIMIT 1`, path).Scan(&observed)
		} else {
			if bookID == "" {
				err = s.db.QueryRowContext(ctx, `SELECT occurred_at FROM postings WHERE category=? ORDER BY occurred_at DESC,id DESC LIMIT 1`, path).Scan(&observed)
			} else {
				err = s.db.QueryRowContext(ctx, `SELECT occurred_at FROM postings WHERE category=? AND book_id=? ORDER BY occurred_at DESC,id DESC LIMIT 1`, path, bookID).Scan(&observed)
			}
		}
		if errors.Is(err, sql.ErrNoRows) {
			fields = append(fields, field)
			continue
		}
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(observed) == "" {
			field.Status = "current"
			field.Reason = "observation timestamp unavailable"
			fields = append(fields, field)
			continue
		}
		at, parseErr := time.Parse(time.RFC3339Nano, observed)
		if parseErr != nil {
			return nil, parseErr
		}
		field.ObservedAt = &at
		field.Status = "current"
		if s.now().UTC().Sub(at) > time.Duration(window)*24*time.Hour {
			field.Status = "stale"
			field.Reason = "observation is older than the declared staleness window"
		}
		fields = append(fields, field)
	}
	return fields, nil
}

var operatorPaths = []string{
	"cash", "monthlyBurn.aiApi", "monthlyBurn.infrastructure", "monthlyBurn.saas", "monthlyBurn.tooling",
	"timeAllocation.product", "timeAllocation.services", "timeAllocation.ops",
	"servicesRevenue.leadGen", "servicesRevenue.doneForYou", "servicesRevenue.consulting", "servicesTime.hoursThisWindow", "subscriptions.mrr",
}

// ImportOperatorInputs verifies the source shape and admits only populated,
// non-pending numeric fields. Null and pending-operator are absent facts, never
// zeroes. The source bytes are supplied by the caller and are never deleted.
func (s *Store) ImportOperatorInputs(ctx context.Context, data []byte, adapterID, bookID, accountID string) (*OperatorImportReport, error) {
	return s.importOperatorInputs(ctx, data, "operator-inputs.json", adapterID, bookID, accountID, true)
}

func (s *Store) ImportOperatorInputsJSON(ctx context.Context, data []byte, apply bool, adapterID, bookID, accountID string) (*OperatorImportReport, error) {
	return s.importOperatorInputs(ctx, data, "money-ledger-console", adapterID, bookID, accountID, apply)
}

// ImportOperatorInputsFile is the fixture-only migration seam. The caller
// chooses the source explicitly; the importer never discovers or mutates a
// live shared/operator-inputs.json path.
func (s *Store) ImportOperatorInputsFile(ctx context.Context, sourcePath, adapterID, bookID, accountID string) (*OperatorImportReport, error) {
	return s.ImportOperatorInputsSource(ctx, sourcePath, OperatorSourceModeFixture, true, adapterID, bookID, accountID)
}

type OperatorSourceMode string

const (
	OperatorSourceModeFixture  OperatorSourceMode = "fixture"
	OperatorSourceModeOperator OperatorSourceMode = "operator"
)

// ImportOperatorInputsSource is the safe operator seam. The caller chooses the
// source mode and must opt into apply; dry-run reports never call the journal or
// write the source file.
func (s *Store) ImportOperatorInputsSource(ctx context.Context, sourcePath string, mode OperatorSourceMode, apply bool, adapterID, bookID, accountID string) (*OperatorImportReport, error) {
	if strings.TrimSpace(sourcePath) == "" {
		return nil, errors.New("operator-inputs source path is required")
	}
	cleaned := strings.ReplaceAll(sourcePath, "\\", "/")
	if mode == OperatorSourceModeFixture && !strings.Contains(cleaned, "/testdata/") && !strings.HasPrefix(cleaned, "testdata/") {
		return nil, errors.New("operator-inputs fixture mode requires a testdata path")
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("read operator-inputs source: %w", err)
	}
	return s.importOperatorInputs(ctx, data, sourcePath, adapterID, bookID, accountID, apply)
}

func (s *Store) importOperatorInputs(ctx context.Context, data []byte, sourcePath, adapterID, bookID, accountID string, apply bool) (*OperatorImportReport, error) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("operator-inputs JSON: %w", err)
	}
	if strings.TrimSpace(adapterID) == "" || strings.TrimSpace(bookID) == "" || strings.TrimSpace(accountID) == "" {
		return nil, errors.New("adapter_id, book_id, and account_id are required")
	}
	report := &OperatorImportReport{}
	type prepared struct {
		path, kind, unit, status, reason string
		value                            float64
		window                           int
		observed                         *time.Time
	}
	preparedFields := make([]prepared, 0, len(operatorPaths))
	for _, path := range operatorPaths {
		value, present := nestedValue(root, strings.Split(path, "."))
		if !present {
			return nil, fmt.Errorf("operator-inputs field %s is missing", path)
		}
		kind, unit, window := operatorFieldClass(path)
		field := OperatorFieldReport{Path: path, Status: "absent", Kind: kind, Unit: unit, WindowDays: window}
		report.Read++
		obj, _ := value.(map[string]any)
		status, _ := obj["status"].(string)
		if status == "pending-operator" || obj["value"] == nil {
			if status == "" || obj["value"] == nil && status != "pending-operator" {
				status = "absent"
			}
			field.Status = status
			preparedFields = append(preparedFields, prepared{path: path, kind: kind, unit: unit, status: status, window: window})
			report.Fields = append(report.Fields, field)
			continue
		}
		number, ok := obj["value"].(float64)
		if !ok {
			return nil, fmt.Errorf("operator-inputs field %s has non-numeric value", path)
		}
		var observed *time.Time
		if raw, ok := obj["updatedAt"].(string); ok && strings.TrimSpace(raw) != "" {
			parsed, parseErr := time.Parse(time.RFC3339, raw)
			if parseErr != nil {
				return nil, fmt.Errorf("operator-inputs field %s updatedAt: %w", path, parseErr)
			}
			parsed = parsed.UTC()
			observed = &parsed
		}
		field.Status, field.ObservedAt = status, observed
		if observed != nil && s.now().UTC().Sub(*observed) > time.Duration(window)*24*time.Hour {
			field.Status, field.Reason = "stale", "observation is older than the declared staleness window"
		}
		preparedFields = append(preparedFields, prepared{path: path, kind: kind, unit: unit, status: field.Status, reason: field.Reason, value: number, window: window, observed: observed})
		if kind == "derived-rate" {
			field.Status, field.Reason = "rejected", "derived rate is computed from journal telemetry and cannot be imported as a posting"
			report.Findings++
		} else if field.Status == "stale" {
			report.Findings++
		} else if kind == "measure" {
			field.Status = "current-measure"
		} else {
			field.Status = "current"
		}
		report.Fields = append(report.Fields, field)
	}
	if !apply {
		return report, nil
	}
	for i := range preparedFields {
		field := &report.Fields[i]
		p := preparedFields[i]
		if p.status == "pending-operator" || p.status == "absent" || p.status == "not-applicable-pre-launch" || p.kind == "derived-rate" || p.status == "stale" || p.kind == "measure" {
			continue
		}
		now := s.now().UTC()
		event := &sharedpb.MoneyEvent{ExternalId: "operator-inputs:" + p.path, AdapterId: adapterID, AccountId: accountID, BookId: bookID, AmountMinor: int64(p.value), Currency: "USD", OccurredAt: timestamppb.New(now), FetchedAt: timestamppb.New(now), Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, Category: p.path, Description: "Imported operator input from " + sourcePath}
		_, duplicate, err := s.journal.Ingest(ctx, event, "operator")
		if err != nil {
			return nil, fmt.Errorf("operator-inputs field %s: %w", p.path, err)
		}
		field.Written, field.Status = !duplicate, "current"
		if field.Written {
			report.Written++
		}
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM operator_measures WHERE source_path=?`, sourcePath); err != nil {
		return nil, err
	}
	for _, p := range preparedFields {
		if p.kind == "measure" && p.status != "pending-operator" && p.status != "absent" && p.status != "not-applicable-pre-launch" {
			observed := ""
			if p.observed != nil {
				observed = p.observed.Format(time.RFC3339Nano)
			}
			if _, err := s.db.ExecContext(ctx, `INSERT INTO operator_measures(path,value,unit,window_days,observed_at,status,source_path) VALUES(?,?,?,?,?,?,?)`, p.path, p.value, p.unit, p.window, observed, p.status, sourcePath); err != nil {
				return nil, err
			}
		}
	}
	for _, p := range preparedFields {
		if p.kind == "derived-rate" && p.status != "pending-operator" && p.status != "absent" && p.status != "not-applicable-pre-launch" {
			if _, err := s.db.ExecContext(ctx, `INSERT INTO operator_input_findings(path,reason,created_at) VALUES(?,?,?)`, p.path, "derived rate is not an admissible journal input", s.now().UTC().Format(time.RFC3339Nano)); err != nil {
				return nil, err
			}
		}
	}
	report.Applied = true
	return report, nil
}

func operatorFieldClass(path string) (kind, unit string, window int) {
	switch {
	case strings.HasPrefix(path, "timeAllocation."), path == "servicesTime.hoursThisWindow":
		return "measure", "share-of-time-budget", 7
	case path == "subscriptions.mrr":
		return "derived-rate", "currency-major-per-month", 30
	case strings.HasPrefix(path, "servicesRevenue."):
		return "monetary", "currency-minor", 30
	default:
		return "monetary", "currency-minor", 30
	}
}

func nestedValue(root map[string]any, parts []string) (any, bool) {
	var current any = root
	for _, part := range parts {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}
