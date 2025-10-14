package dataprocessor

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// TransformationType represents the type of transformation
type TransformationType string

const (
	TransformFilter    TransformationType = "filter"
	TransformMap       TransformationType = "map"
	TransformJoin      TransformationType = "join"
	TransformAggregate TransformationType = "aggregate"
	TransformPivot     TransformationType = "pivot"
	TransformSort      TransformationType = "sort"
	TransformSQL       TransformationType = "sql"
)

// Transformation represents a data transformation operation
type Transformation struct {
	Type       TransformationType     `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
	SQL        string                 `json:"sql,omitempty"`
}

// TransformResult contains the result of a transformation
type TransformResult struct {
	Data           []map[string]interface{} `json:"data"`
	Schema         Schema                   `json:"schema"`
	RowCount       int                      `json:"row_count"`
	ExecutionStats ExecutionStats           `json:"execution_stats"`
}

// ExecutionStats contains execution statistics
type ExecutionStats struct {
	ExecutionTimeMs int    `json:"execution_time_ms"`
	RowsProcessed   int    `json:"rows_processed"`
	MemoryUsedBytes int64  `json:"memory_used_bytes"`
	Success         bool   `json:"success"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

// Transformer handles data transformation operations
type Transformer struct{}

// NewTransformer creates a new transformer instance
func NewTransformer() *Transformer {
	return &Transformer{}
}

// Transform applies transformations to data
func (t *Transformer) Transform(data []map[string]interface{}, transformations []Transformation) (*TransformResult, error) {
	startTime := getCurrentTimeMs()
	result := data

	for _, transformation := range transformations {
		var err error
		result, err = t.applyTransformation(result, transformation)
		if err != nil {
			return nil, fmt.Errorf("transformation failed: %w", err)
		}
	}

	schema := t.inferSchema(result)

	return &TransformResult{
		Data:     result,
		Schema:   schema,
		RowCount: len(result),
		ExecutionStats: ExecutionStats{
			ExecutionTimeMs: getCurrentTimeMs() - startTime,
			RowsProcessed:   len(data),
			Success:         true,
		},
	}, nil
}

// applyTransformation applies a single transformation
func (t *Transformer) applyTransformation(data []map[string]interface{}, transformation Transformation) ([]map[string]interface{}, error) {
	switch transformation.Type {
	case TransformFilter:
		return t.applyFilter(data, transformation.Parameters)
	case TransformMap:
		return t.applyMap(data, transformation.Parameters)
	case TransformAggregate:
		return t.applyAggregate(data, transformation.Parameters)
	case TransformPivot:
		return t.applyPivot(data, transformation.Parameters)
	case TransformJoin:
		return t.applyJoin(data, transformation.Parameters)
	case TransformSort:
		return t.applySort(data, transformation.Parameters)
	default:
		return nil, fmt.Errorf("unsupported transformation type: %s", transformation.Type)
	}
}

// applyFilter filters data based on conditions
func (t *Transformer) applyFilter(data []map[string]interface{}, params map[string]interface{}) ([]map[string]interface{}, error) {
	condition, ok := params["condition"].(string)
	if !ok {
		return nil, fmt.Errorf("filter requires 'condition' parameter")
	}

	result := []map[string]interface{}{}

	for _, row := range data {
		if t.evaluateCondition(row, condition) {
			result = append(result, row)
		}
	}

	return result, nil
}

// applyMap transforms each row using a mapping function
func (t *Transformer) applyMap(data []map[string]interface{}, params map[string]interface{}) ([]map[string]interface{}, error) {
	mappings, ok := params["mappings"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("map requires 'mappings' parameter")
	}

	result := make([]map[string]interface{}, len(data))

	for i, row := range data {
		newRow := make(map[string]interface{})

		// Copy existing fields
		for k, v := range row {
			newRow[k] = v
		}

		// Apply mappings
		for targetField, sourceExpr := range mappings {
			if sourceField, ok := sourceExpr.(string); ok {
				if value, exists := row[sourceField]; exists {
					newRow[targetField] = value
				}
			} else {
				// Handle complex expressions
				newRow[targetField] = t.evaluateExpression(row, sourceExpr)
			}
		}

		result[i] = newRow
	}

	return result, nil
}

// applyAggregate performs aggregation operations
func (t *Transformer) applyAggregate(data []map[string]interface{}, params map[string]interface{}) ([]map[string]interface{}, error) {
	groupBy, _ := params["group_by"].([]interface{})
	aggregations, ok := params["aggregations"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("aggregate requires 'aggregations' parameter")
	}

	// Group data
	groups := t.groupData(data, groupBy)

	// Apply aggregations
	result := []map[string]interface{}{}

	for groupKey, groupData := range groups {
		row := make(map[string]interface{})

		// Add group by fields
		if groupBy != nil {
			groupValues := strings.Split(groupKey, "|")
			for i, field := range groupBy {
				if i < len(groupValues) {
					row[field.(string)] = groupValues[i]
				}
			}
		}

		// Apply aggregations
		for targetField, aggConfig := range aggregations {
			if aggMap, ok := aggConfig.(map[string]interface{}); ok {
				aggType := aggMap["type"].(string)
				field := aggMap["field"].(string)

				row[targetField] = t.aggregate(groupData, field, aggType)
			}
		}

		result = append(result, row)
	}

	return result, nil
}

// applyPivot pivots data from rows to columns
func (t *Transformer) applyPivot(data []map[string]interface{}, params map[string]interface{}) ([]map[string]interface{}, error) {
	indexField, ok := params["index"].(string)
	if !ok {
		return nil, fmt.Errorf("pivot requires 'index' parameter")
	}

	columnsField, ok := params["columns"].(string)
	if !ok {
		return nil, fmt.Errorf("pivot requires 'columns' parameter")
	}

	valuesField, ok := params["values"].(string)
	if !ok {
		return nil, fmt.Errorf("pivot requires 'values' parameter")
	}

	// Create pivot table
	pivotMap := make(map[string]map[string]interface{})
	columnValues := make(map[string]bool)

	for _, row := range data {
		indexValue := fmt.Sprintf("%v", row[indexField])
		columnValue := fmt.Sprintf("%v", row[columnsField])
		value := row[valuesField]

		if _, exists := pivotMap[indexValue]; !exists {
			pivotMap[indexValue] = make(map[string]interface{})
			pivotMap[indexValue][indexField] = row[indexField]
		}

		pivotMap[indexValue][columnValue] = value
		columnValues[columnValue] = true
	}

	// Convert to result format
	result := []map[string]interface{}{}
	for _, pivotRow := range pivotMap {
		result = append(result, pivotRow)
	}

	return result, nil
}

// applyJoin joins data with another dataset
func (t *Transformer) applyJoin(data []map[string]interface{}, params map[string]interface{}) ([]map[string]interface{}, error) {
	// Simplified join implementation
	// In production, this would handle different join types and data sources

	joinData, ok := params["join_data"].([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("join requires 'join_data' parameter")
	}

	leftKey, ok := params["left_key"].(string)
	if !ok {
		return nil, fmt.Errorf("join requires 'left_key' parameter")
	}

	rightKey, ok := params["right_key"].(string)
	if !ok {
		return nil, fmt.Errorf("join requires 'right_key' parameter")
	}

	joinType, _ := params["type"].(string)
	if joinType == "" {
		joinType = "inner"
	}

	// Build index for right dataset
	rightIndex := make(map[string][]map[string]interface{})
	for _, row := range joinData {
		key := fmt.Sprintf("%v", row[rightKey])
		rightIndex[key] = append(rightIndex[key], row)
	}

	// Perform join
	result := []map[string]interface{}{}

	for _, leftRow := range data {
		leftKeyValue := fmt.Sprintf("%v", leftRow[leftKey])
		rightRows := rightIndex[leftKeyValue]

		if len(rightRows) > 0 {
			// Join found
			for _, rightRow := range rightRows {
				joinedRow := make(map[string]interface{})

				// Add left row fields
				for k, v := range leftRow {
					joinedRow[k] = v
				}

				// Add right row fields (prefix with table name to avoid conflicts)
				for k, v := range rightRow {
					if k != rightKey {
						joinedRow[k] = v
					}
				}

				result = append(result, joinedRow)
			}
		} else if joinType == "left" {
			// Left join - include left row even without match
			result = append(result, leftRow)
		}
	}

	return result, nil
}

// applySort sorts data by specified fields
func (t *Transformer) applySort(data []map[string]interface{}, params map[string]interface{}) ([]map[string]interface{}, error) {
	field, ok := params["field"].(string)
	if !ok {
		return nil, fmt.Errorf("sort requires 'field' parameter")
	}

	ascending := true
	if order, ok := params["order"].(string); ok {
		ascending = strings.ToLower(order) != "desc"
	}

	// Create a copy of the data to avoid modifying the original
	result := make([]map[string]interface{}, len(data))
	copy(result, data)

	// Sort the data
	sort.Slice(result, func(i, j int) bool {
		vi := result[i][field]
		vj := result[j][field]

		// Handle comparison based on type
		switch vi := vi.(type) {
		case float64:
			if vj, ok := vj.(float64); ok {
				if ascending {
					return vi < vj
				}
				return vi > vj
			}
		case int:
			if vj, ok := vj.(int); ok {
				if ascending {
					return vi < vj
				}
				return vi > vj
			}
		case string:
			if vj, ok := vj.(string); ok {
				if ascending {
					return vi < vj
				}
				return vi > vj
			}
		}

		// Default string comparison
		if ascending {
			return fmt.Sprintf("%v", vi) < fmt.Sprintf("%v", vj)
		}
		return fmt.Sprintf("%v", vi) > fmt.Sprintf("%v", vj)
	})

	return result, nil
}

// groupData groups data by specified fields
func (t *Transformer) groupData(data []map[string]interface{}, groupBy []interface{}) map[string][]map[string]interface{} {
	groups := make(map[string][]map[string]interface{})

	for _, row := range data {
		key := ""
		if groupBy != nil {
			keyParts := []string{}
			for _, field := range groupBy {
				if fieldName, ok := field.(string); ok {
					keyParts = append(keyParts, fmt.Sprintf("%v", row[fieldName]))
				}
			}
			key = strings.Join(keyParts, "|")
		}

		groups[key] = append(groups[key], row)
	}

	return groups
}

// aggregate performs aggregation on grouped data
func (t *Transformer) aggregate(data []map[string]interface{}, field string, aggType string) interface{} {
	switch aggType {
	case "count":
		return len(data)

	case "sum":
		sum := 0.0
		for _, row := range data {
			if val, ok := row[field].(float64); ok {
				sum += val
			} else if val, ok := row[field].(int); ok {
				sum += float64(val)
			}
		}
		return sum

	case "avg":
		sum := 0.0
		count := 0
		for _, row := range data {
			if val, ok := row[field].(float64); ok {
				sum += val
				count++
			} else if val, ok := row[field].(int); ok {
				sum += float64(val)
				count++
			}
		}
		if count > 0 {
			return sum / float64(count)
		}
		return 0.0

	case "min":
		var min interface{}
		for _, row := range data {
			val := row[field]
			if min == nil || t.compare(val, min) < 0 {
				min = val
			}
		}
		return min

	case "max":
		var max interface{}
		for _, row := range data {
			val := row[field]
			if max == nil || t.compare(val, max) > 0 {
				max = val
			}
		}
		return max

	default:
		return nil
	}
}

// evaluateCondition evaluates a filter condition
func (t *Transformer) evaluateCondition(row map[string]interface{}, condition string) bool {
	// Simple condition evaluation
	// In production, use a proper expression parser

	// Handle simple comparisons like "age > 25"
	parts := strings.Fields(condition)
	if len(parts) == 3 {
		field := parts[0]
		operator := parts[1]
		valueStr := parts[2]

		fieldValue := row[field]

		switch operator {
		case ">":
			return t.compareValues(fieldValue, valueStr) > 0
		case ">=":
			return t.compareValues(fieldValue, valueStr) >= 0
		case "<":
			return t.compareValues(fieldValue, valueStr) < 0
		case "<=":
			return t.compareValues(fieldValue, valueStr) <= 0
		case "=", "==":
			return t.compareValues(fieldValue, valueStr) == 0
		case "!=":
			return t.compareValues(fieldValue, valueStr) != 0
		}
	}

	return false
}

// evaluateExpression evaluates a mapping expression
func (t *Transformer) evaluateExpression(row map[string]interface{}, expr interface{}) interface{} {
	// Simple expression evaluation
	// In production, use a proper expression evaluator

	if strExpr, ok := expr.(string); ok {
		// Handle field references
		if val, exists := row[strExpr]; exists {
			return val
		}
	}

	return expr
}

// compareValues compares a field value with a string value
func (t *Transformer) compareValues(fieldValue interface{}, strValue string) int {
	// Convert string value to appropriate type for comparison
	switch v := fieldValue.(type) {
	case int:
		if intVal, err := parseInt(strValue); err == nil {
			return compareInt(v, intVal)
		}
	case float64:
		if floatVal, err := parseFloat(strValue); err == nil {
			return compareFloat(v, floatVal)
		}
	case string:
		return strings.Compare(v, strValue)
	}

	return 0
}

// compare compares two values
func (t *Transformer) compare(a, b interface{}) int {
	// Type-aware comparison
	switch va := a.(type) {
	case int:
		if vb, ok := b.(int); ok {
			return compareInt(va, vb)
		}
	case float64:
		if vb, ok := b.(float64); ok {
			return compareFloat(va, vb)
		}
	case string:
		if vb, ok := b.(string); ok {
			return strings.Compare(va, vb)
		}
	}

	return 0
}

// inferSchema infers the schema from result data
func (t *Transformer) inferSchema(data []map[string]interface{}) Schema {
	if len(data) == 0 {
		return Schema{Columns: []Column{}, RowCount: 0}
	}

	// Collect all unique keys
	keyMap := make(map[string]string)
	for _, row := range data {
		for key, value := range row {
			if _, exists := keyMap[key]; !exists {
				keyMap[key] = t.inferType(value)
			}
		}
	}

	// Create columns
	columns := []Column{}
	for key, dataType := range keyMap {
		columns = append(columns, Column{
			Name:     key,
			Type:     dataType,
			Nullable: false,
		})
	}

	// Sort columns for consistency
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].Name < columns[j].Name
	})

	return Schema{
		Columns:      columns,
		RowCount:     len(data),
		QualityScore: 0.9,
	}
}

// inferType infers the type of a value
func (t *Transformer) inferType(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch reflect.TypeOf(value).Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.String:
		return "string"
	case reflect.Slice:
		return "array"
	case reflect.Map:
		return "object"
	default:
		return "unknown"
	}
}

// Helper functions

func getCurrentTimeMs() int {
	// Simplified - in production use proper time tracking
	return 0
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func compareInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareFloat(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
