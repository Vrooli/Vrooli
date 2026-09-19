package main

import (
	"fmt"
	"log/slog"
	"os"

	apiserver "github.com/vrooli/api-core/server"
	vrooliapi "github.com/vrooli/vrooli/internal/api"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/logx"
)

func installAPILogger() (*slog.Logger, func()) {
	logger, _, restore := logx.InstallAndReport(logx.Options{
		Component:      "vrooli-api",
		SetDefault:     true,
		RedirectStdlib: true,
	})
	return logger, restore
}

func main() {
	logger, restoreLogger := installAPILogger()
	defer restoreLogger()

	if err := enforceStrictFingerprint(); err != nil {
		logger.Error("Stale fingerprint check failed", logx.ErrorArgs(err)...)
		os.Exit(1)
	}

	port := os.Getenv("VROOLI_API_PORT")
	if port == "" {
		port = "8092"
	}

	logger.Info(
		"Build metadata loaded",
		logx.AttrFingerprint, buildinfo.Fingerprint,
		logx.AttrCommit, buildinfo.GitCommit,
		logx.AttrBuildTime, buildinfo.BuildTime,
		logx.AttrPort, port,
	)

	app := vrooliapi.BuildRuntimeApp(vrooliapi.RuntimeConfig{
		Root:       vrooliapi.ResolveRepoRoot(),
		Home:       vrooliapi.DefaultHomeDir(),
		Logger:     logger,
		LookPathFn: vrooliapi.DefaultLookPath,
		CommandFn:  vrooliapi.DefaultCommandOutput,
	})
	if err := apiserver.Run(apiserver.Config{
		Handler: app.Router(),
		Port:    port,
		Logger: func(format string, args ...interface{}) {
			logger.Info(fmt.Sprintf(format, args...))
		},
	}); err != nil {
		logger.Error("API server failed", logx.ErrorArgs(err)...)
		os.Exit(1)
	}
}

func enforceStrictFingerprint() error {
	if os.Getenv("VROOLI_STRICT_FINGERPRINT") != "1" {
		return nil
	}

	current, err := buildinfo.CurrentFingerprint()
	if err != nil {
		return err
	}
	if current == buildinfo.Fingerprint {
		return nil
	}
	return fmt.Errorf("binary fingerprint %s does not match current sources %s", buildinfo.Fingerprint, current)
}
