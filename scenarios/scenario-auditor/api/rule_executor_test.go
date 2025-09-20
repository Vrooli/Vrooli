package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRuleExecutor_New(t *testing.T) {
	tempDir := t.TempDir()
	
	executor := NewRuleExecutor(tempDir, 10*time.Second)
	
	if executor == nil {
		t.Fatal("Expected executor to be created")
	}
	
	if executor.rulesDir != tempDir {
		t.Errorf("Expected rulesDir %s, got %s", tempDir, executor.rulesDir)
	}
	
	if executor.timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", executor.timeout)
	}
	
	if executor.cache == nil {
		t.Error("Expected cache to be initialized")
	}
	
	if executor.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}
}

func TestRuleExecutor_ExecuteRule(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewRuleExecutor(tempDir, 5*time.Second)
	
	testFile := filepath.Join(tempDir, "test.go")
	os.WriteFile(testFile, []byte(`
package main

func main() {
	println("Hello")
}
`), 0644)
	
	tests := []struct {
		name         string
		rule         Rule
		targetPath   string
		expectedPass bool
		expectError  bool
	}{
		{
			name: "simple passing rule",
			rule: Rule{
				ID:   "test-rule-1",
				Name: "Test Rule 1",
				Check: func(path string) (bool, string, error) {
					return true, "Check passed", nil
				},
			},
			targetPath:   testFile,
			expectedPass: true,
			expectError:  false,
		},
		{
			name: "failing rule with message",
			rule: Rule{
				ID:   "test-rule-2",
				Name: "Test Rule 2",
				Check: func(path string) (bool, string, error) {
					return false, "Missing required header", nil
				},
			},
			targetPath:   testFile,
			expectedPass: false,
			expectError:  false,
		},
		{
			name: "rule with error",
			rule: Rule{
				ID:   "test-rule-3",
				Name: "Test Rule 3",
				Check: func(path string) (bool, string, error) {
					return false, "", fmt.Errorf("failed to read file")
				},
			},
			targetPath:   testFile,
			expectedPass: false,
			expectError:  true,
		},
		{
			name: "rule checking file content",
			rule: Rule{
				ID:   "content-check",
				Name: "Content Check",
				Check: func(path string) (bool, string, error) {
					content, err := os.ReadFile(path)
					if err != nil {
						return false, "", err
					}
					if strings.Contains(string(content), "Hello") {
						return true, "Found Hello", nil
					}
					return false, "Hello not found", nil
				},
			},
			targetPath:   testFile,
			expectedPass: true,
			expectError:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := executor.ExecuteRule(ctx, tt.rule, tt.targetPath)
			
			if (err != nil) != tt.expectError {
				t.Errorf("ExecuteRule() error = %v, expectError %v", err, tt.expectError)
				return
			}
			
			if !tt.expectError {
				if result.RuleID != tt.rule.ID {
					t.Errorf("Expected rule ID %s, got %s", tt.rule.ID, result.RuleID)
				}
				
				if result.Passed != tt.expectedPass {
					t.Errorf("Expected passed=%v, got %v", tt.expectedPass, result.Passed)
				}
				
				if result.TargetPath != tt.targetPath {
					t.Errorf("Expected target path %s, got %s", tt.targetPath, result.TargetPath)
				}
				
				if result.ExecutedAt.IsZero() {
					t.Error("Expected ExecutedAt to be set")
				}
			}
		})
	}
}

func TestRuleExecutor_ExecuteRules(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewRuleExecutor(tempDir, 5*time.Second)
	
	testFile1 := filepath.Join(tempDir, "file1.go")
	testFile2 := filepath.Join(tempDir, "file2.go")
	
	os.WriteFile(testFile1, []byte("package main"), 0644)
	os.WriteFile(testFile2, []byte("package test"), 0644)
	
	rules := []Rule{
		{
			ID:   "rule1",
			Name: "Rule 1",
			Check: func(path string) (bool, string, error) {
				return true, "OK", nil
			},
		},
		{
			ID:   "rule2",
			Name: "Rule 2",
			Check: func(path string) (bool, string, error) {
				content, _ := os.ReadFile(path)
				if strings.Contains(string(content), "main") {
					return true, "Found main", nil
				}
				return false, "No main", nil
			},
		},
	}
	
	ctx := context.Background()
	results, err := executor.ExecuteRules(ctx, rules, testFile1)
	
	if err != nil {
		t.Fatalf("ExecuteRules() error = %v", err)
	}
	
	if len(results) != len(rules) {
		t.Errorf("Expected %d results, got %d", len(rules), len(results))
	}
	
	for i, result := range results {
		if result.RuleID != rules[i].ID {
			t.Errorf("Result %d: expected rule ID %s, got %s", i, rules[i].ID, result.RuleID)
		}
	}
}

func TestRuleExecutor_ExecuteWithTimeout(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewRuleExecutor(tempDir, 100*time.Millisecond)
	
	slowRule := Rule{
		ID:   "slow-rule",
		Name: "Slow Rule",
		Check: func(path string) (bool, string, error) {
			time.Sleep(500 * time.Millisecond)
			return true, "OK", nil
		},
	}
	
	ctx := context.Background()
	_, err := executor.ExecuteRule(ctx, slowRule, "test.go")
	
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Error("Expected timeout error")
	}
}

func TestRuleExecutor_Caching(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewRuleExecutor(tempDir, 5*time.Second)
	
	callCount := 0
	rule := Rule{
		ID:   "cached-rule",
		Name: "Cached Rule",
		Check: func(path string) (bool, string, error) {
			callCount++
			return true, fmt.Sprintf("Call %d", callCount), nil
		},
	}
	
	testFile := filepath.Join(tempDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)
	
	ctx := context.Background()
	
	result1, _ := executor.ExecuteRule(ctx, rule, testFile)
	result2, _ := executor.ExecuteRule(ctx, rule, testFile)
	
	if callCount != 1 {
		t.Errorf("Expected rule to be called once (cached), got %d calls", callCount)
	}
	
	if result1.Message != result2.Message {
		t.Error("Expected cached result to be returned")
	}
	
	executor.ClearCache()
	
	executor.ExecuteRule(ctx, rule, testFile)
	
	if callCount != 2 {
		t.Errorf("Expected rule to be called again after cache clear, got %d calls", callCount)
	}
}

func TestRuleExecutor_Metrics(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewRuleExecutor(tempDir, 5*time.Second)
	
	rules := []Rule{
		{
			ID: "pass-rule",
			Check: func(path string) (bool, string, error) {
				return true, "OK", nil
			},
		},
		{
			ID: "fail-rule",
			Check: func(path string) (bool, string, error) {
				return false, "Failed", nil
			},
		},
		{
			ID: "error-rule",
			Check: func(path string) (bool, string, error) {
				return false, "", fmt.Errorf("error")
			},
		},
	}
	
	ctx := context.Background()
	for _, rule := range rules {
		executor.ExecuteRule(ctx, rule, "test.go")
	}
	
	metrics := executor.GetMetrics()
	
	if metrics.TotalExecutions != 3 {
		t.Errorf("Expected 3 total executions, got %d", metrics.TotalExecutions)
	}
	
	if metrics.PassedRules != 1 {
		t.Errorf("Expected 1 passed rule, got %d", metrics.PassedRules)
	}
	
	if metrics.FailedRules != 1 {
		t.Errorf("Expected 1 failed rule, got %d", metrics.FailedRules)
	}
	
	if metrics.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", metrics.ErrorCount)
	}
	
	if metrics.CacheHits != 0 {
		t.Errorf("Expected 0 cache hits, got %d", metrics.CacheHits)
	}
}

func TestRuleExecutor_ConcurrentExecution(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewRuleExecutor(tempDir, 5*time.Second)
	
	rule := Rule{
		ID: "concurrent-rule",
		Check: func(path string) (bool, string, error) {
			time.Sleep(10 * time.Millisecond)
			return true, "OK", nil
		},
	}
	
	ctx := context.Background()
	numGoroutines := 10
	results := make(chan *RuleResult, numGoroutines)
	errors := make(chan error, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			result, err := executor.ExecuteRule(ctx, rule, fmt.Sprintf("file%d.go", id))
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}
	
	successCount := 0
	errorCount := 0
	
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-results:
			successCount++
		case <-errors:
			errorCount++
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent executions")
		}
	}
	
	if successCount != numGoroutines {
		t.Errorf("Expected %d successful executions, got %d", numGoroutines, successCount)
	}
	
	if errorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", errorCount)
	}
}

func TestRuleExecutor_ValidateRule(t *testing.T) {
	executor := NewRuleExecutor("", 5*time.Second)
	
	tests := []struct {
		name        string
		rule        Rule
		expectValid bool
	}{
		{
			name: "valid rule",
			rule: Rule{
				ID:   "valid-rule",
				Name: "Valid Rule",
				Check: func(path string) (bool, string, error) {
					return true, "OK", nil
				},
			},
			expectValid: true,
		},
		{
			name: "missing ID",
			rule: Rule{
				Name: "No ID Rule",
				Check: func(path string) (bool, string, error) {
					return true, "OK", nil
				},
			},
			expectValid: false,
		},
		{
			name: "missing check function",
			rule: Rule{
				ID:   "no-check",
				Name: "No Check Rule",
			},
			expectValid: false,
		},
		{
			name: "empty name",
			rule: Rule{
				ID:   "empty-name",
				Name: "",
				Check: func(path string) (bool, string, error) {
					return true, "OK", nil
				},
			},
			expectValid: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidateRule(tt.rule)
			
			if tt.expectValid && err != nil {
				t.Errorf("Expected rule to be valid, got error: %v", err)
			}
			
			if !tt.expectValid && err == nil {
				t.Error("Expected rule to be invalid, got no error")
			}
		})
	}
}

func TestRuleExecutor_BatchExecution(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewRuleExecutor(tempDir, 5*time.Second)
	
	numFiles := 5
	numRules := 3
	
	for i := 0; i < numFiles; i++ {
		file := filepath.Join(tempDir, fmt.Sprintf("file%d.go", i))
		os.WriteFile(file, []byte(fmt.Sprintf("package file%d", i)), 0644)
	}
	
	var rules []Rule
	for i := 0; i < numRules; i++ {
		ruleID := fmt.Sprintf("rule%d", i)
		rules = append(rules, Rule{
			ID:   ruleID,
			Name: fmt.Sprintf("Rule %d", i),
			Check: func(path string) (bool, string, error) {
				return true, "OK", nil
			},
		})
	}
	
	ctx := context.Background()
	files, _ := filepath.Glob(filepath.Join(tempDir, "*.go"))
	
	allResults := make([][]RuleResult, 0)
	
	for _, file := range files {
		results, err := executor.ExecuteRules(ctx, rules, file)
		if err != nil {
			t.Errorf("Failed to execute rules for %s: %v", file, err)
			continue
		}
		allResults = append(allResults, results)
	}
	
	if len(allResults) != numFiles {
		t.Errorf("Expected results for %d files, got %d", numFiles, len(allResults))
	}
	
	for i, results := range allResults {
		if len(results) != numRules {
			t.Errorf("File %d: expected %d rule results, got %d", i, numRules, len(results))
		}
	}
}

func TestRuleExecutor_CustomValidation(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewRuleExecutor(tempDir, 5*time.Second)
	
	testFile := filepath.Join(tempDir, "config.yaml")
	os.WriteFile(testFile, []byte(`
version: 1.0
settings:
  enabled: true
  timeout: 30
`), 0644)
	
	yamlRule := Rule{
		ID:   "yaml-validation",
		Name: "YAML Validation",
		Check: func(path string) (bool, string, error) {
			if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
				return true, "Not a YAML file", nil
			}
			
			content, err := os.ReadFile(path)
			if err != nil {
				return false, "", err
			}
			
			if !strings.Contains(string(content), "version:") {
				return false, "Missing version field", nil
			}
			
			return true, "Valid YAML", nil
		},
	}
	
	ctx := context.Background()
	result, err := executor.ExecuteRule(ctx, yamlRule, testFile)
	
	if err != nil {
		t.Fatalf("Failed to execute rule: %v", err)
	}
	
	if !result.Passed {
		t.Errorf("Expected YAML validation to pass, got: %s", result.Message)
	}
}

func BenchmarkRuleExecution(b *testing.B) {
	tempDir := b.TempDir()
	executor := NewRuleExecutor(tempDir, 5*time.Second)
	
	testFile := filepath.Join(tempDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)
	
	rule := Rule{
		ID:   "bench-rule",
		Name: "Benchmark Rule",
		Check: func(path string) (bool, string, error) {
			content, _ := os.ReadFile(path)
			return strings.Contains(string(content), "package"), "OK", nil
		},
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.ExecuteRule(ctx, rule, testFile)
	}
}

func BenchmarkConcurrentRuleExecution(b *testing.B) {
	tempDir := b.TempDir()
	executor := NewRuleExecutor(tempDir, 5*time.Second)
	
	testFile := filepath.Join(tempDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)
	
	rule := Rule{
		ID:   "bench-concurrent",
		Name: "Concurrent Benchmark",
		Check: func(path string) (bool, string, error) {
			return true, "OK", nil
		},
	}
	
	ctx := context.Background()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			executor.ExecuteRule(ctx, rule, testFile)
		}
	})
}

type RuleExecutor struct {
	rulesDir string
	timeout  time.Duration
	cache    *RuleCache
	metrics  *ExecutorMetrics
	mu       sync.RWMutex
}

type RuleCache struct {
	entries map[string]*CacheEntry
	mu      sync.RWMutex
}

type CacheEntry struct {
	Result    *RuleResult
	CachedAt  time.Time
	ExpiresAt time.Time
}

type RuleResult struct {
	RuleID     string
	RuleName   string
	Passed     bool
	Message    string
	TargetPath string
	ExecutedAt time.Time
}

type ExecutorMetrics struct {
	TotalExecutions int64
	PassedRules     int64
	FailedRules     int64
	ErrorCount      int64
	CacheHits       int64
	CacheMisses     int64
	mu              sync.Mutex
}

func NewRuleExecutor(rulesDir string, timeout time.Duration) *RuleExecutor {
	return &RuleExecutor{
		rulesDir: rulesDir,
		timeout:  timeout,
		cache:    &RuleCache{entries: make(map[string]*CacheEntry)},
		metrics:  &ExecutorMetrics{},
	}
}

func (e *RuleExecutor) ExecuteRule(ctx context.Context, rule Rule, targetPath string) (*RuleResult, error) {
	if err := e.ValidateRule(rule); err != nil {
		return nil, err
	}
	
	cacheKey := fmt.Sprintf("%s:%s", rule.ID, targetPath)
	if cached := e.cache.get(cacheKey); cached != nil {
		e.metrics.recordCacheHit()
		return cached, nil
	}
	
	e.metrics.recordCacheMiss()
	
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	
	resultChan := make(chan *RuleResult)
	errorChan := make(chan error)
	
	go func() {
		passed, message, err := rule.Check(targetPath)
		if err != nil {
			errorChan <- err
			return
		}
		
		result := &RuleResult{
			RuleID:     rule.ID,
			RuleName:   rule.Name,
			Passed:     passed,
			Message:    message,
			TargetPath: targetPath,
			ExecutedAt: time.Now(),
		}
		
		resultChan <- result
	}()
	
	select {
	case result := <-resultChan:
		e.cache.set(cacheKey, result, 5*time.Minute)
		e.metrics.record(result)
		return result, nil
	case err := <-errorChan:
		e.metrics.recordError()
		return nil, err
	case <-ctx.Done():
		e.metrics.recordError()
		return nil, fmt.Errorf("rule execution timeout")
	}
}

func (e *RuleExecutor) ExecuteRules(ctx context.Context, rules []Rule, targetPath string) ([]RuleResult, error) {
	results := make([]RuleResult, 0, len(rules))
	
	for _, rule := range rules {
		result, err := e.ExecuteRule(ctx, rule, targetPath)
		if err != nil {
			continue
		}
		results = append(results, *result)
	}
	
	return results, nil
}

func (e *RuleExecutor) ValidateRule(rule Rule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if rule.Check == nil {
		return fmt.Errorf("rule check function is required")
	}
	return nil
}

func (e *RuleExecutor) ClearCache() {
	e.cache.mu.Lock()
	defer e.cache.mu.Unlock()
	e.cache.entries = make(map[string]*CacheEntry)
}

func (e *RuleExecutor) GetMetrics() ExecutorMetrics {
	e.metrics.mu.Lock()
	defer e.metrics.mu.Unlock()
	return *e.metrics
}

func (c *RuleCache) get(key string) *RuleResult {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.entries[key]
	if !exists {
		return nil
	}
	
	if time.Now().After(entry.ExpiresAt) {
		delete(c.entries, key)
		return nil
	}
	
	return entry.Result
}

func (c *RuleCache) set(key string, result *RuleResult, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.entries[key] = &CacheEntry{
		Result:    result,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}
}

func (m *ExecutorMetrics) record(result *RuleResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.TotalExecutions++
	if result.Passed {
		m.PassedRules++
	} else {
		m.FailedRules++
	}
}

func (m *ExecutorMetrics) recordError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorCount++
}

func (m *ExecutorMetrics) recordCacheHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheHits++
}

func (m *ExecutorMetrics) recordCacheMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheMisses++
}