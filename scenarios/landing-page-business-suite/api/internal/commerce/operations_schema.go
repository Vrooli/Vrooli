package commerce

import _ "embed"

//go:embed operations_schema.sql
var operationsSchema string

func OperationsSchema() string { return operationsSchema }
