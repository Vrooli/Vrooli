package atspi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	atspiRoleApplication = 75
	atspiRoleWindow      = 23
)

var atspiStateNames = []string{
	"", "active", "armed", "busy", "checked", "collapsed", "defunct", "editable", "enabled", "expandable", "expanded", "focusable", "focused", "has_tooltip", "horizontal", "iconified", "modal", "multi_line", "multiselectable", "opaque", "pressed", "resizable", "selectable", "selected", "sensitive", "showing", "single_line", "stale", "transient", "vertical", "visible", "manages_descendants", "indeterminate", "required", "truncated", "animated", "invalid_entry", "supports_autocompletion", "selectable_text", "is_default", "visited", "checkable", "has_popup", "read_only", "last_defined",
}

// Interfaces returns the bounded interface list for one exact accessible
// object. Callers use it to distinguish unsupported optional capabilities from
// a broken or unauthorised object.
func (c *Client) Interfaces(ctx context.Context, ref Ref) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.identity(ctx, ref) != nil {
		return nil, ErrRefused
	}
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Accessible.GetInterfaces")
	var interfaces []string
	if err != nil || dbus.Store(body, &interfaces) != nil || len(interfaces) > 64 {
		return nil, ErrRefused
	}
	bytes := 0
	for _, name := range interfaces {
		if name == "" || len(name) > 256 {
			return nil, ErrRefused
		}
		bytes += len(name)
	}
	if bytes > 16*1024 {
		return nil, ErrRefused
	}
	return interfaces, nil
}

// Description reads the optional accessible description as a presentation
// label. It is bounded and never persisted by the helper.
func (c *Client) Description(ctx context.Context, ref Ref) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.identity(ctx, ref) != nil {
		return "", ErrRefused
	}
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.freedesktop.DBus.Properties.Get", "org.a11y.atspi.Accessible", "Description")
	var value dbus.Variant
	if err != nil || dbus.Store(body, &value) != nil {
		return "", ErrRefused
	}
	description, ok := value.Value().(string)
	if !ok || !validText(description) {
		return "", ErrRefused
	}
	return description, nil
}

// State returns the known AT-SPI state names for one exact object.
func (c *Client) State(ctx context.Context, ref Ref) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.identity(ctx, ref) != nil {
		return nil, ErrRefused
	}
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Accessible.GetState")
	if err != nil {
		return nil, ErrRefused
	}
	var values []uint32
	if dbus.Store(body, &values) == nil {
		states := make([]string, 0, len(values))
		for _, value := range values {
			if value == 0 || value >= uint32(len(atspiStateNames)) || value == 44 || atspiStateNames[value] == "" {
				continue
			}
			states = append(states, atspiStateNames[value])
		}
		return states, nil
	}
	var bits uint64
	if dbus.Store(body, &bits) != nil {
		var narrow uint32
		if dbus.Store(body, &narrow) != nil {
			return nil, ErrRefused
		}
		bits = uint64(narrow)
	}
	states := make([]string, 0, len(atspiStateNames))
	for bit, name := range atspiStateNames {
		if name != "" && name != "last_defined" && bits&(uint64(1)<<uint(bit)) != 0 {
			states = append(states, name)
		}
	}
	return states, nil
}

// Bounds returns screen-coordinate extents when the exact object implements
// Component. Unsupported components are represented by ok=false.
func (c *Client) Bounds(ctx context.Context, ref Ref) (x int32, y int32, width uint32, height uint32, ok bool, err error) {
	interfaces, err := c.Interfaces(ctx, ref)
	if err != nil {
		return 0, 0, 0, 0, false, err
	}
	if !containsInterface(interfaces, "org.a11y.atspi.Component") {
		return 0, 0, 0, 0, false, nil
	}
	body, callErr := c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Component.GetExtents", int32(0))
	var rect struct {
		X      int32
		Y      int32
		Width  int32
		Height int32
	}
	if callErr != nil || dbus.Store(body, &rect) != nil || rect.Width <= 0 || rect.Height <= 0 {
		return 0, 0, 0, 0, false, ErrRefused
	}
	return rect.X, rect.Y, uint32(rect.Width), uint32(rect.Height), true, nil
}

// Actions returns bounded native action names for one exact object.
func (c *Client) Actions(ctx context.Context, ref Ref) ([]string, bool, error) {
	interfaces, err := c.Interfaces(ctx, ref)
	if err != nil {
		return nil, false, err
	}
	if !containsInterface(interfaces, "org.a11y.atspi.Action") {
		return nil, false, nil
	}
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Action.GetNActions")
	var count int32
	if err != nil || dbus.Store(body, &count) != nil || count < 0 || count > 64 {
		return nil, false, ErrRefused
	}
	actions := make([]string, 0, count)
	for i := int32(0); i < count; i++ {
		body, err = c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Action.GetName", i)
		var name string
		if err != nil || dbus.Store(body, &name) != nil || name == "" || len(name) > 256 {
			return nil, false, ErrRefused
		}
		actions = append(actions, name)
	}
	return actions, true, nil
}

func containsInterface(interfaces []string, wanted string) bool {
	for _, name := range interfaces {
		if name == wanted {
			return true
		}
	}
	return false
}

// Focus acquires focus through the exact Component object immediately before
// an accessibility mutation. A missing/false native acknowledgement refuses.
func (c *Client) Focus(ctx context.Context, ref Ref) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.identity(ctx, ref) != nil {
		return ErrRefused
	}
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Component.GrabFocus")
	if err != nil {
		return ErrRefused
	}
	if len(body) == 0 {
		return nil
	}
	var accepted bool
	if dbus.Store(body, &accepted) != nil || !accepted {
		return ErrRefused
	}
	return nil
}

// InvokeNamed maps a semantic action to an action exposed by the exact
// accessible object. It never chooses a name based on list position.
func (c *Client) InvokeNamed(ctx context.Context, ref Ref, wanted string) error {
	wanted = strings.ToLower(strings.TrimSpace(wanted))
	if wanted == "" {
		return ErrRefused
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.identity(ctx, ref) != nil || c.guard(ctx, ref) != nil {
		return ErrRefused
	}
	actions, ok, err := c.Actions(ctx, ref)
	if err != nil || !ok {
		return ErrRefused
	}
	for index, action := range actions {
		if strings.ToLower(strings.TrimSpace(action)) != wanted {
			continue
		}
		body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Action.DoAction", int32(index))
		if err != nil {
			return ErrOutcomeUnknown
		}
		if len(body) == 0 {
			return nil
		}
		var accepted bool
		if dbus.Store(body, &accepted) != nil || !accepted {
			return ErrOutcomeUnknown
		}
		return nil
	}
	return fmt.Errorf("%w: semantic action %q unavailable", ErrRefused, wanted)
}

// Children stays within one exact application owner. Embedded foreign-process
// objects require separate admission; they are not silently followed.
func (c *Client) Children(ctx context.Context, ref Ref) ([]Ref, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.identity(ctx, ref) != nil {
		return nil, ErrRefused
	}
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Accessible.GetChildren")
	var children []struct {
		Owner string
		Path  dbus.ObjectPath
	}
	if err != nil || dbus.Store(body, &children) != nil || len(children) > 128 {
		return nil, ErrRefused
	}
	result := make([]Ref, 0, len(children))
	for _, child := range children {
		if child.Owner != ref.Owner || !child.Path.IsValid() {
			return nil, ErrRefused
		}
		next := ref
		next.Path = child.Path
		result = append(result, next)
	}
	return result, nil
}

func (c *Client) Name(ctx context.Context, ref Ref) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.identity(ctx, ref) != nil {
		return "", ErrRefused
	}
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.freedesktop.DBus.Properties.Get", "org.a11y.atspi.Accessible", "Name")
	var value dbus.Variant
	if err != nil || dbus.Store(body, &value) != nil {
		return "", ErrRefused
	}
	name, ok := value.Value().(string)
	if !ok || len(name) > 4096 {
		return "", ErrRefused
	}
	return name, nil
}

// Role returns the native semantic role after revalidating exact identity.
func (c *Client) Role(ctx context.Context, ref Ref) (uint32, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.identity(ctx, ref) != nil {
		return 0, ErrRefused
	}
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Accessible.GetRole")
	var role uint32
	if err != nil || dbus.Store(body, &role) != nil {
		return 0, ErrRefused
	}
	return role, nil
}

// Invoke activates the first native action exposed by an exact accessible
// element. The identity and action count are rechecked immediately before the
// mutation; a transport failure is reported as an unknown outcome so callers
// do not blindly retry a possibly delivered activation.
func (c *Client) Invoke(ctx context.Context, ref Ref) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.identity(ctx, ref) != nil {
		return ErrRefused
	}
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Action.GetNActions")
	var count int32
	if err != nil {
		return ErrRefused
	}
	if dbus.Store(body, &count) != nil {
		var unsigned uint32
		if dbus.Store(body, &unsigned) != nil || unsigned > 64 {
			return ErrRefused
		}
		count = int32(unsigned)
	}
	if count <= 0 || count > 64 {
		return ErrRefused
	}
	if c.guard(ctx, ref) != nil {
		return ErrRefused
	}
	body, err = c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Action.DoAction", int32(0))
	if err != nil {
		return ErrOutcomeUnknown
	}
	// A few AT-SPI bridges expose DoAction as a void method even though the
	// canonical interface returns a boolean. A successful, empty reply still
	// confirms that the request crossed the native boundary.
	if len(body) == 0 {
		return nil
	}
	var accepted bool
	if dbus.Store(body, &accepted) != nil || !accepted {
		return ErrOutcomeUnknown
	}
	return nil
}
