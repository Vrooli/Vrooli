package credentialauthority

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

const collapseExemptionMarker = "credential-authority:allow-collapse"

type collapseFinding struct {
	Filename string
	Line     int
	Message  string
}

func (f collapseFinding) String() string {
	return fmt.Sprintf("%s:%d: %s", f.Filename, f.Line, f.Message)
}

// scanCredentialCollapses finds direct Authority.Resolve calls whose function
// neither handles the public failure taxonomy nor carries a reviewed collapse
// exemption. It deliberately follows only values constructed by this package's
// Default function, avoiding false positives on unrelated types with a Resolve
// method.
func scanCredentialCollapses(filename string, source []byte) ([]collapseFinding, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, filename, source, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	authorityAliases := credentialAuthorityImportAliases(file)
	if len(authorityAliases) == 0 {
		return nil, nil
	}
	authorityFields := credentialAuthorityStructFields(file, authorityAliases)

	lines := strings.Split(string(source), "\n")
	var findings []collapseFinding
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		authorityVariables := defaultAuthorityVariables(function.Body, authorityAliases)
		for variable := range typedAuthorityParameters(function.Type, authorityAliases) {
			authorityVariables[variable] = struct{}{}
		}
		receiver, receiverFields := authorityReceiverFields(function, authorityFields)
		if len(authorityVariables) == 0 && len(receiverFields) == 0 {
			continue
		}
		if functionHandlesTaxonomy(function.Body, authorityAliases) {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || !isAuthorityResolveCall(call, authorityVariables, receiver, receiverFields) {
				return true
			}
			line := fileSet.Position(call.Pos()).Line
			if hasCollapseExemption(lines, line) {
				return true
			}
			findings = append(findings, collapseFinding{
				Filename: filename,
				Line:     line,
				Message:  "credential authority Resolve collapses the failure taxonomy; use Require/ResolveOrMint, handle ErrUnconfigured, ErrProviderUnavailable, and ErrProviderAbsent explicitly, or add a reviewed //" + collapseExemptionMarker + " reason",
			})
			return true
		})
	}
	return findings, nil
}

func credentialAuthorityStructFields(file *ast.File, aliases map[string]struct{}) map[string]map[string]struct{} {
	fields := make(map[string]map[string]struct{})
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, specification := range general.Specs {
			typeSpec, ok := specification.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structure, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range structure.Fields.List {
				if !isCredentialAuthorityType(field.Type, aliases) {
					continue
				}
				if fields[typeSpec.Name.Name] == nil {
					fields[typeSpec.Name.Name] = make(map[string]struct{})
				}
				for _, name := range field.Names {
					fields[typeSpec.Name.Name][name.Name] = struct{}{}
				}
			}
		}
	}
	return fields
}

func typedAuthorityParameters(function *ast.FuncType, aliases map[string]struct{}) map[string]struct{} {
	variables := make(map[string]struct{})
	if function == nil || function.Params == nil {
		return variables
	}
	for _, field := range function.Params.List {
		if !isCredentialAuthorityType(field.Type, aliases) {
			continue
		}
		for _, name := range field.Names {
			variables[name.Name] = struct{}{}
		}
	}
	return variables
}

func authorityReceiverFields(function *ast.FuncDecl, fields map[string]map[string]struct{}) (string, map[string]struct{}) {
	if function.Recv == nil || len(function.Recv.List) != 1 || len(function.Recv.List[0].Names) != 1 {
		return "", nil
	}
	receiverType := function.Recv.List[0].Type
	if pointer, ok := receiverType.(*ast.StarExpr); ok {
		receiverType = pointer.X
	}
	identifier, ok := receiverType.(*ast.Ident)
	if !ok {
		return "", nil
	}
	return function.Recv.List[0].Names[0].Name, fields[identifier.Name]
}

func isCredentialAuthorityType(expression ast.Expr, aliases map[string]struct{}) bool {
	if pointer, ok := expression.(*ast.StarExpr); ok {
		expression = pointer.X
	}
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Authority" {
		return false
	}
	packageName, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	_, ok = aliases[packageName.Name]
	return ok
}

func credentialAuthorityImportAliases(file *ast.File) map[string]struct{} {
	aliases := make(map[string]struct{})
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil || path != "github.com/vrooli/vrooli/packages/credential-authority-go" {
			continue
		}
		alias := "credentialauthority"
		if imported.Name != nil {
			alias = imported.Name.Name
		}
		if alias != "." && alias != "_" {
			aliases[alias] = struct{}{}
		}
	}
	return aliases
}

func defaultAuthorityVariables(body *ast.BlockStmt, authorityAliases map[string]struct{}) map[string]struct{} {
	variables := make(map[string]struct{})
	ast.Inspect(body, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for index, expression := range assignment.Rhs {
			call, ok := expression.(*ast.CallExpr)
			if !ok || !isPackageCall(call, authorityAliases, "Default") || index >= len(assignment.Lhs) {
				continue
			}
			identifier, ok := assignment.Lhs[index].(*ast.Ident)
			if ok {
				variables[identifier.Name] = struct{}{}
			}
		}
		return true
	})
	return variables
}

func functionHandlesTaxonomy(body *ast.BlockStmt, authorityAliases map[string]struct{}) bool {
	handled := make(map[string]bool)
	ast.Inspect(body, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		packageName, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		if _, ok := authorityAliases[packageName.Name]; !ok {
			return true
		}
		switch selector.Sel.Name {
		case "ErrUnconfigured", "ErrProviderUnavailable", "ErrProviderAbsent":
			handled[selector.Sel.Name] = true
		}
		return true
	})
	return handled["ErrUnconfigured"] && handled["ErrProviderUnavailable"] && handled["ErrProviderAbsent"]
}

func isAuthorityResolveCall(call *ast.CallExpr, authorityVariables map[string]struct{}, receiver string, receiverFields map[string]struct{}) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Resolve" {
		return false
	}
	if identifier, ok := selector.X.(*ast.Ident); ok {
		_, ok = authorityVariables[identifier.Name]
		return ok
	}
	field, ok := selector.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	receiverIdentifier, ok := field.X.(*ast.Ident)
	if !ok || receiverIdentifier.Name != receiver {
		return false
	}
	_, ok = receiverFields[field.Sel.Name]
	return ok
}

func isPackageCall(call *ast.CallExpr, aliases map[string]struct{}, method string) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != method {
		return false
	}
	packageName, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	_, ok = aliases[packageName.Name]
	return ok
}

func hasCollapseExemption(lines []string, callLine int) bool {
	// Markers belong immediately above the resolved call. Requiring proximity
	// prevents one broad file-level marker from exempting unrelated call sites.
	start := max(0, callLine-5)
	end := min(len(lines), callLine-1)
	markerIndex := -1
	for index := start; index < end; index++ {
		if strings.Contains(lines[index], collapseExemptionMarker) {
			markerIndex = index
		}
	}
	if markerIndex < 0 {
		return false
	}
	for index := markerIndex - 1; index >= start; index-- {
		trimmed := strings.TrimSpace(lines[index])
		if !strings.HasPrefix(trimmed, "//") {
			return false
		}
		comment := strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
		if comment == "" {
			continue
		}
		return !strings.Contains(comment, collapseExemptionMarker)
	}
	return false
}

func TestCollapseGuardRejectsCollapsingCallSite(t *testing.T) {
	source := []byte(`package fixture
import credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
func resolve() string {
	authority, err := credentialauthority.Default()
	if err != nil { return "" }
	value, err := authority.Resolve("vrooli/fixture", "secret")
	if err != nil { return "" }
	return value
}`)
	findings, err := scanCredentialCollapses("collapse.go", source)
	if err != nil {
		t.Fatalf("scanCredentialCollapses() error = %v", err)
	}
	if len(findings) != 1 || findings[0].Line != 6 {
		t.Fatalf("scanCredentialCollapses() findings = %#v, want one at line 6", findings)
	}
	message := findings[0].String()
	for _, fragment := range []string{"collapse.go:6", "Require/ResolveOrMint", "ErrProviderUnavailable"} {
		if !strings.Contains(message, fragment) {
			t.Errorf("finding = %q, missing %q", message, fragment)
		}
	}
}

func TestCollapseGuardAcceptsExplicitTaxonomyHandling(t *testing.T) {
	source := []byte(`package fixture
import (
	"errors"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)
func resolve() (string, error) {
	authority, err := credentialauthority.Default()
	if err != nil { return "", err }
	value, err := authority.Resolve("vrooli/fixture", "secret")
	switch {
	case err == nil:
		return value, nil
	case errors.Is(err, credentialauthority.ErrUnconfigured):
		return "", err
	case errors.Is(err, credentialauthority.ErrProviderUnavailable):
		return "", err
	case errors.Is(err, credentialauthority.ErrProviderAbsent):
		return "", err
	default:
		return "", err
	}
}`)
	findings, err := scanCredentialCollapses("handled.go", source)
	if err != nil || len(findings) != 0 {
		t.Fatalf("scanCredentialCollapses() findings = %#v, error = %v", findings, err)
	}
}

func TestCollapseGuardAcceptsReasonedExemption(t *testing.T) {
	source := []byte(`package fixture
import credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
func probe() string {
	authority, _ := credentialauthority.Default()
	// The inventory probe reports unknown while the bootstrap store is offline.
	//credential-authority:allow-collapse
	value, err := authority.Resolve("vrooli/fixture", "secret")
	if err != nil { return "" }
	return value
}`)
	findings, err := scanCredentialCollapses("exempt.go", source)
	if err != nil || len(findings) != 0 {
		t.Fatalf("scanCredentialCollapses() findings = %#v, error = %v", findings, err)
	}
}

func TestCollapseGuardRejectsUnreasonedExemption(t *testing.T) {
	source := []byte(`package fixture
import credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
func probe() string {
	authority, _ := credentialauthority.Default()
	//credential-authority:allow-collapse
	value, _ := authority.Resolve("vrooli/fixture", "secret")
	return value
}`)
	findings, err := scanCredentialCollapses("unreasoned.go", source)
	if err != nil || len(findings) != 1 {
		t.Fatalf("scanCredentialCollapses() findings = %#v, error = %v", findings, err)
	}
}

func TestCollapseGuardRejectsTypedAuthorityParameter(t *testing.T) {
	source := []byte(`package fixture
import credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
func resolve(authority *credentialauthority.Authority) (string, error) {
	return authority.Resolve("vrooli/fixture", "secret")
}`)
	findings, err := scanCredentialCollapses("parameter.go", source)
	if err != nil || len(findings) != 1 {
		t.Fatalf("scanCredentialCollapses() findings = %#v, error = %v", findings, err)
	}
}

func TestCollapseGuardRejectsTypedAuthorityReceiverField(t *testing.T) {
	source := []byte(`package fixture
import credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
type resolver struct { authority *credentialauthority.Authority }
func (r resolver) resolve() (string, error) {
	return r.authority.Resolve("vrooli/fixture", "secret")
}`)
	findings, err := scanCredentialCollapses("receiver.go", source)
	if err != nil || len(findings) != 1 {
		t.Fatalf("scanCredentialCollapses() findings = %#v, error = %v", findings, err)
	}
}
