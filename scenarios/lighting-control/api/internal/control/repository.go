package control

import (
	"context"

	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
)

type ZoneMapping struct {
	ControllerID string
	ZoneID       string
	Fingerprint  string
	Name         string
	LEDCount     uint32
	Static       bool
}

type Repository interface {
	SyncControllers(context.Context, []*controlv1.Controller, []ZoneMapping) error
	ListControllers(context.Context) ([]*controlv1.Controller, error)
	SetEnrollment(context.Context, string, bool) (*controlv1.Controller, error)
	ListZones(context.Context, string) ([]ZoneMapping, error)
	SaveDesired(context.Context, *controlv1.ZoneState, *controlv1.ZoneApplyResult) error
	ListDesired(context.Context) ([]*controlv1.ZoneStatus, error)
	SetManualOverride(context.Context, *controlv1.ManualOverride) error
	GetManualOverride(context.Context) (*controlv1.ManualOverride, error)
	ClearManualOverride(context.Context) error
}
