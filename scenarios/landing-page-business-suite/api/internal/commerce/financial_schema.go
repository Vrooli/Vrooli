package commerce

import _ "embed"

//go:embed financial_schema.sql
var financialSchema string

func FinancialSchema() string { return financialSchema }
