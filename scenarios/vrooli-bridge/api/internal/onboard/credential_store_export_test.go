package onboard

import "time"

// SetAgentUnlockPollForTest shortens the post-rotation agent wait and returns
// a restore func.
func SetAgentUnlockPollForTest(interval time.Duration, attempts int) func() {
	previousInterval, previousAttempts := agentUnlockPollInterval, agentUnlockPollAttempts
	agentUnlockPollInterval, agentUnlockPollAttempts = interval, attempts
	return func() { agentUnlockPollInterval, agentUnlockPollAttempts = previousInterval, previousAttempts }
}
