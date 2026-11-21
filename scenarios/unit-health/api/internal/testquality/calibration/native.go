package calibration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type NativeExpectation struct {
	CaseID                string `json:"caseId"`
	Name                  string `json:"name"`
	Status                string `json:"status"`
	FailureContains       string `json:"failureContains,omitempty"`
	RetryCount            int    `json:"retryCount"`
	RetainedErrorContains string `json:"retainedErrorContains,omitempty"`
	Limitation            string `json:"limitation,omitempty"`
}
type NativeSpecification struct {
	Root       string              `json:"root"`
	Files      []string            `json:"files"`
	Version    string              `json:"version"`
	Provenance string              `json:"provenance"`
	Cases      []NativeExpectation `json:"cases"`
}
type NativeObservation struct {
	Version        string `json:"version"`
	RunnerExitCode int    `json:"runnerExitCode"`
	Cases          []struct {
		Name            string   `json:"name"`
		Status          string   `json:"status"`
		FailureMessages []string `json:"failureMessages"`
	} `json:"cases"`
	Reporter struct {
		Events []struct {
			Name       string        `json:"name"`
			State      string        `json:"state"`
			RetryCount int           `json:"retryCount"`
			Errors     []NativeError `json:"errors"`
		} `json:"events"`
		Errors []NativeError `json:"errors"`
	} `json:"reporter"`
}
type NativeError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}
type NativeReport struct {
	Version        string   `json:"version"`
	UnknownVersion bool     `json:"unknownVersion"`
	Total          int      `json:"total"`
	Matched        int      `json:"matched"`
	Differences    []string `json:"differences"`
	Limitations    []string `json:"limitations"`
}

func LoadNativeSpecification(root string) (NativeSpecification, error) {
	data, err := os.ReadFile(filepath.Join(root, "native-expected.json"))
	if err != nil {
		return NativeSpecification{}, err
	}
	var spec NativeSpecification
	if err := json.Unmarshal(data, &spec); err != nil {
		return spec, err
	}
	if spec.Version == "" || spec.Provenance == "" || len(spec.Cases) == 0 {
		return spec, fmt.Errorf("native specification lacks version, provenance or cases")
	}
	if !filepath.IsLocal(spec.Root) || len(spec.Files) == 0 {
		return spec, fmt.Errorf("native fixture root/files required")
	}
	for _, file := range spec.Files {
		if !filepath.IsLocal(file) {
			return spec, fmt.Errorf("non-local native fixture path")
		}
		if _, err := os.ReadFile(filepath.Join(root, spec.Root, file)); err != nil {
			return spec, err
		}
	}
	seen := map[string]bool{}
	for _, c := range spec.Cases {
		if c.CaseID == "" || c.Name == "" || seen[c.Name] || (c.Status != "passed" && c.Status != "failed" && c.Status != "skipped") || c.RetryCount < 0 {
			return spec, fmt.Errorf("invalid native expectation %q", c.Name)
		}
		if c.Status == "failed" && c.FailureContains == "" {
			return spec, fmt.Errorf("failed native expectation %s must identify intended failure", c.Name)
		}
		seen[c.Name] = true
	}
	return spec, nil
}

// CompareNative distinguishes intentional fixture failures from a broken
// harness. Assertions are matched by independent names/outcomes, never by exit
// code alone. Unknown runner versions cannot satisfy this version's contract.
func CompareNative(spec NativeSpecification, observed NativeObservation) NativeReport {
	r := NativeReport{Version: observed.Version, Total: len(spec.Cases), Differences: []string{}, Limitations: []string{}}
	if observed.Version != spec.Version {
		r.UnknownVersion = true
		r.Differences = append(r.Differences, "unsupported runner version")
		return r
	}
	for _, e := range observed.Reporter.Errors {
		r.Differences = append(r.Differences, "unhandled runner error: "+e.Name+": "+e.Message)
	}
	seen := map[string]bool{}
	for _, expected := range spec.Cases {
		seen[expected.Name] = true
		matches := 0
		valid := true
		for _, got := range observed.Cases {
			if got.Name != expected.Name {
				continue
			}
			matches++
			if got.Status != expected.Status {
				r.Differences = append(r.Differences, fmt.Sprintf("%s: expected %s, observed %s", expected.Name, expected.Status, got.Status))
				valid = false
			}
			if expected.FailureContains != "" && !strings.Contains(strings.Join(got.FailureMessages, "\n"), expected.FailureContains) {
				r.Differences = append(r.Differences, expected.Name+": intended failure not observed")
				valid = false
			}
		}
		if matches != 1 {
			r.Differences = append(r.Differences, fmt.Sprintf("%s: expected one observation, got %d", expected.Name, matches))
			valid = false
		}
		maxRetry := 0
		retained := false
		for _, e := range observed.Reporter.Events {
			if e.Name == expected.Name {
				if e.RetryCount > maxRetry {
					maxRetry = e.RetryCount
				}
				for _, err := range e.Errors {
					if strings.Contains(err.Message, expected.RetainedErrorContains) {
						retained = true
					}
				}
			}
		}
		if maxRetry != expected.RetryCount {
			r.Differences = append(r.Differences, expected.Name+": retry count mismatch")
			valid = false
		}
		if expected.RetainedErrorContains != "" && !retained {
			r.Differences = append(r.Differences, expected.Name+": prior retry error not retained")
			valid = false
		}
		if expected.Limitation != "" {
			r.Limitations = append(r.Limitations, expected.CaseID+": "+expected.Limitation)
		}
		if valid {
			r.Matched++
		}
	}
	for _, got := range observed.Cases {
		if !seen[got.Name] {
			r.Differences = append(r.Differences, "unexpected native case: "+got.Name)
		}
	}
	sort.Strings(r.Differences)
	sort.Strings(r.Limitations)
	return r
}
