package main

import "time"

// ambientSeam is the CLI's one door to the wall clock and the process
// environment. Commands read time and env through it so a test can substitute
// them; the real implementations are wired in main.go, the composition root.
type ambientSeam struct {
	Now       func() time.Time
	Getenv    func(string) string
	LookupEnv func(string) (string, bool)
}
