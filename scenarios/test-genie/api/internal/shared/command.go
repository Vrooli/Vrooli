package shared

// LookupFunc is a function that looks up a command by name.
// This is used across lint and dependencies packages.
type LookupFunc func(name string) (string, error)
