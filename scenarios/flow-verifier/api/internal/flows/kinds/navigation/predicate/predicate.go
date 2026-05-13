// Package predicate parses and evaluates the navigation predicate DSL.
//
// Grammar (case-sensitive keywords; uppercase):
//
//	expr        := or_expr
//	or_expr     := and_expr ('OR' and_expr)*
//	and_expr    := unary ('AND' unary)*
//	unary       := 'NOT' unary | primary
//	primary     := '(' expr ')' | comparison
//	comparison  := IDENT op rhs
//	op          := '=' | '!=' | 'IN' | 'CONTAINS'
//	rhs         := list | atom               (atom for =, !=, CONTAINS)
//	list        := '[' atom (',' atom)* ']'  (only after IN)
//	atom        := IDENT ('=' IDENT)?        (the suffix supports CONTAINS rhs
//	                                          like `auth=logged_in`)
//
// The Eval function takes a Lookup func mapping identifier → string and
// returns the boolean evaluation. CONTAINS does a substring match on the
// raw string value the lookup returns (used by deep-link policy:
// `requires CONTAINS auth=logged_in`).
package predicate

import (
	"fmt"
	"strings"
)

// Predicate is a parsed expression ready to evaluate.
type Predicate struct {
	node node
	raw  string
}

// Raw returns the source text the Predicate was parsed from.
func (p Predicate) Raw() string { return p.raw }

// Lookup resolves a bare identifier to a string. Unknown identifiers
// return ("", false); the caller decides whether that is an error.
type Lookup func(name string) (string, bool)

// Eval evaluates the predicate against the given lookup.
func (p Predicate) Eval(lookup Lookup) (bool, error) {
	if p.node == nil {
		return true, nil
	}
	return p.node.eval(lookup)
}

// Parse parses src into a Predicate. Empty src yields an always-true
// Predicate; that's the same as omitting the field.
func Parse(src string) (Predicate, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return Predicate{}, nil
	}
	tokens, err := tokenize(src)
	if err != nil {
		return Predicate{}, fmt.Errorf("predicate %q: %w", src, err)
	}
	p := &parser{tokens: tokens}
	root, err := p.parseExpr()
	if err != nil {
		return Predicate{}, fmt.Errorf("predicate %q: %w", src, err)
	}
	if p.pos != len(tokens) {
		return Predicate{}, fmt.Errorf("predicate %q: trailing tokens at %q", src, tokens[p.pos].text)
	}
	return Predicate{node: root, raw: src}, nil
}

// ── AST ────────────────────────────────────────────────────────────────

type node interface {
	eval(lookup Lookup) (bool, error)
}

type orNode struct{ left, right node }

func (n orNode) eval(l Lookup) (bool, error) {
	a, err := n.left.eval(l)
	if err != nil {
		return false, err
	}
	if a {
		return true, nil
	}
	return n.right.eval(l)
}

type andNode struct{ left, right node }

func (n andNode) eval(l Lookup) (bool, error) {
	a, err := n.left.eval(l)
	if err != nil {
		return false, err
	}
	if !a {
		return false, nil
	}
	return n.right.eval(l)
}

type notNode struct{ inner node }

func (n notNode) eval(l Lookup) (bool, error) {
	v, err := n.inner.eval(l)
	return !v, err
}

type cmpNode struct {
	ident string
	op    string
	rhs   string
	list  []string
}

func (n cmpNode) eval(lookup Lookup) (bool, error) {
	v, ok := lookup(n.ident)
	if !ok {
		return false, fmt.Errorf("unknown identifier %q", n.ident)
	}
	switch n.op {
	case "=":
		return v == n.rhs, nil
	case "!=":
		return v != n.rhs, nil
	case "CONTAINS":
		return strings.Contains(v, n.rhs), nil
	case "IN":
		for _, item := range n.list {
			if v == item {
				return true, nil
			}
		}
		return false, nil
	}
	return false, fmt.Errorf("unsupported operator %q", n.op)
}

// ── Tokenizer ──────────────────────────────────────────────────────────

type tokKind int

const (
	tokIdent tokKind = iota
	tokOp
	tokKeyword
	tokLParen
	tokRParen
	tokLBracket
	tokRBracket
	tokComma
)

type token struct {
	kind tokKind
	text string
}

var keywords = map[string]bool{
	"AND": true, "OR": true, "NOT": true, "IN": true, "CONTAINS": true,
}

func tokenize(src string) ([]token, error) {
	var out []token
	i := 0
	isWord := func(b byte) bool {
		return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') || b == '_' || b == '-' || b == '.'
	}
	for i < len(src) {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n':
			i++
		case c == '(':
			out = append(out, token{tokLParen, "("})
			i++
		case c == ')':
			out = append(out, token{tokRParen, ")"})
			i++
		case c == '[':
			out = append(out, token{tokLBracket, "["})
			i++
		case c == ']':
			out = append(out, token{tokRBracket, "]"})
			i++
		case c == ',':
			out = append(out, token{tokComma, ","})
			i++
		case c == '=':
			out = append(out, token{tokOp, "="})
			i++
		case c == '!':
			if i+1 < len(src) && src[i+1] == '=' {
				out = append(out, token{tokOp, "!="})
				i += 2
			} else {
				return nil, fmt.Errorf("unexpected %q", c)
			}
		case isWord(c):
			j := i
			for j < len(src) && isWord(src[j]) {
				j++
			}
			word := src[i:j]
			if keywords[word] {
				out = append(out, token{tokKeyword, word})
			} else {
				out = append(out, token{tokIdent, word})
			}
			i = j
		default:
			return nil, fmt.Errorf("unexpected character %q", c)
		}
	}
	return out, nil
}

// ── Parser ─────────────────────────────────────────────────────────────

type parser struct {
	tokens []token
	pos    int
}

func (p *parser) peek() (token, bool) {
	if p.pos >= len(p.tokens) {
		return token{}, false
	}
	return p.tokens[p.pos], true
}

func (p *parser) consume() token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func (p *parser) parseExpr() (node, error) { return p.parseOr() }

func (p *parser) parseOr() (node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for {
		t, ok := p.peek()
		if !ok || !(t.kind == tokKeyword && t.text == "OR") {
			return left, nil
		}
		p.consume()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = orNode{left, right}
	}
}

func (p *parser) parseAnd() (node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		t, ok := p.peek()
		if !ok || !(t.kind == tokKeyword && t.text == "AND") {
			return left, nil
		}
		p.consume()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = andNode{left, right}
	}
}

func (p *parser) parseUnary() (node, error) {
	t, ok := p.peek()
	if !ok {
		return nil, fmt.Errorf("expected expression")
	}
	if t.kind == tokKeyword && t.text == "NOT" {
		p.consume()
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return notNode{inner}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (node, error) {
	t, ok := p.peek()
	if !ok {
		return nil, fmt.Errorf("expected primary")
	}
	if t.kind == tokLParen {
		p.consume()
		inner, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		closeTok, ok := p.peek()
		if !ok || closeTok.kind != tokRParen {
			return nil, fmt.Errorf("expected ')'")
		}
		p.consume()
		return inner, nil
	}
	return p.parseComparison()
}

func (p *parser) parseComparison() (node, error) {
	t, ok := p.peek()
	if !ok || t.kind != tokIdent {
		return nil, fmt.Errorf("expected identifier")
	}
	p.consume()
	op, ok := p.peek()
	if !ok {
		return nil, fmt.Errorf("expected operator after %q", t.text)
	}
	switch {
	case op.kind == tokOp:
		p.consume()
		rhs, err := p.parseAtom()
		if err != nil {
			return nil, err
		}
		return cmpNode{ident: t.text, op: op.text, rhs: rhs}, nil
	case op.kind == tokKeyword && op.text == "IN":
		p.consume()
		items, err := p.parseList()
		if err != nil {
			return nil, err
		}
		return cmpNode{ident: t.text, op: "IN", list: items}, nil
	case op.kind == tokKeyword && op.text == "CONTAINS":
		p.consume()
		rhs, err := p.parseAtom()
		if err != nil {
			return nil, err
		}
		return cmpNode{ident: t.text, op: "CONTAINS", rhs: rhs}, nil
	}
	return nil, fmt.Errorf("expected operator after %q, got %q", t.text, op.text)
}

// parseAtom reads an identifier with an optional `=ident` suffix. The
// suffix lets `CONTAINS auth=logged_in` parse as a single rhs string.
func (p *parser) parseAtom() (string, error) {
	t, ok := p.peek()
	if !ok || t.kind != tokIdent {
		return "", fmt.Errorf("expected value")
	}
	p.consume()
	out := t.text
	next, ok := p.peek()
	if ok && next.kind == tokOp && next.text == "=" {
		p.consume()
		val, ok := p.peek()
		if !ok || val.kind != tokIdent {
			return "", fmt.Errorf("expected value after '='")
		}
		p.consume()
		out = out + "=" + val.text
	}
	return out, nil
}

func (p *parser) parseList() ([]string, error) {
	t, ok := p.peek()
	if !ok || t.kind != tokLBracket {
		return nil, fmt.Errorf("expected '['")
	}
	p.consume()
	var items []string
	for {
		item, err := p.parseAtom()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		next, ok := p.peek()
		if !ok {
			return nil, fmt.Errorf("expected ',' or ']'")
		}
		if next.kind == tokRBracket {
			p.consume()
			return items, nil
		}
		if next.kind != tokComma {
			return nil, fmt.Errorf("expected ',' or ']', got %q", next.text)
		}
		p.consume()
	}
}
