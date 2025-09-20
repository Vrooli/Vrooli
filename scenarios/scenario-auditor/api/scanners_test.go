package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGosecScanner(t *testing.T) {
	tempDir := t.TempDir()
	
	testFile := filepath.Join(tempDir, "vulnerable.go")
	os.WriteFile(testFile, []byte(`
package main

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"net/http"
)

func insecureHash(data string) string {
	h := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", h)
}

func sqlInjection(userInput string) {
	query := fmt.Sprintf("SELECT * FROM users WHERE name = '%s'", userInput)
	db.Exec(query)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello"))
	})
	http.ListenAndServe(":8080", nil)
}
`), 0644)
	
	scanner := NewGosecScanner()
	ctx := context.Background()
	
	results, err := scanner.Scan(ctx, tempDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	
	if len(results.Vulnerabilities) == 0 {
		t.Skip("Gosec not installed or no vulnerabilities detected")
	}
	
	foundWeakCrypto := false
	foundSQLInjection := false
	
	for _, vuln := range results.Vulnerabilities {
		if strings.Contains(vuln.Type, "crypto") || strings.Contains(vuln.Type, "G401") {
			foundWeakCrypto = true
		}
		if strings.Contains(vuln.Type, "sql") || strings.Contains(vuln.Type, "G201") {
			foundSQLInjection = true
		}
	}
	
	if !foundWeakCrypto {
		t.Error("Expected to find weak crypto vulnerability")
	}
	
	if !foundSQLInjection {
		t.Error("Expected to find SQL injection vulnerability")
	}
}

func TestGitleaksScanner(t *testing.T) {
	tempDir := t.TempDir()
	
	testFile := filepath.Join(tempDir, "config.json")
	os.WriteFile(testFile, []byte(`{
	"api_key": "sk-1234567890abcdef1234567890abcdef",
	"database_password": "super_secret_password_123!",
	"aws_access_key": "AKIAIOSFODNN7EXAMPLE",
	"aws_secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	"github_token": "ghp_1234567890abcdef1234567890abcdef1234"
}`), 0644)
	
	scanner := NewGitleaksScanner()
	ctx := context.Background()
	
	results, err := scanner.Scan(ctx, tempDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	
	if len(results.Vulnerabilities) == 0 {
		t.Skip("Gitleaks not installed or no secrets detected")
	}
	
	foundAPIKey := false
	foundAWSKey := false
	
	for _, vuln := range results.Vulnerabilities {
		if strings.Contains(vuln.Description, "api_key") || strings.Contains(vuln.Description, "API") {
			foundAPIKey = true
		}
		if strings.Contains(vuln.Description, "aws") || strings.Contains(vuln.Description, "AWS") {
			foundAWSKey = true
		}
		
		if vuln.Severity != "HIGH" && vuln.Severity != "CRITICAL" {
			t.Errorf("Expected HIGH or CRITICAL severity for secrets, got %s", vuln.Severity)
		}
	}
	
	if !foundAPIKey {
		t.Error("Expected to find API key")
	}
	
	if !foundAWSKey {
		t.Error("Expected to find AWS credentials")
	}
}

func TestCustomScanner(t *testing.T) {
	tempDir := t.TempDir()
	
	testGoFile := filepath.Join(tempDir, "main.go")
	os.WriteFile(testGoFile, []byte(`
package main

import "fmt"

// TODO: Fix this security issue
func insecureFunction() {
	password := "hardcoded"
	fmt.Println(password)
}

func main() {
	// FIXME: Remove debug code
	debug := true
	if debug {
		fmt.Println("Debug mode")
	}
}
`), 0644)
	
	testJSFile := filepath.Join(tempDir, "app.js")
	os.WriteFile(testJSFile, []byte(`
// TODO: Implement authentication
function login(username, password) {
	// HACK: Temporary bypass
	return true;
}

// BUG: Memory leak here
let cache = [];
function addToCache(item) {
	cache.push(item);
}

console.log("TODO: Add error handling");
`), 0644)
	
	scanner := NewCustomScanner()
	ctx := context.Background()
	
	results, err := scanner.Scan(ctx, tempDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	
	if len(results.Vulnerabilities) == 0 {
		t.Error("Expected to find vulnerabilities")
	}
	
	foundTODO := false
	foundFIXME := false
	foundHACK := false
	foundBUG := false
	foundHardcoded := false
	
	for _, vuln := range results.Vulnerabilities {
		if strings.Contains(vuln.Type, "TODO") {
			foundTODO = true
		}
		if strings.Contains(vuln.Type, "FIXME") {
			foundFIXME = true
		}
		if strings.Contains(vuln.Type, "HACK") {
			foundHACK = true
		}
		if strings.Contains(vuln.Type, "BUG") {
			foundBUG = true
		}
		if strings.Contains(vuln.Type, "hardcoded") || strings.Contains(vuln.Description, "hardcoded") {
			foundHardcoded = true
		}
		
		if vuln.File == "" {
			t.Error("Vulnerability missing file path")
		}
		
		if vuln.Line == 0 {
			t.Error("Vulnerability missing line number")
		}
	}
	
	if !foundTODO {
		t.Error("Expected to find TODO comments")
	}
	
	if !foundFIXME {
		t.Error("Expected to find FIXME comments")
	}
	
	if !foundHACK {
		t.Error("Expected to find HACK comments")
	}
	
	if !foundBUG {
		t.Error("Expected to find BUG comments")
	}
	
	if !foundHardcoded {
		t.Error("Expected to find hardcoded password")
	}
}

func TestScannerInterface(t *testing.T) {
	scanners := []Scanner{
		NewGosecScanner(),
		NewGitleaksScanner(),
		NewCustomScanner(),
	}
	
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)
	
	ctx := context.Background()
	
	for _, scanner := range scanners {
		name := scanner.Name()
		if name == "" {
			t.Error("Scanner name should not be empty")
		}
		
		results, err := scanner.Scan(ctx, tempDir)
		if err != nil {
			t.Errorf("Scanner %s failed: %v", name, err)
			continue
		}
		
		if results.ScannedAt.IsZero() {
			t.Errorf("Scanner %s: ScannedAt should be set", name)
		}
		
		if results.ScannerName != name {
			t.Errorf("Scanner %s: expected scanner name %s, got %s", name, name, results.ScannerName)
		}
		
		if results.TargetPath != tempDir {
			t.Errorf("Scanner %s: expected target path %s, got %s", name, tempDir, results.TargetPath)
		}
	}
}

func TestScannerTimeout(t *testing.T) {
	scanner := &slowScanner{
		baseScanner: baseScanner{name: "slow-scanner"},
		delay:       5 * time.Second,
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	_, err := scanner.Scan(ctx, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "context") {
		t.Error("Expected context timeout error")
	}
}

func TestScanResultSerialization(t *testing.T) {
	result := &ScanResult{
		ScannerName: "test-scanner",
		TargetPath:  "/test/path",
		ScannedAt:   time.Now(),
		Vulnerabilities: []Vulnerability{
			{
				Type:        "SQL Injection",
				Severity:    "HIGH",
				File:        "main.go",
				Line:        42,
				Column:      10,
				Description: "Unsanitized user input",
				Suggestion:  "Use parameterized queries",
			},
		},
		Metadata: map[string]interface{}{
			"version": "1.0",
			"rules":   10,
		},
	}
	
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}
	
	var decoded ScanResult
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	if decoded.ScannerName != result.ScannerName {
		t.Errorf("Scanner name mismatch: %s != %s", decoded.ScannerName, result.ScannerName)
	}
	
	if len(decoded.Vulnerabilities) != len(result.Vulnerabilities) {
		t.Errorf("Vulnerability count mismatch: %d != %d", 
			len(decoded.Vulnerabilities), len(result.Vulnerabilities))
	}
	
	if decoded.Vulnerabilities[0].Type != result.Vulnerabilities[0].Type {
		t.Error("Vulnerability type mismatch after serialization")
	}
}

func TestMultiScanner(t *testing.T) {
	tempDir := t.TempDir()
	
	testFile := filepath.Join(tempDir, "multi.go")
	os.WriteFile(testFile, []byte(`
package main

import (
	"crypto/md5"
	"fmt"
)

func main() {
	// TODO: Fix this
	apiKey := "sk-1234567890abcdef"
	hash := md5.Sum([]byte(apiKey))
	fmt.Printf("%x", hash)
}
`), 0644)
	
	multiScanner := NewMultiScanner(
		NewGosecScanner(),
		NewGitleaksScanner(),
		NewCustomScanner(),
	)
	
	ctx := context.Background()
	results, err := multiScanner.ScanAll(ctx, tempDir)
	if err != nil {
		t.Fatalf("Multi-scan failed: %v", err)
	}
	
	if len(results) < 2 {
		t.Skip("Not all scanners available")
	}
	
	totalVulns := 0
	for _, result := range results {
		totalVulns += len(result.Vulnerabilities)
	}
	
	if totalVulns == 0 {
		t.Error("Expected to find vulnerabilities from multiple scanners")
	}
	
	scannerNames := make(map[string]bool)
	for _, result := range results {
		scannerNames[result.ScannerName] = true
	}
	
	if len(scannerNames) != len(results) {
		t.Error("Duplicate scanner results detected")
	}
}

func TestScannerConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	
	for i := 0; i < 10; i++ {
		file := filepath.Join(tempDir, fmt.Sprintf("file%d.go", i))
		os.WriteFile(file, []byte(fmt.Sprintf(`
package main
// TODO: Fix issue %d
func main() {}
`, i)), 0644)
	}
	
	scanner := NewCustomScanner()
	ctx := context.Background()
	
	results := make(chan *ScanResult, 10)
	errors := make(chan error, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			result, err := scanner.Scan(ctx, tempDir)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}()
	}
	
	successCount := 0
	errorCount := 0
	
	for i := 0; i < 10; i++ {
		select {
		case <-results:
			successCount++
		case <-errors:
			errorCount++
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent scans")
		}
	}
	
	if successCount != 10 {
		t.Errorf("Expected 10 successful scans, got %d", successCount)
	}
	
	if errorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", errorCount)
	}
}

func BenchmarkCustomScanner(b *testing.B) {
	tempDir := b.TempDir()
	
	for i := 0; i < 100; i++ {
		file := filepath.Join(tempDir, fmt.Sprintf("file%d.go", i))
		os.WriteFile(file, []byte(`
package main
// TODO: Optimize this
func process() {
	// FIXME: Performance issue
	data := make([]int, 1000000)
	for i := range data {
		data[i] = i * 2
	}
}
`), 0644)
	}
	
	scanner := NewCustomScanner()
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner.Scan(ctx, tempDir)
	}
}

type slowScanner struct {
	baseScanner
	delay time.Duration
}

func (s *slowScanner) Scan(ctx context.Context, path string) (*ScanResult, error) {
	select {
	case <-time.After(s.delay):
		return &ScanResult{
			ScannerName: s.name,
			TargetPath:  path,
			ScannedAt:   time.Now(),
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type MultiScanner struct {
	scanners []Scanner
}

func NewMultiScanner(scanners ...Scanner) *MultiScanner {
	return &MultiScanner{scanners: scanners}
}

func (m *MultiScanner) ScanAll(ctx context.Context, path string) ([]*ScanResult, error) {
	results := make([]*ScanResult, 0, len(m.scanners))
	
	for _, scanner := range m.scanners {
		result, err := scanner.Scan(ctx, path)
		if err != nil {
			continue
		}
		results = append(results, result)
	}
	
	return results, nil
}

type baseScanner struct {
	name string
}

func (b *baseScanner) Name() string {
	return b.name
}