package main

import (
	repocontract "github.com/vrooli/repo-contract-go"
)

func resolveScenariosRoot() string {
	contract, root, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		return ""
	}
	scenariosRoot, err := contract.TopLevelDir(root, "scenarios")
	if err != nil {
		return ""
	}
	return scenariosRoot
}
