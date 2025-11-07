package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SaaSDetectionService scans and analyzes scenarios for SaaS characteristics
type SaaSDetectionService struct {
	scenariosPath string
	dbService     *DatabaseService
}

func NewSaaSDetectionService(scenariosPath string, dbService *DatabaseService) *SaaSDetectionService {
	return &SaaSDetectionService{
		scenariosPath: scenariosPath,
		dbService:     dbService,
	}
}

func (sds *SaaSDetectionService) ScanScenarios(forceRescan bool, scenarioFilter string) (*ScanResponse, error) {
	scenariosDir := sds.scenariosPath
	if scenariosDir == "" {
		scenariosDir = DefaultScenariosPath
	}

	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenarios directory: %w", err)
	}

	var totalScenarios, saasScenarios, newlyDetected int
	var scenarios []SaaSScenario

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scenarioName := entry.Name()
		if scenarioFilter != "" && !strings.Contains(scenarioName, scenarioFilter) {
			continue
		}

		totalScenarios++

		// Check if scenario is SaaS
		isSaaS, scenario := sds.analyzeSaaSCharacteristics(scenarioName, scenariosDir)
		if isSaaS {
			saasScenarios++

			// Check if this is newly detected
			existing, err := sds.getExistingScenario(scenarioName)
			if err != nil || existing == nil {
				newlyDetected++
			}

			// Store in database
			scenario.ID = uuid.New().String()
			scenario.LastScan = time.Now()

			err = sds.dbService.CreateSaaSScenario(&scenario)
			if err != nil {
				log.Printf("Failed to store scenario %s: %v", scenarioName, err)
			}

			scenarios = append(scenarios, scenario)
		}
	}

	return &ScanResponse{
		TotalScenarios: totalScenarios,
		SaaSScenarios:  saasScenarios,
		NewlyDetected:  newlyDetected,
		Scenarios:      scenarios,
	}, nil
}

func (sds *SaaSDetectionService) analyzeSaaSCharacteristics(scenarioName, scenariosDir string) (bool, SaaSScenario) {
	cleanName, err := validateScenarioPath(scenarioName)
	if err != nil {
		log.Printf("Invalid scenario path %s: %v", scenarioName, err)
		return false, SaaSScenario{ScenarioName: scenarioName}
	}
	scenarioPath := filepath.Join(scenariosDir, cleanName)
	scenario := SaaSScenario{
		ScenarioName: cleanName,
		Metadata:     make(map[string]interface{}),
	}

	var confidenceScore float64
	var characteristics []string

	// 1. Analyze service.json for SaaS indicators
	serviceFile := filepath.Join(scenarioPath, ".vrooli", "service.json")
	if serviceContent, err := os.ReadFile(serviceFile); err == nil {
		var serviceConfig map[string]interface{}
		if json.Unmarshal(serviceContent, &serviceConfig) == nil {

			// Extract display name and description
			if service, ok := serviceConfig["service"].(map[string]interface{}); ok {
				if displayName, ok := service["displayName"].(string); ok {
					scenario.DisplayName = displayName
				}
				if description, ok := service["description"].(string); ok {
					scenario.Description = description
				}

				// Check tags for SaaS indicators
				if tags, ok := service["tags"].([]interface{}); ok {
					saasIndicators := []string{"multi-tenant", "billing", "analytics", "a-b-testing",
						"saas", "business-application", "subscription", "api-service"}

					for _, tag := range tags {
						if tagStr, ok := tag.(string); ok {
							for _, indicator := range saasIndicators {
								if strings.Contains(tagStr, indicator) {
									confidenceScore += SaaSTagMatchScore
									characteristics = append(characteristics, fmt.Sprintf("tag:%s", tagStr))
								}
							}
						}
					}
				}
			}

			// Check for database requirements (typical for SaaS)
			if resources, ok := serviceConfig["resources"].(map[string]interface{}); ok {
				if postgres, ok := resources["postgres"].(map[string]interface{}); ok {
					if enabled, ok := postgres["enabled"].(bool); ok && enabled {
						confidenceScore += SaaSPostgresScore
						characteristics = append(characteristics, "database:postgres")
					}
				}
			}
		}
	}

	// 2. Analyze PRD.md for business value and revenue indicators
	prdFile := filepath.Join(scenarioPath, "PRD.md")
	if prdContent, err := os.ReadFile(prdFile); err == nil {
		content := string(prdContent)

		// Look for revenue indicators
		revenuePatterns := []string{
			`\$\d+K`,
			`revenue.*potential`,
			`business.*value`,
			`subscription`,
			`pricing`,
			`enterprise`,
			`saas`,
			`multi-tenant`,
		}

		for _, pattern := range revenuePatterns {
			if matched, _ := regexp.MatchString("(?i)"+pattern, content); matched {
				confidenceScore += SaaSRevenuePatternScore
				characteristics = append(characteristics, fmt.Sprintf("prd:%s", pattern))
			}
		}

		// Extract revenue potential
		revenueRegex := regexp.MustCompile(`(?i)revenue.*potential.*\$(\d+K?\s*-\s*\$?\d+K?)`)
		if matches := revenueRegex.FindStringSubmatch(content); len(matches) > 1 {
			scenario.RevenuePotential = matches[1]
			confidenceScore += SaaSRevenueExtractionScore
		}

		// Determine SaaS type based on content
		if strings.Contains(strings.ToLower(content), "b2b") {
			scenario.SaaSType = "b2b_tool"
		} else if strings.Contains(strings.ToLower(content), "api") {
			scenario.SaaSType = "api_service"
		} else if strings.Contains(strings.ToLower(content), "marketplace") {
			scenario.SaaSType = "marketplace"
		} else {
			scenario.SaaSType = "b2c_app"
		}
	}

	// 3. Check for existing UI (indicates user-facing application)
	uiPath := filepath.Join(scenarioPath, "ui")
	if _, err := os.Stat(uiPath); err == nil {
		confidenceScore += SaaSUIPresenceScore
		characteristics = append(characteristics, "has_ui")
	}

	// 4. Check for API (indicates service offering)
	apiPath := filepath.Join(scenarioPath, "api")
	if _, err := os.Stat(apiPath); err == nil {
		confidenceScore += SaaSAPIPresenceScore
		characteristics = append(characteristics, "has_api")
	}

	// 5. Check for existing landing page
	landingPath := filepath.Join(scenarioPath, "landing")
	if _, err := os.Stat(landingPath); err == nil {
		scenario.HasLandingPage = true
		scenario.LandingPageURL = fmt.Sprintf("/scenarios/%s/landing/", scenarioName)
		characteristics = append(characteristics, "existing_landing")
	}

	scenario.ConfidenceScore = confidenceScore
	scenario.Metadata["characteristics"] = characteristics
	scenario.Metadata["analysis_version"] = SaaSDetectionAnalysisVersion

	// Consider it a SaaS if confidence score is above threshold
	isSaaS := confidenceScore >= SaaSConfidenceThreshold

	return isSaaS, scenario
}

func (sds *SaaSDetectionService) getExistingScenario(scenarioName string) (*SaaSScenario, error) {
	query := `SELECT id FROM saas_scenarios WHERE scenario_name = $1`
	var id string
	err := sds.dbService.db.QueryRow(query, scenarioName).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &SaaSScenario{ID: id}, nil
}
