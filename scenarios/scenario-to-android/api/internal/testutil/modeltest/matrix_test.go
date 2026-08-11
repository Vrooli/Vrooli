package modeltest_test

import (
	"errors"
	"testing"

	"scenario-to-android/internal/testutil/modeltest"

	"github.com/stretchr/testify/require"
)

type status string

const (
	statusIdle status = "idle"
	statusBusy status = "busy"
	statusDone status = "done"
)

type event string

const (
	eventStart  event = "start"
	eventFinish event = "finish"
)

func transition(s status, e event) (status, error) {
	switch {
	case s == statusIdle && e == eventStart:
		return statusBusy, nil
	case s == statusBusy && e == eventFinish:
		return statusDone, nil
	default:
		return s, errors.New("illegal transition")
	}
}

func validRows() []modeltest.MatrixRow[status, event] {
	return []modeltest.MatrixRow[status, event]{
		{Name: "idle_start", From: statusIdle, Event: eventStart, To: statusBusy},
		{Name: "idle_finish", From: statusIdle, Event: eventFinish, To: statusIdle, WantErr: true},
		{Name: "busy_start", From: statusBusy, Event: eventStart, To: statusBusy, WantErr: true},
		{Name: "busy_finish", From: statusBusy, Event: eventFinish, To: statusDone},
		{Name: "done_start", From: statusDone, Event: eventStart, To: statusDone, WantErr: true},
		{Name: "done_finish", From: statusDone, Event: eventFinish, To: statusDone, WantErr: true},
	}
}

func TestValidateTransitionMatrix_AcceptsCompleteMatrix(t *testing.T) {
	errs := modeltest.ValidateTransitionMatrix(
		[]status{statusIdle, statusBusy, statusDone},
		[]event{eventStart, eventFinish},
		validRows(),
		transition,
	)
	require.Empty(t, errs)
}

func TestValidateTransitionMatrix_RejectsStructuralDrift(t *testing.T) {
	rows := validRows()
	rows = append(rows[:1], rows[2:]...)
	rows = append(rows, modeltest.MatrixRow[status, event]{
		Name:  "unknown",
		From:  "ghost",
		Event: eventStart,
		To:    statusBusy,
	})
	rows = append(rows, modeltest.MatrixRow[status, event]{
		Name:  "duplicate",
		From:  statusIdle,
		Event: eventStart,
		To:    statusBusy,
	})

	errs := modeltest.ValidateTransitionMatrix(
		[]status{statusIdle, statusBusy, statusDone},
		[]event{eventStart, eventFinish},
		rows,
		transition,
	)
	require.NotEmpty(t, errs)
	require.Contains(t, joined(errs), `unknown from status ghost`)
	require.Contains(t, joined(errs), `duplicate pair idle/start`)
	require.Contains(t, joined(errs), `missing pair idle/finish`)
}

func TestValidateTransitionMatrix_RejectsBehaviorDrift(t *testing.T) {
	rows := validRows()
	rows[0].To = statusDone
	rows[1].WantErr = false

	errs := modeltest.ValidateTransitionMatrix(
		[]status{statusIdle, statusBusy, statusDone},
		[]event{eventStart, eventFinish},
		rows,
		transition,
	)
	require.NotEmpty(t, errs)
	require.Contains(t, joined(errs), `idle_start: got status busy, want done`)
	require.Contains(t, joined(errs), `idle_finish: unexpected error`)
}

func joined(errs []error) string {
	out := ""
	for _, err := range errs {
		out += err.Error() + "\n"
	}
	return out
}
