package main

import (
	"os"

	repocontract "github.com/vrooli/repo-contract-go"
)

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func FindRepoRoot(start string) (string, error) {
	return repocontract.FindRepoRoot(start)
}
