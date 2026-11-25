package mocks

import (
	"context"

	"document-manager/internal/derivation"
)

type Runner struct {
	Err   error
	Model derivation.Model
	Calls int
}

func (r *Runner) Run(context.Context, derivation.Handler, derivation.Input) (derivation.Model, error) {
	r.Calls++
	if r.Err != nil {
		return derivation.Model{}, r.Err
	}
	return r.Model, nil
}

type Store struct {
	Next    int
	Results []derivation.Result
}

func (s *Store) NextVersion(context.Context, string) (int, error) {
	s.Next++
	return s.Next, nil
}

func (s *Store) Append(_ context.Context, result derivation.Result) error {
	s.Results = append(s.Results, result)
	return nil
}
