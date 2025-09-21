package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// RuleImplementationStatus captures whether a rule implementation was loaded successfully.
type RuleImplementationStatus struct {
	Valid   bool   `json:"valid"`
	Error   string `json:"error,omitempty"`
	Details string `json:"details,omitempty"`
}

// ruleExecutor defines the behaviour required to run a rule against supplied content.
type ruleExecutor interface {
	Execute(content string, pathHint string, scenario string) ([]Violation, error)
}

// dynamicGoRuleExecutor executes Go-based rules interpreted at runtime.
type dynamicGoRuleExecutor struct {
	fn       reflect.Value
	ruleID   string
	category string
}

func (e *dynamicGoRuleExecutor) Execute(content string, pathHint string, scenario string) ([]Violation, error) {
	if !e.fn.IsValid() {
		return nil, errors.New("rule function is not available")
	}

	args := []reflect.Value{
		reflect.ValueOf([]byte(content)),
		reflect.ValueOf(pathHint),
		reflect.ValueOf(scenario),
	}

	results := e.fn.Call(args)
	if len(results) == 0 {
		return nil, nil
	}

	var callErr error
	if len(results) > 1 {
		if last := results[1]; last.IsValid() && last.Type().Implements(errorType) {
			if !last.IsNil() {
				callErr = last.Interface().(error)
			}
		}
	}

	if callErr != nil {
		return nil, callErr
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

	symbol := discoverRuleSymbol(file)
	if symbol.FuncName == "" {
		status.Error = "no exported Check function found"
		return nil, status
	}

	pkgName := file.Name.Name

	interpreter := interp.New(interp.Options{})
	interpreter.Use(stdlib.Symbols)

	typesPath := filepath.Join(filepath.Dir(filepath.Dir(rule.FilePath)), "types.go")
	if _, err := os.Stat(typesPath); err == nil {
		if _, err := interpreter.EvalPath(typesPath); err != nil {
			status.Error = fmt.Sprintf("failed to load types.go: %v", err)
			return nil, status
		}
	}

	candidate := filepath.Clean(rule.FilePath)
	if _, err := interpreter.EvalPath(candidate); err != nil {
		status.Error = fmt.Sprintf("failed to load %s: %v", filepath.Base(candidate), err)
		return nil, status
	}

	wrapperName := fmt.Sprintf("__rule_wrapper_%s", rule.ID)
	wrapperCode, err := buildWrapperCode(wrapperName, pkgName, symbol)
	if err != nil {
		status.Error = err.Error()
		return nil, status
	}

	if _, err := interpreter.Eval(wrapperCode); err != nil {
		status.Error = fmt.Sprintf("failed to build wrapper: %v", err)
		return nil, status
	}

	callableValue, err := interpreter.Eval(wrapperName)
	if err != nil {
		status.Error = fmt.Sprintf("failed to resolve wrapper: %v", err)
		return nil, status
	}

	if callableValue.Kind() != reflect.Func {
		status.Error = fmt.Sprintf("wrapper for %s is not callable", symbol.FuncName)
		return nil, status
	}

	status.Valid = true
	status.Details = fmt.Sprintf("Loaded %s.%s", pkgName, symbol.FuncName)

	executor := &dynamicGoRuleExecutor{
		fn:       callableValue,
		ruleID:   rule.ID,
		category: rule.Category,
	}

	return executor, status
}

func buildWrapperCode(wrapperName, pkgName string, symbol ruleSymbol) (string, error) {
	var setup []string
	var callTarget string

	if symbol.ReceiverName != "" {
		constructor := fmt.Sprintf("%s.%s{}", pkgName, symbol.ReceiverName)
		if symbol.PointerReceiver {
			constructor = "&" + constructor
		}
		setup = append(setup, fmt.Sprintf("rule := %s", constructor))
	}

	args := []string{}
	switch symbol.ContentParam {
	case paramBytes:
		args = append(args, "content")
	case paramString:
		args = append(args, "string(content)")
	}
	if symbol.HasPathParam {
		args = append(args, "filepath")
	}
	if symbol.HasScenarioParam {
		args = append(args, "scenario")
	}

	if symbol.ReceiverName != "" {
		callTarget = fmt.Sprintf("rule.%s(%s)", symbol.FuncName, strings.Join(args, ", "))
	} else {
		callTarget = fmt.Sprintf("%s.%s(%s)", pkgName, symbol.FuncName, strings.Join(args, ", "))
	}

	if symbol.ContentParam == paramNone {
		setup = append(setup, "_ = content")
	}
	if !symbol.HasPathParam {
		setup = append(setup, "_ = filepath")
	}
	if !symbol.HasScenarioParam {
		setup = append(setup, "_ = scenario")
	}

	var body []string
	body = append(body, setup...)

	if symbol.ReturnsValue {
		if symbol.ReturnsError {
			body = append(body, fmt.Sprintf("result, err := %s", callTarget))
			body = append(body, "if err != nil {", "return nil, err", "}")
			body = append(body, "return result, nil")
		} else {
			body = append(body, fmt.Sprintf("result := %s", callTarget))
			body = append(body, "return result, nil")
		}
	} else {
		if symbol.ReturnsError {
			body = append(body, fmt.Sprintf("_, err := %s", callTarget))
			body = append(body, "return nil, err")
		} else {
			body = append(body, callTarget)
			body = append(body, "return nil, nil")
		}
	}

	code := fmt.Sprintf("var %s = func(content []byte, filepath string, scenario string) (interface{}, error) {\n%s\n}", wrapperName, strings.Join(body, "\n"))
	return code, nil
}

type ruleSymbol struct {
	FuncName         string
	ReceiverName     string
	PointerReceiver  bool
	ContentParam     paramKind
	HasPathParam     bool
	HasScenarioParam bool
	ReturnsValue     bool
	ReturnsError     bool
}

type paramKind int

const (
	paramNone paramKind = iota
	paramBytes
	paramString
)

func discoverRuleSymbol(file *ast.File) ruleSymbol {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}
		if !fn.Name.IsExported() || !strings.HasPrefix(fn.Name.Name, "Check") {
			continue
		}

		symbol := ruleSymbol{FuncName: fn.Name.Name}

		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			receiver := fn.Recv.List[0]
			switch expr := receiver.Type.(type) {
			case *ast.StarExpr:
				if ident, ok := expr.X.(*ast.Ident); ok {
					symbol.ReceiverName = ident.Name
					symbol.PointerReceiver = true
				}
			case *ast.Ident:
				symbol.ReceiverName = expr.Name
			}
		}

		if fn.Type.Params != nil && len(fn.Type.Params.List) > 0 {
			params := fn.Type.Params.List
			if len(params) >= 1 {
				symbol.ContentParam = classifyParam(params[0])
				if symbol.ContentParam == paramNone && len(params) == 1 {
					// treat single string parameter as path argument
					symbol.HasPathParam = true
				}
			}
			if len(params) >= 2 {
				if classifyParam(params[1]) == paramString {
					symbol.HasPathParam = true
				}
			} else if len(params) == 1 && symbol.ContentParam != paramNone {
				// only content parameter supplied, no filepath
			}
			if len(params) >= 3 {
				if classifyParam(params[2]) == paramString {
					symbol.HasScenarioParam = true
				}
			}
		}

		if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
			symbol.ReturnsValue = true
			results := fn.Type.Results.List
			last := results[len(results)-1]
			if isErrorType(last.Type) {
				symbol.ReturnsError = true
				if len(results) == 1 {
					symbol.ReturnsValue = false
				}
			}
		}

		return symbol
	}
	return ruleSymbol{}
}

func classifyParam(field *ast.Field) paramKind {
	switch expr := field.Type.(type) {
	case *ast.ArrayType:
		if ident, ok := expr.Elt.(*ast.Ident); ok && ident.Name == "byte" {
			return paramBytes
		}
	case *ast.Ident:
		if expr.Name == "string" {
			return paramString
		}
	}
	return paramNone
}

func isErrorType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "error"
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
