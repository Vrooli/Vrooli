// Package filepreview is the reusable file-preview subsystem for web-console.
// It resolves a raw path string (from a message link or inline code chip) into
// rich preview metadata, classifies it into a preview Kind, and issues opaque
// short-lived preview ids the REST blob endpoint serves bytes against.
//
// The package is transport-neutral: it has no knowledge of Connect, mux, or
// the session store. Package main adapts the live session cwd into Resolve and
// wires the Connect handler + blob route around Store. This keeps path
// resolution, classification, and the id store independently unit-testable and
// reusable by future web-console surfaces (records, BAS runs, plan drill-downs)
// without dragging conversation semantics along.
package filepreview

import "github.com/vrooli/api-core/filekind"

// Kind classifies how a resolved file should be previewed in the UI. The
// classifier itself lives in api-core/filekind so every app that previews files
// picks the same viewer; these are thin bindings so the rest of this package
// keeps its vocabulary.
type Kind = filekind.Kind

const (
	KindMarkdown     = filekind.KindMarkdown
	KindCode         = filekind.KindCode
	KindText         = filekind.KindText
	KindSVG          = filekind.KindSVG
	KindImage        = filekind.KindImage
	KindPDF          = filekind.KindPDF
	KindAudio        = filekind.KindAudio
	KindVideo        = filekind.KindVideo
	KindCSV          = filekind.KindCSV
	KindSpreadsheet  = filekind.KindSpreadsheet
	KindDocument     = filekind.KindDocument
	KindPresentation = filekind.KindPresentation
	KindEbook        = filekind.KindEbook
	KindModel3D      = filekind.KindModel3D
	KindFont         = filekind.KindFont
	KindDatabase     = filekind.KindDatabase
	KindArchive      = filekind.KindArchive
	KindDiff         = filekind.KindDiff
	KindDirectory    = filekind.KindDirectory
	KindUnsupported  = filekind.KindUnsupported
)

// MIMEDirectory is the conventional media type reported for a directory
// target. Directories have no bytes; this only labels the resolved target.
const MIMEDirectory = filekind.MIMEDirectory

// classification is the result of inspecting a file's extension and (for
// unknown extensions) a content sniff.
type classification struct {
	kind     Kind
	mimeType string
}

// classify determines the preview Kind and MIME type for a resolved file.
// sniff supplies up to 512 leading bytes for content detection; it is only
// called for extensions with no Kind mapping (so well-known agent artifacts
// never pay a read).
func classify(path string, sniff func() ([]byte, error)) classification {
	r := filekind.Classify(path, sniff)
	return classification{kind: r.Kind, mimeType: r.MIME}
}

// classifyByExtension returns the Kind implied by a name's extension alone,
// with ok=false when the extension is unmapped. Directory listings use this so
// rendering a page of entries costs zero file reads.
func classifyByExtension(name string) (Kind, bool) { return filekind.ByExtension(name) }

// IsMediaFilename reports whether the filename has a supported audio or video
// extension. HTML runtime resources share this registry with standalone previews.
func IsMediaFilename(name string) bool { return filekind.IsMedia(name) }

// mimeForKind resolves the Content-Type for a kind+extension.
func mimeForKind(k Kind, ext string) string { return filekind.MIMEFor(k, ext) }
