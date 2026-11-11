package rehearsals

// safe_code.go implements a deliberately small interpreter for n8n Code nodes.
// It never compiles or evaluates JavaScript. Source is tokenized, parsed into a
// closed AST, then interpreted with bounded steps, items, strings and output.

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

const (
	maxSafeCodeBytes       = 16 * 1024
	maxSafeCodeSteps       = 20_000
	maxSafeCodeItemBytes   = 64 * 1024
	maxSafeCodeInputBytes  = 1 << 20
	maxSafeCodeOutputBytes = 1 << 20
	maxSafeCodeNodeNames   = 64
	maxSafeCodeEachItems   = 128
)

type (
	codeToken struct{ text, kind string }
	codeExpr  struct {
		op          string
		value       any
		left, right *codeExpr
		args        []*codeExpr
		params      []string
		fields      []string
		body        []codeStmt
	}
)

type codeSwitchCase struct {
	test      *codeExpr
	isDefault bool
	body      []codeStmt
}
type codeStmt struct {
	kind, name, op  string
	params, fields  []string
	expr            *codeExpr
	target          *codeExpr
	cases           []codeSwitchCase
	body, alternate []codeStmt
	init            *codeStmt
	test            *codeExpr
	update          *codeExpr
}
type safeCodeProgram struct{ statements []codeStmt }
type safeLocalFunction struct {
	name   string
	params []string
	body   []codeStmt
}

func parseSafeCode(source string) (*safeCodeProgram, error) {
	return parseSafeCodeMode(source, false, false)
}

func parseSafeLegacyFunction(source string) (*safeCodeProgram, error) {
	return parseSafeCodeMode(source, true, false)
}

func parseSafeCodeMode(source string, legacyFunction, currentJSON bool) (*safeCodeProgram, error) {
	if len(source) == 0 || len(source) > maxSafeCodeBytes {
		return nil, fmt.Errorf("source must be 1..%d bytes", maxSafeCodeBytes)
	}
	tokens, err := lexSafeCode(source)
	if err != nil {
		return nil, err
	}
	if len(tokens) > 4096 {
		return nil, fmt.Errorf("token count exceeds 4096")
	}
	p := codeParser{tokens: tokens, allowPropertyWrites: true}
	_ = currentJSON
	program, err := p.statements("")
	if err != nil {
		return nil, err
	}
	if p.peek().text != "" {
		return nil, fmt.Errorf("unexpected token %q", p.peek().text)
	}
	lastExecutable := -1
	for i, statement := range program {
		if statement.kind != "function" {
			lastExecutable = i
		}
	}
	if lastExecutable < 0 || program[lastExecutable].kind != "return" {
		return nil, fmt.Errorf("program must end with return before optional function declarations")
	}
	scope := map[string]bool{"$input": false, "$json": false, "$now": false, "$execution": false, "$": false, "$items": false, "String": false, "Number": false, "Array": false, "Math": false, "DateTime": false, "JSON": false, "Object": false, "parseInt": false}
	_ = currentJSON // $json is bounded to the current/first input item in either mode.
	if legacyFunction {
		scope["__legacy_function__"] = true
	}
	if err := validateSafeStatements(program, scope); err != nil {
		return nil, err
	}
	return &safeCodeProgram{statements: program}, nil
}

func lexSafeCode(source string) ([]codeToken, error) {
	if strings.ContainsAny(source, "\x00") {
		return nil, fmt.Errorf("NUL byte is forbidden")
	}
	var out []codeToken
	for i := 0; i < len(source); {
		c := source[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			i++
			continue
		}
		if c == '/' && i+1 < len(source) && source[i+1] == '/' {
			for i < len(source) && source[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(source) && source[i+1] == '*' {
			end := strings.Index(source[i+2:], "*/")
			if end < 0 {
				return nil, fmt.Errorf("unterminated block comment")
			}
			i += end + 4
			continue
		}
		if c == '`' {
			end := safeTemplateLiteralEnd(source, i+1)
			if end < 0 {
				return nil, fmt.Errorf("unterminated or unsupported template literal")
			}
			out = append(out, codeToken{source[i+1 : end], "template"})
			i = end + 1
			continue
		}
		if c == '/' { // Regexes are an email predicate or one HTML-escape character.
			end := safeRegexLiteralEnd(source, i+1)
			if end < 0 {
				return nil, fmt.Errorf("unterminated or unsupported regex")
			}
			pattern := source[i+1 : end]
			j := end + 1
			for j < len(source) && ((source[j] >= 'a' && source[j] <= 'z') || (source[j] >= 'A' && source[j] <= 'Z')) {
				j++
			}
			flags := source[end+1 : j]
			if !safeCodeRegex(pattern, flags) {
				return nil, fmt.Errorf("regex literal is outside the safe email or HTML-escaping subset")
			}
			out = append(out, codeToken{pattern + "\x00" + flags, "regex"})
			i = j
			continue
		}
		if c == '\'' || c == '"' {
			quote := c
			j := i + 1
			var b strings.Builder
			for j < len(source) && source[j] != quote {
				if source[j] == '\\' {
					j++
					if j >= len(source) {
						break
					}
					switch source[j] {
					case 'n':
						b.WriteByte('\n')
					case 'r':
						b.WriteByte('\r')
					case 't':
						b.WriteByte('\t')
					default:
						b.WriteByte(source[j])
					}
				} else {
					b.WriteByte(source[j])
				}
				j++
			}
			if j >= len(source) {
				return nil, fmt.Errorf("unterminated string")
			}
			out = append(out, codeToken{b.String(), "string"})
			i = j + 1
			continue
		}
		if isIdentStart(c) {
			j := i + 1
			for j < len(source) && isIdent(source[j]) {
				j++
			}
			out = append(out, codeToken{source[i:j], "ident"})
			i = j
			continue
		}
		if c >= '0' && c <= '9' {
			j := i + 1
			for j < len(source) && source[j] >= '0' && source[j] <= '9' {
				j++
			}
			n, _ := strconv.Atoi(source[i:j])
			out = append(out, codeToken{strconv.Itoa(n), "number"})
			i = j
			continue
		}
		matched := ""
		for _, op := range []string{"===", "!==", "...", "??", "||", "&&", "?.", "+=", "=>", "++", "<=", ">=", "==", "!="} {
			if strings.HasPrefix(source[i:], op) {
				matched = op
				break
			}
		}
		if matched != "" {
			out = append(out, codeToken{matched, "punct"})
			i += len(matched)
			continue
		}
		if strings.ContainsRune("{}[]().,;:=!<>+-*?", rune(c)) {
			out = append(out, codeToken{string(c), "punct"})
			i++
			continue
		}
		return nil, fmt.Errorf("character %q is unsupported", c)
	}
	return out, nil
}

func safeTemplateLiteralEnd(source string, start int) int {
	for index := start; index < len(source); index++ {
		if source[index] == '\\' {
			index++
			continue
		}
		if source[index] == '`' {
			return index
		}
	}
	return -1
}

func parseSafeTemplate(source string) (*codeExpr, error) {
	parts := make([]string, 0, 2)
	values := make([]*codeExpr, 0, 1)
	for offset := 0; offset < len(source); {
		start := offset
		for offset < len(source) {
			if offset+1 < len(source) && source[offset] == '$' && source[offset+1] == '{' {
				break
			}
			if source[offset] == '\\' && offset+1 < len(source) {
				offset++
			}
			offset++
		}
		parts = append(parts, decodeSafeTemplateText(source[start:offset]))
		if offset+1 >= len(source) {
			break
		}
		expressionStart := offset + 2
		end, err := safeTemplateExpressionEnd(source, expressionStart)
		if err != nil {
			return nil, err
		}
		tokens, err := lexSafeCode(source[expressionStart:end])
		if err != nil {
			return nil, err
		}
		parser := codeParser{tokens: tokens}
		value, err := parser.expr(0)
		if err != nil {
			return nil, fmt.Errorf("template interpolation: %w", err)
		}
		if parser.peek().text != "" {
			return nil, fmt.Errorf("template interpolation contains trailing token %q", parser.peek().text)
		}
		values = append(values, value)
		offset = end + 1
	}
	if len(parts) == 0 {
		parts = append(parts, "")
	}
	if len(parts) == len(values) {
		parts = append(parts, "")
	}
	return &codeExpr{op: "template", value: parts, args: values}, nil
}

func safeTemplateExpressionEnd(source string, start int) (int, error) {
	depth := 0
	for index := start; index < len(source); index++ {
		switch source[index] {
		case '\'', '"':
			quote := source[index]
			for index++; index < len(source); index++ {
				if source[index] == '\\' {
					index++
					continue
				}
				if source[index] == quote {
					break
				}
			}
		case '{':
			depth++
		case '}':
			if depth == 0 {
				return index, nil
			}
			depth--
		}
	}
	return 0, fmt.Errorf("unterminated template interpolation")
}

func decodeSafeTemplateText(value string) string {
	var decoded strings.Builder
	for index := 0; index < len(value); index++ {
		if value[index] != '\\' || index+1 >= len(value) {
			decoded.WriteByte(value[index])
			continue
		}
		index++
		switch value[index] {
		case 'n':
			decoded.WriteByte('\n')
		case 'r':
			decoded.WriteByte('\r')
		case 't':
			decoded.WriteByte('\t')
		default:
			decoded.WriteByte(value[index])
		}
	}
	return decoded.String()
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}
func isIdent(c byte) bool { return isIdentStart(c) || c >= '0' && c <= '9' }

type codeParser struct {
	tokens              []codeToken
	pos                 int
	depth               int
	allowPropertyWrites bool
}

func (p *codeParser) peek() codeToken {
	if p.pos >= len(p.tokens) {
		return codeToken{}
	}
	return p.tokens[p.pos]
}

func (p *codeParser) take(s string) bool {
	if p.peek().text == s {
		p.pos++
		return true
	}
	return false
}

func (p *codeParser) need(s string) error {
	if !p.take(s) {
		return fmt.Errorf("expected %q near %q", s, p.peek().text)
	}
	return nil
}

func (p *codeParser) statements(stop string) ([]codeStmt, error) {
	var out []codeStmt
	for p.peek().text != "" && p.peek().text != stop {
		s, e := p.statement()
		if e != nil {
			return nil, e
		}
		out = append(out, s)
		if len(out) > 256 {
			return nil, fmt.Errorf("too many statements")
		}
		p.take(";")
	}
	if stop != "" {
		if err := p.need(stop); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (p *codeParser) statement() (codeStmt, error) {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 64 {
		return codeStmt{}, fmt.Errorf("statement nesting exceeds 64")
	}
	if p.take("function") {
		name := p.peek().text
		if p.peek().kind != "ident" || !safeCodeIdentifier(name) {
			return codeStmt{}, fmt.Errorf("function name must be a bounded local identifier")
		}
		p.pos++
		if err := p.need("("); err != nil {
			return codeStmt{}, err
		}
		var params []string
		if !p.take(")") {
			for {
				param := p.peek().text
				if p.peek().kind != "ident" || !safeCodeIdentifier(param) {
					return codeStmt{}, fmt.Errorf("function parameter must be a bounded local identifier")
				}
				p.pos++
				params = append(params, param)
				if p.take(")") {
					break
				}
				if err := p.need(","); err != nil {
					return codeStmt{}, err
				}
			}
		}
		body, err := p.block()
		return codeStmt{kind: "function", name: name, params: params, body: body}, err
	}
	if p.take("const") {
		if p.take("{") {
			var fields, names []string
			if !p.take("}") {
				for {
					key := p.peek()
					if key.kind != "ident" || !safeCodePropertyName(key.text) {
						return codeStmt{}, fmt.Errorf("object destructuring requires safe static property names")
					}
					p.pos++
					name := key.text
					if p.take(":") {
						name = p.peek().text
						if p.peek().kind != "ident" || !safeCodeIdentifier(name) {
							return codeStmt{}, fmt.Errorf("object destructuring alias must be a bounded local identifier")
						}
						p.pos++
					}
					fields = append(fields, key.text)
					names = append(names, name)
					if p.take("}") {
						break
					}
					if err := p.need(","); err != nil {
						return codeStmt{}, err
					}
				}
			}
			if len(names) == 0 || len(names) > 64 {
				return codeStmt{}, fmt.Errorf("object destructuring requires 1..64 static fields")
			}
			if err := p.need("="); err != nil {
				return codeStmt{}, err
			}
			expr, err := p.expr(0)
			return codeStmt{kind: "objectdecl", params: names, fields: fields, expr: expr}, err
		}
		name := p.peek().text
		if p.peek().kind != "ident" {
			return codeStmt{}, fmt.Errorf("expected local variable")
		}
		p.pos++
		if err := p.need("="); err != nil {
			return codeStmt{}, err
		}
		e, err := p.expr(0)
		return codeStmt{kind: "decl", name: name, expr: e}, err
	}
	if p.take("let") {
		name := p.peek().text
		if p.peek().kind != "ident" {
			return codeStmt{}, fmt.Errorf("expected local variable")
		}
		p.pos++
		if p.take("=") {
			e, err := p.expr(0)
			return codeStmt{kind: "letdecl", name: name, expr: e}, err
		}
		return codeStmt{kind: "letdecl", name: name}, nil
	}
	if p.take("var") {
		name := p.peek().text
		if p.peek().kind != "ident" {
			return codeStmt{}, fmt.Errorf("expected local variable")
		}
		p.pos++
		if p.take("=") {
			e, err := p.expr(0)
			return codeStmt{kind: "letdecl", name: name, expr: e}, err
		}
		return codeStmt{kind: "letdecl", name: name}, nil
	}
	if p.take("switch") {
		if err := p.need("("); err != nil {
			return codeStmt{}, err
		}
		test, err := p.expr(0)
		if err != nil {
			return codeStmt{}, err
		}
		if err = p.need(")"); err != nil {
			return codeStmt{}, err
		}
		if err = p.need("{"); err != nil {
			return codeStmt{}, err
		}
		var cases []codeSwitchCase
		for p.peek().text != "}" && p.peek().text != "" {
			item := codeSwitchCase{}
			if p.take("case") {
				item.test, err = p.expr(0)
				if err != nil {
					return codeStmt{}, err
				}
				if item.test.op != "lit" {
					return codeStmt{}, fmt.Errorf("switch labels must be static literals")
				}
				if err = p.need(":"); err != nil {
					return codeStmt{}, err
				}
			} else if p.take("default") {
				item.isDefault = true
				if err = p.need(":"); err != nil {
					return codeStmt{}, err
				}
			} else {
				return codeStmt{}, fmt.Errorf("switch body requires case or default")
			}
			for p.peek().text != "case" && p.peek().text != "default" && p.peek().text != "}" && p.peek().text != "" {
				stmt, stmtErr := p.statement()
				if stmtErr != nil {
					return codeStmt{}, stmtErr
				}
				item.body = append(item.body, stmt)
				p.take(";")
			}
			cases = append(cases, item)
			if len(cases) > 64 {
				return codeStmt{}, fmt.Errorf("switch case bound exceeded")
			}
		}
		if err = p.need("}"); err != nil {
			return codeStmt{}, err
		}
		return codeStmt{kind: "switch", expr: test, cases: cases}, nil
	}
	if p.take("break") {
		return codeStmt{kind: "break"}, nil
	}
	if p.take("if") {
		if err := p.need("("); err != nil {
			return codeStmt{}, err
		}
		test, err := p.expr(0)
		if err != nil {
			return codeStmt{}, err
		}
		if err = p.need(")"); err != nil {
			return codeStmt{}, err
		}
		body, err := p.blockOrStatement()
		if err != nil {
			return codeStmt{}, err
		}
		var alt []codeStmt
		if p.take("else") {
			alt, err = p.blockOrStatement()
		}
		return codeStmt{kind: "if", test: test, body: body, alternate: alt}, err
	}
	if p.take("for") {
		if err := p.need("("); err != nil {
			return codeStmt{}, err
		}
		if p.take("let") {
			name := p.peek().text
			if p.peek().kind != "ident" {
				return codeStmt{}, fmt.Errorf("expected loop variable")
			}
			p.pos++
			if err := p.need("="); err != nil {
				return codeStmt{}, err
			}
			initial, err := p.expr(0)
			if err != nil {
				return codeStmt{}, err
			}
			if err = p.need(";"); err != nil {
				return codeStmt{}, err
			}
			test, err := p.expr(0)
			if err != nil {
				return codeStmt{}, err
			}
			if err = p.need(";"); err != nil {
				return codeStmt{}, err
			}
			if p.peek().kind != "ident" {
				return codeStmt{}, fmt.Errorf("loop update must increment its local index")
			}
			updateName := p.peek().text
			p.pos++
			if updateName != name {
				return codeStmt{}, fmt.Errorf("bounded for-loop update must increment its own index")
			}
			if err := p.need("++"); err != nil {
				return codeStmt{}, err
			}
			update := &codeExpr{op: "post++", value: updateName}
			if err = p.need(")"); err != nil {
				return codeStmt{}, err
			}
			body, err := p.block()
			return codeStmt{kind: "for", name: name, init: &codeStmt{kind: "letdecl", name: name, expr: initial}, test: test, update: update, body: body}, err
		}
		if p.take("const") || p.take("var") {
			name := p.peek().text
			if p.peek().kind != "ident" {
				return codeStmt{}, fmt.Errorf("expected loop variable")
			}
			p.pos++
			if err := p.need("of"); err != nil {
				return codeStmt{}, err
			}
			collection, err := p.expr(0)
			if err != nil {
				return codeStmt{}, err
			}
			if err = p.need(")"); err != nil {
				return codeStmt{}, err
			}
			body, err := p.block()
			return codeStmt{kind: "forof", name: name, expr: collection, body: body}, err
		}
		name := p.peek().text
		if p.peek().kind != "ident" {
			return codeStmt{}, fmt.Errorf("expected loop variable")
		}
		p.pos++
		if err := p.need("of"); err != nil {
			return codeStmt{}, err
		}
		collection, err := p.expr(0)
		if err != nil {
			return codeStmt{}, err
		}
		if err = p.need(")"); err != nil {
			return codeStmt{}, err
		}
		body, err := p.block()
		return codeStmt{kind: "forof", name: name, expr: collection, body: body}, err
	}
	if p.take("continue") {
		return codeStmt{kind: "continue"}, nil
	}
	if p.take("try") {
		body, err := p.block()
		if err != nil {
			return codeStmt{}, err
		}
		if err = p.need("catch"); err != nil {
			return codeStmt{}, err
		}
		if p.take("(") {
			if p.peek().kind != "ident" {
				return codeStmt{}, fmt.Errorf("expected catch binding")
			}
			p.pos++
			if err = p.need(")"); err != nil {
				return codeStmt{}, err
			}
		}
		alternate, err := p.block()
		return codeStmt{kind: "try", body: body, alternate: alternate}, err
	}
	if p.take("return") {
		e, err := p.expr(0)
		return codeStmt{kind: "return", expr: e}, err
	}
	e, err := p.expr(0)
	if err != nil {
		return codeStmt{}, err
	}
	if e.op == "var" && (p.take("=") || p.take("+=")) {
		operator := p.tokens[p.pos-1].text
		if operator == "=" {
			names := []string{e.value.(string)}
			for p.peek().kind == "ident" && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].text == "=" {
				names = append(names, p.peek().text)
				p.pos += 2
				if len(names) > 16 {
					return codeStmt{}, fmt.Errorf("chained assignment exceeds 16 local bindings")
				}
			}
			if len(names) > 1 {
				value, valueErr := p.expr(0)
				return codeStmt{kind: "assignchain", params: names, expr: value}, valueErr
			}
		}
		value, valueErr := p.expr(0)
		return codeStmt{kind: "assign", name: e.value.(string), expr: value, op: operator}, valueErr
	}
	if (e.op == "prop" || e.op == "index") && p.take("=") {
		if !p.allowPropertyWrites {
			return codeStmt{}, fmt.Errorf("object property assignment is supported only in bounded legacy Function mode")
		}
		value, valueErr := p.expr(0)
		return codeStmt{kind: "setprop", target: e, expr: value}, valueErr
	}
	return codeStmt{kind: "expr", expr: e}, nil
}

func (p *codeParser) blockOrStatement() ([]codeStmt, error) {
	if p.take("{") {
		return p.statements("}")
	}
	s, e := p.statement()
	if e != nil {
		return nil, e
	}
	p.take(";")
	return []codeStmt{s}, nil
}

func (p *codeParser) block() ([]codeStmt, error) {
	if err := p.need("{"); err != nil {
		return nil, err
	}
	return p.statements("}")
}

func validateSafeStatements(statements []codeStmt, scope map[string]bool) error {
	for _, statement := range statements {
		if statement.kind == "function" {
			if _, exists := scope[statement.name]; exists {
				return fmt.Errorf("redeclaration of %q is unsupported", statement.name)
			}
			scope[statement.name] = false
		}
	}
	for _, statement := range statements {
		switch statement.kind {
		case "function":
			inner := cloneBoolMap(scope)
			for _, param := range statement.params {
				if _, exists := inner[param]; exists {
					return fmt.Errorf("duplicate or shadowed function parameter %q", param)
				}
				inner[param] = false
			}
			if err := validateSafeStatements(statement.body, inner); err != nil {
				return err
			}
		case "decl":
			if _, exists := scope[statement.name]; exists {
				return fmt.Errorf("redeclaration of %q is unsupported", statement.name)
			}
			if statement.expr.op == "lambda" {
				inner := cloneBoolMap(scope)
				inner[statement.expr.value.(string)] = false
				if statement.expr.left != nil {
					if err := validateSafeExpr(statement.expr.left, inner); err != nil {
						return err
					}
				} else if err := validateSafeStatements(statement.expr.body, inner); err != nil {
					return err
				}
			} else if err := validateSafeExpr(statement.expr, scope); err != nil {
				return err
			}
			scope[statement.name] = false
		case "letdecl":
			if _, exists := scope[statement.name]; exists {
				return fmt.Errorf("redeclaration of %q is unsupported", statement.name)
			}
			if statement.expr != nil {
				if err := validateSafeExpr(statement.expr, scope); err != nil {
					return err
				}
			}
			scope[statement.name] = true
		case "objectdecl":
			if err := validateSafeExpr(statement.expr, scope); err != nil {
				return err
			}
			if len(statement.params) == 0 || len(statement.params) != len(statement.fields) {
				return fmt.Errorf("object destructuring bindings are invalid")
			}
			for _, name := range statement.params {
				if _, exists := scope[name]; exists {
					return fmt.Errorf("redeclaration of %q is unsupported", name)
				}
				scope[name] = false
			}
		case "assign":
			mutable, exists := scope[statement.name]
			if !exists && !scope["__legacy_function__"] {
				return fmt.Errorf("assignment target %q must be declared with let", statement.name)
			}
			if exists && !mutable {
				return fmt.Errorf("assignment requires a declared let variable")
			}
			if err := validateSafeExpr(statement.expr, scope); err != nil {
				return err
			}
			// n8n's legacy Function node examples often use an undeclared
			// temporary. Treat it as an isolated function-local binding; it can
			// never write a process or host global.
			scope[statement.name] = true
		case "assignchain":
			for _, name := range statement.params {
				mutable, exists := scope[name]
				if !exists || !mutable {
					return fmt.Errorf("chained assignment requires declared mutable local %q", name)
				}
			}
			if err := validateSafeExpr(statement.expr, scope); err != nil {
				return err
			}
		case "setprop":
			if statement.target == nil {
				return fmt.Errorf("only static object property assignment is supported")
			}
			root, ok := rootSafeVariable(statement.target)
			if !ok || strings.HasPrefix(root, "$") || !scopeHas(scope, root) {
				return fmt.Errorf("property assignment requires a local object binding")
			}
			if statement.target.op == "prop" {
				if err := validateSafeExpr(statement.target, scope); err != nil {
					return err
				}
			} else if statement.target.op == "index" && statement.target.right != nil {
				if err := validateSafeExpr(statement.target.left, scope); err != nil {
					return err
				}
				if err := validateSafeExpr(statement.target.right, scope); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("only static object property assignment is supported")
			}
			if err := validateSafeExpr(statement.expr, scope); err != nil {
				return err
			}
		case "expr", "return":
			if err := validateSafeExpr(statement.expr, scope); err != nil {
				return err
			}
		case "if":
			if err := validateSafeExpr(statement.test, scope); err != nil {
				return err
			}
			if err := validateSafeStatements(statement.body, cloneBoolMap(scope)); err != nil {
				return err
			}
			if err := validateSafeStatements(statement.alternate, cloneBoolMap(scope)); err != nil {
				return err
			}
		case "forof":
			if err := validateSafeExpr(statement.expr, scope); err != nil {
				return err
			}
			inner := cloneBoolMap(scope)
			inner[statement.name] = false
			if err := validateSafeStatements(statement.body, inner); err != nil {
				return err
			}
		case "for":
			inner := cloneBoolMap(scope)
			if statement.init == nil || statement.init.kind != "letdecl" {
				return fmt.Errorf("bounded for loop requires a let index")
			}
			if err := validateSafeExpr(statement.init.expr, scope); err != nil {
				return err
			}
			inner[statement.name] = true
			if err := validateSafeExpr(statement.test, inner); err != nil {
				return err
			}
			if err := validateSafeExpr(statement.update, inner); err != nil {
				return err
			}
			if err := validateSafeStatements(statement.body, inner); err != nil {
				return err
			}
		case "switch":
			if err := validateSafeExpr(statement.expr, scope); err != nil {
				return err
			}
			for _, branch := range statement.cases {
				if branch.test != nil {
					if err := validateSafeExpr(branch.test, scope); err != nil {
						return err
					}
				}
				if err := validateSafeStatements(branch.body, cloneBoolMap(scope)); err != nil {
					return err
				}
			}
		case "break":
		case "continue":
			// Runtime only accepts this inside bounded loops.
		case "try":
			if err := validateSafeStatements(statement.body, cloneBoolMap(scope)); err != nil {
				return err
			}
			if err := validateSafeStatements(statement.alternate, cloneBoolMap(scope)); err != nil {
				return err
			}
		default:
			return fmt.Errorf("statement %q is not allowed", statement.kind)
		}
	}
	return nil
}

func rootSafeVariable(expr *codeExpr) (string, bool) {
	for expr != nil && (expr.op == "prop" || expr.op == "index" || expr.op == "optprop" || expr.op == "optindex") {
		expr = expr.left
	}
	if expr == nil || expr.op != "var" {
		return "", false
	}
	name, ok := expr.value.(string)
	return name, ok
}

func cloneBoolMap(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func validateSafeExpr(expr *codeExpr, scope map[string]bool) error {
	if expr == nil {
		return fmt.Errorf("missing expression")
	}
	switch expr.op {
	case "lit":
		return nil
	case "regex":
		if regex, ok := expr.value.(safeRegex); ok && safeCodeRegex(regex.pattern, regex.flags()) {
			return nil
		}
		return fmt.Errorf("regex outside allowlist")
	case "template":
		for _, child := range expr.args {
			if err := validateSafeExpr(child, scope); err != nil {
				return err
			}
		}
		return nil
	case "date":
		if expr.left != nil {
			return validateSafeExpr(expr.left, scope)
		}
		return nil
	case "set":
		if expr.left != nil {
			return validateSafeExpr(expr.left, scope)
		}
		return nil
	case "var":
		if expr.value == "__legacy_function__" {
			return fmt.Errorf("identifier %q is not allowed", expr.value)
		}
		if _, exists := scope[expr.value.(string)]; exists {
			return nil
		}
		return fmt.Errorf("identifier %q is not allowed", expr.value)
	case "post++":
		name, _ := expr.value.(string)
		if mutable, exists := scope[name]; exists && mutable {
			return nil
		}
		return fmt.Errorf("loop increment requires a mutable local index")
	case "array":
		for _, child := range expr.args {
			if err := validateSafeExpr(child, scope); err != nil {
				return err
			}
		}
		return nil
	case "object":
		for _, child := range expr.args {
			if err := validateSafeExpr(child, scope); err != nil {
				return err
			}
		}
		return nil
	case "spread":
		return validateSafeExpr(expr.left, scope)
	case "prop":
		if err := validateSafeExpr(expr.left, scope); err != nil {
			return err
		}
		if !safeCodePropertyName(expr.value.(string)) {
			return fmt.Errorf("property %q is not allowed", expr.value)
		}
		return nil
	case "call":
		if err := validateSafeExpr(expr.left, scope); err != nil {
			return err
		}
		switch expr.value {
		case "all", "first", "toISOString", "toISO", "getDay", "getDate", "getMonth", "getFullYear", "test", "trim", "toLowerCase", "replace", "match", "slice", "split", "push", "join", "map", "filter", "forEach", "includes", "indexOf", "has", "concat", "from", "isArray", "fromCodePoint", "floor", "round", "random", "fromISO", "diff", "toFormat", "stringify", "parse", "keys":
		default:
			return fmt.Errorf("method %q is not allowed", expr.value)
		}
		if expr.value == "from" {
			if expr.left.op != "var" || expr.left.value != "Array" || len(expr.args) != 2 {
				return fmt.Errorf("Array.from requires a static length and bounded zero-argument callback")
			}
			descriptor := expr.args[0]
			if descriptor.op != "object" || len(descriptor.fields) != 1 || descriptor.fields[0] != "length" || descriptor.args[0].op != "lit" {
				return fmt.Errorf("Array.from supports only a static integer length")
			}
			length, ok := descriptor.args[0].value.(float64)
			if !ok || length < 0 || length > maxSyntheticItems || length != math.Trunc(length) {
				return fmt.Errorf("Array.from length must be an integer within the %d-item bound", maxSyntheticItems)
			}
			callback := expr.args[1]
			if callback.op != "lambda0" || callback.left == nil && len(callback.body) == 0 {
				return fmt.Errorf("Array.from callback must be a bounded zero-argument function")
			}
			if callback.left != nil {
				return validateSafeExpr(callback.left, scope)
			}
			return validateSafeStatements(callback.body, cloneBoolMap(scope))
		}
		if expr.value == "isArray" && (expr.left.op != "var" || expr.left.value != "Array" || len(expr.args) != 1) {
			return fmt.Errorf("Array.isArray requires one bounded value")
		}
		for index, a := range expr.args {
			if (expr.value == "map" || expr.value == "filter" || expr.value == "forEach") && index == 0 {
				if a.op != "lambda" || len(safeLambdaParameters(a)) != 1 {
					return fmt.Errorf("array %s requires one bounded arrow callback", expr.value)
				}
				inner := cloneBoolMap(scope)
				inner[safeLambdaParameters(a)[0]] = true
				if a.left != nil {
					if err := validateSafeExpr(a.left, inner); err != nil {
						return err
					}
				} else if err := validateSafeStatements(a.body, inner); err != nil {
					return err
				}
				continue
			}
			if expr.value == "replace" && index == 1 && a.op == "lambda" {
				params := safeLambdaParameters(a)
				if len(params) < 1 || len(params) > 4 {
					return fmt.Errorf("replace callback requires one to four bounded parameters")
				}
				inner := cloneBoolMap(scope)
				for _, param := range params {
					if !safeCodeIdentifier(param) {
						return fmt.Errorf("replace callback parameter is invalid")
					}
					inner[param] = true
				}
				if a.left != nil {
					if err := validateSafeExpr(a.left, inner); err != nil {
						return err
					}
				} else if err := validateSafeStatements(a.body, inner); err != nil {
					return err
				}
				continue
			}
			if err := validateSafeExpr(a, scope); err != nil {
				return err
			}
		}
		if expr.value == "all" || expr.value == "first" {
			validInput := expr.left.op == "var" && expr.left.value == "$input"
			validPriorNode := isSafeNamedNodeLookup(expr.left, scope)
			if (!validInput && !validPriorNode) || len(expr.args) != 0 {
				return fmt.Errorf("%s requires a direct input or static prior-node receiver and no arguments", expr.value)
			}
		}
		if expr.value == "map" || expr.value == "filter" || expr.value == "forEach" {
			if len(expr.args) != 1 || expr.args[0].op != "lambda" {
				return fmt.Errorf("array %s supports one bounded arrow callback", expr.value)
			}
		}
		if expr.value == "floor" && (expr.left.op != "var" || expr.left.value != "Math" || len(expr.args) != 1) {
			return fmt.Errorf("Math.floor requires one numeric argument")
		}
		if expr.value == "random" && (expr.left.op != "var" || expr.left.value != "Math" || len(expr.args) != 0) {
			return fmt.Errorf("Math.random takes no arguments")
		}
		if expr.value == "includes" && len(expr.args) != 1 {
			return fmt.Errorf("array includes supports one argument")
		}
		if expr.value == "concat" && len(expr.args) != 1 {
			return fmt.Errorf("array concat supports one bounded argument")
		}
		if expr.value == "slice" {
			if len(expr.args) < 1 || len(expr.args) > 2 {
				return fmt.Errorf("slice supports one or two bounded integer arguments")
			}
		}
		return nil
	case "lambda", "lambda0":
		return fmt.Errorf("arrow callback is allowed only as an array map/filter argument")
	case "invoke":
		if expr.left.op == "var" && expr.left.value == "$" {
			if len(expr.args) != 1 {
				return fmt.Errorf("named-node lookup requires one bounded node-name value")
			}
			if expr.args[0].op == "lit" {
				name, ok := expr.args[0].value.(string)
				if !ok || !safeCodeNodeName(name) {
					return fmt.Errorf("named-node lookup requires a bounded static node name")
				}
			} else if name, ok := expr.args[0].value.(string); expr.args[0].op != "var" || !ok || !scopeHas(scope, name) {
				return fmt.Errorf("named-node lookup requires one bounded node-name value")
			}
			return nil
		}
		if expr.left.op == "var" && expr.left.value == "$items" {
			if len(expr.args) != 1 || expr.args[0].op != "lit" {
				return fmt.Errorf("$items requires one static prior-node name")
			}
			name, ok := expr.args[0].value.(string)
			if !ok || !safeCodeNodeName(name) {
				return fmt.Errorf("$items requires a bounded prior-node name")
			}
			return nil
		}
		staticBuiltin := expr.left.op == "var" && (expr.left.value == "String" || expr.left.value == "parseInt" || expr.left.value == "Number")
		staticStringCodePoint := expr.left.op == "prop" && expr.left.value == "fromCodePoint" && expr.left.left.op == "var" && expr.left.left.value == "String"
		if !staticBuiltin && !staticStringCodePoint {
			name, _ := expr.left.value.(string)
			if expr.left.op != "var" || !scopeHas(scope, name) {
				return fmt.Errorf("function invocation is not allowed")
			}
		}
		for _, a := range expr.args {
			if err := validateSafeExpr(a, scope); err != nil {
				return err
			}
		}
		return nil
	case "!":
		return validateSafeExpr(expr.left, scope)
	case "typeof":
		return validateSafeExpr(expr.left, scope)
	case "index", "optindex":
		if err := validateSafeExpr(expr.left, scope); err != nil {
			return err
		}
		return validateSafeExpr(expr.right, scope)
	case "optprop":
		if !safeCodePropertyName(expr.value.(string)) {
			return fmt.Errorf("property %q is not allowed", expr.value)
		}
		return validateSafeExpr(expr.left, scope)
	case "?:", "pair":
		if err := validateSafeExpr(expr.left, scope); err != nil {
			return err
		}
		return validateSafeExpr(expr.right, scope)
	case "??", "||", "&&", "===", "==", "!==", "!=", "in", ">", ">=", "<", "<=", "+", "-", "*":
		if err := validateSafeExpr(expr.left, scope); err != nil {
			return err
		}
		return validateSafeExpr(expr.right, scope)
	default:
		return fmt.Errorf("operator %q is not allowed", expr.op)
	}
}

func scopeHas(scope map[string]bool, name string) bool {
	_, exists := scope[name]
	return exists
}

func (p *codeParser) expr(min int) (*codeExpr, error) {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 64 {
		return nil, fmt.Errorf("expression nesting exceeds 64")
	}
	var left *codeExpr
	t := p.peek()
	p.pos++
	switch t.kind {
	case "string":
		left = &codeExpr{op: "lit", value: t.text}
	case "number":
		n, _ := strconv.Atoi(t.text)
		left = &codeExpr{op: "lit", value: float64(n)}
	case "regex":
		parts := strings.SplitN(t.text, "\x00", 2)
		flags := ""
		if len(parts) == 2 {
			flags = parts[1]
		}
		left = &codeExpr{op: "regex", value: safeRegex{pattern: parts[0], global: strings.Contains(flags, "g"), ignoreCase: strings.Contains(flags, "i")}}
	case "template":
		expr, err := parseSafeTemplate(t.text)
		if err != nil {
			return nil, err
		}
		left = expr
	case "ident":
		switch t.text {
		case "true":
			left = &codeExpr{op: "lit", value: true}
		case "false":
			left = &codeExpr{op: "lit", value: false}
		case "null":
			left = &codeExpr{op: "lit", value: nil}
		case "typeof":
			x, e := p.expr(8)
			left = &codeExpr{op: "typeof", left: x}
			if e != nil {
				return nil, e
			}
		case "new":
			if p.peek().text == "Set" {
				p.pos++
				if err := p.need("("); err != nil {
					return nil, err
				}
				var values *codeExpr
				if !p.take(")") {
					var err error
					values, err = p.expr(0)
					if err != nil {
						return nil, err
					}
					if err = p.need(")"); err != nil {
						return nil, err
					}
				}
				left = &codeExpr{op: "set", left: values}
				break
			}
			if p.peek().text != "Date" {
				return nil, fmt.Errorf("only synthetic Date() or bounded Set() is supported")
			}
			p.pos++
			if err := p.need("("); err != nil {
				return nil, err
			}
			var dateInput *codeExpr
			if !p.take(")") {
				var err error
				dateInput, err = p.expr(0)
				if err != nil {
					return nil, err
				}
				if err = p.need(")"); err != nil {
					return nil, err
				}
			}
			left = &codeExpr{op: "date", left: dateInput}
		default:
			left = &codeExpr{op: "var", value: t.text}
		}
	case "punct":
		if t.text == "!" {
			x, e := p.expr(8)
			left = &codeExpr{op: "!", left: x}
			if e != nil {
				return nil, e
			}
		} else if t.text == "(" {
			start := p.pos
			if p.take(")") && p.take("=>") {
				if p.take("{") {
					body, e := p.statements("}")
					if e != nil {
						return nil, e
					}
					left = &codeExpr{op: "lambda0", body: body}
				} else {
					body, e := p.expr(0)
					if e != nil {
						return nil, e
					}
					left = &codeExpr{op: "lambda0", left: body}
				}
			} else {
				p.pos = start
				params := []string{}
				parsedLambda := false
				if p.peek().kind == "ident" {
					for {
						if p.peek().kind != "ident" {
							params = nil
							break
						}
						params = append(params, p.peek().text)
						p.pos++
						if !p.take(",") {
							break
						}
					}
					if params != nil && p.take(")") && p.take("=>") {
						if len(params) > 4 {
							return nil, fmt.Errorf("arrow functions support at most four bounded parameters")
						}
						if p.take("{") {
							body, e := p.statements("}")
							if e != nil {
								return nil, e
							}
							left = &codeExpr{op: "lambda", value: params[0], params: params, body: body}
						} else {
							body, e := p.expr(0)
							if e != nil {
								return nil, e
							}
							left = &codeExpr{op: "lambda", value: params[0], params: params, left: body}
						}
						parsedLambda = true
					}
				}
				if !parsedLambda {
					p.pos = start
					x, e := p.expr(0)
					if e != nil {
						return nil, e
					}
					if e = p.need(")"); e != nil {
						return nil, e
					}
					left = x
				}
			}
		} else if t.text == "[" {
			var xs []*codeExpr
			if !p.take("]") {
				for {
					x, e := p.expr(0)
					if e != nil {
						return nil, e
					}
					xs = append(xs, x)
					if p.take("]") {
						break
					}
					if e = p.need(","); e != nil {
						return nil, e
					}
					if p.take("]") {
						break
					}
				}
			}
			left = &codeExpr{op: "array", args: xs}
		} else if t.text == "{" {
			var fields []string
			var xs []*codeExpr
			if !p.take("}") {
				for {
					if p.take("...") {
						x, e := p.expr(0)
						if e != nil {
							return nil, e
						}
						fields = append(fields, "")
						xs = append(xs, &codeExpr{op: "spread", left: x})
						if p.take("}") {
							break
						}
						if e = p.need(","); e != nil {
							return nil, e
						}
						continue
					}
					key := p.peek().text
					keyKind := p.peek().kind
					if keyKind != "ident" && keyKind != "string" {
						return nil, fmt.Errorf("object keys must be static identifiers or strings (got %q)", key)
					}
					p.pos++
					fields = append(fields, key)
					var x *codeExpr
					var e error
					if p.take(":") {
						x, e = p.expr(0)
					} else if keyKind == "ident" {
						x = &codeExpr{op: "var", value: key}
					} else {
						return nil, fmt.Errorf("object field requires a value")
					}
					if e != nil {
						return nil, e
					}
					xs = append(xs, x)
					if p.take("}") {
						break
					}
					if e = p.need(","); e != nil {
						return nil, e
					}
					if p.take("}") {
						break
					}
				}
			}
			left = &codeExpr{op: "object", fields: fields, args: xs}
		} else {
			return nil, fmt.Errorf("unexpected token %q", t.text)
		}
	default:
		return nil, fmt.Errorf("unexpected token %q", t.text)
	}
	for {
		if p.take("?.") {
			if p.take("[") {
				index, e := p.expr(0)
				if e != nil {
					return nil, e
				}
				if e = p.need("]"); e != nil {
					return nil, e
				}
				left = &codeExpr{op: "optindex", left: left, right: index}
			} else {
				prop := p.peek().text
				if p.peek().kind != "ident" {
					return nil, fmt.Errorf("expected optional property name")
				}
				p.pos++
				if p.take("(") {
					var args []*codeExpr
					if !p.take(")") {
						for {
							x, e := p.expr(0)
							if e != nil {
								return nil, e
							}
							args = append(args, x)
							if p.take(")") {
								break
							}
							if e = p.need(","); e != nil {
								return nil, e
							}
						}
					}
					left = &codeExpr{op: "call", value: prop, left: left, args: args}
				} else {
					left = &codeExpr{op: "optprop", value: prop, left: left}
				}
			}
			continue
		}
		if p.take("[") {
			index, e := p.expr(0)
			if e != nil {
				return nil, e
			}
			if e = p.need("]"); e != nil {
				return nil, e
			}
			left = &codeExpr{op: "index", left: left, right: index}
			continue
		}
		if p.take(".") {
			prop := p.peek().text
			if p.peek().kind != "ident" {
				return nil, fmt.Errorf("expected property name")
			}
			p.pos++
			if p.take("(") {
				var args []*codeExpr
				if !p.take(")") {
					for {
						x, e := p.expr(0)
						if e != nil {
							return nil, e
						}
						args = append(args, x)
						if p.take(")") {
							break
						}
						if e = p.need(","); e != nil {
							return nil, e
						}
					}
				}
				left = &codeExpr{op: "call", value: prop, left: left, args: args}
			} else {
				left = &codeExpr{op: "prop", value: prop, left: left}
			}
			continue
		}
		if p.take("(") {
			var args []*codeExpr
			if !p.take(")") {
				for {
					x, e := p.expr(0)
					if e != nil {
						return nil, e
					}
					args = append(args, x)
					if p.take(")") {
						break
					}
					if e = p.need(","); e != nil {
						return nil, e
					}
				}
			}
			left = &codeExpr{op: "invoke", left: left, args: args}
			continue
		}
		if p.peek().text == "=>" {
			if min != 0 || left.op != "var" {
				return nil, fmt.Errorf("arrow functions support one named parameter and an expression body")
			}
			p.pos++
			if p.take("{") {
				body, e := p.statements("}")
				if e != nil {
					return nil, e
				}
				left = &codeExpr{op: "lambda", value: left.value, body: body}
			} else {
				body, e := p.expr(0)
				if e != nil {
					return nil, e
				}
				left = &codeExpr{op: "lambda", value: left.value, left: body}
			}
			continue
		}
		op := p.peek().text
		prec := map[string]int{"??": 1, "||": 1, "&&": 2, "===": 3, "!==": 3, "==": 3, "!=": 3, "in": 3, ">": 4, ">=": 4, "<": 4, "<=": 4, "+": 5, "-": 6, "*": 7}[op]
		if prec < min || prec == 0 {
			break
		}
		p.pos++
		right, e := p.expr(prec + 1)
		if e != nil {
			return nil, e
		}
		left = &codeExpr{op: op, left: left, right: right}
	}
	if min == 0 && p.take("?") {
		ifTrue, err := p.expr(0)
		if err != nil {
			return nil, err
		}
		if err = p.need(":"); err != nil {
			return nil, err
		}
		ifFalse, err := p.expr(0)
		if err != nil {
			return nil, err
		}
		left = &codeExpr{op: "?:", left: left, right: &codeExpr{op: "pair", left: ifTrue, right: ifFalse}}
	}
	return left, nil
}

type safeCodeContext struct {
	locals      map[string]any
	nodeOutputs map[string]safeInput
	now         time.Time
	rng         *rand.Rand
	steps       int
	callDepth   int
	rootLocals  map[string]any
	returnValue any
}

var errSafeContinue = fmt.Errorf("safe-code continue")
var errSafeReturn = fmt.Errorf("safe-code return")
var errSafeBreak = fmt.Errorf("safe-code break")

func executeSafeCode(source string, input []map[string]any, now time.Time) ([]map[string]any, error) {
	return executeSafeCodeWithNodeOutputs(source, input, nil, now)
}

func normalizeSafeExpression(source string) (string, error) {
	source = strings.TrimSpace(source)
	if strings.HasPrefix(source, "={{") && strings.HasSuffix(source, "}}") {
		source = strings.TrimSpace(source[3 : len(source)-2])
	}
	if source == "" || len(source) > 4096 {
		return "", fmt.Errorf("expression must be 1..4096 bytes")
	}
	source = strings.ReplaceAll(source, "$json", "$input.first().json")
	wrapped := "return [{json: {value: " + source + "}}];"
	if _, err := parseSafeCode(wrapped); err != nil {
		return "", err
	}
	return source, nil
}

func executeSafeExpression(source string, input map[string]any, now time.Time) (any, error) {
	return executeSafeExpressionWithNodeOutputs(source, input, nil, now)
}

func executeSafeExpressionWithNodeOutputs(source string, input map[string]any, nodeOutputs map[string][]map[string]any, now time.Time) (any, error) {
	wrapped := "return [{json: {value: " + source + "}}];"
	items, err := executeSafeCodeWithNodeOutputs(wrapped, []map[string]any{input}, nodeOutputs, now)
	if err != nil {
		return nil, err
	}
	if len(items) != 1 {
		return nil, fmt.Errorf("expression must emit one bounded value")
	}
	data, ok := items[0]["json"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expression emitted an invalid value object")
	}
	return data["value"], nil
}

func executeSafeCodeWithNodeOutputs(source string, input []map[string]any, nodeOutputs map[string][]map[string]any, now time.Time) ([]map[string]any, error) {
	return executeSafeCodeWithNodeOutputsMode(source, input, nodeOutputs, now, false, false)
}

func executeSafeLegacyFunctionWithNodeOutputs(source string, input []map[string]any, nodeOutputs map[string][]map[string]any, now time.Time) ([]map[string]any, error) {
	return executeSafeCodeWithNodeOutputsMode(source, input, nodeOutputs, now, true, false)
}

func executeSafeCodePerItemWithNodeOutputs(source string, input map[string]any, nodeOutputs map[string][]map[string]any, now time.Time) (map[string]any, error) {
	output, err := executeSafeCodeWithNodeOutputsMode(source, []map[string]any{input}, nodeOutputs, now, false, true)
	if err != nil {
		return nil, err
	}
	if len(output) != 1 {
		return nil, fmt.Errorf("runOnceForEachItem Code must return exactly one item")
	}
	return output[0], nil
}

func executeSafeCodeWithNodeOutputsMode(source string, input []map[string]any, nodeOutputs map[string][]map[string]any, now time.Time, legacyFunction, runOnceForEachItem bool) ([]map[string]any, error) {
	program, err := parseSafeCodeMode(source, legacyFunction, runOnceForEachItem)
	if err != nil {
		return nil, err
	}
	if len(input) > maxSyntheticItems {
		return nil, fmt.Errorf("input item bound exceeded")
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("encode input items: %w", err)
	}
	if len(inputJSON) > maxSafeCodeInputBytes {
		return nil, fmt.Errorf("input byte bound exceeded")
	}
	for index, item := range input {
		encoded, e := json.Marshal(item)
		if e != nil {
			return nil, e
		}
		if len(encoded) > maxSafeCodeItemBytes {
			return nil, fmt.Errorf("input item %d exceeds %d bytes", index, maxSafeCodeItemBytes)
		}
	}
	items := make([]any, 0, len(input))
	for _, item := range input {
		items = append(items, map[string]any{"json": item})
	}
	if len(nodeOutputs) > maxSafeCodeNodeNames {
		return nil, fmt.Errorf("prior-node context exceeds %d nodes", maxSafeCodeNodeNames)
	}
	nodeOutputBytes := 0
	named := make(map[string]safeInput, len(nodeOutputs))
	namedItemCount := 0
	for name, outputs := range nodeOutputs {
		if !safeCodeNodeName(name) {
			return nil, fmt.Errorf("prior-node context contains an invalid node name")
		}
		if len(outputs) > maxSyntheticItems-namedItemCount {
			return nil, fmt.Errorf("prior-node context exceeds %d total items", maxSyntheticItems)
		}
		namedItemCount += len(outputs)
		wrapped := make([]any, 0, len(outputs))
		for index, output := range outputs {
			encoded, encodeErr := json.Marshal(output)
			if encodeErr != nil {
				return nil, fmt.Errorf("encode prior node %q item %d: %w", name, index, encodeErr)
			}
			if len(encoded) > maxSafeCodeItemBytes {
				return nil, fmt.Errorf("prior node %q item %d exceeds %d bytes", name, index, maxSafeCodeItemBytes)
			}
			nodeOutputBytes += len(encoded)
			wrapped = append(wrapped, map[string]any{"json": output})
		}
		named[name] = safeInput{items: wrapped}
	}
	if nodeOutputBytes+len(inputJSON) > maxSafeCodeInputBytes {
		return nil, fmt.Errorf("input and prior-node context byte bound exceeded")
	}
	seedInput := append(append([]byte(source), inputJSON...), []byte(now.UTC().Format(time.RFC3339Nano))...)
	seed := sha256.Sum256(seedInput)
	ctx := &safeCodeContext{locals: map[string]any{"__input": safeInput{items: items}, "$now": safeDate{at: now.UTC()}}, nodeOutputs: named, now: now.UTC(), rng: rand.New(rand.NewSource(int64(binary.BigEndian.Uint64(seed[:8]))))}
	returned, err := ctx.run(program.statements)
	if err == errSafeReturn {
		returned = ctx.returnValue
		err = nil
	}
	if err != nil {
		return nil, err
	}
	if runOnceForEachItem {
		item, ok := returned.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("runOnceForEachItem Code must return one item object")
		}
		jsonValue, wrapped := item["json"].(map[string]any)
		if !wrapped {
			// n8n's pinned JS task runner accepts a raw object from each-item Code
			// and wraps it as the output item's json, unless it already carries json.
			// https://github.com/n8n-io/n8n/blob/n8n@2.48.0/packages/@n8n/task-runner/src/js-task-runner/js-task-runner.ts
			jsonValue = item
			item = map[string]any{"json": jsonValue}
		}
		if len(jsonValue) > 256 {
			return nil, fmt.Errorf("runOnceForEachItem Code must return a bounded json object")
		}
		encoded, marshalErr := json.Marshal(item)
		if marshalErr != nil || len(encoded) > maxSafeCodeItemBytes {
			return nil, fmt.Errorf("runOnceForEachItem Code output item exceeds %d bytes", maxSafeCodeItemBytes)
		}
		return []map[string]any{item}, nil
	}
	if item, ok := returned.(map[string]any); ok {
		// n8n 2.48.0's Code node accepts one normalized item object in all-items
		// mode as well as an item array:
		// https://github.com/n8n-io/n8n/blob/n8n@2.48.0/packages/nodes-base/nodes/Code/result-validation.ts
		jsonValue, valid := item["json"].(map[string]any)
		if !valid || len(jsonValue) > 256 || len(item) > 5 {
			return nil, fmt.Errorf("single Code output must contain one bounded json object")
		}
		for key := range item {
			if key != "json" && key != "binary" && key != "pairedItem" && key != "error" && key != "index" {
				return nil, fmt.Errorf("single Code output contains unsupported top-level field %q", key)
			}
		}
		encoded, marshalErr := json.Marshal(item)
		if marshalErr != nil || len(encoded) > maxSafeCodeItemBytes {
			return nil, fmt.Errorf("single Code output item exceeds %d bytes", maxSafeCodeItemBytes)
		}
		return []map[string]any{item}, nil
	}
	arr, ok := returned.(*safeArray)
	if !ok || len(arr.values) > maxSyntheticItems {
		return nil, fmt.Errorf("return must be a bounded item array")
	}
	out := make([]map[string]any, 0, len(arr.values))
	for _, v := range arr.values {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("each returned item must be an object")
		}
		if _, ok = m["json"].(map[string]any); !ok {
			return nil, fmt.Errorf("each returned item needs a json object")
		}
		out = append(out, m)
	}
	b, e := json.Marshal(out)
	if e != nil {
		return nil, e
	}
	if len(b) > maxSafeCodeOutputBytes {
		return nil, fmt.Errorf("output byte bound exceeded")
	}
	return out, nil
}

func (c *safeCodeContext) tick() error {
	c.steps++
	if c.steps > maxSafeCodeSteps {
		return fmt.Errorf("interpreter step bound exceeded")
	}
	return nil
}

func (c *safeCodeContext) run(ss []codeStmt) (any, error) {
	if c.callDepth == 0 && c.rootLocals == nil {
		c.rootLocals = c.locals
	}
	for _, statement := range ss {
		if statement.kind == "function" {
			c.locals[statement.name] = safeLocalFunction{name: statement.name, params: statement.params, body: statement.body}
		}
	}
	var ret any
	for _, s := range ss {
		if e := c.tick(); e != nil {
			return nil, e
		}
		switch s.kind {
		case "function":
			continue
		case "decl":
			if _, exists := c.locals[s.name]; exists {
				return nil, fmt.Errorf("duplicate local %s", s.name)
			}
			v, e := c.eval(s.expr)
			if e != nil {
				return nil, e
			}
			c.locals[s.name] = v
		case "letdecl":
			if _, exists := c.locals[s.name]; exists {
				return nil, fmt.Errorf("duplicate local %s", s.name)
			}
			var v any
			var e error
			if s.expr != nil {
				v, e = c.eval(s.expr)
				if e != nil {
					return nil, e
				}
			}
			c.locals[s.name] = v
		case "objectdecl":
			value, err := c.eval(s.expr)
			if err != nil {
				return nil, err
			}
			object, ok := value.(map[string]any)
			if !ok || len(s.params) == 0 || len(s.params) != len(s.fields) {
				return nil, fmt.Errorf("object destructuring requires a bounded plain object")
			}
			for index, name := range s.params {
				c.locals[name] = object[s.fields[index]]
			}
		case "assign":
			current, exists := c.locals[s.name]
			v, e := c.eval(s.expr)
			if e != nil {
				return nil, e
			}
			if s.op == "+=" {
				if !exists {
					return nil, fmt.Errorf("+= assignment target %q is unavailable", s.name)
				}
				if a, ok := current.(float64); ok {
					if b, ok := v.(float64); ok {
						v = a + b
					} else {
						return nil, fmt.Errorf("numeric += requires numeric values")
					}
				} else if a, ok := current.(string); ok {
					v = a + jsString(v)
				} else {
					return nil, fmt.Errorf("+= type is unsupported")
				}
			}
			c.locals[s.name] = v
		case "assignchain":
			value, err := c.eval(s.expr)
			if err != nil {
				return nil, err
			}
			for _, name := range s.params {
				c.locals[name] = value
			}
		case "setprop":
			parent, e := c.eval(s.target.left)
			if e != nil {
				return nil, e
			}
			object, ok := parent.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("property assignment requires a local plain object")
			}
			var key string
			if s.target.op == "prop" {
				key = s.target.value.(string)
			} else {
				keyValue, keyErr := c.eval(s.target.right)
				if keyErr != nil {
					return nil, keyErr
				}
				var ok bool
				key, ok = keyValue.(string)
				if !ok {
					return nil, fmt.Errorf("computed property assignment key must evaluate to a string")
				}
			}
			if !safeCodePropertyName(key) {
				return nil, fmt.Errorf("property %q is not safe to assign", key)
			}
			value, e := c.eval(s.expr)
			if e != nil {
				return nil, e
			}
			object[key] = value
		case "expr":
			if _, e := c.eval(s.expr); e != nil {
				return nil, e
			}
		case "if":
			v, e := c.eval(s.test)
			if e != nil {
				return nil, e
			}
			branch := s.alternate
			if truthy(v) {
				branch = s.body
			}
			branchResult, branchErr := c.run(branch)
			if branchErr == errSafeReturn {
				return branchResult, errSafeReturn
			}
			if branchErr != nil {
				return nil, branchErr
			}
		case "continue":
			return nil, errSafeContinue
		case "break":
			return nil, errSafeBreak
		case "switch":
			value, err := c.eval(s.expr)
			if err != nil {
				return nil, err
			}
			selected := -1
			fallback := -1
			for i, branch := range s.cases {
				if branch.isDefault {
					fallback = i
					continue
				}
				match, err := c.eval(branch.test)
				if err != nil {
					return nil, err
				}
				if sameSafePrimitive(value, match) {
					selected = i
					break
				}
			}
			if selected < 0 {
				selected = fallback
			}
			if selected >= 0 {
				for _, branch := range s.cases[selected:] {
					branchResult, branchErr := c.run(branch.body)
					if branchErr == errSafeBreak {
						break
					}
					if branchErr == errSafeReturn {
						return branchResult, errSafeReturn
					}
					if branchErr != nil {
						return nil, branchErr
					}
				}
			}
		case "try":
			if value, e := c.run(s.body); e != nil {
				if e == errSafeReturn {
					return value, e
				}
				if e == errSafeContinue {
					return nil, e
				}
				if value, e = c.run(s.alternate); e != nil {
					return value, e
				}
			}
		case "forof":
			v, e := c.eval(s.expr)
			if e != nil {
				return nil, e
			}
			array, ok := v.(*safeArray)
			if !ok || len(array.values) > maxSyntheticItems {
				return nil, fmt.Errorf("for-of needs a bounded array")
			}
			for _, item := range array.values {
				if e = c.tick(); e != nil {
					return nil, e
				}
				before := make(map[string]bool, len(c.locals))
				for key := range c.locals {
					before[key] = true
				}
				c.locals[s.name] = item
				bodyResult, bodyErr := c.run(s.body)
				if bodyErr == errSafeReturn {
					return bodyResult, errSafeReturn
				}
				if bodyErr != nil && bodyErr != errSafeContinue {
					return nil, bodyErr
				}
				for key := range c.locals {
					if !before[key] {
						delete(c.locals, key)
					}
				}
			}
			delete(c.locals, s.name)
		case "for":
			initial, e := c.eval(s.init.expr)
			if e != nil {
				return nil, e
			}
			c.locals[s.name] = initial
			for iteration := 0; iteration < maxSyntheticItems; iteration++ {
				condition, e := c.eval(s.test)
				if e != nil {
					return nil, e
				}
				if !truthy(condition) {
					break
				}
				if e = c.tick(); e != nil {
					return nil, e
				}
				before := make(map[string]bool, len(c.locals))
				for key := range c.locals {
					before[key] = true
				}
				bodyResult, bodyErr := c.run(s.body)
				if bodyErr == errSafeReturn {
					return bodyResult, errSafeReturn
				}
				if bodyErr != nil && bodyErr != errSafeContinue {
					return nil, bodyErr
				}
				for key := range c.locals {
					if !before[key] {
						delete(c.locals, key)
					}
				}
				index, ok := c.locals[s.name].(float64)
				if !ok {
					return nil, fmt.Errorf("bounded for-loop index must remain numeric")
				}
				c.locals[s.name] = index + 1
				if iteration == maxSyntheticItems-1 {
					return nil, fmt.Errorf("bounded for-loop exceeded item limit")
				}
			}
			delete(c.locals, s.name)
		case "return":
			ret, e := c.eval(s.expr)
			if e == nil {
				c.returnValue = ret
				return ret, errSafeReturn
			}
			return nil, e
		default:
			return nil, fmt.Errorf("unsupported statement %q", s.kind)
		}
	}
	return ret, nil
}

func (c *safeCodeContext) eval(e *codeExpr) (any, error) {
	if e == nil {
		return nil, fmt.Errorf("missing expression")
	}
	if err := c.tick(); err != nil {
		return nil, err
	}
	switch e.op {
	case "lit":
		return e.value, nil
	case "regex":
		return e.value.(safeRegex), nil
	case "template":
		parts, ok := e.value.([]string)
		if !ok || len(parts) != len(e.args)+1 {
			return nil, fmt.Errorf("template literal structure is invalid")
		}
		var output strings.Builder
		for index, expression := range e.args {
			output.WriteString(parts[index])
			value, err := c.eval(expression)
			if err != nil {
				return nil, err
			}
			output.WriteString(jsString(value))
			if output.Len() > maxSafeCodeItemBytes {
				return nil, fmt.Errorf("template output item bound exceeded")
			}
		}
		output.WriteString(parts[len(parts)-1])
		if output.Len() > maxSafeCodeItemBytes {
			return nil, fmt.Errorf("template output item bound exceeded")
		}
		return output.String(), nil
	case "date":
		if e.left == nil {
			return safeDate{at: c.now}, nil
		}
		value, err := c.eval(e.left)
		if err != nil {
			return nil, err
		}
		raw, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("Date input must be an ISO date string")
		}
		for _, layout := range []string{time.RFC3339, "2006-01-02", "2006-01-02T15:04:05.000Z"} {
			if parsed, parseErr := time.Parse(layout, raw); parseErr == nil {
				return safeDate{at: parsed.UTC()}, nil
			}
		}
		return nil, fmt.Errorf("Date input is not a valid ISO date")
	case "set":
		set := safeSet{values: map[string]any{}}
		if e.left == nil {
			return set, nil
		}
		value, err := c.eval(e.left)
		if err != nil {
			return nil, err
		}
		array, ok := value.(*safeArray)
		if !ok || len(array.values) > maxSyntheticItems {
			return nil, fmt.Errorf("Set constructor requires a bounded array")
		}
		for _, item := range array.values {
			key, valid := safeSetKey(item)
			if !valid {
				return nil, fmt.Errorf("Set values must be bounded primitive scalars")
			}
			set.values[key] = item
		}
		return set, nil
	case "post++":
		return nil, fmt.Errorf("post-increment is allowed only as a bounded for-loop update")
	case "typeof":
		v, err := c.eval(e.left)
		if err != nil {
			return nil, err
		}
		switch v.(type) {
		case nil:
			return "object", nil
		case string:
			return "string", nil
		case bool:
			return "boolean", nil
		case float64:
			return "number", nil
		default:
			return "object", nil
		}
	case "var":
		v, ok := c.locals[e.value.(string)]
		if ok {
			return v, nil
		}
		if e.value == "$input" {
			input, _ := c.locals["__input"].(safeInput)
			return input, nil
		}
		if e.value == "$json" {
			input, _ := c.locals["__input"].(safeInput)
			if len(input.items) == 0 {
				return nil, fmt.Errorf("$json requires one current item")
			}
			item, _ := input.items[0].(map[string]any)
			jsonValue, _ := item["json"].(map[string]any)
			return jsonValue, nil
		}
		if e.value == "$now" {
			return safeDate{at: c.now}, nil
		}
		if e.value == "$execution" {
			return safeExecution{resumeURL: "https://n8n.synthetic.invalid"}, nil
		}
		if e.value == "String" {
			return safeBuiltin("String"), nil
		}
		if e.value == "Number" {
			return safeBuiltin("Number"), nil
		}
		if e.value == "Array" {
			return safeBuiltin("Array"), nil
		}
		if e.value == "Math" {
			return safeBuiltin("Math"), nil
		}
		if e.value == "DateTime" {
			return safeBuiltin("DateTime"), nil
		}
		if e.value == "parseInt" {
			return safeBuiltin("parseInt"), nil
		}
		if e.value == "$" {
			return safeBuiltin("named-node"), nil
		}
		if e.value == "$items" {
			return safeBuiltin("items"), nil
		}
		if e.value == "JSON" {
			return safeBuiltin("JSON"), nil
		}
		if e.value == "Object" {
			return safeBuiltin("Object"), nil
		}
		return nil, fmt.Errorf("identifier %q is not allowed", e.value)
	case "array":
		a := &safeArray{values: make([]any, 0, len(e.args)), bytes: 2}
		for _, x := range e.args {
			v, er := c.eval(x)
			if er != nil {
				return nil, er
			}
			a.values = append(a.values, v)
		}
		return a, nil
	case "object":
		m := map[string]any{}
		for i, x := range e.args {
			v, er := c.eval(x)
			if er != nil {
				return nil, er
			}
			if x.op == "spread" {
				other, ok := v.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("object spread requires a plain JSON object")
				}
				for key, value := range other {
					if !safeCodePropertyName(key) {
						return nil, fmt.Errorf("object spread contains unsafe property %q", key)
					}
					m[key] = value
				}
				continue
			}
			m[e.fields[i]] = v
		}
		return m, nil
	case "spread":
		return c.eval(e.left)
	case "lambda", "lambda0":
		return e, nil
	case "!":
		v, er := c.eval(e.left)
		return !truthy(v), er
	case "??":
		a, er := c.eval(e.left)
		if er != nil {
			return nil, er
		}
		if a != nil {
			return a, nil
		}
		return c.eval(e.right)
	case "||":
		a, er := c.eval(e.left)
		if er != nil {
			return nil, er
		}
		if truthy(a) {
			return a, nil
		}
		return c.eval(e.right)
	case "&&":
		a, er := c.eval(e.left)
		if er != nil {
			return nil, er
		}
		if !truthy(a) {
			return a, nil
		}
		return c.eval(e.right)
	case "?:":
		condition, er := c.eval(e.left)
		if er != nil {
			return nil, er
		}
		if truthy(condition) {
			return c.eval(e.right.left)
		}
		return c.eval(e.right.right)
	case "in":
		key, er := c.eval(e.left)
		if er != nil {
			return nil, er
		}
		object, er := c.eval(e.right)
		if er != nil {
			return nil, er
		}
		if x, ok := object.(map[string]any); ok {
			_, exists := x[jsString(key)]
			return exists, nil
		}
		return nil, fmt.Errorf("in requires a bounded object")
	case "===", "==", "!==", "!=":
		a, er := c.eval(e.left)
		if er != nil {
			return nil, er
		}
		b, er := c.eval(e.right)
		if er != nil {
			return nil, er
		}
		eq := fmt.Sprint(a) == fmt.Sprint(b)
		if e.op == "!==" || e.op == "!=" {
			eq = !eq
		}
		return eq, nil
	case ">", ">=", "<", "<=":
		a, er := c.eval(e.left)
		if er != nil {
			return nil, er
		}
		b, er := c.eval(e.right)
		if er != nil {
			return nil, er
		}
		var x, y float64
		switch n := a.(type) {
		case float64:
			x = n
		case int:
			x = float64(n)
		case string:
			x = float64(len(n))
		default:
			return nil, fmt.Errorf("comparison type not allowed")
		}
		switch n := b.(type) {
		case float64:
			y = n
		case int:
			y = float64(n)
		default:
			return nil, fmt.Errorf("comparison value not allowed")
		}
		switch e.op {
		case ">":
			return x > y, nil
		case ">=":
			return x >= y, nil
		case "<":
			return x < y, nil
		default:
			return x <= y, nil
		}
	case "+":
		a, er := c.eval(e.left)
		if er != nil {
			return nil, er
		}
		b, er := c.eval(e.right)
		if er != nil {
			return nil, er
		}
		if x, ok := numberValue(a); ok {
			if y, ok := numberValue(b); ok {
				return x + y, nil
			}
		}
		return fmt.Sprint(a) + fmt.Sprint(b), nil
	case "-", "*":
		a, er := c.eval(e.left)
		if er != nil {
			return nil, er
		}
		b, er := c.eval(e.right)
		if er != nil {
			return nil, er
		}
		x, okX := numberValue(a)
		y, okY := numberValue(b)
		if !okX || !okY {
			return nil, fmt.Errorf("arithmetic requires numeric operands")
		}
		if e.op == "-" {
			return x - y, nil
		}
		return x * y, nil
	case "prop":
		v, er := c.eval(e.left)
		if er != nil {
			return nil, er
		}
		return safeProperty(v, e.value.(string))
	case "optprop":
		v, er := c.eval(e.left)
		if er != nil {
			return nil, er
		}
		if v == nil {
			return nil, nil
		}
		return safeProperty(v, e.value.(string))
	case "index", "optindex":
		v, er := c.eval(e.left)
		if er != nil {
			return nil, er
		}
		if e.op == "optindex" && v == nil {
			return nil, nil
		}
		key, er := c.eval(e.right)
		if er != nil {
			return nil, er
		}
		switch x := v.(type) {
		case map[string]any:
			return x[jsString(key)], nil
		case []any:
			n, ok := safeIndexNumber(key)
			if !ok || n < 0 || n >= float64(len(x)) || n != math.Trunc(n) {
				return nil, nil
			}
			return x[int(n)], nil
		case *safeArray:
			n, ok := safeIndexNumber(key)
			if !ok || n < 0 || n >= float64(len(x.values)) || n != math.Trunc(n) {
				return nil, nil
			}
			return x.values[int(n)], nil
		case string:
			n, ok := safeIndexNumber(key)
			if !ok || n < 0 || n >= float64(len(x)) || n != math.Trunc(n) {
				return nil, nil
			}
			return string(x[int(n)]), nil
		default:
			return nil, fmt.Errorf("computed access is unsupported on %T", v)
		}
	case "call":
		receiver, er := c.eval(e.left)
		if er != nil {
			return nil, er
		}
		method := e.value.(string)
		if method == "concat" {
			array, ok := receiver.(*safeArray)
			if !ok || len(e.args) != 1 || len(array.values) > maxSyntheticItems {
				return nil, fmt.Errorf("array concat requires one bounded item array")
			}
			value, evalErr := c.eval(e.args[0])
			if evalErr != nil {
				return nil, evalErr
			}
			result := append([]any{}, array.values...)
			if other, ok := value.(*safeArray); ok {
				if len(result)+len(other.values) > maxSyntheticItems {
					return nil, fmt.Errorf("array concat output item bound exceeded")
				}
				result = append(result, other.values...)
			} else {
				if len(result) >= maxSyntheticItems {
					return nil, fmt.Errorf("array concat output item bound exceeded")
				}
				result = append(result, value)
			}
			return boundedSafeArray(result)
		}
		if method == "map" || method == "filter" || method == "forEach" {
			if len(e.args) != 1 || e.args[0].op != "lambda" {
				return nil, fmt.Errorf("array %s requires one bounded arrow callback", method)
			}
			array, ok := receiver.(*safeArray)
			if !ok {
				return nil, fmt.Errorf("array %s requires an item array (got %T)", method, receiver)
			}
			if len(array.values) > maxSyntheticItems {
				return nil, fmt.Errorf("array %s input item bound exceeded", method)
			}
			result := make([]any, 0, len(array.values))
			for _, item := range array.values {
				value, evalErr := c.evalLambda(e.args[0], item)
				if evalErr != nil {
					return nil, evalErr
				}
				if method == "map" {
					result = append(result, value)
				} else if method == "filter" && truthy(value) {
					result = append(result, item)
				}
			}
			if method == "forEach" {
				return nil, nil
			}
			return boundedSafeArray(result)
		}
		args := make([]any, 0, len(e.args))
		for _, x := range e.args {
			v, ee := c.eval(x)
			if ee != nil {
				return nil, ee
			}
			args = append(args, v)
		}
		if method == "slice" {
			if array, ok := receiver.(*safeArray); ok {
				return safeArraySlice(array, args)
			}
		}
		if builtin, ok := receiver.(safeBuiltin); ok && builtin == "Array" && method == "isArray" && len(args) == 1 {
			_, ok := args[0].(*safeArray)
			return ok, nil
		}
		if array, ok := receiver.(safeBuiltin); ok && array == "Array" && method == "from" {
			return c.safeArrayFrom(args)
		}
		if inputMethod, ok := receiver.(safeBuiltin); ok && (inputMethod == "input.all" || inputMethod == "input.first") {
			if len(args) != 0 {
				return nil, fmt.Errorf("$input accessors take no arguments")
			}
			input, _ := c.locals["__input"].(safeInput)
			if inputMethod == "input.all" {
				return &safeArray{values: input.items}, nil
			}
			if len(input.items) == 0 {
				return nil, fmt.Errorf("$input.first requires at least one input item")
			}
			return input.items[0], nil
		}
		if mathBuiltin, ok := receiver.(safeBuiltin); ok && mathBuiltin == "Math" {
			switch method {
			case "floor":
				if len(args) != 1 {
					return nil, fmt.Errorf("Math.floor requires one numeric argument")
				}
				n, ok := numberValue(args[0])
				if !ok {
					return nil, fmt.Errorf("Math.floor requires a numeric argument")
				}
				return math.Floor(n), nil
			case "random":
				if len(args) != 0 {
					return nil, fmt.Errorf("Math.random takes no arguments")
				}
				return c.rng.Float64(), nil
			case "round":
				if len(args) != 1 {
					return nil, fmt.Errorf("Math.round requires one numeric argument")
				}
				n, ok := numberValue(args[0])
				if !ok {
					return nil, fmt.Errorf("Math.round requires a numeric argument")
				}
				return math.Round(n), nil
			}
		}
		if receiver == safeBuiltin("DateTime") && method == "fromISO" {
			if len(args) != 1 {
				return nil, fmt.Errorf("DateTime.fromISO requires one ISO date string")
			}
			raw, ok := args[0].(string)
			if !ok || len(raw) > 256 {
				return nil, fmt.Errorf("DateTime.fromISO requires one bounded ISO date string")
			}
			parsed, err := parseSafeISODate(raw)
			if err != nil {
				return nil, err
			}
			return safeDateTime{at: parsed}, nil
		}
		if namedNode, ok := receiver.(safeBuiltin); ok && namedNode == "named-node" {
			if len(args) != 1 {
				return nil, fmt.Errorf("named-node lookup requires one static node name")
			}
			name, ok := args[0].(string)
			if !ok || !safeCodeNodeName(name) {
				return nil, fmt.Errorf("named-node lookup requires a bounded static node name")
			}
			items, exists := c.nodeOutputs[name]
			if !exists {
				return nil, fmt.Errorf("prior node %q has not executed or has no available output", name)
			}
			return safeNodeLookup{items: items.items}, nil
		}
		if lookup, ok := receiver.(safeNodeLookup); ok && method == "first" {
			if len(args) != 0 || len(lookup.items) == 0 {
				return nil, fmt.Errorf("prior node first() requires available output")
			}
			return lookup.items[0], nil
		}
		return safeCall(c, receiver, method, args)
	case "invoke":
		fn, er := c.eval(e.left)
		if er != nil {
			return nil, er
		}
		args := make([]any, 0, len(e.args))
		for _, x := range e.args {
			v, ee := c.eval(x)
			if ee != nil {
				return nil, ee
			}
			args = append(args, v)
		}
		if local, ok := fn.(safeLocalFunction); ok {
			return c.callLocalFunction(local, args)
		}
		if fn == safeBuiltin("Object.keys") && len(args) == 1 {
			return safeObjectKeys(args[0])
		}
		if fn == safeBuiltin("Array.isArray") && len(args) == 1 {
			_, ok := args[0].(*safeArray)
			return ok, nil
		}
		if fn == safeBuiltin("String") && len(args) == 1 {
			return jsString(args[0]), nil
		}
		if fn == safeBuiltin("Number") && len(args) == 1 {
			switch value := args[0].(type) {
			case nil:
				return float64(0), nil
			case float64:
				return value, nil
			case int:
				return float64(value), nil
			case json.Number:
				number, err := value.Float64()
				if err == nil {
					return number, nil
				}
			case bool:
				if value {
					return float64(1), nil
				}
				return float64(0), nil
			case string:
				if strings.TrimSpace(value) == "" {
					return float64(0), nil
				}
				number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
				if err == nil {
					return number, nil
				}
			}
			return nil, fmt.Errorf("Number requires a safely parseable scalar")
		}
		if fn == safeBuiltin("String.fromCodePoint") && len(args) == 1 {
			number, ok := numberValue(args[0])
			if !ok || number < 0 || number > 0x10ffff || number != math.Trunc(number) || number >= 0xd800 && number <= 0xdfff {
				return nil, fmt.Errorf("String.fromCodePoint requires one Unicode scalar")
			}
			var result strings.Builder
			result.WriteRune(rune(number))
			return result.String(), nil
		}
		if lambda, ok := fn.(*codeExpr); ok && lambda.op == "lambda" && len(args) == 1 {
			return c.evalLambda(lambda, args[0])
		}
		if fn == safeBuiltin("parseInt") && len(args) >= 1 && len(args) <= 2 {
			text := strings.TrimSpace(jsString(args[0]))
			radix := 10
			if len(args) == 2 {
				n, ok := numberValue(args[1])
				if !ok || n != 10 {
					return nil, fmt.Errorf("parseInt only supports decimal radix")
				}
			}
			if text == "" {
				return nil, nil
			}
			value, err := strconv.ParseInt(text, radix, 64)
			if err != nil {
				return nil, nil
			}
			return float64(value), nil
		}
		if fn == safeBuiltin("named-node") && len(args) == 1 {
			name, ok := args[0].(string)
			if !ok || !safeCodeNodeName(name) {
				return nil, fmt.Errorf("named-node lookup requires a bounded static node name")
			}
			items, exists := c.nodeOutputs[name]
			if !exists {
				return nil, fmt.Errorf("prior node %q has not executed or has no available output", name)
			}
			return safeNodeLookup{items: items.items}, nil
		}
		if fn == safeBuiltin("items") && len(args) == 1 {
			name, ok := args[0].(string)
			if !ok || !safeCodeNodeName(name) {
				return nil, fmt.Errorf("$items requires a bounded prior-node name")
			}
			items, exists := c.nodeOutputs[name]
			if !exists {
				return nil, fmt.Errorf("prior node %q has not executed or has no available output", name)
			}
			return boundedSafeArray(items.items)
		}
		return nil, fmt.Errorf("function invocation is not allowed")
	default:
		return nil, fmt.Errorf("operator %q is not allowed", e.op)
	}
}

func safeArraySlice(array *safeArray, args []any) (*safeArray, error) {
	if array == nil || len(args) < 1 || len(args) > 2 || len(array.values) > maxSyntheticItems {
		return nil, fmt.Errorf("array slice requires a bounded one- or two-index range")
	}
	toIndex := func(raw any, fallback int) (int, error) {
		if raw == nil {
			return fallback, nil
		}
		number, ok := numberValue(raw)
		if !ok || math.IsNaN(number) || math.IsInf(number, 0) || math.Abs(number) > maxSafeCodeSliceUnits {
			return 0, fmt.Errorf("array slice index is outside its bound")
		}
		index := int(math.Trunc(number))
		if index < 0 {
			index += len(array.values)
		}
		if index < 0 {
			index = 0
		}
		if index > len(array.values) {
			index = len(array.values)
		}
		return index, nil
	}
	start, err := toIndex(args[0], 0)
	if err != nil {
		return nil, err
	}
	end := len(array.values)
	if len(args) == 2 {
		end, err = toIndex(args[1], len(array.values))
		if err != nil {
			return nil, err
		}
	}
	if end < start {
		end = start
	}
	return boundedSafeArray(array.values[start:end])
}

func safeSetKey(value any) (string, bool) {
	switch item := value.(type) {
	case nil:
		return "null", true
	case string:
		return "string:" + item, len(item) <= maxSafeCodeItemBytes
	case bool:
		return "bool:" + strconv.FormatBool(item), true
	case float64:
		if math.IsNaN(item) || math.IsInf(item, 0) {
			return "", false
		}
		return "number:" + strconv.FormatFloat(item, 'g', -1, 64), true
	case int:
		return "number:" + strconv.Itoa(item), true
	case json.Number:
		return "number:" + item.String(), true
	default:
		return "", false
	}
}

type (
	safeInput      struct{ items []any }
	safeNodeLookup struct{ items []any }
	safeExecution  struct{ resumeURL string }
	safeDate       struct{ at time.Time }
	safeDateTime   struct{ at time.Time }
	safeSet        struct{ values map[string]any }
	safeRegex      struct {
		pattern    string
		global     bool
		ignoreCase bool
	}
	safeBuiltin string
	safeArray   struct {
		values []any
		bytes  int
	}
)

func (r safeRegex) flags() string {
	var flags string
	if r.global {
		flags += "g"
	}
	if r.ignoreCase {
		flags += "i"
	}
	return flags
}

func safeRegexLiteralEnd(source string, start int) int {
	inClass := false
	for index := start; index < len(source); index++ {
		switch source[index] {
		case '\\':
			index++
		case '[':
			inClass = true
		case ']':
			inClass = false
		case '/':
			if !inClass {
				return index
			}
		}
	}
	return -1
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("JSON has trailing content")
	}
	return nil
}

func toSafeJSON(value any, depth int) (any, error) {
	if depth > 32 {
		return nil, fmt.Errorf("JSON nesting exceeds 32 levels")
	}
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, child := range typed {
			if !safeJSONPropertyName(key) {
				return nil, fmt.Errorf("JSON property %q is unsupported", key)
			}
			converted, err := toSafeJSON(child, depth+1)
			if err != nil {
				return nil, err
			}
			result[key] = converted
		}
		return result, nil
	case []any:
		if len(typed) > maxSyntheticItems {
			return nil, fmt.Errorf("JSON array item bound exceeded")
		}
		values := make([]any, len(typed))
		for index, child := range typed {
			converted, err := toSafeJSON(child, depth+1)
			if err != nil {
				return nil, err
			}
			values[index] = converted
		}
		return boundedSafeArray(values)
	case json.Number:
		return typed, nil
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return nil, fmt.Errorf("JSON number must be finite")
		}
		return typed, nil
	case int:
		return typed, nil
	case *safeArray:
		if typed == nil || len(typed.values) > maxSyntheticItems {
			return nil, fmt.Errorf("JSON array item bound exceeded")
		}
		values := make([]any, len(typed.values))
		for index, child := range typed.values {
			converted, err := toSafeJSON(child, depth+1)
			if err != nil {
				return nil, err
			}
			values[index] = converted
		}
		return boundedSafeArray(values)
	case string, bool, nil:
		return typed, nil
	default:
		return nil, fmt.Errorf("JSON value type %T is unsupported", typed)
	}
}

func (a *safeArray) MarshalJSON() ([]byte, error) { return json.Marshal(a.values) }

func safeProperty(v any, k string) (any, error) {
	switch x := v.(type) {
	case map[string]any:
		return toSafeJSON(x[k], 0)
	case *safeArray:
		if k == "length" {
			return float64(len(x.values)), nil
		}
	case string:
		if k == "length" {
			return float64(len(x)), nil
		}
		if k == "match" {
			return safeBuiltin("string.match"), nil
		}
	case safeInput:
		if k == "all" || k == "first" {
			return safeBuiltin("input." + k), nil
		}
		if k == "item" {
			if len(x.items) != 1 {
				return nil, fmt.Errorf("$input.item requires exactly one current item")
			}
			item, _ := x.items[0].(map[string]any)
			jsonValue, _ := item["json"].(map[string]any)
			if jsonValue == nil {
				return nil, fmt.Errorf("$input.item requires one JSON object")
			}
			return jsonValue, nil
		}
	case safeNodeLookup:
		if k == "item" {
			if len(x.items) != 1 {
				return nil, fmt.Errorf("prior-node .item requires exactly one linked output item")
			}
			return x.items[0], nil
		}
		if k == "all" || k == "first" {
			return safeBuiltin("node." + k), nil
		}
	case safeExecution:
		if k == "resumeUrl" {
			return x.resumeURL, nil
		}
	case safeDate:
		if k == "toISOString" || k == "toISO" {
			return safeBuiltin("date." + k), nil
		}
		if k == "getDay" || k == "getDate" || k == "getMonth" || k == "getFullYear" {
			return safeBuiltin("date." + k), nil
		}
	case safeSet:
		if k == "has" {
			return safeBuiltin("set.has"), nil
		}
	case safeRegex:
		if k == "test" {
			return safeBuiltin("regex.test:" + x.pattern), nil
		}
	case safeBuiltin:
		if x == "String" && k == "fromCodePoint" {
			return safeBuiltin("String.fromCodePoint"), nil
		}
		if x == "Object" && k == "keys" {
			return safeBuiltin("Object.keys"), nil
		}
		if x == "JSON" && (k == "stringify" || k == "parse") {
			return safeBuiltin("JSON." + k), nil
		}
		if x == "Array" && k == "from" {
			return safeBuiltin("Array"), nil
		}
		if x == "Array" && k == "isArray" {
			return safeBuiltin("Array.isArray"), nil
		}
		if x == "Math" && (k == "floor" || k == "random") {
			return safeBuiltin("Math"), nil
		}
	}
	return nil, fmt.Errorf("property %q is not allowed on %T", k, v)
}

func safeCall(c *safeCodeContext, receiver any, method string, args []any) (any, error) {
	b, ok := receiver.(safeBuiltin)
	if ok {
		if b == "Object" && method == "keys" && len(args) == 1 {
			return safeObjectKeys(args[0])
		}
		if b == "JSON" {
			return safeJSONCall(method, args)
		}
		switch string(b) {
		case "JSON.stringify":
			if len(args) != 1 {
				return nil, fmt.Errorf("JSON.stringify requires one value")
			}
			encoded, err := json.Marshal(args[0])
			if err != nil || len(encoded) > maxSafeCodeItemBytes {
				return nil, fmt.Errorf("JSON.stringify value exceeds the bounded JSON size")
			}
			return string(encoded), nil
		case "JSON.parse":
			if len(args) != 1 {
				return nil, fmt.Errorf("JSON.parse requires one string")
			}
			raw, ok := args[0].(string)
			if !ok || len(raw) > maxSafeCodeItemBytes {
				return nil, fmt.Errorf("JSON.parse requires a bounded JSON string")
			}
			var decoded any
			decoder := json.NewDecoder(strings.NewReader(raw))
			decoder.UseNumber()
			if err := decoder.Decode(&decoded); err != nil {
				return nil, err
			}
			if err := ensureJSONEnd(decoder); err != nil {
				return nil, err
			}
			return toSafeJSON(decoded, 0)
		case "String":
			if method == "call" {
				if len(args) != 1 {
					return nil, fmt.Errorf("String requires one argument")
				}
				return fmt.Sprint(args[0]), nil
			}
		case "input.all":
			if len(args) != 0 {
				return nil, fmt.Errorf("$input.all takes no arguments")
			}
			return safeInputItems{}, nil
		case "input.first":
			if len(args) != 0 {
				return nil, fmt.Errorf("$input.first takes no arguments")
			}
			return safeBuiltin("input.first-result"), nil
		case "date.toISOString":
			return "", fmt.Errorf("date receiver lost")
		}
		if strings.HasPrefix(string(b), "regex.test:") {
			if len(args) != 1 {
				return nil, fmt.Errorf("regex.test requires one argument")
			}
			r, e := regexp.Compile(strings.TrimPrefix(string(b), "regex.test:"))
			if e != nil {
				return nil, e
			}
			return r.MatchString(fmt.Sprint(args[0])), nil
		}
	}
	switch x := receiver.(type) {
	case safeDate:
		if method == "toISOString" && len(args) == 0 {
			return x.at.Format("2006-01-02T15:04:05.000Z"), nil
		}
		if method == "toISO" && len(args) == 0 {
			return x.at.UTC().Format(time.RFC3339Nano), nil
		}
		if len(args) == 0 {
			switch method {
			case "getDay":
				return float64(x.at.Weekday()), nil
			case "getDate":
				return float64(x.at.Day()), nil
			case "getMonth":
				return float64(x.at.Month() - 1), nil
			case "getFullYear":
				return float64(x.at.Year()), nil
			}
		}
	case safeDateTime:
		if method == "toFormat" && len(args) == 1 {
			format, ok := args[0].(string)
			if !ok || format != "yyyy-MM-dd HH:mm:ss" {
				return nil, fmt.Errorf("DateTime.toFormat supports only the bounded yyyy-MM-dd HH:mm:ss format")
			}
			return x.at.UTC().Format("2006-01-02 15:04:05"), nil
		}
		if method == "diff" && len(args) == 2 {
			other, ok := args[0].(safeDateTime)
			units, unitsOK := args[1].(*safeArray)
			if !ok || !unitsOK || len(units.values) != 3 || units.values[0] != "hours" || units.values[1] != "minutes" || units.values[2] != "seconds" {
				return nil, fmt.Errorf("DateTime.diff supports only another ISO date and ordered hours, minutes, seconds units")
			}
			seconds := x.at.Sub(other.at).Seconds()
			hours := math.Trunc(seconds / 3600)
			seconds -= hours * 3600
			minutes := math.Trunc(seconds / 60)
			seconds -= minutes * 60
			return map[string]any{"hours": hours, "minutes": minutes, "seconds": seconds}, nil
		}
	case safeSet:
		if method == "has" && len(args) == 1 {
			key, ok := safeSetKey(args[0])
			if !ok {
				return false, nil
			}
			_, exists := x.values[key]
			return exists, nil
		}
	case safeInput:
		if method == "all" && len(args) == 0 {
			return &safeArray{values: x.items}, nil
		}
		if method == "first" && len(args) == 0 {
			if len(x.items) == 0 {
				return nil, fmt.Errorf("$input.first requires at least one input item")
			}
			return x.items[0], nil
		}
	case safeNodeLookup:
		if method == "all" && len(args) == 0 {
			return boundedSafeArray(x.items)
		}
		if method == "item" && len(args) == 0 {
			if len(x.items) != 1 {
				return nil, fmt.Errorf("prior-node .item requires exactly one linked output item")
			}
			return x.items[0], nil
		}
	case safeRegex:
		if method == "test" && len(args) == 1 {
			r, e := compileSafeRegex(x)
			if e != nil {
				return nil, e
			}
			return r.MatchString(fmt.Sprint(args[0])), nil
		}
	case string:
		if method == "match" && len(args) == 1 {
			pattern, ok := args[0].(safeRegex)
			if !ok || pattern.global {
				return nil, fmt.Errorf("match requires one bounded non-global approved regex")
			}
			regex, err := compileSafeRegex(pattern)
			if err != nil {
				return nil, err
			}
			matched := regex.FindString(x)
			if matched == "" {
				return nil, nil
			}
			return boundedSafeArray([]any{matched})
		}
		if method == "indexOf" && len(args) == 1 {
			needle, ok := args[0].(string)
			if !ok {
				return float64(-1), nil
			}
			return float64(strings.Index(x, needle)), nil
		}
		if method == "replace" && len(args) == 2 {
			if callback, ok := args[1].(*codeExpr); ok && callback.op == "lambda" {
				pattern, ok := args[0].(safeRegex)
				if !ok {
					return nil, fmt.Errorf("replace callback requires an approved bounded regex")
				}
				return safeRegexReplaceCallback(x, pattern, callback, c)
			}
			replacement, ok := args[1].(string)
			if !ok || len(replacement) > maxSafeCodeItemBytes {
				return nil, fmt.Errorf("replace requires one bounded string replacement")
			}
			switch pattern := args[0].(type) {
			case string:
				if pattern == "" || len(pattern) > 256 {
					return nil, fmt.Errorf("replace requires a nonempty bounded literal search")
				}
				if index := strings.Index(x, pattern); index >= 0 {
					x = x[:index] + replacement + x[index+len(pattern):]
				}
			case safeRegex:
				re, err := compileSafeRegex(pattern)
				if err != nil {
					return nil, err
				}
				if pattern.global {
					x = re.ReplaceAllString(x, replacement)
				} else if index := re.FindStringIndex(x); index != nil {
					x = x[:index[0]] + replacement + x[index[1]:]
				}
			default:
				return nil, fmt.Errorf("replace search must be a bounded literal or approved regex")
			}
			if len(x) > maxSafeCodeItemBytes {
				return nil, fmt.Errorf("replace output item bound exceeded")
			}
			return x, nil
		}
		if len(args) == 0 {
			switch method {
			case "trim":
				return strings.TrimSpace(x), nil
			case "toLowerCase":
				return strings.ToLower(x), nil
			}
		}
		if method == "slice" && len(args) == 2 {
			start, startOK := args[0].(float64)
			end, endOK := args[1].(float64)
			if !startOK || !endOK || start != 0 || end < 0 || end > maxSafeCodeSliceUnits || end != float64(int(end)) {
				return nil, fmt.Errorf("slice supports only slice(0, boundedLength)")
			}
			units := utf16.Encode([]rune(x))
			if int(end) < len(units) {
				if int(end) > 0 && units[int(end)-1] >= 0xD800 && units[int(end)-1] <= 0xDBFF && units[int(end)] >= 0xDC00 && units[int(end)] <= 0xDFFF {
					return nil, fmt.Errorf("slice boundary splits a Unicode surrogate pair; unsupported safe string output")
				}
				units = units[:int(end)]
			}
			return string(utf16.Decode(units)), nil
		}
		if method == "split" && len(args) == 1 {
			separator, ok := args[0].(string)
			if !ok || separator == "" {
				return nil, fmt.Errorf("split requires one nonempty literal separator")
			}
			parts := strings.Split(x, separator)
			if len(parts) > maxSyntheticItems {
				return nil, fmt.Errorf("split item bound exceeded")
			}
			values := make([]any, len(parts))
			for i, part := range parts {
				values[i] = part
			}
			return boundedSafeArray(values)
		}
	case *safeArray:
		if method == "includes" && len(args) == 1 {
			for _, item := range x.values {
				if sameSafePrimitive(item, args[0]) {
					return true, nil
				}
			}
			return false, nil
		}
		if method == "push" && len(args) == 1 {
			if len(x.values) >= maxSyntheticItems {
				return nil, fmt.Errorf("array item bound exceeded")
			}
			encoded, err := json.Marshal(args[0])
			if err != nil {
				return nil, fmt.Errorf("encode pushed value: %w", err)
			}
			additional := len(encoded)
			if len(x.values) > 0 {
				additional++
			}
			if x.bytes+additional > maxSafeCodeOutputBytes {
				return nil, fmt.Errorf("output byte bound exceeded")
			}
			x.bytes += additional
			x.values = append(x.values, args[0])
			return float64(len(x.values)), nil
		}
		if method == "join" && len(args) == 1 {
			return strings.Join(anyStrings(x.values), fmt.Sprint(args[0])), nil
		}
	case safeBuiltin:
		if x == "String" && method == "fromCodePoint" && len(args) == 1 {
			number, ok := numberValue(args[0])
			if !ok || number < 0 || number > 0x10ffff || number != math.Trunc(number) || number >= 0xd800 && number <= 0xdfff {
				return nil, fmt.Errorf("String.fromCodePoint requires one Unicode scalar")
			}
			var result strings.Builder
			result.WriteRune(rune(number))
			return result.String(), nil
		}
		if x == "String" && method == "call" && len(args) == 1 {
			return fmt.Sprint(args[0]), nil
		}
	}
	return nil, fmt.Errorf("call %q on %T is not allowed", method, receiver)
}

func safeRegexReplaceCallback(input string, pattern safeRegex, callback *codeExpr, context *safeCodeContext) (string, error) {
	regex, err := compileSafeRegex(pattern)
	if err != nil {
		return "", err
	}
	limit := -1
	if pattern.global {
		limit = maxSyntheticItems
	}
	matches := regex.FindAllStringSubmatchIndex(input, limit)
	if len(matches) > maxSyntheticItems {
		return "", fmt.Errorf("regex replacement match bound exceeded")
	}
	var output strings.Builder
	position := 0
	params := safeLambdaParameters(callback)
	for _, match := range matches {
		if len(match) < 2 || match[0] < position {
			continue
		}
		output.WriteString(input[position:match[0]])
		callbackArgs := make([]any, len(params))
		for index := range params {
			switch {
			case index == 0:
				callbackArgs[index] = input[match[0]:match[1]]
			case index*2 < len(match):
				start, end := match[index*2], match[index*2+1]
				if start < 0 || end < 0 {
					callbackArgs[index] = ""
				} else {
					callbackArgs[index] = input[start:end]
				}
			case index == len(match)/2:
				callbackArgs[index] = float64(match[0])
			case index == len(match)/2+1:
				callbackArgs[index] = input
			default:
				callbackArgs[index] = nil
			}
		}
		replacement, callbackErr := context.evalLambdaArguments(callback, callbackArgs)
		if callbackErr != nil {
			return "", callbackErr
		}
		output.WriteString(jsString(replacement))
		position = match[1]
		if output.Len() > maxSafeCodeItemBytes {
			return "", fmt.Errorf("replace output item bound exceeded")
		}
	}
	output.WriteString(input[position:])
	if output.Len() > maxSafeCodeItemBytes {
		return "", fmt.Errorf("replace output item bound exceeded")
	}
	return output.String(), nil
}

func safeJSONCall(method string, args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("JSON.%s requires one argument", method)
	}
	switch method {
	case "stringify":
		encoded, err := json.Marshal(args[0])
		if err != nil || len(encoded) > maxSafeCodeItemBytes {
			return nil, fmt.Errorf("JSON.stringify value exceeds the bounded JSON size")
		}
		return string(encoded), nil
	case "parse":
		raw, ok := args[0].(string)
		if !ok || len(raw) > maxSafeCodeItemBytes {
			return nil, fmt.Errorf("JSON.parse requires a bounded JSON string")
		}
		var decoded any
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&decoded); err != nil {
			return nil, err
		}
		if err := ensureJSONEnd(decoder); err != nil {
			return nil, err
		}
		return toSafeJSON(decoded, 0)
	default:
		return nil, fmt.Errorf("JSON method %q is not allowed", method)
	}
}

type safeInputItems struct{}

func (c *safeCodeContext) callLocalFunction(fn safeLocalFunction, args []any) (any, error) {
	if len(args) != len(fn.params) {
		return nil, fmt.Errorf("function %s requires %d arguments", fn.name, len(fn.params))
	}
	if c.callDepth >= 64 {
		return nil, fmt.Errorf("local function call depth exceeds 64")
	}
	previous := c.locals
	previousReturnValue := c.returnValue
	c.returnValue = nil
	base := c.rootLocals
	if base == nil {
		base = previous
	}
	next := make(map[string]any, len(base)+len(args))
	for k, v := range base {
		next[k] = v
	}
	c.locals = next
	c.callDepth++
	defer func() { c.callDepth--; c.locals = previous }()
	for i, name := range fn.params {
		c.locals[name] = args[i]
	}
	value, err := c.run(fn.body)
	if err == errSafeReturn {
		value = c.returnValue
		err = nil
	}
	c.returnValue = previousReturnValue
	return value, err
}

func (c *safeCodeContext) evalLambda(lambda *codeExpr, value any) (any, error) {
	return c.evalLambdaArguments(lambda, []any{value})
}

func (c *safeCodeContext) evalLambdaArguments(lambda *codeExpr, values []any) (any, error) {
	if lambda == nil || lambda.op != "lambda" {
		return nil, fmt.Errorf("callback is not a bounded arrow expression")
	}
	params := safeLambdaParameters(lambda)
	if len(params) != len(values) {
		return nil, fmt.Errorf("callback argument count does not match bounded parameters")
	}
	previousLocals := c.locals
	previousReturn := c.returnValue
	nextLocals := make(map[string]any, len(previousLocals)+len(params))
	for key, value := range previousLocals {
		nextLocals[key] = value
	}
	c.locals = nextLocals
	for index, name := range params {
		if !safeCodeIdentifier(name) {
			c.locals = previousLocals
			return nil, fmt.Errorf("callback parameter is invalid")
		}
		c.locals[name] = values[index]
	}
	defer func() {
		c.locals = previousLocals
		c.returnValue = previousReturn
	}()
	var result any
	var err error
	if lambda.left != nil {
		result, err = c.eval(lambda.left)
	} else {
		c.returnValue = nil
		result, err = c.run(lambda.body)
		if err == errSafeReturn {
			result = c.returnValue
			err = nil
		}
	}
	return result, err
}

func safeLambdaParameters(lambda *codeExpr) []string {
	if lambda == nil || lambda.op != "lambda" {
		return nil
	}
	if len(lambda.params) > 0 {
		return lambda.params
	}
	name, ok := lambda.value.(string)
	if !ok {
		return nil
	}
	return []string{name}
}

func (c *safeCodeContext) safeArrayFrom(args []any) (*safeArray, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Array.from requires a static length and callback")
	}
	descriptor, ok := args[0].(map[string]any)
	if !ok || len(descriptor) != 1 {
		return nil, fmt.Errorf("Array.from supports only a one-field length descriptor")
	}
	length, ok := numberValue(descriptor["length"])
	if !ok || length < 0 || length > maxSyntheticItems || length != math.Trunc(length) {
		return nil, fmt.Errorf("Array.from length is outside the %d-item integer bound", maxSyntheticItems)
	}
	callback, ok := args[1].(*codeExpr)
	if !ok || callback.op != "lambda0" {
		return nil, fmt.Errorf("Array.from callback must be a zero-argument function")
	}
	values := make([]any, 0, int(length))
	for i := 0; i < int(length); i++ {
		if err := c.tick(); err != nil {
			return nil, err
		}
		var value any
		var err error
		if callback.left != nil {
			value, err = c.eval(callback.left)
		} else {
			value, err = c.run(callback.body)
		}
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return boundedSafeArray(values)
}

func numberValue(value any) (float64, bool) {
	switch n := value.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	default:
		return 0, false
	}
}

func parseSafeISODate(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("DateTime.fromISO value is not a bounded ISO timestamp")
}

func boundedSafeArray(values []any) (*safeArray, error) {
	if len(values) > maxSyntheticItems {
		return nil, fmt.Errorf("array item bound exceeded")
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("encode array result: %w", err)
	}
	if len(encoded) > maxSafeCodeOutputBytes {
		return nil, fmt.Errorf("array output byte bound exceeded")
	}
	return &safeArray{values: values, bytes: len(encoded)}, nil
}

func sameSafePrimitive(left, right any) bool {
	switch l := left.(type) {
	case nil:
		return right == nil
	case string:
		r, ok := right.(string)
		return ok && l == r
	case bool:
		r, ok := right.(bool)
		return ok && l == r
	case float64:
		r, ok := right.(float64)
		return ok && l == r
	default:
		return false
	}
}

func isStaticNamedNodeLookup(expr *codeExpr) bool {
	if expr == nil || expr.op != "invoke" || expr.left.op != "var" || expr.left.value != "$" || len(expr.args) != 1 || expr.args[0].op != "lit" {
		return false
	}
	name, ok := expr.args[0].value.(string)
	return ok && safeCodeNodeName(name)
}

func isSafeNamedNodeLookup(expr *codeExpr, scope map[string]bool) bool {
	if isStaticNamedNodeLookup(expr) {
		return true
	}
	if expr == nil || expr.op != "invoke" || expr.left.op != "var" || expr.left.value != "$" || len(expr.args) != 1 || expr.args[0].op != "var" {
		return false
	}
	name, ok := expr.args[0].value.(string)
	return ok && scopeHas(scope, name)
}

func safeCodeNodeName(name string) bool {
	return len(name) > 0 && len(name) <= 128 && !strings.ContainsAny(name, "\x00\r\n")
}

func safeCodeIdentifier(name string) bool {
	if name == "" || !isIdentStart(name[0]) {
		return false
	}
	for i := 1; i < len(name); i++ {
		if !isIdent(name[i]) {
			return false
		}
	}
	return len(name) <= 128
}

func safeCodePropertyName(name string) bool {
	if !safeCodeIdentifier(name) {
		return false
	}
	switch name {
	case "__proto__", "prototype", "constructor", "caller", "callee", "arguments":
		return false
	default:
		return true
	}
}

func safeJSONPropertyName(name string) bool {
	if name == "" || len(name) > 128 || strings.ContainsAny(name, "\x00\r\n") {
		return false
	}
	switch name {
	case "__proto__", "prototype", "constructor", "caller", "callee", "arguments":
		return false
	default:
		return true
	}
}

func anyStrings(a []any) []string {
	r := make([]string, len(a))
	for i, v := range a {
		r[i] = fmt.Sprint(v)
	}
	return r
}

func jsString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func truthy(v any) bool {
	if v == nil || v == false {
		return false
	}
	if s, ok := v.(string); ok {
		return s != ""
	}
	if n, ok := v.(float64); ok {
		return n != 0
	}
	return true
}

const maxSafeCodeSliceUnits = 4096

func safeEmailRegex(pattern string) bool {
	return pattern == "^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$" || pattern == "^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$"
}

func safeCodeRegex(pattern, flags string) bool {
	if safeEmailRegex(pattern) {
		return flags == ""
	}
	if pattern == `^https?:\/\/[^/]+` {
		return flags == ""
	}
	if pattern == "&" || pattern == "<" || pattern == ">" {
		return flags == "" || flags == "g"
	}
	if pattern == "-" {
		return flags == "g"
	}
	switch pattern {
	case `<!--[\s\S]*?-->`, `<[^>]+>`, `&nbsp;`, `&lt;`, `&gt;`, `&quot;`, `&amp;`, `&#(\d+);`, `&#8217;`, `&#8211;`:
		return flags == "g" || flags == ""
	case `<(script|style)[\s\S]*?<\/\1>`, `<br\s*\/?>`, `<\/?(p|div|li|ul|ol|h[1-6]|tr|td|th|table|blockquote|section|article|figure|figcaption)(\s[^>]*)?>`, `&#x([0-9a-f]+);`:
		return flags == "gi"
	case `\s+`:
		return flags == "g"
	default:
		return false
	}
}

func compileSafeRegex(pattern safeRegex) (*regexp.Regexp, error) {
	goPattern := strings.ReplaceAll(pattern.pattern, `\/`, `/`)
	switch pattern.pattern {
	case `<(script|style)[\s\S]*?<\/\1>`:
		goPattern = `<script[\s\S]*?</script>|<style[\s\S]*?</style>`
	case `<br\s*\/?>`:
		goPattern = `<br\s*/?>`
	case `<\/?(p|div|li|ul|ol|h[1-6]|tr|td|th|table|blockquote|section|article|figure|figcaption)(\s[^>]*)?>`:
		goPattern = `</?(p|div|li|ul|ol|h[1-6]|tr|td|th|table|blockquote|section|article|figure|figcaption)(\s[^>]*)?>`
	}
	if pattern.ignoreCase {
		goPattern = "(?i)" + goPattern
	}
	return regexp.Compile(goPattern)
}

func isNumericLiteral(expr *codeExpr, expected float64) bool {
	if expr == nil || expr.op != "lit" {
		return false
	}
	value, ok := expr.value.(float64)
	return ok && value == expected
}

func safeObjectKeys(value any) (*safeArray, error) {
	keys := make([]string, 0, 16)
	switch object := value.(type) {
	case map[string]any:
		if len(object) > 4096 {
			return nil, fmt.Errorf("Object.keys input exceeds the 4096-property bound")
		}
		for key := range object {
			keys = append(keys, key)
		}
		sort.Strings(keys)
	case []any:
		if len(object) > maxSyntheticItems {
			return nil, fmt.Errorf("Object.keys array exceeds the item bound")
		}
		for i := range object {
			keys = append(keys, strconv.Itoa(i))
		}
	case *safeArray:
		if len(object.values) > maxSyntheticItems {
			return nil, fmt.Errorf("Object.keys array exceeds the item bound")
		}
		for i := range object.values {
			keys = append(keys, strconv.Itoa(i))
		}
	default:
		return nil, fmt.Errorf("Object.keys requires a bounded object or array, got %T", value)
	}
	values := make([]any, len(keys))
	for i, key := range keys {
		values[i] = key
	}
	return boundedSafeArray(values)
}

func safeIndexNumber(value any) (float64, bool) {
	if text, ok := value.(string); ok {
		if text == "" || strings.Trim(text, "0123456789") != "" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(text, 64)
		return parsed, err == nil
	}
	return numberValue(value)
}
