package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// LandingPageService handles landing page generation and management
type LandingPageService struct {
	dbService     *DatabaseService
	templatesPath string
}

func NewLandingPageService(dbService *DatabaseService, templatesPath string) *LandingPageService {
	return &LandingPageService{
		dbService:     dbService,
		templatesPath: templatesPath,
	}
}

func (lps *LandingPageService) GenerateLandingPage(req *GenerateRequest) (*GenerateResponse, error) {
	// Get the template
	templates, err := lps.dbService.GetTemplates("", "")
	if err != nil {
		return nil, err
	}

	var selectedTemplate Template
	if req.TemplateID != "" {
		// Find specific template
		for _, t := range templates {
			if t.ID == req.TemplateID {
				selectedTemplate = t
				break
			}
		}
	} else {
		// Use first available template
		if len(templates) > 0 {
			selectedTemplate = templates[0]
		}
	}

	// Create landing page
	landingPageID := uuid.New().String()
	landingPage := &LandingPage{
		ID:                 landingPageID,
		ScenarioID:         req.ScenarioID,
		TemplateID:         selectedTemplate.ID,
		Variant:            "control",
		Title:              "Generated Landing Page",
		Description:        "AI-generated landing page",
		Content:            req.CustomContent,
		SEOMetadata:        make(map[string]interface{}),
		PerformanceMetrics: make(map[string]interface{}),
		Status:             "draft",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	err = lps.dbService.CreateLandingPage(landingPage)
	if err != nil {
		return nil, err
	}

	// Generate variants for A/B testing if enabled
	variants := []string{"control"}
	if req.EnableABTesting {
		// Create A/B test variants
		for _, variant := range []string{"a", "b"} {
			variantPage := *landingPage
			variantPage.ID = uuid.New().String()
			variantPage.Variant = variant
			variantPage.Title = fmt.Sprintf("%s - Variant %s", landingPage.Title, strings.ToUpper(variant))

			err = lps.dbService.CreateLandingPage(&variantPage)
			if err != nil {
				log.Printf("Failed to create variant %s: %v", variant, err)
			} else {
				variants = append(variants, variant)
			}
		}
	}

	return &GenerateResponse{
		LandingPageID:    landingPageID,
		PreviewURL:       fmt.Sprintf("/preview/%s", landingPageID),
		DeploymentStatus: "ready",
		ABTestVariants:   variants,
	}, nil
}
