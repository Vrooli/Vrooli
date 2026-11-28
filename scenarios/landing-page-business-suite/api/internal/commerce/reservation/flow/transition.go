package flow

import "landing-page-business-suite-api/internal/commerce/reservation/flow/generated"

// TransitionReservation delegates lifecycle topology to the generated contract.
func TransitionReservation(status generated.ReservationStatus, event generated.ReservationEvent) (generated.ReservationStatus, error) {
	return generated.TransitionReservationStatus(status, event)
}
