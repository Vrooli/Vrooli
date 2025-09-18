package rules

import (
	"regexp"
	"strings"
)

/*
Rule: Resource Cleanup
Description: Ensure proper cleanup of resources (connections, files, etc.)
Reason: Prevents resource leaks, port exhaustion, and memory issues in production
Category: api
Severity: critical
Standard: resource-management-v1

<test-case id="http-client-no-timeout" should-fail="true">
  <description>HTTP client without timeout</description>
  <input language="go">
func makeRequest(url string) (*Response, error) {
    client := &http.Client{}
    resp, err := client.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return processResponse(resp)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>HTTP Client Without Timeout</expected-message>
</test-case>

<test-case id="response-body-not-closed" should-fail="true">
  <description>HTTP response body not closed</description>
  <input language="go">
func fetchData(url string) ([]byte, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    // Missing: defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>response body must be closed</expected-message>
</test-case>

<test-case id="file-not-closed" should-fail="true">
  <description>File handle not properly closed</description>
  <input language="go">
func readConfig() (*Config, error) {
    file, err := os.Open("config.json")
    if err != nil {
        return nil, err
    }
    // Missing: defer file.Close()
    var config Config
    decoder := json.NewDecoder(file)
    err = decoder.Decode(&config)
    return &config, err
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>File Not Closed</expected-message>
</test-case>

<test-case id="db-rows-not-closed" should-fail="true">
  <description>Database rows not closed after query</description>
  <input language="go">
func getUsers(db *sql.DB) ([]User, error) {
    rows, err := db.Query("SELECT id, name FROM users")
    if err != nil {
        return nil, err
    }
    // Missing: defer rows.Close()
    
    var users []User
    for rows.Next() {
        var user User
        rows.Scan(&user.ID, &user.Name)
        users = append(users, user)
    }
    return users, nil
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Database Rows Not Closed</expected-message>
</test-case>

<test-case id="goroutine-no-context" should-fail="true">
  <description>Goroutine without context for cancellation</description>
  <input language="go">
func startWorker() {
    go func() {
        for {
            doWork()
            time.Sleep(1 * time.Second)
        }
    }()
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Goroutine Without Context</expected-message>
</test-case>

<test-case id="proper-resource-cleanup" should-fail="false">
  <description>Proper resource management with all cleanups</description>
  <input language="go">
func fetchAndStore(url string, filepath string) error {
    // HTTP client with timeout
    client := &http.Client{
        Timeout: 30 * time.Second,
    }
    
    resp, err := client.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // File with proper cleanup
    file, err := os.Create(filepath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    _, err = io.Copy(file, resp.Body)
    return err
}
  </input>
</test-case>

<test-case id="proper-db-cleanup" should-fail="false">
  <description>Database operations with proper cleanup</description>
  <input language="go">
func queryDatabase(db *sql.DB) ([]Result, error) {
    rows, err := db.Query("SELECT * FROM results")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var results []Result
    for rows.Next() {
        var r Result
        if err := rows.Scan(&r.ID, &r.Value); err != nil {
            return nil, err
        }
        results = append(results, r)
    }
    return results, rows.Err()
}
  </input>
</test-case>

<test-case id="goroutine-with-context" should-fail="false">
  <description>Goroutine with proper context handling</description>
  <input language="go">
func startWorkerWithContext(ctx context.Context) {
    go func(ctx context.Context) {
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                doWork()
            }
        }
    }(ctx)
}
  </input>
</test-case>
*/

// CheckResourceCleanup detects potential resource leaks
func CheckResourceCleanup(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")
	
	// Check 1: HTTP clients without timeouts
	httpClientPattern := regexp.MustCompile(`&http\.Client\{`)
	httpGetPattern := regexp.MustCompile(`http\.(Get|Post|Head)\(`)
	
	for i, line := range lines {
		if httpClientPattern.MatchString(line) {
			// Check if timeout is set within next few lines
			hasTimeout := false
			for j := i; j < min(i+5, len(lines)); j++ {
				if strings.Contains(lines[j], "Timeout") {
					hasTimeout = true
					break
				}
			}
			
			if !hasTimeout {
				violations = append(violations, Violation{
					Type:        "resource_cleanup",
					Severity:    "critical",
					Title:       "HTTP Client Without Timeout",
					Description: "HTTP client created without timeout can cause resource leaks",
					FilePath:    filePath,
					LineNumber:  i + 1,
					CodeSnippet: line,
					Recommendation: "Add Timeout: 30 * time.Second to http.Client",
					Standard:    "resource-management-v1",
				})
			}
		}
		
		// Check for http.Get/Post without timeout
		if httpGetPattern.MatchString(line) {
			violations = append(violations, Violation{
				Type:        "resource_cleanup",
				Severity:    "high",
				Title:       "Default HTTP Client Usage",
				Description: "Using default HTTP client without timeout",
				FilePath:    filePath,
				LineNumber:  i + 1,
				CodeSnippet: line,
				Recommendation: "Create custom client with timeout or use context with timeout",
				Standard:    "resource-management-v1",
			})
		}
	}
	
	// Check 2: HTTP response bodies not closed
	respPattern := regexp.MustCompile(`(\w+)\s*,\s*err\s*:=\s*http\.(Get|Post|Do)\(`)
	clientDoPattern := regexp.MustCompile(`(\w+)\s*,\s*err\s*:=\s*\w+\.Do\(`)
	
	for i, line := range lines {
		var respVar string
		if matches := respPattern.FindStringSubmatch(line); matches != nil {
			respVar = matches[1]
		} else if matches := clientDoPattern.FindStringSubmatch(line); matches != nil {
			respVar = matches[1]
		}
		
		if respVar != "" {
			// Look for defer close within next 5 lines
			hasDeferClose := false
			for j := i + 1; j < min(i+6, len(lines)); j++ {
				if strings.Contains(lines[j], "defer "+respVar+".Body.Close()") {
					hasDeferClose = true
					break
				}
			}
			
			if !hasDeferClose {
				violations = append(violations, Violation{
					Type:        "resource_cleanup",
					Severity:    "critical",
					Title:       "HTTP Response Body Not Closed",
					Description: "HTTP response body must be closed to prevent resource leak",
					FilePath:    filePath,
					LineNumber:  i + 1,
					CodeSnippet: line,
					Recommendation: "Add defer " + respVar + ".Body.Close() after error check",
					Standard:    "resource-management-v1",
				})
			}
		}
	}
	
	// Check 3: File operations without defer Close()
	fileOpenPattern := regexp.MustCompile(`(\w+)\s*,\s*err\s*:=\s*os\.(Open|Create|OpenFile)\(`)
	
	for i, line := range lines {
		if matches := fileOpenPattern.FindStringSubmatch(line); matches != nil {
			fileVar := matches[1]
			
			// Look for defer close
			hasDeferClose := false
			for j := i + 1; j < min(i+5, len(lines)); j++ {
				if strings.Contains(lines[j], "defer "+fileVar+".Close()") {
					hasDeferClose = true
					break
				}
			}
			
			if !hasDeferClose {
				violations = append(violations, Violation{
					Type:        "resource_cleanup",
					Severity:    "high",
					Title:       "File Not Closed",
					Description: "File handle not closed with defer",
					FilePath:    filePath,
					LineNumber:  i + 1,
					CodeSnippet: line,
					Recommendation: "Add defer " + fileVar + ".Close() after error check",
					Standard:    "resource-management-v1",
				})
			}
		}
	}
	
	// Check 4: Database rows not closed
	rowsPattern := regexp.MustCompile(`(\w+)\s*,\s*err\s*:=\s*\w+\.(Query|QueryRow)\(`)
	
	for i, line := range lines {
		if matches := rowsPattern.FindStringSubmatch(line); matches != nil {
			rowsVar := matches[1]
			
			// QueryRow doesn't need Close, only Query
			if strings.Contains(line, "QueryRow") {
				continue
			}
			
			// Look for defer close
			hasDeferClose := false
			for j := i + 1; j < min(i+5, len(lines)); j++ {
				if strings.Contains(lines[j], "defer "+rowsVar+".Close()") {
					hasDeferClose = true
					break
				}
			}
			
			if !hasDeferClose {
				violations = append(violations, Violation{
					Type:        "resource_cleanup",
					Severity:    "high",
					Title:       "Database Rows Not Closed",
					Description: "Database rows must be closed after use",
					FilePath:    filePath,
					LineNumber:  i + 1,
					CodeSnippet: line,
					Recommendation: "Add defer " + rowsVar + ".Close() after error check",
					Standard:    "database-v1",
				})
			}
		}
	}
	
	// Check 5: Goroutines without context
	goroutinePattern := regexp.MustCompile(`go\s+func\s*\(`)
	
	for i, line := range lines {
		if goroutinePattern.MatchString(line) {
			// Check if context is passed
			hasContext := false
			for j := i; j < min(i+3, len(lines)); j++ {
				if strings.Contains(lines[j], "context.Context") ||
				   strings.Contains(lines[j], "ctx") {
					hasContext = true
					break
				}
			}
			
			if !hasContext {
				violations = append(violations, Violation{
					Type:        "resource_cleanup",
					Severity:    "high",
					Title:       "Goroutine Without Context",
					Description: "Goroutine lacks context for cancellation",
					FilePath:    filePath,
					LineNumber:  i + 1,
					CodeSnippet: line,
					Recommendation: "Pass context.Context to enable graceful shutdown",
					Standard:    "concurrency-v1",
				})
			}
		}
	}
	
	return violations
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}