package control

import (
	"context"

	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
)

type ZoneMapping struct {
	ControllerID       string
	ZoneID             string
	Fingerprint        string
	Name               string
	LEDCount           uint32
	Static             bool
	LEDMin, LEDMax     uint32
	ConfiguredLEDCount uint32
	Effects            []string
}

// SelectionResult separates a superseded request from an already committed one.
type SelectionResult int

const (
	SelectionSuperseded SelectionResult = iota
	SelectionCompleted
	SelectionCommitted
)

type Repository interface {
	SaveZoneConfiguration(context.Context, string, string, uint32) error
	SyncControllers(context.Context, []*controlv1.Controller, []ZoneMapping) error
	ListControllers(context.Context) ([]*controlv1.Controller, error)
	SetEnrollment(context.Context, string, bool) (*controlv1.Controller, error)
	ListZones(context.Context, string) ([]ZoneMapping, error)
	SaveDesired(context.Context, *controlv1.ZoneState, *controlv1.ZoneApplyResult) error
	SavePolicyDesired(context.Context, *controlv1.ZoneState, *controlv1.ZoneApplyResult) error
	SaveApply(context.Context, *controlv1.ZoneState, *controlv1.ZoneApplyResult) error
	ListDesired(context.Context) ([]*controlv1.ZoneStatus, error)
	ListManualDesired(context.Context) ([]*controlv1.ZoneStatus, error)
	DesiredSource(context.Context, string, string) (string, error)
	PolicyRevision(context.Context) (int64, error)
	SelectPolicy(context.Context, PolicySelection, *controlv1.ManualOverride) (SelectionResult, error)
	SetManualOverride(context.Context, *controlv1.ManualOverride) error
	GetManualOverride(context.Context) (*controlv1.ManualOverride, error)
	ClearManualOverride(context.Context) error
}
