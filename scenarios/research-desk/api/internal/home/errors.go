package home

import "errors"

// The causes of a refused or failed call. Every package wraps one of them
// and internal/wire.Error maps each to one wire code (rulings E4).
var (
	// ErrNotFound: the commission, topic, review, item or subject named does
	// not exist.
	ErrNotFound = errors.New("not found")
	// ErrInvalid: the request itself is wrong (bad option, text over the
	// bound, a path outside the allow-list).
	ErrInvalid = errors.New("refused")
	// ErrRefused: the request is well formed but the present state does not
	// allow it (read-only instance, unreadable file, a goal that is not
	// active).
	ErrRefused = errors.New("refused")
	// ErrUnavailable: a dependency the call needs cannot be reached.
	ErrUnavailable = errors.New("unavailable")
	// ErrNoResearchHome: the research home itself cannot be read; the
	// operator sets or creates it.
	ErrNoResearchHome = Cause(ErrRefused, errors.New("research home unreadable"))
)

// Cause marks err with one of the causes above and keeps its message.
func Cause(cause, err error) error {
	if err == nil || errors.Is(err, cause) {
		return err
	}
	return caused{cause: cause, err: err}
}

type caused struct{ cause, err error }

func (c caused) Error() string   { return c.err.Error() }
func (c caused) Unwrap() []error { return []error{c.err, c.cause} }
