package androidadb

import (
	"context"
	"fmt"
)

func (a *Adapter) GoHome(ctx context.Context) error {
	_, err := a.runner.Run(ctx, "adb", a.args("shell", "input", "keyevent", "KEYCODE_HOME")...)
	return err
}

func (a *Adapter) DismissKeyguard(ctx context.Context) error {
	_, err := a.runner.Run(ctx, "adb", a.args("shell", "wm", "dismiss-keyguard")...)
	return err
}

func (a *Adapter) SetScreenState(ctx context.Context, state string) error {
	var key string
	switch state {
	case "on":
		key = "KEYCODE_WAKEUP"
	case "off":
		key = "KEYCODE_SLEEP"
	default:
		return fmt.Errorf("screen state must be on or off")
	}
	_, err := a.runner.Run(ctx, "adb", a.args("shell", "input", "keyevent", key)...)
	return err
}
