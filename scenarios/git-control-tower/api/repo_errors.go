package main

import "fmt"

// RepoErrorKind describes repo resolution failures.
type RepoErrorKind string

const (
	RepoErrorNotFound RepoErrorKind = "not_found"
	RepoErrorInvalid  RepoErrorKind = "invalid"
	RepoErrorMissing  RepoErrorKind = "missing"
)

// RepoError indicates a repo resolution or validation failure.
type RepoError struct {
	Kind    RepoErrorKind
	Message string
	Cause   error
}

func (e RepoError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return fmt.Sprintf("repo error (%s)", e.Kind)
}

func (e RepoError) Unwrap() error {
	return e.Cause
}

func newRepoError(kind RepoErrorKind, message string, cause error) RepoError {
	return RepoError{Kind: kind, Message: message, Cause: cause}
}
