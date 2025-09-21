package main

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ScanResult struct {
	ScannerName     string
	TargetPath      string
	ScannedAt       time.Time
	Vulnerabilities []Vulnerability
}

type Vulnerability struct {
	Type        string
	Severity    string
	File        string
	Line        int
	Column      int
	Description string
	Suggestion  string
}

type Scanner interface {
	Name() string
	Scan(ctx context.Context, path string) (*ScanResult, error)
}

type basicScanner struct {
	name     string
	detector func(filePath string, content string) []Vulnerability
}

func (s *basicScanner) Name() string {
	return s.name
}

func (s *basicScanner) Scan(ctx context.Context, path string) (*ScanResult, error) {
	result := &ScanResult{
		ScannerName: s.name,
		TargetPath:  path,
		ScannedAt:   time.Now(),
	}

	if s.detector == nil {
		return result, nil
	}

	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if d.IsDir() {
			return nil
		}

		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		vulns := s.detector(p, string(data))
		if len(vulns) > 0 {
			result.Vulnerabilities = append(result.Vulnerabilities, vulns...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func NewGosecScanner() Scanner {
	return &basicScanner{
		name: "gosec",
		detector: func(filePath string, content string) []Vulnerability {
			var vulns []Vulnerability
			lower := strings.ToLower(content)
			if strings.Contains(lower, "md5.sum") || strings.Contains(lower, "crypto/md5") {
				vulns = append(vulns, Vulnerability{
					Type:        "weak-crypto",
					Severity:    "MEDIUM",
					File:        filepath.Base(filePath),
					Description: "Potential use of weak cryptography",
					Suggestion:  "Use a modern hash such as SHA-256",
				})
			}
			if strings.Contains(lower, "select * from") && strings.Contains(lower, "fmt.Sprintf") {
				vulns = append(vulns, Vulnerability{
					Type:        "sql-injection",
					Severity:    "HIGH",
					File:        filepath.Base(filePath),
					Description: "Potential SQL injection risk",
					Suggestion:  "Use parameterized queries",
				})
			}
			return vulns
		},
	}
}

func NewGitleaksScanner() Scanner {
	secrets := []string{"api_key", "aws_access_key", "aws_secret_key", "github_token"}
	return &basicScanner{
		name: "gitleaks",
		detector: func(filePath string, content string) []Vulnerability {
			var vulns []Vulnerability
			lower := strings.ToLower(content)
			for _, token := range secrets {
				if strings.Contains(lower, token) {
					vulns = append(vulns, Vulnerability{
						Type:        "hardcoded-secret",
						Severity:    "HIGH",
						File:        filepath.Base(filePath),
						Description: "Potential secret detected: " + token,
						Suggestion:  "Remove hardcoded secrets and use a secrets manager",
					})
				}
			}
			return vulns
		},
	}
}

func NewCustomScanner() Scanner {
	indicators := []struct {
		token      string
		vulnType   string
		severity   string
		suggestion string
	}{
		{"todo", "todo-comment", "LOW", "Resolve TODO items"},
		{"fixme", "fixme-comment", "MEDIUM", "Address FIXMEs before release"},
		{"hack", "hack-comment", "LOW", "Avoid leaving HACK comments"},
		{"password", "hardcoded-password", "HIGH", "Remove hardcoded passwords"},
	}

	return &basicScanner{
		name: "custom",
		detector: func(filePath string, content string) []Vulnerability {
			var vulns []Vulnerability
			lower := strings.ToLower(content)
			for _, indicator := range indicators {
				if strings.Contains(lower, indicator.token) {
					vulns = append(vulns, Vulnerability{
						Type:        indicator.vulnType,
						Severity:    indicator.severity,
						File:        filepath.Base(filePath),
						Description: "Indicator detected: " + indicator.token,
						Suggestion:  indicator.suggestion,
					})
				}
			}
			return vulns
		},
	}
}
