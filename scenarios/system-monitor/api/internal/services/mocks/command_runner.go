package mocks

import (
	"context"
	"fmt"
)

// CommandRunner is a configurable test double for services.CommandRunner.
type CommandRunner struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
}

func NewCommandRunner() *CommandRunner {
	return &CommandRunner{exitCode: 0}
}

func (m *CommandRunner) WithStdout(stdout string) *CommandRunner {
	m.stdout = stdout
	return m
}

func (m *CommandRunner) WithStderr(stderr string) *CommandRunner {
	m.stderr = stderr
	return m
}

func (m *CommandRunner) WithExitCode(exitCode int) *CommandRunner {
	m.exitCode = exitCode
	return m
}

func (m *CommandRunner) WithError(err error) *CommandRunner {
	m.err = err
	return m
}

func (m *CommandRunner) WithErrorf(format string, args ...interface{}) *CommandRunner {
	m.err = fmt.Errorf(format, args...)
	return m
}

func (m *CommandRunner) Run(_ context.Context, _ string, _ []string, _ string) (string, string, int, error) {
	return m.stdout, m.stderr, m.exitCode, m.err
}
