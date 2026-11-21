package resources

import (
	"errors"
	"github.com/vrooli/vrooli/internal/lightingruntime"
)

func (c *Controller) verifyOpenRGBNativeInitialLightingIntent(a openRGBInitialAttempt) error {
	owner, err := lightingruntime.NewController(c.Home, lightingruntime.Owners{})
	if err != nil {
		return err
	}
	intent, err := owner.Current()
	if err != nil {
		return err
	}
	return verifyOpenRGBNativeInitialLightingSelection(intent, a)
}
func verifyOpenRGBNativeInitialLightingSelection(intent lightingruntime.Intent, a openRGBInitialAttempt) error {
	expected := lightingruntime.Selection{Version: a.Contract.Artifact.Version, SHA256: a.Contract.Artifact.SHA256, DevicePolicyID: a.Policy.ID}
	if intent.Phase != lightingruntime.Enabled || intent.Selection == nil || *intent.Selection != expected {
		return errors.New("native initial admission no longer matches enabled lighting intent")
	}
	return nil
}
