package graph

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/printer"
	"go/token"
)

// functionSyntaxAttributes records reusable source facts once in the parser.
// Detectors interpret these facts; the parser does not decide what is debt.
func functionSyntaxAttributes(fset *token.FileSet, fn *ast.FuncDecl) map[string]string {
	attrs := map[string]string{"signature": exprString(fset, fn.Type)}
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		receiver := fn.Recv.List[0].Type
		if pointer, ok := receiver.(*ast.StarExpr); ok {
			receiver = pointer.X
		}
		switch indexed := receiver.(type) {
		case *ast.IndexExpr:
			receiver = indexed.X
		case *ast.IndexListExpr:
			receiver = indexed.X
		}
		attrs["receiver"] = exprString(fset, receiver)
	}
	if fn.Body == nil {
		return attrs
	}
	attrs["body_lines"] = intAttr(fset.Position(fn.Body.End()).Line - fset.Position(fn.Body.Pos()).Line + 1)
	complexity := 1
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
			complexity++
		case *ast.CaseClause:
			if len(node.List) > 0 {
				complexity++
			}
		case *ast.CommClause:
			if node.Comm != nil {
				complexity++
			}
		case *ast.BinaryExpr:
			if node.Op == token.LAND || node.Op == token.LOR {
				complexity++
			}
		}
		return true
	})
	attrs["cyclomatic_complexity"] = intAttr(complexity)
	var body bytes.Buffer
	if err := printer.Fprint(&body, token.NewFileSet(), fn.Body); err == nil {
		sum := sha256.Sum256(body.Bytes())
		attrs["body_hash"] = hex.EncodeToString(sum[:])
	}
	return attrs
}
