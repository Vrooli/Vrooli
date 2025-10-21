package main

import (
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

var (
	uiDirOnce sync.Once
	uiDir     string
)

func resolveStaticUIDir() string {
	uiDirOnce.Do(func() {
		uiDir = detectStaticUIDir()
	})
	return uiDir
}

func detectStaticUIDir() string {
	var bases []string

	if env := os.Getenv("SCENARIO_ROOT"); env != "" {
		bases = append(bases, env)
	}

	if wd, err := os.Getwd(); err == nil {
		bases = append(bases, wd)
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		bases = append(bases, exeDir, filepath.Dir(exeDir))
	}

	seen := map[string]struct{}{}
	for _, base := range bases {
		if base == "" {
			continue
		}
		cleanBase := filepath.Clean(base)
		if _, ok := seen[cleanBase]; ok {
			continue
		}
		seen[cleanBase] = struct{}{}

		for _, candidate := range []string{
			filepath.Join(cleanBase, "ui"),
			filepath.Join(cleanBase, "..", "ui"),
		} {
			info, err := os.Stat(candidate)
			if err != nil {
				continue
			}
			if info.IsDir() {
				return candidate
			}
		}
	}

	return ""
}

func newStaticHandler() http.Handler {
	dir := resolveStaticUIDir()
	if dir == "" {
		return nil
	}
	return http.FileServer(http.Dir(dir))
}
