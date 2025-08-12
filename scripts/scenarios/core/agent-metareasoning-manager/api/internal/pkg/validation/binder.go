package validation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// RequestBinder handles binding and validation of HTTP requests
type RequestBinder interface {
	// BindJSON binds JSON request body to a struct and validates it
	BindJSON(r *http.Request, obj interface{}) error
	
	// BindQuery binds query parameters to a struct and validates it
	BindQuery(r *http.Request, obj interface{}) error
	
	// BindAndValidate binds request data and validates the result
	BindAndValidate(r *http.Request, obj interface{}) error
}

// DefaultRequestBinder implements RequestBinder
type DefaultRequestBinder struct {
	validator Validator
}

// NewDefaultRequestBinder creates a new request binder with validation
func NewDefaultRequestBinder(validator Validator) RequestBinder {
	return &DefaultRequestBinder{
		validator: validator,
	}
}

// BindJSON binds JSON request body to a struct and validates it
func (b *DefaultRequestBinder) BindJSON(r *http.Request, obj interface{}) error {
	// Parse JSON body
	if err := json.NewDecoder(r.Body).Decode(obj); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	
	// Validate the bound object
	if err := b.validator.Validate(obj); err != nil {
		return err
	}
	
	return nil
}

// BindQuery binds query parameters to a struct and validates it
func (b *DefaultRequestBinder) BindQuery(r *http.Request, obj interface{}) error {
	// Bind query parameters
	if err := b.bindQueryParams(r.URL.Query(), obj); err != nil {
		return fmt.Errorf("query binding failed: %w", err)
	}
	
	// Validate the bound object
	if err := b.validator.Validate(obj); err != nil {
		return err
	}
	
	return nil
}

// BindAndValidate binds request data based on content type and validates
func (b *DefaultRequestBinder) BindAndValidate(r *http.Request, obj interface{}) error {
	contentType := r.Header.Get("Content-Type")
	
	// Handle JSON requests
	if strings.Contains(contentType, "application/json") {
		return b.BindJSON(r, obj)
	}
	
	// Handle form/query parameters for GET requests or form-encoded POST
	if r.Method == "GET" || strings.Contains(contentType, "application/x-www-form-urlencoded") {
		return b.BindQuery(r, obj)
	}
	
	// Default to query parameter binding
	return b.BindQuery(r, obj)
}

// bindQueryParams binds URL query parameters to a struct
func (b *DefaultRequestBinder) bindQueryParams(values url.Values, obj interface{}) error {
	val := reflect.ValueOf(obj)
	typ := reflect.TypeOf(obj)
	
	// Handle pointers
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return fmt.Errorf("object cannot be nil")
		}
		val = val.Elem()
		typ = typ.Elem()
	}
	
	// Only bind to structs
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("can only bind to struct types")
	}
	
	// Iterate through fields
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		
		// Skip unexported fields
		if !field.CanSet() {
			continue
		}
		
		// Get parameter name from tag or field name
		paramName := b.getParamName(fieldType)
		if paramName == "" {
			continue
		}
		
		// Handle inline structs (for pagination, etc.)
		if paramName == ",inline" && field.Kind() == reflect.Struct {
			if err := b.bindQueryParams(values, field.Addr().Interface()); err != nil {
				return err
			}
			continue
		}
		
		// Get parameter value
		paramValue := values.Get(paramName)
		if paramValue == "" {
			// Check for default value
			defaultValue := fieldType.Tag.Get("default")
			if defaultValue != "" {
				paramValue = defaultValue
			} else {
				continue
			}
		}
		
		// Set field value
		if err := b.setFieldValue(field, paramValue); err != nil {
			return fmt.Errorf("failed to set field %s: %w", fieldType.Name, err)
		}
	}
	
	return nil
}

// getParamName extracts parameter name from struct tag
func (b *DefaultRequestBinder) getParamName(field reflect.StructField) string {
	// Check query tag first
	if tag := field.Tag.Get("query"); tag != "" {
		return tag
	}
	
	// Check json tag
	if tag := field.Tag.Get("json"); tag != "" {
		parts := strings.Split(tag, ",")
		return parts[0]
	}
	
	// Use field name in lowercase
	return strings.ToLower(field.Name)
}

// setFieldValue sets a struct field value from string
func (b *DefaultRequestBinder) setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
		
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %s", value)
		}
		field.SetBool(boolVal)
		
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer value: %s", value)
		}
		field.SetInt(intVal)
		
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value: %s", value)
		}
		field.SetUint(uintVal)
		
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float value: %s", value)
		}
		field.SetFloat(floatVal)
		
	case reflect.Slice:
		return b.setSliceValue(field, value)
		
	default:
		// Handle custom types by checking if they implement TextUnmarshaler
		if field.CanAddr() {
			unmarshaler, ok := field.Addr().Interface().(interface {
				UnmarshalText([]byte) error
			})
			if ok {
				return unmarshaler.UnmarshalText([]byte(value))
			}
		}
		
		// Handle string-based enums
		if field.Kind() == reflect.String || 
		   (field.Kind() == reflect.Interface && field.Type().String() == "interface {}") {
			field.SetString(value)
			return nil
		}
		
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	
	return nil
}

// setSliceValue sets a slice field value from comma-separated string
func (b *DefaultRequestBinder) setSliceValue(field reflect.Value, value string) error {
	if value == "" {
		return nil
	}
	
	parts := strings.Split(value, ",")
	slice := reflect.MakeSlice(field.Type(), len(parts), len(parts))
	
	for i, part := range parts {
		part = strings.TrimSpace(part)
		elem := slice.Index(i)
		
		if err := b.setFieldValue(elem, part); err != nil {
			return fmt.Errorf("failed to set slice element %d: %w", i, err)
		}
	}
	
	field.Set(slice)
	return nil
}