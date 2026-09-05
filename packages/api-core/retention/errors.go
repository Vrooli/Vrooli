package retention

import "errors"

var (
	// ErrUnknownUnit is returned when a max_age or max_bytes value carries a
	// unit the parser does not recognize. Units are mandatory precisely so a
	// mistake here is loud: a silently misread byte ceiling is a disk outage.
	ErrUnknownUnit = errors.New("retention: unknown unit")

	// ErrNoBound is returned when a budget declares neither max_age nor
	// max_bytes. Such a budget cannot be enforced and must not be accepted as
	// compliant.
	ErrNoBound = errors.New("retention: budget declares no bound")

	// ErrPrunerNotRegistered is returned when a budget names a custom pruner
	// that nothing registered. The engine fails loudly rather than falling back
	// to the builtin pruner, because a silent fallback would apply a generic age
	// rule to data that has a domain selection rule and delete the wrong items.
	ErrPrunerNotRegistered = errors.New("retention: custom pruner not registered")

	// ErrInvalidTarget is returned when a target is missing a field its kind
	// requires, or names a kind the engine does not implement.
	ErrInvalidTarget = errors.New("retention: invalid target")

	// ErrNoBuiltinFactory is returned when a budget asks for the builtin pruner
	// but the engine was constructed without one.
	ErrNoBuiltinFactory = errors.New("retention: no builtin pruner factory configured")

	// ErrInsufficientSpace is returned when an operation needs more free space
	// than the filesystem has. It is a refusal before any write, not a failure
	// part way through one: a retention mechanism that fills the disk it is
	// clearing is worse than one that declines and says so.
	ErrInsufficientSpace = errors.New("retention: insufficient free space")
)
