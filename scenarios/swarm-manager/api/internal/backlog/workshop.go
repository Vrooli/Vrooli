// DOC: docs/concepts/ARCHITECTURE.md#workshop-refinement
//
// This file re-exports workshop types and functions from the shared workshop
// package for convenience within the backlog package.
package backlog

import (
	"swarm-manager/internal/workshop"
)

// Re-export types for use within backlog package.
type (
	WorkshopRound  = workshop.Round
	WorkshopItem   = workshop.Item
	WorkshopOption = workshop.Option
)

func LoadWorkshopRounds(itemDir string) ([]WorkshopRound, error) {
	return workshop.ReadRounds(itemDir)
}

func LoadLatestWorkshopRound(itemDir string) (*WorkshopRound, int, error) {
	return workshop.ReadLatestRound(itemDir)
}

func HasPlanByName(itemDir, filename string) bool {
	return workshop.HasPlanByName(itemDir, filename)
}

func LoadPlanContentByName(itemDir, filename string) string {
	return workshop.LoadPlanContentByName(itemDir, filename)
}

func BuildWorkshopHistory(rounds []WorkshopRound) string {
	return workshop.BuildHistory(rounds)
}
