// Package clock owns the CLI's wall-clock seam. The command layer passes the
// function into long-running workflows so tests can control elapsed-time
// behavior without replacing process-global time.
package clock

import "time"

// Real is the production wall clock.
type Real struct{}

func (Real) Now() time.Time { return time.Now().UTC() }
