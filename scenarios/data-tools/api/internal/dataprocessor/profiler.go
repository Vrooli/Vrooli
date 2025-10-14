package dataprocessor

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// ColumnProfile contains statistical information about a column
type ColumnProfile struct {
	ColumnName     string        `json:"column_name"`
	DataType       string        `json:"data_type"`
	TypeConfidence float64       `json:"type_confidence"`
	NullCount      int           `json:"null_count"`
	UniqueCount    int           `json:"unique_count"`
	MinValue       interface{}   `json:"min_value,omitempty"`
	MaxValue       interface{}   `json:"max_value,omitempty"`
	Mean           *float64      `json:"mean,omitempty"`
	Median         *float64      `json:"median,omitempty"`
	StdDev         *float64      `json:"std_dev,omitempty"`
	Percentile25   *float64      `json:"percentile_25,omitempty"`
	Percentile75   *float64      `json:"percentile_75,omitempty"`
	TopValues      []ValueCount  `json:"top_values,omitempty"`
	SampleValues   []interface{} `json:"sample_values,omitempty"`
}

// ValueCount represents a value and its frequency
type ValueCount struct {
	Value interface{} `json:"value"`
	Count int         `json:"count"`
}

// DatasetProfile contains comprehensive profiling information
type DatasetProfile struct {
	RowCount       int             `json:"row_count"`
	ColumnCount    int             `json:"column_count"`
	ColumnProfiles []ColumnProfile `json:"column_profiles"`
	QualityScore   float64         `json:"quality_score"`
	Completeness   float64         `json:"completeness"`
	Duplicates     int             `json:"duplicates"`
}

// Profiler handles data profiling operations
type Profiler struct{}

// NewProfiler creates a new profiler instance
func NewProfiler() *Profiler {
	return &Profiler{}
}

// ProfileDataset generates a comprehensive statistical profile of a dataset
func (p *Profiler) ProfileDataset(data []map[string]interface{}) *DatasetProfile {
	if len(data) == 0 {
		return &DatasetProfile{
			RowCount:       0,
			ColumnCount:    0,
			ColumnProfiles: []ColumnProfile{},
			QualityScore:   0.0,
		}
	}

	// Collect all column names
	columnNames := p.collectColumnNames(data)

	// Profile each column
	profiles := []ColumnProfile{}
	for _, colName := range columnNames {
		profile := p.profileColumn(data, colName)
		profiles = append(profiles, profile)
	}

	// Calculate overall metrics
	completeness := p.calculateCompleteness(data, columnNames)
	duplicates := p.countDuplicates(data)
	qualityScore := p.calculateOverallQuality(completeness, duplicates, len(data))

	return &DatasetProfile{
		RowCount:       len(data),
		ColumnCount:    len(columnNames),
		ColumnProfiles: profiles,
		QualityScore:   qualityScore,
		Completeness:   completeness,
		Duplicates:     duplicates,
	}
}

// profileColumn generates a profile for a single column
func (p *Profiler) profileColumn(data []map[string]interface{}, columnName string) ColumnProfile {
	values := []interface{}{}
	nullCount := 0

	// Collect values
	for _, row := range data {
		val, exists := row[columnName]
		if !exists || val == nil || val == "" {
			nullCount++
			continue
		}
		values = append(values, val)
	}

	// Infer type with confidence
	dataType, confidence := p.inferTypeWithConfidence(values)

	// Count unique values
	uniqueValues := make(map[interface{}]int)
	for _, val := range values {
		uniqueValues[val]++
	}

	// Basic profile
	profile := ColumnProfile{
		ColumnName:     columnName,
		DataType:       dataType,
		TypeConfidence: confidence,
		NullCount:      nullCount,
		UniqueCount:    len(uniqueValues),
	}

	// Add numeric statistics if applicable
	if dataType == "integer" || dataType == "float" {
		p.addNumericStats(&profile, values)
	}

	// Add top values (for all types)
	profile.TopValues = p.getTopValues(uniqueValues, 5)

	// Add sample values
	sampleSize := 5
	if len(values) < sampleSize {
		sampleSize = len(values)
	}
	profile.SampleValues = values[:sampleSize]

	// Add min/max for strings (by length)
	if dataType == "string" {
		p.addStringStats(&profile, values)
	}

	return profile
}

// inferTypeWithConfidence infers the data type and returns a confidence score
func (p *Profiler) inferTypeWithConfidence(values []interface{}) (string, float64) {
	if len(values) == 0 {
		return "unknown", 0.0
	}

	typeCounts := make(map[string]int)

	for _, val := range values {
		switch v := val.(type) {
		case bool:
			typeCounts["boolean"]++
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			typeCounts["integer"]++
		case float32, float64:
			// Check if it's actually an integer value
			if fVal, ok := val.(float64); ok && fVal == float64(int64(fVal)) {
				typeCounts["integer"]++
			} else {
				typeCounts["float"]++
			}
		case string:
			// Try to parse as different types
			str := strings.TrimSpace(v)
			if str == "" {
				continue
			}

			// Try boolean
			if str == "true" || str == "false" || str == "True" || str == "False" {
				typeCounts["boolean"]++
				continue
			}

			// Try numeric
			if p.isInteger(str) {
				typeCounts["integer"]++
				continue
			}

			if p.isFloat(str) {
				typeCounts["float"]++
				continue
			}

			// Check for date/time patterns
			if p.isDateTime(str) {
				typeCounts["datetime"]++
				continue
			}

			typeCounts["string"]++
		default:
			typeCounts["unknown"]++
		}
	}

	// Find the most common type
	maxType := "string"
	maxCount := 0

	for typ, count := range typeCounts {
		if count > maxCount {
			maxCount = count
			maxType = typ
		}
	}

	// Calculate confidence (percentage of values matching the inferred type)
	confidence := float64(maxCount) / float64(len(values))

	return maxType, confidence
}

// addNumericStats adds statistical measures for numeric columns
func (p *Profiler) addNumericStats(profile *ColumnProfile, values []interface{}) {
	nums := []float64{}

	for _, val := range values {
		var num float64
		switch v := val.(type) {
		case int:
			num = float64(v)
		case int64:
			num = float64(v)
		case float64:
			num = v
		case string:
			// Try to parse string as number
			if parsed, ok := p.parseNumber(v); ok {
				num = parsed
			} else {
				continue
			}
		default:
			continue
		}
		nums = append(nums, num)
	}

	if len(nums) == 0 {
		return
	}

	// Sort for median and percentiles
	sorted := make([]float64, len(nums))
	copy(sorted, nums)
	sort.Float64s(sorted)

	// Min and Max
	profile.MinValue = sorted[0]
	profile.MaxValue = sorted[len(sorted)-1]

	// Mean
	sum := 0.0
	for _, num := range nums {
		sum += num
	}
	mean := sum / float64(len(nums))
	profile.Mean = &mean

	// Median
	median := p.calculateMedian(sorted)
	profile.Median = &median

	// Standard Deviation
	variance := 0.0
	for _, num := range nums {
		diff := num - mean
		variance += diff * diff
	}
	variance /= float64(len(nums))
	stdDev := math.Sqrt(variance)
	profile.StdDev = &stdDev

	// Percentiles
	p25 := p.calculatePercentile(sorted, 25)
	p75 := p.calculatePercentile(sorted, 75)
	profile.Percentile25 = &p25
	profile.Percentile75 = &p75
}

// addStringStats adds statistics for string columns
func (p *Profiler) addStringStats(profile *ColumnProfile, values []interface{}) {
	if len(values) == 0 {
		return
	}

	minLen := math.MaxInt32
	maxLen := 0

	for _, val := range values {
		if str, ok := val.(string); ok {
			length := len(str)
			if length < minLen {
				minLen = length
			}
			if length > maxLen {
				maxLen = length
			}
		}
	}

	if minLen != math.MaxInt32 {
		profile.MinValue = minLen
		profile.MaxValue = maxLen
	}
}

// getTopValues returns the most frequent values
func (p *Profiler) getTopValues(valueCounts map[interface{}]int, limit int) []ValueCount {
	// Convert map to slice
	counts := []ValueCount{}
	for val, count := range valueCounts {
		counts = append(counts, ValueCount{Value: val, Count: count})
	}

	// Sort by count descending
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].Count > counts[j].Count
	})

	// Return top N
	if len(counts) < limit {
		return counts
	}
	return counts[:limit]
}

// Helper methods

func (p *Profiler) collectColumnNames(data []map[string]interface{}) []string {
	nameMap := make(map[string]bool)
	for _, row := range data {
		for key := range row {
			nameMap[key] = true
		}
	}

	names := []string{}
	for name := range nameMap {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (p *Profiler) calculateCompleteness(data []map[string]interface{}, columnNames []string) float64 {
	if len(data) == 0 || len(columnNames) == 0 {
		return 0.0
	}

	totalCells := len(data) * len(columnNames)
	nonNullCells := 0

	for _, row := range data {
		for _, colName := range columnNames {
			if val, exists := row[colName]; exists && val != nil && val != "" {
				nonNullCells++
			}
		}
	}

	return float64(nonNullCells) / float64(totalCells)
}

func (p *Profiler) countDuplicates(data []map[string]interface{}) int {
	seen := make(map[string]bool)
	duplicates := 0

	for _, row := range data {
		// Create a simple hash of the row
		hash := p.hashRow(row)
		if seen[hash] {
			duplicates++
		} else {
			seen[hash] = true
		}
	}

	return duplicates
}

func (p *Profiler) hashRow(row map[string]interface{}) string {
	// Simple string concatenation for hashing
	// In production, use a proper hashing algorithm
	parts := []string{}
	for k, v := range row {
		parts = append(parts, k, ":", p.toString(v), "|")
	}
	sort.Strings(parts)
	return strings.Join(parts, "")
}

func (p *Profiler) toString(val interface{}) string {
	if val == nil {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(string([]byte(fmt.Sprint(val)))))
}

func (p *Profiler) calculateOverallQuality(completeness float64, duplicates, total int) float64 {
	// Quality score factors:
	// - Completeness (60% weight)
	// - Uniqueness (40% weight)

	uniqueness := 1.0
	if total > 0 {
		uniqueness = 1.0 - (float64(duplicates) / float64(total))
	}

	return (completeness * 0.6) + (uniqueness * 0.4)
}

func (p *Profiler) calculateMedian(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0.0
	}
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2.0
	}
	return sorted[n/2]
}

func (p *Profiler) calculatePercentile(sorted []float64, percentile int) float64 {
	if len(sorted) == 0 {
		return 0.0
	}
	index := int(float64(len(sorted)-1) * float64(percentile) / 100.0)
	return sorted[index]
}

func (p *Profiler) isInteger(s string) bool {
	_, err := parseInt64(s)
	return err == nil
}

func (p *Profiler) isFloat(s string) bool {
	_, err := parseFloat64(s)
	return err == nil
}

func (p *Profiler) isDateTime(s string) bool {
	// Simple datetime pattern matching
	// In production, use more sophisticated date parsing
	datePatterns := []string{
		"2006-01-02", "01/02/2006", "02-01-2006",
		"2006-01-02 15:04:05", "01/02/2006 15:04:05",
	}

	for _, pattern := range datePatterns {
		if len(s) == len(pattern) {
			// Basic pattern length check
			return true
		}
	}
	return false
}

func (p *Profiler) parseNumber(s string) (float64, bool) {
	if val, err := parseFloat64(strings.TrimSpace(s)); err == nil {
		return val, true
	}
	return 0, false
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func parseFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}
