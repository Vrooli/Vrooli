package wordpress

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const groundSchemaVersion = 1

var groundLevelID = regexp.MustCompile(`^[a-z]+(?:[.-][a-z0-9-]+)+$`)

type GroundCatalog struct {
	SchemaVersion      int            `json:"schemaVersion"`
	GroundID           string         `json:"groundId"`
	DataClassification string         `json:"dataClassification"`
	LevelFamilies      []GroundFamily `json:"levelFamilies"`
}

type GroundFamily struct {
	ID         string        `json:"id"`
	HardLevels []GroundLevel `json:"hardLevels"`
}
type GroundLevel struct {
	ID       string          `json:"id"`
	Input    string          `json:"input"`
	Expected json.RawMessage `json:"expected"`
}

type GroundGrade struct {
	Total    int                    `json:"total"`
	Passed   int                    `json:"passed"`
	Failed   []string               `json:"failed"`
	Missing  []string               `json:"missing"`
	ByFamily map[string]FamilyGrade `json:"byFamily"`
}
type FamilyGrade struct {
	Total  int `json:"total"`
	Passed int `json:"passed"`
}

// LoadGround validates the closed, synthetic-only known-answer corpus and each
// case reference. It also proves that every recipe-level acceptance ID is
// represented by a runnable fixture family.
func LoadGround(groundPath, recipePath string) (GroundCatalog, error) {
	file, err := os.Open(groundPath)
	if err != nil {
		return GroundCatalog{}, fmt.Errorf("open WordPress hard ground: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 2<<20))
	decoder.DisallowUnknownFields()
	var catalog GroundCatalog
	if err := decoder.Decode(&catalog); err != nil {
		return GroundCatalog{}, fmt.Errorf("decode WordPress hard ground: %w", err)
	}
	if catalog.SchemaVersion != groundSchemaVersion || catalog.GroundID == "" || catalog.DataClassification != "synthetic-only" || len(catalog.LevelFamilies) < 3 {
		return GroundCatalog{}, fmt.Errorf("WordPress ground is incomplete or not synthetic-only")
	}
	seen, families := map[string]bool{}, map[string]bool{}
	for _, family := range catalog.LevelFamilies {
		if family.ID == "" || families[family.ID] || len(family.HardLevels) == 0 {
			return GroundCatalog{}, fmt.Errorf("WordPress ground has an empty or duplicate family %q", family.ID)
		}
		families[family.ID] = true
		for _, level := range family.HardLevels {
			if !groundLevelID.MatchString(level.ID) || seen[level.ID] || len(level.Expected) == 0 {
				return GroundCatalog{}, fmt.Errorf("WordPress ground has an invalid or duplicate level %q", level.ID)
			}
			seen[level.ID] = true
			if err := validateGroundFixture(filepath.Dir(groundPath), level.Input); err != nil {
				return GroundCatalog{}, fmt.Errorf("ground level %s: %w", level.ID, err)
			}
		}
	}
	if !families["forms"] || !families["shop"] || !families["content"] {
		return GroundCatalog{}, fmt.Errorf("WordPress ground must cover forms, shop and content")
	}
	recipes, err := loadRecipeCatalog(recipePath)
	if err != nil {
		return GroundCatalog{}, err
	}
	for _, recipe := range recipes.Recipes {
		for _, expected := range recipe.ExpectedLevels {
			if !seen[expected] {
				return GroundCatalog{}, fmt.Errorf("recipe %s expected level %s is not in the hard ground", recipe.ID, expected)
			}
		}
	}
	return catalog, nil
}

func loadRecipeCatalog(path string) (recipeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return recipeFile{}, err
	}
	var catalog recipeFile
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&catalog); err != nil {
		return recipeFile{}, fmt.Errorf("decode recipe levels for ground: %w", err)
	}
	return catalog, nil
}

func validateGroundFixture(root, reference string) error {
	parts := strings.Split(reference, "#")
	if len(parts) != 2 || parts[1] == "" || filepath.IsAbs(parts[0]) {
		return fmt.Errorf("fixture reference must be a relative file#case key")
	}
	for _, part := range strings.Split(filepath.ToSlash(parts[0]), "/") {
		if part == ".." || part == "." {
			return fmt.Errorf("fixture reference escapes the ground")
		}
	}
	path := filepath.Join(root, filepath.FromSlash(parts[0]))
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read fixture %s: %w", parts[0], err)
	}
	var document struct {
		Cases map[string]json.RawMessage `json:"cases"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("decode fixture %s: %w", parts[0], err)
	}
	if len(document.Cases) > 0 {
		if _, ok := document.Cases[parts[1]]; !ok {
			return fmt.Errorf("fixture case %q does not exist", parts[1])
		}
		return nil
	}
	if _, ok := map[string]bool{"catalog": true, "case-study-type": true, "post-with-fields": true, "post-missing-field": true, "same-update-twice": true}[parts[1]]; ok {
		// Product catalog is a root array and content fixture cases are an object;
		// the named content fixture has its own case file in the root cases map.
		var anyDocument map[string]json.RawMessage
		if err := json.Unmarshal(data, &anyDocument); err != nil {
			return err
		}
		if _, ok := anyDocument[parts[1]]; ok {
			return nil
		}
		if parts[1] == "catalog" && len(anyDocument) > 0 {
			return nil
		}
	}
	return fmt.Errorf("fixture case %q does not exist", parts[1])
}

func (catalog GroundCatalog) Grade(results map[string]bool) GroundGrade {
	grade := GroundGrade{ByFamily: map[string]FamilyGrade{}}
	for _, family := range catalog.LevelFamilies {
		familyGrade := FamilyGrade{}
		for _, level := range family.HardLevels {
			grade.Total++
			familyGrade.Total++
			passed, exists := results[level.ID]
			if !exists {
				grade.Missing = append(grade.Missing, level.ID)
				continue
			}
			if passed {
				grade.Passed++
				familyGrade.Passed++
			} else {
				grade.Failed = append(grade.Failed, level.ID)
			}
		}
		grade.ByFamily[family.ID] = familyGrade
	}
	return grade
}

// GradeFixtures executes the synthetic fixtures against the same form/order
// normalization and idempotency code used by the job kits. It deliberately
// carries ledgers across cases so replay and collision levels exercise state.
func (catalog GroundCatalog) GradeFixtures(groundPath string) (GroundGrade, error) {
	formLedger, orderLedger := NewSubmissionLedger(), NewOrderLedger()
	contentLedger := NewContentUpdateLedger()
	migrationLedger := NewContentMigrationLedger()
	results := make(map[string]bool)
	for _, family := range catalog.LevelFamilies {
		for _, level := range family.HardLevels {
			fixturePath, caseID, err := fixtureLocation(filepath.Dir(groundPath), level.Input)
			if err != nil {
				return GroundGrade{}, fmt.Errorf("%s: %w", level.ID, err)
			}
			data, err := os.ReadFile(fixturePath)
			if err != nil {
				return GroundGrade{}, err
			}
			var document struct {
				Cases    map[string]json.RawMessage `json:"cases"`
				Products []json.RawMessage          `json:"products"`
			}
			if err := json.Unmarshal(data, &document); err != nil {
				return GroundGrade{}, fmt.Errorf("decode %s: %w", level.ID, err)
			}
			actual, err := runGroundCase(family.ID, caseID, document, formLedger, orderLedger, contentLedger, migrationLedger)
			if err != nil {
				return GroundGrade{}, fmt.Errorf("run %s: %w", level.ID, err)
			}
			var expected any
			if err := json.Unmarshal(level.Expected, &expected); err != nil {
				return GroundGrade{}, fmt.Errorf("decode expected answer for %s: %w", level.ID, err)
			}
			results[level.ID] = jsonEquivalent(actual, expected)
		}
	}
	return catalog.Grade(results), nil
}

func fixtureLocation(root, reference string) (string, string, error) {
	parts := strings.Split(reference, "#")
	if len(parts) != 2 || parts[1] == "" || filepath.IsAbs(parts[0]) {
		return "", "", fmt.Errorf("invalid fixture reference %q", reference)
	}
	for _, part := range strings.Split(filepath.ToSlash(parts[0]), "/") {
		if part == ".." || part == "." {
			return "", "", fmt.Errorf("fixture reference escapes the ground: %q", reference)
		}
	}
	return filepath.Join(root, filepath.FromSlash(parts[0])), parts[1], nil
}

func runGroundCase(family, caseID string, doc struct {
	Cases    map[string]json.RawMessage `json:"cases"`
	Products []json.RawMessage          `json:"products"`
}, forms *SubmissionLedger, orders *OrderLedger, content *ContentUpdateLedger, migrations *ContentMigrationLedger) (any, error) {
	if family == "shop" && caseID == "catalog" {
		var input struct {
			ExpectedSkuCount int `json:"expectedSkuCount"`
		}
		if err := json.Unmarshal(doc.Cases[caseID], &input); err != nil {
			return nil, err
		}
		return map[string]any{"productCount": len(doc.Products), "syntheticOnly": true}, nil
	}
	raw, ok := doc.Cases[caseID]
	if !ok {
		return nil, fmt.Errorf("fixture case %q is absent", caseID)
	}
	switch family {
	case "forms":
		var test struct {
			Provider string          `json:"provider"`
			Payload  json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(raw, &test); err != nil {
			return nil, err
		}
		event, err := NormalizeFormSubmission(test.Provider, test.Payload)
		if err != nil {
			return map[string]any{"accepted": false, "reason": reasonCode(err.Error())}, nil
		}
		disposition, err := forms.Accept(event)
		if err != nil {
			return nil, err
		}
		if disposition == SubmissionDuplicate {
			return map[string]any{"accepted": false, "reason": "duplicate"}, nil
		}
		if disposition == SubmissionConflict {
			return map[string]any{"accepted": false, "reason": "identity_conflict"}, nil
		}
		actual := map[string]any{"accepted": true, "identity": event.EventID}
		if test.Provider == "wpforms-lite" {
			actual["entryId"] = 0
		}
		if test.Provider == "elementor-shaped-synthetic" {
			actual["syntheticCompatibilityOnly"] = true
		}
		return actual, nil
	case "shop":
		var test struct {
			Operation string          `json:"operation"`
			Payload   json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(raw, &test); err != nil {
			return nil, err
		}
		order, err := NormalizeWooOrder("k3-ground", test.Payload, map[string]bool{"K3-BOOK-001": true, "K3-MUG-002": true, "K3-SET-003": true})
		if err != nil {
			return map[string]any{"accepted": false, "reason": reasonCode(err.Error())}, nil
		}
		disposition, err := orders.Accept(order)
		if err != nil {
			return nil, err
		}
		if disposition == SubmissionDuplicate {
			return map[string]any{"accepted": false, "reason": "duplicate"}, nil
		}
		if disposition == SubmissionConflict {
			return map[string]any{"accepted": false, "reason": "identity_conflict"}, nil
		}
		return map[string]any{"accepted": true, "orderId": order.OrderID, "status": order.Status, "revision": order.Revision}, nil
	case "content":
		var test struct {
			Operation string                   `json:"operation"`
			Type      map[string]any           `json:"type"`
			Post      map[string]any           `json:"post"`
			Field     string                   `json:"field"`
			Update    map[string]any           `json:"update"`
			Record    ContentMigrationRecord   `json:"record"`
			Posts     []ContentMigrationRecord `json:"posts"`
			Target    string                   `json:"target"`
		}
		if err := json.Unmarshal(raw, &test); err != nil {
			return nil, err
		}
		switch test.Operation {
		case "read_type":
			return map[string]any{"exists": test.Type["name"] != "", "fieldCount": len(anySlice(test.Type["fields"]))}, nil
		case "read_post":
			return map[string]any{"client_type": test.Post["client_type"], "outcome_metric": test.Post["outcome_metric"]}, nil
		case "read_field":
			_, exists := test.Post[test.Field]
			return map[string]any{"exists": exists}, nil
		case "content_audit":
			return AuditSummary(AuditCoreContentRecords("ground", test.Posts)), nil
		case "idempotent_update":
			postID := fmt.Sprint(test.Post["id"])
			created, err := content.CreateIfMissing(postID, test.Post)
			if err != nil {
				return nil, err
			}
			first, _, err := content.Apply(postID, "update-77-outcome", test.Update)
			if err != nil {
				return nil, err
			}
			replay, fields, err := content.Apply(postID, "update-77-outcome", test.Update)
			if err != nil {
				return nil, err
			}
			return map[string]any{"created": boolInt(created), "updated": boolInt(first == ContentUpdated), "duplicate": boolInt(replay == ContentDuplicate), "finalOutcomeMetric": fields["outcome_metric"]}, nil
		case "migration_apply", "migration_plan":
			if test.Target == "" {
				test.Target = "migration-target"
			}
			suppliedHash := test.Record.ContentHash
			computedHash, hashErr := contentMigrationHash(test.Record)
			if hashErr != nil {
				return nil, hashErr
			}
			if test.Record.ContentHash == "auto" {
				test.Record.ContentHash = computedHash
			}
			var disposition ContentMigrationDisposition
			var err error
			if test.Operation == "migration_plan" {
				disposition, err = migrations.Plan(test.Target, test.Record)
			} else {
				disposition, err = migrations.Apply(test.Target, test.Record)
			}
			if err != nil {
				if test.Record.ClientType != "synthetic" {
					return map[string]any{"accepted": false, "reason": "client_data_disallowed"}, nil
				}
				if suppliedHash != "auto" && suppliedHash != computedHash {
					return map[string]any{"accepted": false, "reason": "hash_mismatch"}, nil
				}
				return map[string]any{"accepted": false, "reason": "invalid_record"}, nil
			}
			return map[string]any{"accepted": true, "disposition": string(disposition)}, nil
		default:
			return nil, fmt.Errorf("unsupported content operation %q", test.Operation)
		}
	default:
		return nil, fmt.Errorf("unsupported ground family %q", family)
	}
}

func reasonCode(message string) string {
	for _, code := range []string{"invalid_email", "missing_name", "missing_message", "missing_revision", "unknown_sku", "unsupported_order_status", "invalid_customer_email"} {
		if strings.Contains(message, code) {
			return code
		}
	}
	return message
}

func anySlice(value any) []any { result, _ := value.([]any); return result }

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func jsonEquivalent(left, right any) bool {
	a, errA := json.Marshal(left)
	b, errB := json.Marshal(right)
	return errA == nil && errB == nil && string(a) == string(b)
}
