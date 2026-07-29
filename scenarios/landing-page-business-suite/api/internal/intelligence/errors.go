// Package intelligence owns AI-provider contracts and provider-specific error semantics.
package intelligence

import "errors"

// ErrProvider indicates that the upstream AI provider rejected or could not
// complete a request. HTTP adapters map this domain error to Bad Gateway.
var ErrProvider = errors.New("AI provider error")
