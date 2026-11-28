//go:build windows

package main

func execProcess(argv0 string, argv []string, env []string) error {
	return runAndWait(argv0, argv, env)
}
