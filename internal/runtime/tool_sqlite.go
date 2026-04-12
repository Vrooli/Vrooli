package runtime

func newSQLiteTool() handler {
	return newToolHandler("sqlite", []string{"sqlite3"}, []string{"--version"}, "sqlite3", map[string]string{
		"apt-get": "sqlite3",
		"brew":    "sqlite",
	}, "Install sqlite3 for SQLite-backed tooling")
}
