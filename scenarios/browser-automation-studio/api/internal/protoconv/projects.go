package protoconv

import (
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	basprojects "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ProjectToProto converts a database.ProjectIndex (DB index-only row) into the generated proto message.
// Project metadata like description is sourced from filesystem protojson, not the database.
func ProjectToProto(project *database.ProjectIndex) *basprojects.Project {
	if project == nil {
		return nil
	}
	pb := &basprojects.Project{
		Id:          project.ID.String(),
		Name:        project.Name,
		FolderPath:  project.FolderPath,
		CreatedAt:   timestamppb.New(project.CreatedAt),
		UpdatedAt:   timestamppb.New(project.UpdatedAt),
	}
	return pb
}

// ProjectStatsToProto converts aggregated project stats to proto.
func ProjectStatsToProto(stats *database.ProjectStats) *basprojects.ProjectStats {
	if stats == nil {
		return nil
	}
	pb := &basprojects.ProjectStats{
		ProjectId:      stats.ProjectID.String(),
		WorkflowCount:  int32(stats.WorkflowCount),
		ExecutionCount: int32(stats.ExecutionCount),
	}
	if stats.LastExecution != nil {
		pb.LastExecution = timestamppb.New(*stats.LastExecution)
	}
	return pb
}

// ProjectStatsFromMap constructs stats from a loose map (handler/service legacy).
func ProjectStatsFromMap(stats map[string]any, projectID uuid.UUID) *database.ProjectStats {
	if stats == nil {
		return nil
	}
	result := &database.ProjectStats{
		ProjectID: projectID,
	}
	if v, ok := stats["workflow_count"].(int); ok {
		result.WorkflowCount = v
	}
	if v, ok := stats["workflow_count"].(float64); ok {
		result.WorkflowCount = int(v)
	}
	if v, ok := stats["execution_count"].(int); ok {
		result.ExecutionCount = v
	}
	if v, ok := stats["execution_count"].(float64); ok {
		result.ExecutionCount = int(v)
	}
	if ts, ok := stats["last_execution"]; ok {
		switch t := ts.(type) {
		case *time.Time:
			result.LastExecution = t
		case time.Time:
			result.LastExecution = &t
		}
	}
	return result
}

// ProjectWithStatsToProto bundles project plus stats.
func ProjectWithStatsToProto(project *database.ProjectIndex, stats *database.ProjectStats) *basprojects.ProjectWithStats {
	if project == nil {
		return nil
	}
	return &basprojects.ProjectWithStats{
		Project: ProjectToProto(project),
		Stats:   ProjectStatsToProto(stats),
	}
}
