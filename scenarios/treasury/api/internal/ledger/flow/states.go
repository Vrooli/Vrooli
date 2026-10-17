package flow

type (
	Status string
	Event  string
)

const (
	StatusQueued   Status = "queued"
	StatusAccepted Status = "accepted"

	EventDeliveryFailed   Event = "delivery_failed"
	EventDeliveryAccepted Event = "delivery_accepted"
)

type State struct{ Status Status }

func InitialState() State { return State{Status: StatusQueued} }
