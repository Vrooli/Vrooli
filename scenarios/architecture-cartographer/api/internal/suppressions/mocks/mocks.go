// Package mocks holds fakes for the suppressions seams.
package mocks

import (
	"context"

	"architecture-cartographer/internal/suppressions"
)

// FakeScanner returns canned markers.
type FakeScanner struct {
	Markers []suppressions.Marker
	Err     error
	Calls   []string
}

var _ suppressions.Scanner = (*FakeScanner)(nil)

func (f *FakeScanner) Scan(_ context.Context, scenarioDir string) ([]suppressions.Marker, error) {
	f.Calls = append(f.Calls, scenarioDir)
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Markers, nil
}

// FakeProvider returns canned active markers.
type FakeProvider struct {
	Markers []suppressions.Marker
	Err     error
}

var _ suppressions.Provider = (*FakeProvider)(nil)

func (f *FakeProvider) Active(context.Context, string) ([]suppressions.Marker, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Markers, nil
}

// FakeWriter records WriteMarker calls instead of touching the filesystem.
type FakeWriter struct {
	Written []WriteCall
	Err     error
}

// WriteCall captures one WriteMarker invocation.
type WriteCall struct {
	Path   string
	Line   int
	Marker suppressions.Marker
}

var _ suppressions.Writer = (*FakeWriter)(nil)

func (f *FakeWriter) WriteMarker(absPath string, line int, m suppressions.Marker) error {
	if f.Err != nil {
		return f.Err
	}
	f.Written = append(f.Written, WriteCall{Path: absPath, Line: line, Marker: m})
	return nil
}
