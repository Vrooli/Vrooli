// Package markedrefs parses Vrooli's machine-readable inline reference syntax.
//
// Marked references appear inside markdown inline code spans:
//
//	`path:docs/README.md`
//	`topic[example]:foo/bar/*`
//
// The package deliberately does not perform domain validation. It only parses
// and classifies markers and qualifiers so callers can apply their own checks:
// prompt-manager validates topic references, documentation tools validate
// paths, and other scenarios can add domain-specific handling without
// duplicating the syntax parser.
package markedrefs
