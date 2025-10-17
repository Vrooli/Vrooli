package main

import (
	"strings"
)

// Get default model capabilities as fallback
func getDefaultModelCapabilities() []ModelCapability {
	return []ModelCapability{
		{
			ModelName:       "llama3.2:1b",
			Capabilities:    []string{"completion", "reasoning"},
			RamRequiredGB:   2.0,
			Speed:          "fast",
			CostPer1KTokens: 0.001,
			QualityTier:    "basic",
			BestFor:        []string{"simple-completion", "basic-reasoning"},
		},
		{
			ModelName:       "llama3.2:3b", 
			Capabilities:    []string{"completion", "reasoning", "code"},
			RamRequiredGB:   4.0,
			Speed:          "medium",
			CostPer1KTokens: 0.003,
			QualityTier:    "good",
			BestFor:        []string{"completion", "reasoning", "simple-code"},
		},
		{
			ModelName:       "llama3.2:8b",
			Capabilities:    []string{"completion", "reasoning", "code", "analysis"},
			RamRequiredGB:   8.0,
			Speed:          "slow",
			CostPer1KTokens: 0.008,
			QualityTier:    "high",
			BestFor:        []string{"complex-reasoning", "code-generation", "analysis"},
		},
		{
			ModelName:       "codellama:7b",
			Capabilities:    []string{"code", "completion"},
			RamRequiredGB:   7.0,
			Speed:          "medium",
			CostPer1KTokens: 0.007,
			QualityTier:    "high",
			BestFor:        []string{"code-generation", "code-analysis"},
		},
	}
}

// Convert Ollama models to model capabilities
func convertOllamaModelsToCapabilities(ollamaModels []OllamaModel) []ModelCapability {
	capabilities := make([]ModelCapability, 0, len(ollamaModels))
	
	for _, model := range ollamaModels {
		// Determine capabilities based on model name patterns
		var modelCaps []string
		var speed string
		var qualityTier string
		var costPer1K float64
		var bestFor []string
		
		modelName := strings.ToLower(model.Name)
		sizeGB := float64(model.Size) / (1024 * 1024 * 1024)
		
		// Classify model capabilities based on name patterns
		if strings.Contains(modelName, "code") || strings.Contains(modelName, "llama") {
			modelCaps = []string{"completion", "reasoning", "code"}
			bestFor = []string{"code-generation", "completion", "reasoning"}
		} else if strings.Contains(modelName, "embed") {
			modelCaps = []string{"embedding"}
			bestFor = []string{"text-embedding", "similarity"}
		} else {
			modelCaps = []string{"completion", "reasoning"}
			bestFor = []string{"completion", "reasoning"}
		}
		
		// Determine quality and speed based on size
		if sizeGB < 3 {
			speed = "fast"
			qualityTier = "basic"
			costPer1K = 0.001
		} else if sizeGB < 6 {
			speed = "medium"
			qualityTier = "good"
			costPer1K = 0.003
		} else {
			speed = "slow"
			qualityTier = "high"
			costPer1K = 0.008
		}
		
		// Add analysis capability for larger models
		if sizeGB > 6 {
			modelCaps = append(modelCaps, "analysis")
			bestFor = append(bestFor, "complex-reasoning", "analysis")
		}
		
		capability := ModelCapability{
			ModelName:       model.Name,
			Capabilities:    modelCaps,
			RamRequiredGB:   sizeGB,
			Speed:          speed,
			CostPer1KTokens: costPer1K,
			QualityTier:    qualityTier,
			BestFor:        bestFor,
		}
		
		capabilities = append(capabilities, capability)
	}
	
	return capabilities
}