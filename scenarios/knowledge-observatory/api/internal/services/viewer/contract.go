package viewer

import (
	"path/filepath"
	"strings"

	"knowledge-observatory/internal/doccontract"
	"knowledge-observatory/internal/doctemplates"
)

func (s *Service) resolveContractDoc(scenarioName, id string) (doccontract.Document, string, error) {
	scenarioName = strings.TrimSpace(scenarioName)
	if scenarioName == "" {
		return doccontract.Document{}, "", ErrScenarioRequired
	}
	if strings.Contains(scenarioName, "/") || strings.Contains(scenarioName, "\\") || strings.Contains(scenarioName, "..") {
		return doccontract.Document{}, "", ErrScenarioInvalid
	}
	scenarioPath := filepath.Join(s.scenariosRoot, scenarioName)
	resolved, err := doctemplates.NewResolverFromScenariosRoot(s.scenariosRoot).ResolveScenario(scenarioPath)
	if err != nil {
		return doccontract.Document{}, "", err
	}
	doc, ok := resolved.Contract.ResolveIdentifier(id)
	if !ok {
		return doccontract.Document{}, "", ErrDocTypeInvalid
	}
	return doc, scenarioPath, nil
}

func splitScenarioRepoPath(repoRel string) (string, string, bool) {
	repoRel = filepath.ToSlash(strings.TrimSpace(repoRel))
	parts := strings.Split(repoRel, "/")
	if len(parts) < 3 || parts[0] != "scenarios" {
		return "", "", false
	}
	return parts[1], strings.Join(parts[2:], "/"), true
}
