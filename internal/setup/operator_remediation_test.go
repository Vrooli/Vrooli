package setup

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The rule this test enforces, stated once:
//
//	An operator-facing remediation string must name a command the operator can
//	run inside the Vrooli setup and onboarding flow. `vrooli setup
//	--sudo-mode=ask` is that command: it prompts in process and leaves the rest
//	of the run unelevated. `sudo vrooli setup` is not: it takes the operator
//	outside the flow and runs everything as root.
//
// The check walks syntax trees rather than text, because several packages
// legitimately describe `sudo vrooli` as a mechanism in comments and package
// documentation. Describing how privilege works is not instructing the
// operator, and a text scan cannot tell the two apart.
var operatorRemediationFiles = []string{
	"../../scenarios/vrooli-onboarding/api/v2_apply.go",
	"../privilegebroker/install.go",
	"../volumeremediation/service.go",
	"../volumeremediation/elevated.go",
	"../safeguards/autoheal-recovery-privileges/handler.go",
	"../safeguards/autoheal-watchdog/handler.go",
	"../safeguards/onboarding-apply-privileges/handler.go",
	"../safeguards/tpm-credential-access/handler.go",
	"../hostreqkit/install.go",
	"../cli/projectcli/lifecycle.go",
	"requirements_report.go",
}

// operatorFacingFields names the struct fields whose string values reach an
// operator as an instruction.
var operatorFacingFields = map[string]bool{
	"Remediation":     true,
	"Recovery":        true,
	"OperatorCommand": true,
	"Notes":           true,
	"Reason":          true,
	"hint":            true,
	"command":         true,
}

const forbiddenOperatorInstruction = "sudo vrooli"

func TestOperatorRemediationNamesAnInFlowCommand(t *testing.T) {
	for _, path := range operatorRemediationFiles {
		if findings := operatorInstructionFindings(t, path); len(findings) > 0 {
			t.Errorf("%s instructs the operator outside the flow: %v", filepath.Base(path), findings)
		}
	}
}

// The enforcement is only worth having if it actually fires. Prove it against
// a synthetic source rather than by temporarily breaking a real file.
func TestOperatorRemediationCheckRejectsAReintroducedLiteral(t *testing.T) {
	source := `package sample

type status struct {
	Recovery string
	Detail   string
}

func build() status {
	return status{Recovery: "Re-run sudo vrooli setup to repair it.", Detail: "the sudo vrooli shim resolves"}
}
`
	findings := operatorInstructionFindingsIn(t, "sample.go", source)
	if len(findings) != 1 {
		t.Fatalf("findings = %v, want exactly the Recovery literal", findings)
	}
	if !strings.Contains(findings[0], "Recovery") {
		t.Fatalf("finding = %q, want the Recovery field", findings[0])
	}
}

// A comment describing the mechanism is not an instruction and must not fire.
func TestOperatorRemediationCheckIgnoresComments(t *testing.T) {
	source := `package sample

// When the operator runs sudo vrooli setup the process becomes root.
type status struct{ Recovery string }

func build() status {
	// sudo vrooli setup inherits root's PATH.
	return status{Recovery: "Re-run ` + "`vrooli setup --sudo-mode=ask`" + `."}
}
`
	if findings := operatorInstructionFindingsIn(t, "sample.go", source); len(findings) != 0 {
		t.Fatalf("comments produced findings: %v", findings)
	}
}

func operatorInstructionFindings(t *testing.T, path string) []string {
	t.Helper()
	return operatorInstructionFindingsIn(t, path, nil)
}

func operatorInstructionFindingsIn(t *testing.T, path string, src any) []string {
	t.Helper()
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, src, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	var findings []string
	record := func(field string, node ast.Node) {
		value, ok := stringLiteral(node)
		if !ok || !strings.Contains(value, forbiddenOperatorInstruction) {
			return
		}
		findings = append(findings, field+" at "+fileSet.Position(node.Pos()).String())
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.KeyValueExpr:
			key, ok := typed.Key.(*ast.Ident)
			if !ok || !operatorFacingFields[key.Name] {
				return true
			}
			record(key.Name, typed.Value)
			// A composite value such as a slice of notes carries instructions
			// in its elements rather than in the value itself.
			if composite, isComposite := typed.Value.(*ast.CompositeLit); isComposite {
				for _, element := range composite.Elts {
					record(key.Name, element)
				}
			}
		case *ast.AssignStmt:
			for index, target := range typed.Lhs {
				selector, ok := target.(*ast.SelectorExpr)
				if !ok || !operatorFacingFields[selector.Sel.Name] || index >= len(typed.Rhs) {
					continue
				}
				record(selector.Sel.Name, typed.Rhs[index])
			}
		case *ast.CallExpr:
			// append(status.Notes, "...") and fmt.Sprintf into a Notes append
			// are the two shapes a safeguard uses to instruct the operator.
			if !appendsToOperatorFacingField(typed) {
				return true
			}
			for _, argument := range typed.Args[1:] {
				record("Notes", argument)
				if inner, isCall := argument.(*ast.CallExpr); isCall {
					for _, nested := range inner.Args {
						record("Notes", nested)
					}
				}
				if binary, isBinary := argument.(*ast.BinaryExpr); isBinary {
					record("Notes", binary.X)
					record("Notes", binary.Y)
				}
			}
		}
		return true
	})
	return findings
}

func appendsToOperatorFacingField(call *ast.CallExpr) bool {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok || ident.Name != "append" || len(call.Args) < 2 {
		return false
	}
	selector, ok := call.Args[0].(*ast.SelectorExpr)
	return ok && operatorFacingFields[selector.Sel.Name]
}

func stringLiteral(node ast.Node) (string, bool) {
	switch typed := node.(type) {
	case *ast.BasicLit:
		if typed.Kind != token.STRING {
			return "", false
		}
		value, err := strconv.Unquote(typed.Value)
		if err != nil {
			return typed.Value, true
		}
		return value, true
	case *ast.BinaryExpr:
		left, leftOK := stringLiteral(typed.X)
		right, rightOK := stringLiteral(typed.Y)
		if !leftOK && !rightOK {
			return "", false
		}
		return left + right, true
	}
	return "", false
}
