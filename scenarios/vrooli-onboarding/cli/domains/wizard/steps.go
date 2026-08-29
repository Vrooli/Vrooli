package wizard

import (
	"fmt"
)

// stepHandler is the seam between the server-declared step model and the
// terminal surface. Keeping the id-to-handler map explicit makes an added API
// step fail loudly instead of silently disappearing from the CLI.
type stepHandler func(*wizardSession) error

type wizardSession struct {
	runStep func(string) error
}

type unimplementedStepError struct{ ID string }

func (e unimplementedStepError) Error() string {
	return fmt.Sprintf("onboarding step %q has no CLI handler", e.ID)
}

func dispatchStep(id string) stepHandler {
	return func(session *wizardSession) error {
		if session == nil || session.runStep == nil {
			return fmt.Errorf("onboarding step %q has no session dispatcher", id)
		}
		return session.runStep(id)
	}
}

var stepHandlers = map[string]stepHandler{
	"welcome":        dispatchStep("welcome"),
	"scenarios":      dispatchStep("scenarios"),
	"core-set":       dispatchStep("core-set"),
	"resources":      dispatchStep("resources"),
	"credentials":    dispatchStep("credentials"),
	"integrations":   dispatchStep("integrations"),
	"host":           dispatchStep("host"),
	"operating-mode": dispatchStep("operating-mode"),
	"apply":          dispatchStep("apply"),
	"validation":     dispatchStep("validation"),
}
