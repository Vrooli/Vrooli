package review

import "swarm-manager/internal/attemptstore"

// These names preserve focused historical tests while production code reaches
// the shared attemptstore through the private review projection helpers.
func LoadRounds(itemDir string) ([]Round, error)             { return readRounds(itemDir) }
func LoadLatestRound(itemDir string) (*Round, int, error)    { return ReadLatestRound(itemDir) }
func LoadRound(itemDir string, roundNum int) (*Round, error) { return readRound(itemDir, roundNum) }
func SaveRound(itemDir string, round Round) error            { return saveRound(itemDir, round) }
func NextRoundNumber(itemDir string) (int, error)            { return nextRoundNumber(itemDir) }
func RoundFilename(n int) string                             { return attemptstore.RoundFilename(n) }
func SaveCapture(itemDir, filename string, data []byte) (string, error) {
	return attemptstore.SaveCapture(itemDir, "review", filename, data)
}

func LoadCapture(itemDir, relativePath string) ([]byte, error) {
	return loadCapture(itemDir, relativePath)
}
