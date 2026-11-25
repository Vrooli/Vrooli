package jobs

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestKindOfReadsTheKindThroughWrappingAndJoins(t *testing.T) {
	refused := Kinded(FailureRefused, errors.New("consent required"))
	cases := []struct {
		name string
		err  error
		want FailureKind
	}{
		{"plain error is a fault", errors.New("cuda out of memory"), FailureFault},
		{"nil is a fault", nil, FailureFault},
		{"kinded", refused, FailureRefused},
		{"wrapped", fmt.Errorf("ai: backend failed: %w", Kinded(FailureInvalidInput, errors.New("no label"))), FailureInvalidInput},
		{"joined", errors.Join(errors.New("cleanup"), refused), FailureRefused},
	}
	for _, tc := range cases {
		if got := KindOf(tc.err); got != tc.want {
			t.Errorf("%s: KindOf = %q, want %q", tc.name, got, tc.want)
		}
	}
	if Kinded(FailureRefused, nil) != nil {
		t.Error("Kinded must not invent an error")
	}
	if got := Kinded(FailureRefused, errors.New("x")).Error(); got != "x" {
		t.Errorf("Kinded changed the message: %q", got)
	}
}

func TestOnlyAFaultCountsAgainstAModelAndARecordFromBeforeTheKindIsAFault(t *testing.T) {
	for kind, counts := range map[FailureKind]bool{
		FailureFault: true, "": true, FailureRefused: false, FailureInvalidInput: false, FailureInterrupted: false, FailureCapacity: false, FailureSafetyUnavailable: false,
	} {
		if kind.CountsAgainstModel() != counts {
			t.Errorf("%q counts against a model = %t, want %t", kind, !counts, counts)
		}
	}
	if ParseFailureKind("") != FailureFault || ParseFailureKind("nonsense") != FailureFault || ParseFailureKind("refused") != FailureRefused {
		t.Error("an unknown or empty recorded kind must read as a fault")
	}
}

func TestAFailedJobRecordsWhoseFaultItWas(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want FailureKind
	}{
		{"fault", errors.New("the runner crashed"), FailureFault},
		{"refused", Kinded(FailureRefused, errors.New("output refused")), FailureRefused},
		{"invalid input", Kinded(FailureInvalidInput, errors.New("this parser has no label for 'book'")), FailureInvalidInput},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := startManager(t, Config{Runner: func(context.Context, Job, func(int, string)) (Result, error) { return Result{}, tc.err }})
			job, err := m.Submit(context.Background(), Spec{Operation: "segment", Lane: LaneCPU})
			if err != nil {
				t.Fatal(err)
			}
			final, err := m.Wait(context.Background(), job.ID)
			if err != nil {
				t.Fatal(err)
			}
			if final.State != StateFailed || final.Meta[MetaFailureKind] != string(tc.want) {
				t.Fatalf("state=%s meta=%v, want failed with kind %q", final.State, final.Meta, tc.want)
			}
		})
	}
}
