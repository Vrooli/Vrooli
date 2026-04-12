package main

import (
	"fmt"
	"path/filepath"
	"sync"

	"scenario-auditor/internal/ruleengine"
)

var (
	ruleServiceOnce sync.Once
	ruleServiceInst *ruleengine.Service
	ruleServiceErr  error
)

func ruleService() (*ruleengine.Service, error) {
	ruleServiceOnce.Do(func() {
		repoRoot, err := resolveRepoRoot()
		if err != nil {
			ruleServiceErr = err
			return
		}

		ruleDirs, err := ruleengine.DiscoverRuleDirs(repoRoot)
		if err != nil {
			ruleServiceErr = err
			return
		}

		scenarioRoot, err := resolveScenarioAuditorRoot()
		if err != nil {
			ruleServiceErr = err
			return
		}
		moduleRoot := filepath.Join(scenarioRoot, "api")
		opts := ruleengine.Options{
			RuleDirs:   ruleDirs,
			ModuleRoot: moduleRoot,
		}

		service, err := ruleengine.NewService(opts)
		if err != nil {
			ruleServiceErr = err
			return
		}

		// Warm the cache early so endpoint handlers see errors immediately.
		if _, err := service.Reload(); err != nil {
			ruleServiceErr = err
			return
		}

		ruleServiceInst = service
	})

	if ruleServiceErr != nil {
		return nil, fmt.Errorf("ruleservice init: %w", ruleServiceErr)
	}
	return ruleServiceInst, nil
}
