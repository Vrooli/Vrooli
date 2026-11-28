//go:build race

package search

// raceDetectorEnabled lets timing-sensitive tests distinguish an instrumented
// build from a normal one. Wall-clock assertions under -race measure the race
// runtime, not the code.
const raceDetectorEnabled = true
