package validation

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Validator provides centralized validation for domain objects
type Validator interface {
	// Validate validates a struct using struct tags
	Validate(obj interface{}) error
	
	// ValidateField validates a single field with rules
	ValidateField(fieldName string, value interface{}, rules string) error
	
	// RegisterCustomValidator adds a custom validation function
	RegisterCustomValidator(name string, fn CustomValidatorFunc)
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// Error implements the error interface
func (ve ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s", ve.Field, ve.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// Error implements the error interface
func (ves ValidationErrors) Error() string {
	if len(ves.Errors) == 0 {
		return "validation failed"
	}
	
	messages := make([]string, len(ves.Errors))
	for i, err := range ves.Errors {
		messages[i] = err.Error()
	}
	
	return strings.Join(messages, "; ")
}

// HasErrors returns true if there are validation errors
func (ves ValidationErrors) HasErrors() bool {
	return len(ves.Errors) > 0
}

// CustomValidatorFunc represents a custom validation function
type CustomValidatorFunc func(value interface{}) error

// DefaultValidator implements the Validator interface
type DefaultValidator struct {
	customValidators map[string]CustomValidatorFunc
}

// NewDefaultValidator creates a new validator with built-in rules
func NewDefaultValidator() Validator {
	v := &DefaultValidator{
		customValidators: make(map[string]CustomValidatorFunc),
	}
	
	// Register built-in custom validators
	v.registerBuiltinValidators()
	
	return v
}

// Validate validates a struct using struct tags
func (v *DefaultValidator) Validate(obj interface{}) error {
	var errors ValidationErrors
	
	val := reflect.ValueOf(obj)
	typ := reflect.TypeOf(obj)
	
	// Handle pointers
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return ValidationError{Field: "root", Message: "object cannot be nil"}
		}
		val = val.Elem()
		typ = typ.Elem()
	}
	
	// Only validate structs
	if val.Kind() != reflect.Struct {
		return ValidationError{Field: "root", Message: "can only validate struct types"}
	}
	
	// Validate each field
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		
		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}
		
		// Get validation rules from tag
		rules := fieldType.Tag.Get("validate")
		if rules == "" {
			continue
		}
		
		// Validate field
		if err := v.ValidateField(fieldType.Name, field.Interface(), rules); err != nil {
			if validationErr, ok := err.(ValidationError); ok {
				errors.Errors = append(errors.Errors, validationErr)
			} else if validationErrs, ok := err.(ValidationErrors); ok {
				errors.Errors = append(errors.Errors, validationErrs.Errors...)
			} else {
				errors.Errors = append(errors.Errors, ValidationError{
					Field:   fieldType.Name,
					Message: err.Error(),
					Value:   field.Interface(),
				})
			}
		}
	}
	
	if errors.HasErrors() {
		return errors
	}
	
	return nil
}

// ValidateField validates a single field with rules
func (v *DefaultValidator) ValidateField(fieldName string, value interface{}, rules string) error {
	var errors ValidationErrors
	
	// Parse rules
	ruleList := strings.Split(rules, ",")
	for _, rule := range ruleList {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		
		// Parse rule and parameters
		parts := strings.Split(rule, "=")
		ruleName := parts[0]
		var ruleParam string
		if len(parts) > 1 {
			ruleParam = parts[1]
		}
		
		// Apply validation rule
		if err := v.applyRule(fieldName, value, ruleName, ruleParam); err != nil {
			errors.Errors = append(errors.Errors, ValidationError{
				Field:   fieldName,
				Message: err.Error(),
				Value:   value,
			})
		}
	}
	
	if errors.HasErrors() {
		return errors
	}
	
	return nil
}

// RegisterCustomValidator adds a custom validation function
func (v *DefaultValidator) RegisterCustomValidator(name string, fn CustomValidatorFunc) {
	v.customValidators[name] = fn
}

// applyRule applies a single validation rule
func (v *DefaultValidator) applyRule(fieldName string, value interface{}, ruleName, ruleParam string) error {
	// Check custom validators first
	if customFn, exists := v.customValidators[ruleName]; exists {
		return customFn(value)
	}
	
	// Built-in validators
	switch ruleName {
	case "required":
		return v.validateRequired(value)
	case "min":
		return v.validateMin(value, ruleParam)
	case "max":
		return v.validateMax(value, ruleParam)
	case "len":
		return v.validateLen(value, ruleParam)
	case "oneof":
		return v.validateOneOf(value, ruleParam)
	case "email":
		return v.validateEmail(value)
	case "url":
		return v.validateURL(value)
	case "json":
		return v.validateJSON(value)
	case "uuid":
		return v.validateUUID(value)
	case "alpha":
		return v.validateAlpha(value)
	case "alphanum":
		return v.validateAlphaNum(value)
	case "numeric":
		return v.validateNumeric(value)
	default:
		return fmt.Errorf("unknown validation rule: %s", ruleName)
	}
}

// Built-in validation functions

func (v *DefaultValidator) validateRequired(value interface{}) error {
	if value == nil {
		return fmt.Errorf("field is required")
	}
	
	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.String:
		if val.String() == "" {
			return fmt.Errorf("field is required")
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if val.Len() == 0 {
			return fmt.Errorf("field is required")
		}
	case reflect.Ptr:
		if val.IsNil() {
			return fmt.Errorf("field is required")
		}
	}
	
	return nil
}

func (v *DefaultValidator) validateMin(value interface{}, param string) error {
	min, err := strconv.Atoi(param)
	if err != nil {
		return fmt.Errorf("invalid min parameter: %s", param)
	}
	
	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.String:
		if val.Len() < min {
			return fmt.Errorf("minimum length is %d", min)
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if val.Len() < min {
			return fmt.Errorf("minimum length is %d", min)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val.Int() < int64(min) {
			return fmt.Errorf("minimum value is %d", min)
		}
	case reflect.Float32, reflect.Float64:
		if val.Float() < float64(min) {
			return fmt.Errorf("minimum value is %d", min)
		}
	default:
		return fmt.Errorf("min validation not supported for type %s", val.Kind())
	}
	
	return nil
}

func (v *DefaultValidator) validateMax(value interface{}, param string) error {
	max, err := strconv.Atoi(param)
	if err != nil {
		return fmt.Errorf("invalid max parameter: %s", param)
	}
	
	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.String:
		if val.Len() > max {
			return fmt.Errorf("maximum length is %d", max)
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if val.Len() > max {
			return fmt.Errorf("maximum length is %d", max)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val.Int() > int64(max) {
			return fmt.Errorf("maximum value is %d", max)
		}
	case reflect.Float32, reflect.Float64:
		if val.Float() > float64(max) {
			return fmt.Errorf("maximum value is %d", max)
		}
	default:
		return fmt.Errorf("max validation not supported for type %s", val.Kind())
	}
	
	return nil
}

func (v *DefaultValidator) validateLen(value interface{}, param string) error {
	expectedLen, err := strconv.Atoi(param)
	if err != nil {
		return fmt.Errorf("invalid len parameter: %s", param)
	}
	
	val := reflect.ValueOf(value)
	var actualLen int
	
	switch val.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		actualLen = val.Len()
	default:
		return fmt.Errorf("len validation not supported for type %s", val.Kind())
	}
	
	if actualLen != expectedLen {
		return fmt.Errorf("length must be exactly %d", expectedLen)
	}
	
	return nil
}

func (v *DefaultValidator) validateOneOf(value interface{}, param string) error {
	options := strings.Split(param, " ")
	valueStr := fmt.Sprintf("%v", value)
	
	for _, option := range options {
		if valueStr == option {
			return nil
		}
	}
	
	return fmt.Errorf("value must be one of: %s", param)
}

func (v *DefaultValidator) validateEmail(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("email validation requires string value")
	}
	
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(str) {
		return fmt.Errorf("invalid email format")
	}
	
	return nil
}

func (v *DefaultValidator) validateURL(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("url validation requires string value")
	}
	
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	if !urlRegex.MatchString(str) {
		return fmt.Errorf("invalid URL format")
	}
	
	return nil
}

func (v *DefaultValidator) validateJSON(value interface{}) error {
	switch v := value.(type) {
	case string:
		var temp interface{}
		if err := json.Unmarshal([]byte(v), &temp); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
	case []byte:
		var temp interface{}
		if err := json.Unmarshal(v, &temp); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
	case json.RawMessage:
		var temp interface{}
		if err := json.Unmarshal(v, &temp); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
	default:
		return fmt.Errorf("json validation requires string, []byte, or json.RawMessage")
	}
	
	return nil
}

func (v *DefaultValidator) validateUUID(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("uuid validation requires string value")
	}
	
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(strings.ToLower(str)) {
		return fmt.Errorf("invalid UUID format")
	}
	
	return nil
}

func (v *DefaultValidator) validateAlpha(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("alpha validation requires string value")
	}
	
	for _, char := range str {
		if !unicode.IsLetter(char) {
			return fmt.Errorf("field must contain only letters")
		}
	}
	
	return nil
}

func (v *DefaultValidator) validateAlphaNum(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("alphanum validation requires string value")
	}
	
	for _, char := range str {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			return fmt.Errorf("field must contain only letters and numbers")
		}
	}
	
	return nil
}

func (v *DefaultValidator) validateNumeric(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("numeric validation requires string value")
	}
	
	for _, char := range str {
		if !unicode.IsDigit(char) && char != '.' && char != '-' && char != '+' {
			return fmt.Errorf("field must be numeric")
		}
	}
	
	return nil
}

// registerBuiltinValidators registers built-in custom validators
func (v *DefaultValidator) registerBuiltinValidators() {
	// Platform validator
	v.RegisterCustomValidator("platform", func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("platform validation requires string value")
		}
		
		validPlatforms := []string{"n8n", "windmill", "both"}
		for _, platform := range validPlatforms {
			if str == platform {
				return nil
			}
		}
		
		return fmt.Errorf("invalid platform: must be one of n8n, windmill, both")
	})
	
	// Execution status validator
	v.RegisterCustomValidator("execution_status", func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("execution_status validation requires string value")
		}
		
		validStatuses := []string{"success", "failed", "running", "pending"}
		for _, status := range validStatuses {
			if str == status {
				return nil
			}
		}
		
		return fmt.Errorf("invalid execution status: must be one of success, failed, running, pending")
	})
}