package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// ValidationErrors represents a collection of validation errors
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	var messages []string
	for _, err := range ve {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return strings.Join(messages, "; ")
}

// GraphValidator provides validation for graph-related operations
type GraphValidator struct {
	plugins map[string]*Plugin
}

// NewGraphValidator creates a new graph validator
func NewGraphValidator(plugins map[string]*Plugin) *GraphValidator {
	return &GraphValidator{
		plugins: plugins,
	}
}

// ValidateCreateGraphRequest validates a create graph request
func (gv *GraphValidator) ValidateCreateGraphRequest(req *CreateGraphRequest) ValidationErrors {
	var errors ValidationErrors

	// Validate name
	if err := gv.validateGraphName(req.Name); err != nil {
		errors = append(errors, *err)
	}

	// Validate type
	if err := gv.validateGraphType(req.Type); err != nil {
		errors = append(errors, *err)
	}

	// Validate description length
	if err := gv.validateDescription(req.Description); err != nil {
		errors = append(errors, *err)
	}

	// Validate tags
	if err := gv.validateTags(req.Tags); err != nil {
		errors = append(errors, err...)
	}

	// Validate data structure if provided
	if req.Data != nil && len(req.Data) > 0 {
		if err := gv.validateGraphData(req.Type, req.Data); err != nil {
			errors = append(errors, *err)
		}
	}

	// Validate metadata
	if err := gv.validateMetadata(req.Metadata); err != nil {
		errors = append(errors, *err)
	}

	return errors
}

// ValidateUpdateGraphRequest validates an update graph request
func (gv *GraphValidator) ValidateUpdateGraphRequest(req *UpdateGraphRequest) ValidationErrors {
	var errors ValidationErrors

	// Validate name if provided
	if req.Name != "" {
		if err := gv.validateGraphName(req.Name); err != nil {
			errors = append(errors, *err)
		}
	}

	// Validate description if provided
	if req.Description != "" {
		if err := gv.validateDescription(req.Description); err != nil {
			errors = append(errors, *err)
		}
	}

	// Validate tags if provided
	if req.Tags != nil {
		if err := gv.validateTags(req.Tags); err != nil {
			errors = append(errors, err...)
		}
	}

	// Validate metadata if provided
	if req.Metadata != nil {
		if err := gv.validateMetadata(req.Metadata); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

// ValidateConversionRequest validates a conversion request
func (gv *GraphValidator) ValidateConversionRequest(req *ConversionRequest) ValidationErrors {
	var errors ValidationErrors

	// Validate target format
	if req.TargetFormat == "" {
		errors = append(errors, ValidationError{
			Field:   "target_format",
			Message: "Target format is required",
		})
	} else if !gv.isValidPluginID(req.TargetFormat) {
		errors = append(errors, ValidationError{
			Field:   "target_format",
			Message: "Invalid target format",
			Value:   req.TargetFormat,
		})
	}

	// Validate options
	if req.Options != nil {
		if err := gv.validateConversionOptions(req.Options); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

// validateGraphName validates graph name
func (gv *GraphValidator) validateGraphName(name string) *ValidationError {
	if name == "" {
		return &ValidationError{
			Field:   "name",
			Message: "Name is required",
		}
	}

	if len(name) < 3 {
		return &ValidationError{
			Field:   "name",
			Message: "Name must be at least 3 characters long",
			Value:   name,
		}
	}

	if len(name) > 255 {
		return &ValidationError{
			Field:   "name",
			Message: "Name must be less than 255 characters",
			Value:   fmt.Sprintf("%d characters", utf8.RuneCountInString(name)),
		}
	}

	// Check for valid characters (letters, numbers, spaces, hyphens, underscores)
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9\s\-_]+$`, name); !matched {
		return &ValidationError{
			Field:   "name",
			Message: "Name can only contain letters, numbers, spaces, hyphens, and underscores",
			Value:   name,
		}
	}

	return nil
}

// validateGraphType validates graph type
func (gv *GraphValidator) validateGraphType(graphType string) *ValidationError {
	if graphType == "" {
		return &ValidationError{
			Field:   "type",
			Message: "Type is required",
		}
	}

	if !gv.isValidPluginID(graphType) {
		return &ValidationError{
			Field:   "type",
			Message: "Invalid graph type",
			Value:   graphType,
		}
	}

	// Check if plugin is enabled
	if plugin, exists := gv.plugins[graphType]; exists && !plugin.Enabled {
		return &ValidationError{
			Field:   "type",
			Message: "Graph type is disabled",
			Value:   graphType,
		}
	}

	return nil
}

// validateDescription validates description
func (gv *GraphValidator) validateDescription(description string) *ValidationError {
	if len(description) > 2000 {
		return &ValidationError{
			Field:   "description",
			Message: "Description must be less than 2000 characters",
			Value:   fmt.Sprintf("%d characters", utf8.RuneCountInString(description)),
		}
	}

	return nil
}

// validateTags validates tags
func (gv *GraphValidator) validateTags(tags []string) ValidationErrors {
	var errors ValidationErrors

	if len(tags) > 20 {
		errors = append(errors, ValidationError{
			Field:   "tags",
			Message: "Maximum 20 tags allowed",
			Value:   fmt.Sprintf("%d tags", len(tags)),
		})
	}

	seenTags := make(map[string]bool)
	for i, tag := range tags {
		// Check for duplicates
		if seenTags[tag] {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("tags[%d]", i),
				Message: "Duplicate tag",
				Value:   tag,
			})
			continue
		}
		seenTags[tag] = true

		// Validate tag format
		if len(tag) == 0 {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("tags[%d]", i),
				Message: "Tag cannot be empty",
			})
			continue
		}

		if len(tag) > 50 {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("tags[%d]", i),
				Message: "Tag must be less than 50 characters",
				Value:   tag,
			})
			continue
		}

		// Check for valid tag characters
		if matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-_]+$`, tag); !matched {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("tags[%d]", i),
				Message: "Tag can only contain letters, numbers, hyphens, and underscores",
				Value:   tag,
			})
		}
	}

	return errors
}

// validateGraphData validates graph data structure
func (gv *GraphValidator) validateGraphData(graphType string, data json.RawMessage) *ValidationError {
	// Basic JSON validation
	var temp interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return &ValidationError{
			Field:   "data",
			Message: "Invalid JSON format",
			Value:   err.Error(),
		}
	}

	// Size validation (max 10MB)
	if len(data) > 10*1024*1024 {
		return &ValidationError{
			Field:   "data",
			Message: "Graph data too large (max 10MB)",
			Value:   fmt.Sprintf("%d bytes", len(data)),
		}
	}

	// Type-specific validation
	switch graphType {
	case "mind-maps":
		return gv.validateMindMapData(data)
	case "bpmn":
		return gv.validateBPMNData(data)
	case "network-graphs":
		return gv.validateNetworkData(data)
	case "mermaid":
		return gv.validateMermaidData(data)
	}

	return nil
}

// validateMindMapData validates mind map specific data
func (gv *GraphValidator) validateMindMapData(data json.RawMessage) *ValidationError {
	var mindMap map[string]interface{}
	if err := json.Unmarshal(data, &mindMap); err != nil {
		return &ValidationError{
			Field:   "data",
			Message: "Invalid mind map data format",
			Value:   err.Error(),
		}
	}

	// Check for root node
	if _, hasRoot := mindMap["root"]; !hasRoot {
		return &ValidationError{
			Field:   "data.root",
			Message: "Mind map must have a root node",
		}
	}

	return nil
}

// validateBPMNData validates BPMN specific data
func (gv *GraphValidator) validateBPMNData(data json.RawMessage) *ValidationError {
	var bpmn map[string]interface{}
	if err := json.Unmarshal(data, &bpmn); err != nil {
		return &ValidationError{
			Field:   "data",
			Message: "Invalid BPMN data format",
			Value:   err.Error(),
		}
	}

	// Check for nodes
	if _, hasNodes := bpmn["nodes"]; !hasNodes {
		return &ValidationError{
			Field:   "data.nodes",
			Message: "BPMN must have nodes",
		}
	}

	return nil
}

// validateNetworkData validates network graph specific data
func (gv *GraphValidator) validateNetworkData(data json.RawMessage) *ValidationError {
	var network map[string]interface{}
	if err := json.Unmarshal(data, &network); err != nil {
		return &ValidationError{
			Field:   "data",
			Message: "Invalid network data format",
			Value:   err.Error(),
		}
	}

	// Check for nodes
	if _, hasNodes := network["nodes"]; !hasNodes {
		return &ValidationError{
			Field:   "data.nodes",
			Message: "Network graph must have nodes",
		}
	}

	return nil
}

// validateMermaidData validates Mermaid specific data
func (gv *GraphValidator) validateMermaidData(data json.RawMessage) *ValidationError {
	var mermaid map[string]interface{}
	if err := json.Unmarshal(data, &mermaid); err != nil {
		return &ValidationError{
			Field:   "data",
			Message: "Invalid Mermaid data format",
			Value:   err.Error(),
		}
	}

	// Check for diagram or type
	if _, hasDiagram := mermaid["diagram"]; !hasDiagram {
		if _, hasType := mermaid["type"]; !hasType {
			return &ValidationError{
				Field:   "data",
				Message: "Mermaid data must have diagram or type field",
			}
		}
	}

	return nil
}

// validateMetadata validates metadata
func (gv *GraphValidator) validateMetadata(metadata map[string]interface{}) *ValidationError {
	if metadata == nil {
		return nil
	}

	// Serialize to check size
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return &ValidationError{
			Field:   "metadata",
			Message: "Invalid metadata format",
			Value:   err.Error(),
		}
	}

	// Size validation (max 1MB)
	if len(metadataBytes) > 1024*1024 {
		return &ValidationError{
			Field:   "metadata",
			Message: "Metadata too large (max 1MB)",
			Value:   fmt.Sprintf("%d bytes", len(metadataBytes)),
		}
	}

	return nil
}

// validateConversionOptions validates conversion options
func (gv *GraphValidator) validateConversionOptions(options map[string]interface{}) *ValidationError {
	if options == nil {
		return nil
	}

	// Serialize to check size
	optionsBytes, err := json.Marshal(options)
	if err != nil {
		return &ValidationError{
			Field:   "options",
			Message: "Invalid options format",
			Value:   err.Error(),
		}
	}

	// Size validation (max 100KB)
	if len(optionsBytes) > 100*1024 {
		return &ValidationError{
			Field:   "options",
			Message: "Conversion options too large (max 100KB)",
			Value:   fmt.Sprintf("%d bytes", len(optionsBytes)),
		}
	}

	return nil
}

// isValidPluginID checks if a plugin ID is valid
func (gv *GraphValidator) isValidPluginID(pluginID string) bool {
	_, exists := gv.plugins[pluginID]
	return exists
}

// Sanitization functions

// SanitizeGraphName sanitizes a graph name
func SanitizeGraphName(name string) string {
	// Trim whitespace
	name = strings.TrimSpace(name)

	// Replace multiple spaces with single space
	re := regexp.MustCompile(`\s+`)
	name = re.ReplaceAllString(name, " ")

	return name
}

// SanitizeDescription sanitizes a description
func SanitizeDescription(description string) string {
	// Trim whitespace
	description = strings.TrimSpace(description)

	// Remove null bytes
	description = strings.ReplaceAll(description, "\x00", "")

	return description
}

// SanitizeTags sanitizes tags
func SanitizeTags(tags []string) []string {
	var sanitized []string
	seen := make(map[string]bool)

	for _, tag := range tags {
		// Trim and lowercase
		tag = strings.ToLower(strings.TrimSpace(tag))

		// Skip empty or duplicate tags
		if tag == "" || seen[tag] {
			continue
		}

		seen[tag] = true
		sanitized = append(sanitized, tag)
	}

	return sanitized
}

// Rate limiting and security functions

// IsValidUserID checks if a user ID is valid format
func IsValidUserID(userID string) bool {
	if len(userID) < 3 || len(userID) > 255 {
		return false
	}

	// Allow alphanumeric, hyphens, underscores, and @ symbol
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-_@\.]+$`, userID)
	return matched
}

// IsValidGraphID checks if a graph ID is valid UUID format
func IsValidGraphID(graphID string) bool {
	uuidPattern := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`
	matched, _ := regexp.MatchString(uuidPattern, graphID)
	return matched
}

// ValidateFileSize checks file size limits
func ValidateFileSize(size int64, maxSize int64) error {
	if size > maxSize {
		return fmt.Errorf("file size %d bytes exceeds maximum of %d bytes", size, maxSize)
	}
	return nil
}

// Business rule validators

// ValidateGraphOwnership checks if user can access/modify a graph
func ValidateGraphOwnership(userID, createdBy string, isPublic bool) bool {
	// System users can access everything
	if userID == "system" {
		return true
	}

	// Owner can always access
	if userID == createdBy {
		return true
	}

	// Public graphs can be read by anyone
	return isPublic
}

// ValidateConversionCompatibility checks if conversion between types is sensible
func ValidateConversionCompatibility(from, to string) error {
	// Same type conversion is redundant
	if from == to {
		return fmt.Errorf("source and target formats are identical")
	}

	// Define incompatible conversions
	incompatible := map[string][]string{
		// Add specific incompatible combinations here if needed
	}

	if blocked, exists := incompatible[from]; exists {
		for _, blocked := range blocked {
			if blocked == to {
				return fmt.Errorf("conversion from %s to %s is not recommended", from, to)
			}
		}
	}

	return nil
}
