package ruleengine

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Service exposes rule discovery, execution, and test utilities for the API.
type Service struct {
	loader *Loader
	tests  *TestHarness

	mu    sync.RWMutex
	rules map[string]Info
}

// NewService constructs a rule engine service with the given options.
func NewService(opts Options) (*Service, error) {
	if strings.TrimSpace(opts.ModuleRoot) == "" {
		return nil, fmt.Errorf("ruleengine: module root must be provided")
	}
	loader, err := NewLoader(opts)
	if err != nil {
		return nil, err
	}
	return &Service{
		loader: loader,
		tests:  NewTestHarness(),
		rules:  make(map[string]Info),
	}, nil
}

// Load returns the cached rule set, reloading from disk if necessary.
func (s *Service) Load() (map[string]Info, error) {
	s.mu.RLock()
	cached := s.rules
	s.mu.RUnlock()

	if len(cached) > 0 {
		return cloneRuleMap(cached), nil
	}

	return s.Reload()
}

// Reload forces a rescan of the rule directories.
func (s *Service) Reload() (map[string]Info, error) {
	rules, err := s.loader.Load()
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.rules = rules
	s.mu.Unlock()

	return cloneRuleMap(rules), nil
}

// Get returns a single rule if it exists.
func (s *Service) Get(id string) (Info, bool, error) {
	rules, err := s.Load()
	if err != nil {
		return Info{}, false, err
	}

	info, ok := rules[id]
	return info, ok, nil
}

// RunTests executes all embedded test cases for the requested rule.
func (s *Service) RunTests(id string) ([]TestResult, error) {
	info, ok, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("rule %s not found", id)
	}

	return s.tests.RunAllTests(id, info)
}

// Validate runs a custom snippet against the requested rule.
func (s *Service) Validate(id string, input, language string) (TestResult, error) {
	if language == "" {
		language = "text"
	}

	info, ok, err := s.Get(id)
	if err != nil {
		return TestResult{}, err
	}
	if !ok {
		return TestResult{}, fmt.Errorf("rule %s not found", id)
	}

	return s.tests.ValidateCustomInput(id, info, input, language)
}

// ClearTestCache clears cached results for a specific rule (or all if id empty).
func (s *Service) ClearTestCache(id string) {
	if id == "" {
		s.tests.ClearAllCache()
		return
	}
	s.tests.ClearCache(id)
}

// TestCacheInfo returns the file hash and whether cached results exist for the rule.
func (s *Service) TestCacheInfo(id string, info Info) (string, bool, error) {
	hash, err := s.tests.GetFileHash(info.FilePath)
	if err != nil {
		return "", false, err
	}
	if cache, ok := s.tests.GetCachedResults(id, hash); ok && cache != nil {
		return hash, true, nil
	}
	return hash, false, nil
}

// RunTestsForInfo executes tests using the already loaded rule info.
func (s *Service) RunTestsForInfo(id string, info Info) ([]TestResult, error) {
	if !info.Implementation.Valid {
		return nil, fmt.Errorf("rule %s implementation unavailable: %s", id, info.Implementation.Error)
	}
	return s.tests.RunAllTests(id, info)
}

// ExtractTestCases returns the embedded test cases for the given rule.
func (s *Service) ExtractTestCases(info Info) ([]TestCase, error) {
	content, err := os.ReadFile(info.FilePath)
	if err != nil {
		return nil, err
	}
	return s.tests.ExtractTestCases(string(content))
}

func cloneRuleMap(src map[string]Info) map[string]Info {
	dst := make(map[string]Info, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
