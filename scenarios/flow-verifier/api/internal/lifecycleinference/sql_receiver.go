package lifecycleinference

import (
	"go/ast"
	"go/token"
)

// isSQLStorageType follows source-local wrappers which promote database/sql
// methods from an embedded DB, Tx, Conn, or Stmt. This recognizes adapters
// such as `type Tx struct { *sql.Tx }` without treating an arbitrary type or
// method named ExecContext as a durable sink.
func isSQLStorageType(expr ast.Expr, source parsedSource, files []parsedSource, visiting map[string]bool) bool {
	switch node := expr.(type) {
	case *ast.StarExpr:
		return isSQLStorageType(node.X, source, files, visiting)
	case *ast.ParenExpr:
		return isSQLStorageType(node.X, source, files, visiting)
	case *ast.SelectorExpr:
		if source.imports[exprIdent(node.X)] == "database/sql" {
			switch node.Sel.Name {
			case "DB", "Tx", "Conn", "Stmt":
				return true
			}
			return false
		}
		if source.imports[exprIdent(node.X)] == "github.com/vrooli/api-core/database" {
			switch node.Sel.Name {
			case "SQLExecutor", "SQLTxExecutor":
				return true
			}
			return false
		}
		return embeddedSQLWrapper(node.Sel.Name, source.imports[exprIdent(node.X)], files, visiting)
	case *ast.Ident:
		return embeddedSQLWrapper(node.Name, source.packageDir, files, visiting) || embeddedSQLWrapper(node.Name, source.importPath, files, visiting)
	default:
		return false
	}
}

func embeddedSQLWrapper(typeName, packageKey string, files []parsedSource, visiting map[string]bool) bool {
	if typeName == "" || packageKey == "" {
		return false
	}
	key := packageKey + "." + typeName
	if visiting[key] {
		return false
	}
	visiting[key] = true
	defer delete(visiting, key)

	for _, source := range files {
		if source.packageDir != packageKey && source.importPath != packageKey {
			continue
		}
		for _, declaration := range source.file.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok || group.Tok != token.TYPE {
				continue
			}
			for _, spec := range group.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != typeName {
					continue
				}
				if isSQLStorageType(typeSpec.Type, source, files, visiting) {
					return true
				}
				structure, ok := typeSpec.Type.(*ast.StructType)
				if !ok || structure.Fields == nil {
					continue
				}
				for _, field := range structure.Fields.List {
					if len(field.Names) == 0 && isSQLStorageType(field.Type, source, files, visiting) {
						return true
					}
				}
			}
		}
	}
	return false
}
