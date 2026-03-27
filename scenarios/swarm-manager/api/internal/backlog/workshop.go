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

// Re-export constants and functions.
var ReadinessDimensions = workshop.ReadinessDimensions

func ComputeEffectiveScores(raw map[string]int, roundsCompleted int, kind BacklogKind) map[string]int {
	return workshop.ComputeEffectiveScores(raw, roundsCompleted, string(kind))
}

func IsReady(effective map[string]int) bool {
	return workshop.IsReady(effective)
}

func LoadWorkshopRounds(itemDir string) ([]WorkshopRound, error) {
	return workshop.LoadRounds(itemDir)
}

func LoadLatestRound(itemDir string) (*WorkshopRound, int, error) {
	return workshop.LoadLatestRound(itemDir)
}

func CountPendingDecisions(round *WorkshopRound) int {
	return workshop.CountPendingDecisions(round)
}

func HasPlan(itemDir string) bool {
	return workshop.HasPlan(itemDir)
}

func HasPlanByName(itemDir, filename string) bool {
	return workshop.HasPlanByName(itemDir, filename)
}

func LoadPlanContent(itemDir string) string {
	return workshop.LoadPlanContent(itemDir)
}

func LoadPlanContentByName(itemDir, filename string) string {
	return workshop.LoadPlanContentByName(itemDir, filename)
}

func BuildWorkshopHistory(rounds []WorkshopRound) string {
	return workshop.BuildHistory(rounds)
}
