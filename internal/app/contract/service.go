package contractapp

import "github.com/vrooli/vrooli/internal/cli/contractcli"

type RootResolver interface {
	ResolveRoot() (string, error)
}

type Service struct {
	ResolveRootFn     func() (string, error)
	ValidateFn        func(string) (contractcli.ValidationOutput, error)
	ShowFn            func() (contractcli.ShowOutput, error)
	ResolveScenarioFn func(string, string, string) (contractcli.ResolveScenarioOutput, error)
	MatchGlobFn       func(string, string) (contractcli.MatchGlobOutput, error)
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

func (s Service) Validate() (contractcli.ValidationOutput, error) {
	root, err := s.ResolveRootFn()
	if err != nil {
		return contractcli.ValidationOutput{}, err
	}
	return s.ValidateFn(root)
}

func (s Service) Show() (contractcli.ShowOutput, error) {
	return s.ShowFn()
}

func (s Service) ResolveScenario(req ResolveScenarioRequest) (contractcli.ResolveScenarioOutput, error) {
	root, err := s.ResolveRootFn()
	if err != nil {
		return contractcli.ResolveScenarioOutput{}, err
	}
	return s.ResolveScenarioFn(root, req.ScenarioName, req.FileKey)
}

func (s Service) MatchGlob(req MatchGlobRequest) (contractcli.MatchGlobOutput, error) {
	return s.MatchGlobFn(req.Pattern, req.Path)
}
