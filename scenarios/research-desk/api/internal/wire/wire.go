// Package wire converts the home reader's evidence-carrying values to the
// shared proto messages. Every domain handler uses it, so evidence has one
// wire shape.
package wire

import (
	"time"

	"research-desk/internal/home"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Timestamp returns nil for the zero time.
func Timestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// Evidence converts one evidence value.
func Evidence(e home.Evidence) *sharedv1.Evidence {
	return &sharedv1.Evidence{Known: e.Known, Source: e.Source, AsOf: Timestamp(e.AsOf), Reason: e.Reason}
}

// EvidenceList converts a list of evidence values.
func EvidenceList(list []home.Evidence) []*sharedv1.Evidence {
	out := make([]*sharedv1.Evidence, 0, len(list))
	for _, e := range list {
		out = append(out, Evidence(e))
	}
	return out
}

// Text converts a derived text value.
func Text(t home.Text) *sharedv1.DerivedText {
	return &sharedv1.DerivedText{Value: t.Value, Evidence: Evidence(t.Evidence)}
}

// Time converts a derived time value; the value is unset when unknown.
func Time(t home.Time) *sharedv1.DerivedTime {
	out := &sharedv1.DerivedTime{Evidence: Evidence(t.Evidence)}
	if t.Known {
		out.Value = Timestamp(t.Value)
	}
	return out
}

// File converts one file fact; nil stays nil.
func File(f *home.FileFact) *sharedv1.FileFact {
	if f == nil {
		return nil
	}
	return &sharedv1.FileFact{Name: f.Name, Path: f.Path, Bytes: f.Bytes, Modified: Timestamp(f.Modified)}
}

// Files converts a list of file facts.
func Files(list []home.FileFact) []*sharedv1.FileFact {
	out := make([]*sharedv1.FileFact, 0, len(list))
	for i := range list {
		out = append(out, File(&list[i]))
	}
	return out
}

// Flag converts a derived yes/no value.
func Flag(f home.Flag) *sharedv1.DerivedFlag {
	return &sharedv1.DerivedFlag{Value: f.Value, Evidence: Evidence(f.Evidence)}
}

// Entry converts one operator-file entry; nil stays nil.
func Entry(e *home.Entry) *sharedv1.OperatorEntry {
	if e == nil {
		return nil
	}
	return &sharedv1.OperatorEntry{At: Timestamp(e.At), Kind: e.Kind, Subject: e.Subject, Via: e.Via, Body: e.Body}
}

// Entries converts an operator file's entries; a missing file has none.
func Entries(log *home.OperatorLog) []*sharedv1.OperatorEntry {
	if log == nil {
		return nil
	}
	out := make([]*sharedv1.OperatorEntry, 0, len(log.Entries))
	for i := range log.Entries {
		out = append(out, Entry(&log.Entries[i]))
	}
	return out
}
