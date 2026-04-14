package contractapp

type RootResolver interface {
	ResolveRoot() (string, error)
}

type Service struct {
	ResolveRootFn     func() (string, error)
	ValidateFn        func(string) (ValidationOutput, error)
	ShowFn            func() (ShowOutput, error)
	ResolveScenarioFn func(string, string, string) (ResolveScenarioOutput, error)
	MatchGlobFn       func(string, string) (MatchGlobOutput, error)
}

type ResolveScenarioRequest struct {
	ScenarioName string
	FileKey      string
}

type MatchGlobRequest struct {
	Pattern string
	Path    string
}

func (s Service) ResolveRoot() (string, error) {
	return s.ResolveRootFn()
}

func (s Service) Validate() (ValidationOutput, error) {
	root, err := s.ResolveRootFn()
	if err != nil {
		return ValidationOutput{}, err
	}
	return s.ValidateFn(root)
}

func (s Service) Show() (ShowOutput, error) {
	return s.ShowFn()
}

func (s Service) ResolveScenario(req ResolveScenarioRequest) (ResolveScenarioOutput, error) {
	root, err := s.ResolveRootFn()
	if err != nil {
		return ResolveScenarioOutput{}, err
	}
	return s.ResolveScenarioFn(root, req.ScenarioName, req.FileKey)
}

func (s Service) MatchGlob(req MatchGlobRequest) (MatchGlobOutput, error) {
	return s.MatchGlobFn(req.Pattern, req.Path)
}
