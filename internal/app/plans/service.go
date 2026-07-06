package plans

import (
	"context"
	"fmt"
	"os"
	"time"
)

const (
	readSourceCanonical = "plan-manager"
	readSourceMirror    = "mirror-fallback"
)

type Service struct {
	Root         string
	Home         string
	Now          func() time.Time
	ReadFile     func(string) ([]byte, error)
	PlanManager  PlanManagerClient
	MirrorReader MirrorFallbackReader
}

type indexFile struct {
	Version int          `json:"version"`
	Plans   []PlanRecord `json:"plans"`
}

func (s Service) List(req ListRequest) (ListOutput, error) {
	workspace, err := s.workspaceScope(req.Workspace)
	if err != nil {
		return ListOutput{}, err
	}
	client, err := s.planManager()
	if err == nil {
		plans, listErr := client.ListPlans(context.Background(), workspace, req.IncludeArchived)
		if listErr == nil {
			return ListOutput{Success: true, Plans: plans, Source: readSourceCanonical}, nil
		}
		if !shouldUseMirrorFallback(listErr) {
			return ListOutput{}, listErr
		}
		err = listErr
	}
	if !shouldUseMirrorFallback(err) {
		return ListOutput{}, err
	}
	records, fallbackErr := s.mirrorReader().List(context.Background(), workspace, req.IncludeArchived)
	if fallbackErr != nil {
		if err != nil {
			return ListOutput{}, fmt.Errorf("plan-manager unavailable and mirror fallback failed: %w", fallbackErr)
		}
		return ListOutput{}, fallbackErr
	}
	return ListOutput{
		Success:  true,
		Plans:    records,
		Source:   readSourceMirror,
		Degraded: true,
		Warning:  fallbackWarning(err),
	}, nil
}

func (s Service) Show(req ShowRequest) (ShowOutput, error) {
	workspace, err := s.workspaceScope(req.Workspace)
	if err != nil {
		return ShowOutput{}, err
	}
	client, err := s.planManager()
	if err == nil {
		record, getErr := client.GetPlan(context.Background(), workspace, req.Ref)
		rendered, renderErr := client.RenderMarkdown(context.Background(), workspace, req.Ref)
		if getErr == nil && renderErr == nil {
			if rendered.Plan.Path != "" {
				record.Path = rendered.Plan.Path
			}
			rendered.Plan = record
			return ShowOutput{Success: true, Plan: rendered.Plan, Content: rendered.Content, Source: readSourceCanonical}, nil
		}
		err = firstNonNil(getErr, renderErr)
		if !shouldUseMirrorFallback(err) {
			return ShowOutput{}, err
		}
	}
	if !shouldUseMirrorFallback(err) {
		return ShowOutput{}, err
	}
	record, content, fallbackErr := s.mirrorReader().Read(context.Background(), workspace, req.Ref)
	if fallbackErr != nil {
		return ShowOutput{}, fallbackErr
	}
	return ShowOutput{
		Success:  true,
		Plan:     record,
		Content:  content,
		Source:   readSourceMirror,
		Degraded: true,
		Warning:  fallbackWarning(err),
	}, nil
}

func (s Service) Path(req ShowRequest) (PathOutput, error) {
	workspace, err := s.workspaceScope(req.Workspace)
	if err != nil {
		return PathOutput{}, err
	}
	client, err := s.planManager()
	if err == nil {
		record, getErr := client.GetPlan(context.Background(), workspace, req.Ref)
		if getErr == nil {
			return PathOutput{Success: true, ID: record.ID, Path: record.Path, Source: readSourceCanonical}, nil
		}
		err = getErr
		if !shouldUseMirrorFallback(err) {
			return PathOutput{}, err
		}
	}
	if !shouldUseMirrorFallback(err) {
		return PathOutput{}, err
	}
	record, fallbackErr := s.mirrorReader().Find(context.Background(), workspace, req.Ref)
	if fallbackErr != nil {
		return PathOutput{}, fallbackErr
	}
	return PathOutput{
		Success:  true,
		ID:       record.ID,
		Path:     record.Path,
		Source:   readSourceMirror,
		Degraded: true,
		Warning:  fallbackWarning(err),
	}, nil
}

func (s Service) planManager() (PlanManagerClient, error) {
	if s.PlanManager != nil {
		return s.PlanManager, nil
	}
	return NewDefaultPlanManagerClient(context.Background())
}

func (s Service) mirrorReader() MirrorFallbackReader {
	if s.MirrorReader != nil {
		return s.MirrorReader
	}
	return OSMirrorFallbackReader{Home: s.Home, Now: s.Now, ReadFile: s.ReadFile}
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s Service) readFile(path string) ([]byte, error) {
	if s.ReadFile != nil {
		return s.ReadFile(path)
	}
	return os.ReadFile(path)
}
