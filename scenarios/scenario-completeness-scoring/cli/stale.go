package main

import (
	"os"
)

func (a *App) warnIfBinaryStale() {
	if a.staleChecker == nil {
		return
	}
	a.staleChecker.ReexecArgs = os.Args[1:]
	a.staleChecker.CheckAndMaybeRebuild()
}
