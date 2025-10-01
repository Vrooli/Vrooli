package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"html"
	"strconv"
	"strings"
)

/*
Rule: Database Backoff With Jitter
Description: Database connection retries must use exponential backoff with jitter
Reason: Prevents thundering herd issues when services reconnect to stateful stores
Category: api
Severity: high
Standard: service-reliability-v1
Targets: api

<test-case id="postgres-backoff-real-jitter" should-fail="false">
  <description>Proper exponential backoff with REAL random jitter for PostgreSQL</description>
  <input language="go">
package main

import (
    "database/sql"
    "fmt"
    "math"
    "math/rand"
    "time"
)

var db *sql.DB

func connectWithBackoff() error {
    db, _ = sql.Open("postgres", "dsn")

    maxRetries := 5
    baseDelay := 200 * time.Millisecond
    maxDelay := 5 * time.Second

    for attempt := 0; attempt &lt; maxRetries; attempt++ {
        if err := db.Ping(); err == nil {
            return nil
        }

        delay := time.Duration(math.Min(float64(baseDelay)*math.Pow(2, float64(attempt)), float64(maxDelay)))
        jitterRange := float64(delay) * 0.25
        jitter := time.Duration(jitterRange * rand.Float64())
        actualDelay := delay + jitter
        time.Sleep(actualDelay)
    }

    return fmt.Errorf("database unavailable after retries")
}
  </input>
</test-case>

<test-case id="postgres-backoff-fake-jitter" should-fail="true">
  <description>REJECT: Deterministic "fake jitter" that doesn't prevent thundering herd</description>
  <input language="go">
package main

import (
    "database/sql"
    "fmt"
    "math"
    "time"
)

var db *sql.DB

func connectWithFakeJitter() error {
    db, _ = sql.Open("postgres", "dsn")

    maxRetries := 5
    baseDelay := 200 * time.Millisecond
    maxDelay := 5 * time.Second

    for attempt := 0; attempt &lt; maxRetries; attempt++ {
        if err := db.Ping(); err == nil {
            return nil
        }

        delay := time.Duration(math.Min(float64(baseDelay)*math.Pow(2, float64(attempt)), float64(maxDelay)))
        jitterRange := float64(delay) * 0.15
        jitter := time.Duration(jitterRange * (float64(attempt+1) / float64(maxRetries)))
        actualDelay := delay + jitter
        time.Sleep(actualDelay)
    }

    return fmt.Errorf("database unavailable after retries")
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>jitter</expected-message>
</test-case>

<test-case id="missing-jitter" should-fail="true">
  <description>Backoff loop without jitter</description>
  <input language="go">
package main

import (
    "database/sql"
    "fmt"
    "math"
    "time"
)

var db *sql.DB

func connectWithoutJitter() error {
    db, _ = sql.Open("postgres", "dsn")

    baseDelay := 250 * time.Millisecond
    maxDelay := 5 * time.Second

    for attempt := 0; attempt &lt; 5; attempt++ {
        if err := db.Ping(); err == nil {
            return nil
        }

        delay := time.Duration(math.Min(float64(baseDelay)*math.Pow(2, float64(attempt)), float64(maxDelay)))
        time.Sleep(delay)
    }

    return fmt.Errorf("database unavailable")
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>jitter</expected-message>
</test-case>

<test-case id="missing-exponential" should-fail="true">
  <description>Retry loop without exponential growth (has fake jitter too)</description>
  <input language="go">
package main

import (
    "database/sql"
    "fmt"
    "time"
)

var db *sql.DB

func connectWithLinearDelay() error {
    db, _ = sql.Open("postgres", "dsn")

    baseDelay := 500 * time.Millisecond

    for attempt := 0; attempt &lt; 5; attempt++ {
        if err := db.Ping(); err == nil {
            return nil
        }

        jitter := time.Duration(float64(baseDelay) * 0.1)
        time.Sleep(baseDelay + jitter)
    }

    return fmt.Errorf("database unavailable")
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>exponential</expected-message>
</test-case>

<test-case id="missing-sleep" should-fail="true">
  <description>Retry loop without sleep between attempts</description>
  <input language="go">
package main

import (
    "database/sql"
    "fmt"
)

var db *sql.DB

func connectWithoutSleep() error {
    db, _ = sql.Open("postgres", "dsn")

    for attempt := 0; attempt &lt; 5; attempt++ {
        if err := db.Ping(); err == nil {
            return nil
        }
    }

    return fmt.Errorf("database unavailable")
}
  </input>
  <expected-violations>3</expected-violations>
  <expected-message>Sleep</expected-message>
</test-case>

<test-case id="time-based-jitter" should-fail="false">
  <description>Time-based jitter using UnixNano() for pseudo-randomness</description>
  <input language="go">
package main

import (
    "database/sql"
    "fmt"
    "math"
    "time"
)

var db *sql.DB

func connectWithTimeJitter() error {
    db, _ = sql.Open("postgres", "dsn")

    maxRetries := 5
    baseDelay := 200 * time.Millisecond
    maxDelay := 5 * time.Second

    for attempt := 0; attempt &lt; maxRetries; attempt++ {
        if err := db.Ping(); err == nil {
            return nil
        }

        delay := time.Duration(math.Min(float64(baseDelay)*math.Pow(2, float64(attempt)), float64(maxDelay)))
        jitter := time.Duration(time.Now().UnixNano() % int64(delay))
        actualDelay := delay + jitter
        time.Sleep(actualDelay)
    }

    return fmt.Errorf("database unavailable after retries")
}
  </input>
</test-case>

<test-case id="rand-intn-jitter" should-fail="false">
  <description>Using rand.Intn() for integer-based jitter</description>
  <input language="go">
package main

import (
    "database/sql"
    "fmt"
    "math"
    "math/rand"
    "time"
)

var db *sql.DB

func connectWithIntnJitter() error {
    db, _ = sql.Open("postgres", "dsn")

    maxRetries := 5
    baseDelay := 200 * time.Millisecond
    maxDelay := 5 * time.Second

    for attempt := 0; attempt &lt; maxRetries; attempt++ {
        if err := db.Ping(); err == nil {
            return nil
        }

        delay := time.Duration(math.Min(float64(baseDelay)*math.Pow(2, float64(attempt)), float64(maxDelay)))
        jitter := time.Duration(rand.Intn(int(delay) / 4))
        actualDelay := delay + jitter
        time.Sleep(actualDelay)
    }

    return fmt.Errorf("database unavailable after retries")
}
  </input>
</test-case>

<test-case id="zero-jitter-fake" should-fail="true">
  <description>Variable named 'jitter' but with zero value (fake jitter)</description>
  <input language="go">
package main

import (
    "database/sql"
    "fmt"
    "math"
    "time"
)

var db *sql.DB

func connectWithZeroJitter() error {
    db, _ = sql.Open("postgres", "dsn")

    maxRetries := 5
    baseDelay := 200 * time.Millisecond
    maxDelay := 5 * time.Second

    for attempt := 0; attempt &lt; maxRetries; attempt++ {
        if err := db.Ping(); err == nil {
            return nil
        }

        delay := time.Duration(math.Min(float64(baseDelay)*math.Pow(2, float64(attempt)), float64(maxDelay)))
        jitter := 0 * time.Millisecond
        actualDelay := delay + jitter
        time.Sleep(actualDelay)
    }

    return fmt.Errorf("database unavailable after retries")
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>jitter</expected-message>
</test-case>

*/

// CheckDatabaseBackoff verifies retry loops use exponential backoff with jitter.
func CheckDatabaseBackoff(content []byte, filePath string) []Violation {
	fset := token.NewFileSet()
	source := html.UnescapeString(string(content))
	file, err := parser.ParseFile(fset, filePath, source, 0)
	if err != nil {
		return nil
	}

	var violations []Violation

	ast.Inspect(file, func(n ast.Node) bool {
		loop, ok := n.(*ast.ForStmt)
		if !ok {
			return true
		}

		analysis := analyzeBackoffLoop(loop)
		if !analysis.hasPing {
			return true
		}

		line := fset.Position(loop.For).Line
		if !analysis.hasSleep {
			violations = append(violations, newBackoffViolation(
				filePath,
				line,
				"Database Retry Missing Sleep",
				"Database retry loops must include time.Sleep between connection attempts",
			))
		}
		if !analysis.hasExponent {
			violations = append(violations, newBackoffViolation(
				filePath,
				line,
				"Database Retry Missing Exponential Backoff",
				"Database retry loops must grow delays exponentially (e.g. math.Pow or bit shifts)",
			))
		}
		if !analysis.hasJitter {
			violations = append(violations, newBackoffViolation(
				filePath,
				line,
				"Database Retry Missing Jitter",
				"Database retry loops must add jitter to prevent thundering herd reconnects",
			))
		}

		return true
	})

	return deduplicateBackoffViolations(violations)
}

type backoffLoopInfo struct {
	hasPing     bool
	hasSleep    bool
	hasExponent bool
	hasJitter   bool
	jitterVars  map[string]bool
}

func analyzeBackoffLoop(loop *ast.ForStmt) backoffLoopInfo {
	info := backoffLoopInfo{
		jitterVars: make(map[string]bool),
	}

	ast.Inspect(loop.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			if isPingCall(node) {
				info.hasPing = true
			}
			if isSleepCall(node) {
				info.hasSleep = true
				if len(node.Args) > 0 && exprHasJitter(node.Args[0], info.jitterVars) {
					info.hasJitter = true
				}
			}
			if callHasExponent(node) {
				info.hasExponent = true
			}
		case *ast.AssignStmt:
			processAssignment(node.Lhs, node.Rhs, &info)
		case *ast.ValueSpec:
			processValueSpec(node, &info)
		case *ast.BinaryExpr:
			if node.Op == token.SHL {
				info.hasExponent = true
			}
			if node.Op == token.ADD {
				if exprHasJitter(node.X, info.jitterVars) || exprHasJitter(node.Y, info.jitterVars) {
					info.hasJitter = true
				}
			}
		}
		return true
	})

	return info
}

func processAssignment(lhs []ast.Expr, rhs []ast.Expr, info *backoffLoopInfo) {
	if len(rhs) == 0 {
		return
	}

	for idx, left := range lhs {
		ident, ok := left.(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}

		expr := rhs[0]
		if len(rhs) > 1 {
			if idx < len(rhs) {
				expr = rhs[idx]
			} else {
				continue
			}
		}

		// IMPORTANT: Only track as jitter var if it contains actual randomness
		// Having "jitter" in the name is NOT enough - must have rand.X() or similar
		if exprHasRandomness(expr, info.jitterVars) {
			info.hasJitter = true
			info.jitterVars[ident.Name] = true
		}

		if exprHasExponent(expr, info.jitterVars) {
			info.hasExponent = true
		}
	}
}

func processValueSpec(spec *ast.ValueSpec, info *backoffLoopInfo) {
	for i, name := range spec.Names {
		if name == nil || name.Name == "_" {
			continue
		}

		if i < len(spec.Values) {
			value := spec.Values[i]

			// Only track as jitter var if it contains actual randomness
			if exprHasRandomness(value, info.jitterVars) {
				info.hasJitter = true
				info.jitterVars[name.Name] = true
			}

			if exprHasExponent(value, info.jitterVars) {
				info.hasExponent = true
			}
		}
	}
}

func isPingCall(call *ast.CallExpr) bool {
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		name := strings.ToLower(fn.Sel.Name)
		if strings.HasPrefix(name, "ping") {
			return true
		}
	case *ast.Ident:
		if strings.Contains(strings.ToLower(fn.Name), "ping") {
			return true
		}
	}
	return false
}

func isSleepCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if strings.ToLower(sel.Sel.Name) != "sleep" {
		return false
	}
	if ident, ok := sel.X.(*ast.Ident); ok {
		return strings.ToLower(ident.Name) == "time"
	}
	return false
}

func callHasExponent(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	name := strings.ToLower(sel.Sel.Name)
	if name == "pow" || name == "exp2" {
		if ident, ok := sel.X.(*ast.Ident); ok {
			return strings.ToLower(ident.Name) == "math"
		}
	}
	for _, arg := range call.Args {
		if exprHasExponent(arg, nil) {
			return true
		}
	}
	return false
}

func exprHasExponent(expr ast.Expr, jitterVars map[string]bool) bool {
	switch e := expr.(type) {
	case *ast.CallExpr:
		if callHasExponent(e) {
			return true
		}
	case *ast.BinaryExpr:
		if e.Op == token.SHL {
			return true
		}
		return exprHasExponent(e.X, jitterVars) || exprHasExponent(e.Y, jitterVars)
	case *ast.ParenExpr:
		return exprHasExponent(e.X, jitterVars)
	case *ast.Ident:
		if jitterVars != nil && jitterVars[e.Name] {
			return false
		}
	}
	return false
}

func exprHasJitter(expr ast.Expr, jitterVars map[string]bool) bool {
	return exprHasRandomness(expr, jitterVars)
}

// exprHasRandomness checks if an expression contains actual random number generation
// that would produce different values on each execution (non-deterministic).
// This prevents "fake jitter" patterns like: jitter = delay * (attempt / maxRetries)
func exprHasRandomness(expr ast.Expr, jitterVars map[string]bool) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		// Check if this identifier was assigned from a random source
		if jitterVars != nil && jitterVars[e.Name] {
			return true
		}
	case *ast.CallExpr:
		// Check for direct random function calls
		if isRandomFunctionCall(e) {
			return true
		}
		// Check for time.Now().UnixNano() which can provide randomness
		if isTimeBasedRandomness(e) {
			return true
		}
		// Recursively check arguments
		for _, arg := range e.Args {
			if exprHasRandomness(arg, jitterVars) {
				return true
			}
		}
	case *ast.BinaryExpr:
		// For binary expressions, check if either side contains randomness
		// But REJECT pure deterministic calculations even if they mention "jitter"
		return exprHasRandomness(e.X, jitterVars) || exprHasRandomness(e.Y, jitterVars)
	case *ast.ParenExpr:
		return exprHasRandomness(e.X, jitterVars)
	case *ast.SelectorExpr:
		// Check the base expression for randomness
		return exprHasRandomness(e.X, jitterVars)
	}
	return false
}

// isRandomFunctionCall detects calls to random number generators:
// - rand.Float64(), rand.Intn(), rand.Int63n()
// - crypto/rand functions
// - Custom RNG packages
func isRandomFunctionCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Get package name
	var pkgName string
	if ident, ok := sel.X.(*ast.Ident); ok {
		pkgName = strings.ToLower(ident.Name)
	}

	// Get method name
	methodName := strings.ToLower(sel.Sel.Name)

	// Check for standard random functions
	if strings.Contains(pkgName, "rand") {
		// rand.Float64(), rand.Intn(), rand.Int63n(), etc.
		if strings.HasPrefix(methodName, "float") ||
			strings.HasPrefix(methodName, "int") ||
			strings.HasPrefix(methodName, "uint") {
			return true
		}
	}

	return false
}

// isTimeBasedRandomness detects time.Now().UnixNano() % something
// which can provide pseudo-randomness for jitter
func isTimeBasedRandomness(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	methodName := strings.ToLower(sel.Sel.Name)

	// Check for UnixNano() call
	if methodName == "unixnano" {
		// Verify it's called on something that might be time.Now()
		if innerCall, ok := sel.X.(*ast.CallExpr); ok {
			if innerSel, ok := innerCall.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := innerSel.X.(*ast.Ident); ok {
					return strings.ToLower(ident.Name) == "time"
				}
			}
		}
	}

	return false
}

func newBackoffViolation(filePath string, line int, title, message string) Violation {
	return Violation{
		Type:           "database_backoff",
		Severity:       "high",
		Title:          title,
		Description:    message,
		FilePath:       filePath,
		LineNumber:     line,
		Recommendation: "Implement exponential backoff with jitter before retrying database connections",
		Standard:       "service-reliability-v1",
	}
}

func deduplicateBackoffViolations(items []Violation) []Violation {
	if len(items) == 0 {
		return items
	}
	seen := make(map[string]bool)
	result := make([]Violation, 0, len(items))
	for _, v := range items {
		key := v.Title + "|" + v.FilePath + "|" + strconv.Itoa(v.LineNumber)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, v)
	}
	return result
}
