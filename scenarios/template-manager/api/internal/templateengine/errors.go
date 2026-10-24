package templateengine

import "errors"

// ErrTemplateNotFound is the sentinel wrapped by the template-loading paths when
// a named template does not exist on disk. Callers branch on it with errors.Is
// instead of matching error strings, and the handler layer can map it onto a
// Connect not-found code where that is the desired contract.
var ErrTemplateNotFound = errors.New("template not found")
