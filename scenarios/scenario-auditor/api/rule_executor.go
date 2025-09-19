package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// RuleImplementationStatus captures whether a rule implementation was loaded successfully.
type RuleImplementationStatus struct {
	Valid   bool   `json:"valid"`
	Error   string `json:"error,omitempty"`
	Details string `json:"details,omitempty"`
}

// ruleExecutor defines the behaviour required to run a rule against supplied content.
type ruleExecutor interface {
	Execute(content string, pathHint string) ([]Violation, error)
}

// dynamicGoRuleExecutor executes Go-based rules interpreted at runtime.
type dynamicGoRuleExecutor struct {
	fn       reflect.Value
	ruleID   string
	category string
}

func (e *dynamicGoRuleExecutor) Execute(content string, pathHint string) ([]Violation, error) {
	if !e.fn.IsValid() {
		return nil, errors.New("rule function is not available")
	}

	args := []reflect.Value{
		reflect.ValueOf([]byte(content)),
		reflect.ValueOf(pathHint),
	}

	results := e.fn.Call(args)
	if len(results) == 0 {
		return nil, nil
	}

	return convertInterpreterResult(results[0], e.ruleID, e.category)
}

// compileGoRule attempts to interpret the rule implementation located at filePath.
func compileGoRule(rule *RuleInfo) (ruleExecutor, RuleImplementationStatus) {
	status := RuleImplementationStatus{Valid: false}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rule.FilePath, nil, parser.SkipObjectResolution)
	if err != nil {
		status.Error = fmt.Sprintf("failed to parse Go source: %v", err)
		return nil, status
	}

	fnName := discoverRuleFunction(file)
	if fnName == "" {
		status.Error = "no exported Check* function found"
		return nil, status
	}

	pkgName := file.Name.Name

	interpreter := interp.New(interp.Options{})
	interpreter.Use(stdlib.Symbols)

	dir := filepath.Dir(rule.FilePath)
	matches, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	sort.Strings(matches)

	for _, candidate := range matches {
		if strings.HasSuffix(candidate, "_test.go") {
			continue
		}
		if _, err := interpreter.EvalPath(candidate); err != nil {
			status.Error = fmt.Sprintf("failed to load %s: %v", filepath.Base(candidate), err)
			return nil, status
		}
	}

	symbolName := fmt.Sprintf("%s.%s", pkgName, fnName)
	value, err := interpreter.Eval(symbolName)
	if err != nil {
		status.Error = fmt.Sprintf("failed to resolve %s: %v", symbolName, err)
		return nil, status
	}

	if value.Kind() != reflect.Func {
		status.Error = fmt.Sprintf("%s is not a function", symbolName)
		return nil, status
	}

	status.Valid = true
	status.Details = fmt.Sprintf("Loaded %s.%s", pkgName, fnName)

	executor := &dynamicGoRuleExecutor{
		fn:       value,
		ruleID:   rule.ID,
		category: rule.Category,
	}

	return executor, status
}

func discoverRuleFunction(file *ast.File) string {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}
		if fn.Name.IsExported() && strings.HasPrefix(fn.Name.Name, "Check") {
			return fn.Name.Name
		}
	}
	return ""
}

func convertInterpreterResult(value reflect.Value, ruleID, category string) ([]Violation, error) {
	value = reflectValue(value)

	switch value.Kind() {
	case reflect.Invalid:
		return nil, nil
	case reflect.Slice:
		var violations []Violation
		for i := 0; i < value.Len(); i++ {
			v, err := convertStructViolation(value.Index(i), ruleID, category)
			if err != nil {
				return nil, err
			}
			if v != nil {
				violations = append(violations, *v)
			}
		}
		return violations, nil
	case reflect.Struct:
		v, err := convertStructViolation(value, ruleID, category)
		if err != nil || v == nil {
			return nil, err
		}
		return []Violation{*v}, nil
	case reflect.Ptr:
		if value.IsNil() {
			return nil, nil
		}
		return convertInterpreterResult(value.Elem(), ruleID, category)
	case reflect.Interface:
		if value.IsNil() {
			return nil, nil
		}
		return convertInterpreterResult(value.Elem(), ruleID, category)
	default:
		return nil, fmt.Errorf("unsupported rule return type: %s", value.Kind())
	}
}

func convertStructViolation(value reflect.Value, ruleID, category string) (*Violation, error) {
	value = reflectValue(value)
	if !value.IsValid() {
		return nil, nil
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil, nil
		}
		value = value.Elem()
	}
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil, nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected violation struct, got %s", value.Kind())
	}

	v := Violation{
		RuleID:   ruleID,
		Category: category,
	}

	if field := value.FieldByName("RuleID"); field.IsValid() {
		v.RuleID = toString(field)
	} else if field := value.FieldByName("ID"); field.IsValid() {
		v.RuleID = toString(field)
	}

	if field := value.FieldByName("Severity"); field.IsValid() {
		v.Severity = toString(field)
	}

	if field := value.FieldByName("Message"); field.IsValid() {
		v.Message = toString(field)
	}

	if field := value.FieldByName("Description"); field.IsValid() && v.Message == "" {
		v.Message = toString(field)
	}

	if field := value.FieldByName("Title"); field.IsValid() {
		v.Title = toString(field)
	}

	if field := value.FieldByName("FilePath"); field.IsValid() {
		path := toString(field)
		v.FilePath = path
		if v.File == "" {
			v.File = path
		}
	} else if field := value.FieldByName("File"); field.IsValid() {
		path := toString(field)
		v.File = path
		if v.FilePath == "" {
			v.FilePath = path
		}
	}

	if field := value.FieldByName("LineNumber"); field.IsValid() {
		v.Line = toInt(field)
	} else if field := value.FieldByName("Line"); field.IsValid() {
		v.Line = toInt(field)
	}

	if field := value.FieldByName("CodeSnippet"); field.IsValid() {
		v.CodeSnippet = toString(field)
	}

	if field := value.FieldByName("Recommendation"); field.IsValid() {
		v.Recommendation = toString(field)
	}

	if field := value.FieldByName("Standard"); field.IsValid() {
		v.Standard = toString(field)
	}

	if field := value.FieldByName("Category"); field.IsValid() {
		v.Category = toString(field)
	}

	return &v, nil
}

func reflectValue(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Ptr) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func toString(value reflect.Value) string {
	value = reflectValue(value)
	if !value.IsValid() {
		return ""
	}
	switch value.Kind() {
	case reflect.String:
		return value.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", value.Int())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", value.Float())
	case reflect.Bool:
		if value.Bool() {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", value.Interface())
	}
}

func toInt(value reflect.Value) int {
	value = reflectValue(value)
	if !value.IsValid() {
		return 0
	}
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(value.Uint())
	default:
		return 0
	}
}
