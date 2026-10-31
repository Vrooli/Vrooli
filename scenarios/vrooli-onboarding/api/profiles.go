package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	profilesdomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/profiles"
)

func onboardingProfilesService() profilesdomain.Service {
	return profilesdomain.Service{
		ProfileDir: func(_ context.Context) (string, error) {
			roots, err := resolveRoots()
			if err != nil {
				return "", err
			}
			candidates := []string{filepath.Join(roots.CatalogRoot, "scenarios", "vrooli-onboarding", "profiles"), filepath.Join(roots.CatalogRoot, "vrooli-onboarding", "profiles")}
			for _, candidate := range candidates {
				if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
					return candidate, nil
				}
			}
			return "", fmt.Errorf("onboarding profile catalog is unavailable; rebuild the bundle with scenarios/vrooli-onboarding/profiles")
		},
		Catalog: func(ctx context.Context) ([]profilesdomain.Scenario, error) {
			models, err := loadScenarioReadModels()
			if err != nil {
				return nil, err
			}
			result := make([]profilesdomain.Scenario, 0, len(models))
			for _, model := range models {
				result = append(result, profilesdomain.Scenario{Name: model.Name, Resources: model.Resources})
			}
			return result, nil
		},
	}
}
