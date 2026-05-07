package memberflow

import "context"

type OperatingGraphFilter struct {
	Team string
	ID   string
}

type OperatingGraphService struct {
	RepoRoot       string
	StoreDir       string
	PromptSections OperatingGraphPromptSectionProvider
}

func (s OperatingGraphService) List(_ context.Context, filter OperatingGraphFilter) (OperatingGraphListResponse, error) {
	blocks, err := LoadOperatingGraphBlocks(s.RepoRoot)
	if err != nil {
		return OperatingGraphListResponse{}, err
	}
	return OperatingGraphListResponse{Graphs: filterOperatingGraphBlocks(blocks, filter.Team, filter.ID)}, nil
}

func (s OperatingGraphService) Validate(ctx context.Context, filter OperatingGraphFilter) (OperatingGraphValidationResponse, error) {
	blocks, runtime, err := s.loadContractInputs(ctx, filter, true)
	if err != nil {
		return OperatingGraphValidationResponse{}, err
	}
	return OperatingGraphValidationResponse{
		Graphs:     blocks,
		Validation: ValidateOperatingGraphs(blocks, runtime, "", ""),
	}, nil
}

func (s OperatingGraphService) Diff(ctx context.Context, filter OperatingGraphFilter) (OperatingGraphDiffResponse, error) {
	blocks, runtime, err := s.loadContractInputs(ctx, filter, false)
	if err != nil {
		return OperatingGraphDiffResponse{}, err
	}
	return OperatingGraphDiffResponse{
		Graphs: blocks,
		Diff:   DiffOperatingGraphs(blocks, runtime, "", ""),
	}, nil
}

func (s OperatingGraphService) Coverage(ctx context.Context, filter OperatingGraphFilter) (OperatingGraphCoverageResponse, error) {
	blocks, runtime, err := s.loadContractInputs(ctx, filter, true)
	if err != nil {
		return OperatingGraphCoverageResponse{}, err
	}
	return OperatingGraphCoverageResponse{
		Graphs:   blocks,
		Coverage: BuildOperatingGraphCoverage(blocks, runtime, "", ""),
	}, nil
}

func (s OperatingGraphService) loadContractInputs(ctx context.Context, filter OperatingGraphFilter, includePromptSections bool) ([]OperatingGraphBlock, OperatingGraphRuntime, error) {
	blocks, err := LoadOperatingGraphBlocks(s.RepoRoot)
	if err != nil {
		return nil, OperatingGraphRuntime{}, err
	}
	runtime, err := BuildOperatingGraphRuntime(s.RepoRoot, s.StoreDir)
	if err != nil {
		return nil, OperatingGraphRuntime{}, err
	}
	if includePromptSections {
		runtime = s.withPromptSections(ctx, runtime)
	}
	return filterOperatingGraphBlocks(blocks, filter.Team, filter.ID), runtime, nil
}

func (s OperatingGraphService) withPromptSections(ctx context.Context, runtime OperatingGraphRuntime) OperatingGraphRuntime {
	if s.PromptSections == nil {
		return runtime
	}
	sectionsByMember := make(map[MemberRef][]OperatingGraphPromptSection, len(runtime.Members))
	for _, m := range runtime.Members {
		sections, err := s.PromptSections.SectionsForMember(ctx, m.Ref.Team, m.Ref.Member)
		if err != nil {
			sectionsByMember[m.Ref] = nil
			continue
		}
		sectionsByMember[m.Ref] = sections
	}
	runtime.PromptSections = sectionsByMember
	return runtime
}
