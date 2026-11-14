package flow

import (
	"testing"

	"landing-page-business-suite-api/internal/commerce/reservation/flow/generated"
)

func TestReservationFormalReplay(t *testing.T) {
	generated.RunReplay(t, TransitionReservation)
}
