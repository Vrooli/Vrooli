package memberflow

import "context"

type OperatingModelService struct {
	RepoRoot       string
	StoreDir       string
	PromptSections OperatingGraphPromptSectionProvider
}

func (s OperatingModelService) List(_ context.Context, filter OperatingModelFilter) (OperatingModelListResponse, error) {
	models, err := LoadOperatingModelDocuments(s.RepoRoot)
	if err != nil {
		return OperatingModelListResponse{}, err
	}
	return OperatingModelListResponse{Models: filterOperatingModelDocuments(models, filter.Team, filter.ID)}, nil
}

func (s OperatingModelService) Validate(ctx context.Context, filter OperatingModelFilter) (OperatingModelValidationResponse, error) {
	models, runtime, err := s.loadContractInputs(ctx, filter, true)
	if err != nil {
		return OperatingModelValidationResponse{}, err
	}
	return OperatingModelValidationResponse{
		Models:     models,
		Validation: ValidateOperatingModels(models, runtime, "", ""),
	}, nil
}

func (s OperatingModelService) Diff(ctx context.Context, filter OperatingModelFilter) (OperatingModelDiffResponse, error) {
	models, runtime, err := s.loadContractInputs(ctx, filter, false)
	if err != nil {
		return OperatingModelDiffResponse{}, err
	}
	blocks := operatingGraphBlocksFromModels(models)
	return OperatingModelDiffResponse{
		Models: models,
		Diff:   DiffOperatingGraphs(blocks, runtime, "", ""),
	}, nil
}

func (s OperatingModelService) Coverage(ctx context.Context, filter OperatingModelFilter) (OperatingModelCoverageResponse, error) {
	models, runtime, err := s.loadContractInputs(ctx, filter, true)
	if err != nil {
		return OperatingModelCoverageResponse{}, err
	}
	return OperatingModelCoverageResponse{
		Models:   models,
		Coverage: BuildOperatingModelCoverage(models, runtime, "", ""),
	}, nil
}

func (s OperatingModelService) loadContractInputs(ctx context.Context, filter OperatingModelFilter, includePromptSections bool) ([]OperatingModelDocument, OperatingGraphRuntime, error) {
	models, err := LoadOperatingModelDocuments(s.RepoRoot)
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
	return filterOperatingModelDocuments(models, filter.Team, filter.ID), runtime, nil
}

func (s OperatingModelService) withPromptSections(ctx context.Context, runtime OperatingGraphRuntime) OperatingGraphRuntime {
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
