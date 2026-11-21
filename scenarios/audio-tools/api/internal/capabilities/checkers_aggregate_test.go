package capabilities

import (
	"context"
	"testing"
)

type staticChecker Status

func (s staticChecker) Check(context.Context) (Status, string) { return Status(s), "test" }

func TestAggregateChecker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []Checker
		want Status
	}{
		{name: "none configured", want: StatusUnavailable},
		{name: "all unavailable", in: []Checker{staticChecker(StatusUnavailable)}, want: StatusUnavailable},
		{name: "one available", in: []Checker{staticChecker(StatusUnavailable), staticChecker(StatusAvailable)}, want: StatusAvailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := (AggregateChecker{Checkers: tt.in}).Check(context.Background())
			if got != tt.want {
				t.Fatalf("status = %q, want %q", got, tt.want)
			}
		})
	}
}
