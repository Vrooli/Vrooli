// This file is intentionally mislocated: it lives under wrongdir/ but
// declares package right. golang.org/x/tools/go/packages surfaces this
// configuration as a load-time error which the extractor classifies as
// a warning.
package right

// Mislocated is a sentinel identifier so the file is non-empty.
func Mislocated() string {
	return "i am in the wrong directory"
}
