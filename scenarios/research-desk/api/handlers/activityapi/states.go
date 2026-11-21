package activityapi

import "research-desk/internal/home"

// stateOrder is the display order of state counts: running first.
var stateOrder = []home.State{home.StateLive, home.StatePaused, home.StateLimited, home.StateStopped, home.StatePrepared, home.StateUnknown, home.StateEnded}
