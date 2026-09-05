// Package schedules hosts the BAS SchedulesService Connect-RPC handler.
//
// SchedulesService owns cron schedule CRUD, manual trigger, on/off toggle,
// and the calendar-occurrences projection used by the Settings → Schedules
// tab and the dashboard Schedules section.
package schedules

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"

	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	schedulesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schedules/schedulesconnect"
)

// Repo is the narrow seam onto database.Repository that SchedulesService needs.
type Repo interface {
	CreateSchedule(ctx context.Context, schedule *database.ScheduleIndex) error
	GetSchedule(ctx context.Context, id uuid.UUID) (*database.ScheduleIndex, error)
	UpdateSchedule(ctx context.Context, schedule *database.ScheduleIndex) error
	DeleteSchedule(ctx context.Context, id uuid.UUID) error
	ListSchedules(ctx context.Context, workflowID *uuid.UUID, activeOnly bool, limit, offset int) ([]*database.ScheduleIndex, error)
	UpdateScheduleLastRun(ctx context.Context, id uuid.UUID, lastRun time.Time) error
}

// Catalog is the narrow seam used to validate workflow existence on create
// and to hydrate workflow_name onto schedule responses.
type Catalog interface {
	GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowSummary, error)
}

// Executor triggers a workflow run; used by Trigger.
type Executor interface {
	ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error)
}

// Deps wires the schedules handler.
type Deps struct {
	Repo     Repo
	Catalog  Catalog
	Executor Executor
	Logger   *logrus.Logger
}

// Module builds the SchedulesService Connect handler.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("schedules.Module requires Deps.Logger")
	}
	if d.Repo == nil {
		panic("schedules.Module requires Deps.Repo")
	}
	if d.Catalog == nil {
		panic("schedules.Module requires Deps.Catalog")
	}
	if d.Executor == nil {
		panic("schedules.Module requires Deps.Executor")
	}
	path, handler := schedulesconnect.NewSchedulesServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}

var _ schedulesconnect.SchedulesServiceHandler = (*service)(nil)
