package testvulns

import (
	"database/sql"
	"fmt"
	"net/http"
)

// SQLInjectionExample demonstrates SQL injection vulnerabilities
func SQLInjectionExample(db *sql.DB, userInput string) {
	// VULNERABILITY: SQL Injection via string concatenation (CWE-89)
	query := "SELECT * FROM users WHERE id = '" + userInput + "'"
	rows, err := db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()
	
	// VULNERABILITY: SQL Injection via fmt.Sprintf
	query2 := fmt.Sprintf("SELECT * FROM products WHERE name = '%s'", userInput)
	db.Query(query2)
	
	// VULNERABILITY: SQL Injection in UPDATE statement
	updateQuery := "UPDATE users SET status = 'active' WHERE email = '" + userInput + "'"
	db.Exec(updateQuery)
}

// CommandInjectionExample demonstrates command injection
func CommandInjectionExample(userInput string) {
	// VULNERABILITY: Command injection (CWE-78)
	cmd := "ls -la " + userInput
	// This would execute the command (commented to prevent actual execution)
	// exec.Command("sh", "-c", cmd).Output()
}

// PathTraversalExample demonstrates path traversal vulnerability
func PathTraversalExample(w http.ResponseWriter, r *http.Request) {
	// VULNERABILITY: Path traversal (CWE-22)
	filename := r.URL.Query().Get("file")
	// Direct use of user input in file path
	content := readFile("./uploads/" + filename)
	w.Write([]byte(content))
}

func readFile(path string) string {
	// Placeholder function
	return ""
}