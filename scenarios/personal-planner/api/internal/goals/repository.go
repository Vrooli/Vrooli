package goals

import "context"

type Repository interface {
	List(context.Context) ([]Goal, error)
	Create(context.Context, Goal) (Goal, error)
	UpdateProgress(context.Context, string, int64, int64) (Goal, error)
	ListMilestones(context.Context, string) ([]Milestone, error)
	CreateMilestone(context.Context, Milestone) (Milestone, error)
	UpdateMilestoneStatus(context.Context, string, string, int64) (Milestone, error)
	WorkItemExists(context.Context, string) (bool, error)
	MilestonesExistForGoal(context.Context, string, []string) (bool, error)
}
