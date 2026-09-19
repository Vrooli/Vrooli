package policy

import (
	"context"
	"errors"
	"time"
)

const TimeFormat = time.RFC3339Nano

var (
	ErrNotFound    = errors.New("policy change not found")
	ErrUnsupported = errors.New("policy action unsupported by configured resolver adapter")
)

type Change struct {
	ID                string
	Target            string
	Action            string
	Status            string
	Values            []string
	Effects           []string
	RollbackSupported bool
	RollbackHandle    string
	ApprovalID        string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ApprovalRecord struct {
	ID        string
	ChangeID  string
	Approved  bool
	Note      string
	CreatedAt time.Time
}

type RollbackRecord struct {
	ID        string
	ChangeID  string
	Status    string
	Details   []string
	CreatedAt time.Time
}

type Profile struct {
	ID                string
	Name              string
	DeviceGroup       string
	FilteringStrength string
	Schedule          string
	OverrideBehavior  string
	Status            string
	Effects           []string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ScheduleEvaluation struct {
	ProfileID    string
	ProfileName  string
	Target       string
	Active       bool
	Status       string
	Effects      []string
	NextChangeAt time.Time
}

type GuidanceCheck struct {
	ID              string
	Title           string
	Status          string
	Evidence        string
	Recommendations []string
}

type GuidanceReport struct {
	ID             string
	Target         string
	Profile        string
	Status         string
	Checks         []GuidanceCheck
	ManualSteps    []string
	AdapterActions []string
	Guardrails     []string
	GeneratedAt    time.Time
}

type Repository interface {
	SaveChange(ctx context.Context, change Change) (Change, error)
	GetChange(ctx context.Context, id string) (Change, error)
	UpdateChange(ctx context.Context, change Change) (Change, error)
	SaveApproval(ctx context.Context, approval ApprovalRecord) (ApprovalRecord, error)
	SaveRollback(ctx context.Context, rollback RollbackRecord) (RollbackRecord, error)
	ListProfiles(ctx context.Context, deviceGroup string) ([]Profile, error)
	UpsertProfile(ctx context.Context, profile Profile) (Profile, error)
	GetProfile(ctx context.Context, id string) (Profile, error)
}

type AdapterPlan struct {
	Effects           []string
	RollbackSupported bool
}

type AdapterApplyResult struct {
	Effects           []string
	RollbackSupported bool
	RollbackHandle    string
}

type AdapterRollbackResult struct {
	Effects []string
}

type ResolverPolicyAdapter interface {
	Preview(ctx context.Context, change Change) (AdapterPlan, error)
	Apply(ctx context.Context, change Change) (AdapterApplyResult, error)
	Rollback(ctx context.Context, change Change) (AdapterRollbackResult, error)
}
