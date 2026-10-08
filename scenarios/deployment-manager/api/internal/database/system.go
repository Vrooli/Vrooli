package database

import _ "embed"

//go:embed system.sql
var systemSQL string

func Schema() string { return systemSQL }
