package main

import (
	"context"
	"testing"

	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
)

func TestMainDelegatesToApplication(t *testing.T) {
	old := runApplication
	defer func() { runApplication = old }()
	called := false
	runApplication = func() error { called = true; return nil }
	main()
	if !called {
		t.Fatal("main did not delegate to application")
	}
}

func TestStartApplicationBuildsDependenciesAndDelegatesServer(t *testing.T) {
	oldPreflight, oldServer := preflightRun, runAPIServer
	defer func() { preflightRun, runAPIServer = oldPreflight, oldServer }()
	preflightRun = func(preflight.Config) bool { return false }
	runAPIServer = func(config apiserver.Config) error {
		if config.Cleanup != nil {
			if err := config.Cleanup(context.Background()); err != nil {
				return err
			}
		}
		return nil
	}
	if err := startApplication(); err != nil {
		t.Fatal(err)
	}
}

func TestStartApplicationReturnsServerError(t *testing.T) {
	oldPreflight, oldServer := preflightRun, runAPIServer
	defer func() { preflightRun, runAPIServer = oldPreflight, oldServer }()
	preflightRun = func(preflight.Config) bool { return false }
	runAPIServer = func(config apiserver.Config) error {
		if config.Cleanup != nil {
			_ = config.Cleanup(context.Background())
		}
		return context.Canceled
	}
	if err := startApplication(); err == nil {
		t.Fatal("server error was swallowed")
	}
}

func TestStartApplicationStopsAfterPreflightReexec(t *testing.T) {
	old := preflightRun
	defer func() { preflightRun = old }()
	preflightRun = func(preflight.Config) bool { return true }
	if err := startApplication(); err != nil {
		t.Fatal(err)
	}
}
