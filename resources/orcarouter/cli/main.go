package main

import (
	"fmt"
	"os"

	"github.com/vrooli/vrooli/resources/internal/aiproviderapp"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func providerConfig() aiproviderapp.Config {
	return aiproviderapp.Config{
		Name:               "orcarouter",
		DisplayName:        "OrcaRouter",
		Version:            "0.1.0",
		APIBaseEnv:         "ORCAROUTER_API_BASE",
		APIBaseDefault:     "https://api.orcarouter.ai/v1",
		APIKeyEnv:          "ORCAROUTER_API_KEY",
		DefaultRoleEnv:     "ORCAROUTER_DEFAULT_ROLE",
		DefaultRole:        "chat.default",
		CredentialIdentity: "vrooli/orcarouter",
		CredentialField:    "api-key",
		PolicyEnv:          "ORCAROUTER_MODEL_POLICY_PATH",
		PolicySourceEnv:    "ORCAROUTER_CLI_SOURCE_ROOT",
	}
}

func main() {
	app, err := aiproviderapp.New(providerConfig(), buildFingerprint, buildTimestamp, buildSourceRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.CLI.Run(os.Args[1:]); err != nil {
		aiproviderapp.HandleProviderError(err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
