package work

import (
	workv1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/work"
	"google.golang.org/protobuf/types/known/timestamppb"
	"personal-planner/internal/work"
)

func domainToProto(item work.WorkItem) *workv1.WorkItem {
	return &workv1.WorkItem{Id: item.ID, Title: item.Title, Description: item.Description, RemainingMinutes: int32(item.RemainingMinutes), SourceLabel: item.SourceLabel, CreatedAt: timestamppb.New(item.CreatedAt.UTC()), UpdatedAt: timestamppb.New(item.UpdatedAt.UTC())}
}
