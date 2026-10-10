// unit_owner_driver.go binds Runtime to its opaque local owner and existing streaming carrier.
package nativeisolation

import (
	"context"
	"io"
	"strings"
)

// fixedUnitDriver is private and cannot be supplied through an interface or
// boolean by callers. Native streaming stays in the owning process and uses
// the existing child carrier. HTTP is explicitly not a native model carrier.
type fixedUnitDriver struct{ owner *FixedUnitOwner }

func (d *fixedUnitDriver) prepare(ctx context.Context, p Plan) error {
	r := p.Request
	for _, value := range p.Env {
		if value != "LANG=C" && value != "LC_ALL=C" {
			r.Env = append(r.Env, value)
		}
	}
	_, e := d.owner.PrepareNative(ctx, r)
	return e
}
func (d *fixedUnitDriver) planForUnit(unit string) (*UnitPlan, error) {
	d.owner.mu.Lock()
	defer d.owner.mu.Unlock()
	for selector, u := range d.owner.units {
		if u.plan.Unit == unit && u.probe == nil {
			return &UnitPlan{d.owner, selector}, nil
		}
	}
	return nil, ErrRefused
}
func (d *fixedUnitDriver) Start(ctx context.Context, p Plan, input io.Reader) (child, error) {
	h, e := d.planForUnit(p.Unit)
	if e != nil {
		return nil, e
	}
	return d.owner.start(ctx, h.selector, input)
}
func (d *fixedUnitDriver) Show(ctx context.Context, unit string) (map[string]string, error) {
	h, e := d.planForUnit(unit)
	if e != nil {
		return nil, e
	}
	d.owner.mu.Lock()
	u := d.owner.units[h.selector]
	started := u.started
	unitName := u.plan.Unit
	d.owner.mu.Unlock()
	if !started {
		return d.owner.driver.Show(ctx, unitName)
	}
	return d.owner.read(ctx, h.selector)
}
func (d *fixedUnitDriver) Signal(ctx context.Context, unit, signal string) error {
	if signal != "SIGTERM" && signal != "SIGKILL" {
		return ErrRefused
	}
	h, e := d.planForUnit(unit)
	if e != nil {
		return e
	}
	return d.owner.stop(ctx, h.selector, strings.EqualFold(signal, "SIGKILL"))
}
