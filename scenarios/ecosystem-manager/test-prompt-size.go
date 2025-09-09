package main

import (
	"fmt"
	"log"
	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

func main() {
	// Create assembler
	assembler, err := prompts.NewAssembler("/home/matthalloran8/Vrooli/scenarios/ecosystem-manager/prompts")
	if err != nil {
		log.Fatal(err)
	}

	// Test task
	task := tasks.TaskItem{
		ID:        "test-task",
		Title:     "Test Resource Generator",
		Type:      "resource", 
		Operation: "generator",
		Category:  "test",
		Priority:  "medium",
	}

	// Generate sections
	sections, err := assembler.GeneratePromptSections(task)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Sections for resource-generator: %d\n", len(sections))
	for i, section := range sections {
		fmt.Printf("  %d. %s\n", i+1, section)
	}

	// Assemble prompt
	prompt, err := assembler.AssemblePromptForTask(task)
	if err != nil {
		log.Fatal(err)
	}

	// Calculate size
	sizeBytes := len(prompt)
	sizeKB := float64(sizeBytes) / 1024.0
	sizeMB := sizeKB / 1024.0

	fmt.Printf("\n=== PROMPT SIZE ANALYSIS ===\n")
	fmt.Printf("Total sections: %d\n", len(sections))
	fmt.Printf("Prompt size: %d characters\n", sizeBytes)
	fmt.Printf("Prompt size: %.2f KB\n", sizeKB)
	fmt.Printf("Prompt size: %.2f MB\n", sizeMB)
	
	// Compare to old size (146KB)
	oldSizeKB := 146.0
	reduction := (oldSizeKB - sizeKB) / oldSizeKB * 100
	fmt.Printf("\nOld size: %.2f KB\n", oldSizeKB)
	fmt.Printf("New size: %.2f KB\n", sizeKB)
	fmt.Printf("Reduction: %.1f%%\n", reduction)
}