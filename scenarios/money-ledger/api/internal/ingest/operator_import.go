package ingest

import (
	"context"
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
	Path    string
	Status  string
	Written bool
}

type OperatorImportReport struct {
	Fields  []OperatorFieldReport
	Read    int
	Written int
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
	return s.importOperatorInputs(ctx, data, "operator-inputs.json", adapterID, bookID, accountID)
}

// ImportOperatorInputsFile is the fixture-only migration seam. The caller
// chooses the source explicitly; the importer never discovers or mutates a
// live shared/operator-inputs.json path.
func (s *Store) ImportOperatorInputsFile(ctx context.Context, sourcePath, adapterID, bookID, accountID string) (*OperatorImportReport, error) {
	if strings.TrimSpace(sourcePath) == "" {
		return nil, errors.New("operator-inputs source path is required")
	}
	cleaned := strings.ReplaceAll(sourcePath, "\\", "/")
	if !strings.Contains(cleaned, "/testdata/") && !strings.HasPrefix(cleaned, "testdata/") {
		return nil, errors.New("operator-inputs importer accepts fixture paths only")
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("read operator-inputs fixture: %w", err)
	}
	return s.importOperatorInputs(ctx, data, sourcePath, adapterID, bookID, accountID)
}

func (s *Store) importOperatorInputs(ctx context.Context, data []byte, sourcePath, adapterID, bookID, accountID string) (*OperatorImportReport, error) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("operator-inputs JSON: %w", err)
	}
	if strings.TrimSpace(adapterID) == "" || strings.TrimSpace(bookID) == "" || strings.TrimSpace(accountID) == "" {
		return nil, errors.New("adapter_id, book_id, and account_id are required")
	}
	report := &OperatorImportReport{}
	for _, path := range operatorPaths {
		value, present := nestedValue(root, strings.Split(path, "."))
		if !present {
			return nil, fmt.Errorf("operator-inputs field %s is missing", path)
		}
		field := OperatorFieldReport{Path: path + ".value", Status: "absent"}
		report.Read++
		obj, _ := value.(map[string]any)
		status, _ := obj["status"].(string)
		if status == "pending-operator" || obj["value"] == nil {
			field.Status = status
			report.Fields = append(report.Fields, field)
			continue
		}
		number, ok := obj["value"].(float64)
		if !ok {
			return nil, fmt.Errorf("operator-inputs field %s has non-numeric value", path)
		}
		now := time.Now().UTC()
		event := &sharedpb.MoneyEvent{ExternalId: "operator-inputs:" + path, AdapterId: adapterID, AccountId: accountID, BookId: bookID, AmountMinor: int64(number), Currency: "USD", OccurredAt: timestamppb.New(now), FetchedAt: timestamppb.New(now), Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, Category: path, Description: "Imported operator input from " + sourcePath}
		_, duplicate, err := s.journal.Ingest(ctx, event, "operator")
		if err != nil {
			return nil, fmt.Errorf("operator-inputs field %s: %w", path, err)
		}
		field.Status, field.Written = "current", !duplicate
		if field.Written {
			report.Written++
		}
		report.Fields = append(report.Fields, field)
	}
	return report, nil
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
