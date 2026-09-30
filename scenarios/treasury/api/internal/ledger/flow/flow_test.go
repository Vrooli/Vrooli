package flow

import "testing"

func TestCompleteStateEventMatrix(t *testing.T) {
	tests := []struct {
		status    Status
		event     Event
		want      Status
		wantError bool
	}{
		{StatusQueued, EventDeliveryFailed, StatusQueued, false},
		{StatusQueued, EventDeliveryAccepted, StatusAccepted, false},
		{StatusAccepted, EventDeliveryFailed, StatusAccepted, true},
		{StatusAccepted, EventDeliveryAccepted, StatusAccepted, true},
	}
	for _, test := range tests {
		t.Run(string(test.status)+"_"+string(test.event), func(t *testing.T) {
			next, err := Transition(State{Status: test.status}, test.event)
			if (err != nil) != test.wantError {
				t.Fatalf("error=%v wantError=%v", err, test.wantError)
			}
			if next.Status != test.want {
				t.Fatalf("status=%s want=%s", next.Status, test.want)
			}
		})
	}
}

func TestRepresentativeTraces(t *testing.T) {
	traces := [][]Event{
		{EventDeliveryAccepted},
		{EventDeliveryFailed, EventDeliveryFailed, EventDeliveryAccepted},
	}
	for _, trace := range traces {
		state := InitialState()
		for _, event := range trace {
			var err error
			state, err = Transition(state, event)
			if err != nil {
				t.Fatalf("trace %v: %v", trace, err)
			}
		}
		if state.Status != StatusAccepted {
			t.Fatalf("trace %v ended at %s", trace, state.Status)
		}
	}
}
