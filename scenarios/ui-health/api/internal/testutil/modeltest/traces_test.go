package modeltest_test

import (
	"testing"

	"ui-health/internal/testutil/modeltest"

	"github.com/stretchr/testify/require"
)

func TestValidateTraces_AcceptsMatchingTraces(t *testing.T) {
	errs := modeltest.ValidateTraces([]modeltest.Trace[status, event]{
		{
			Name:    "happy_path",
			Initial: statusIdle,
			Steps: []modeltest.TraceStep[status, event]{
				{Name: "start", Event: eventStart, Want: statusBusy},
				{Name: "finish", Event: eventFinish, Want: statusDone},
			},
		},
		{
			Name:    "reject_finish_before_start",
			Initial: statusIdle,
			Steps: []modeltest.TraceStep[status, event]{
				{Name: "finish", Event: eventFinish, Want: statusIdle, WantErr: true},
			},
		},
	}, transition)

	require.Empty(t, errs)
}

func TestValidateTraces_RejectsDrift(t *testing.T) {
	errs := modeltest.ValidateTraces([]modeltest.Trace[status, event]{
		{
			Name:    "bad_expectations",
			Initial: statusIdle,
			Steps: []modeltest.TraceStep[status, event]{
				{Name: "start", Event: eventStart, Want: statusDone},
				{Name: "start_again", Event: eventStart, Want: statusDone},
			},
		},
	}, transition)

	require.NotEmpty(t, errs)
	require.Contains(t, joined(errs), `bad_expectations/start: got status busy, want done`)
	require.Contains(t, joined(errs), `bad_expectations/start_again: unexpected error`)
}
