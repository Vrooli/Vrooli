// Package targets owns bounded target resolution without transport concerns.
package targets

import (
	"context"
	"time"
)

type Kind string

const (
	KindPath         Kind = "path"
	KindScenario     Kind = "scenario"
	KindModule       Kind = "module"
	KindProject      Kind = "project"
	KindPackage      Kind = "package"
	KindControlPlane Kind = "control_plane"
)

type Request struct {
	Kind      Kind
	Value     string
	RepoRoot  string
	Languages []string
}

type Target struct {
	Kind       Kind
	Root       string
	RepoRoot   string
	Scenario   string
	Package    string
	Languages  []string
	RootPaths  []string
	ResolvedAt time.Time
}

type Resolver interface {
	Resolve(context.Context, Request) (Target, error)
}

type FileInfo struct {
	Path    string
	Size    int64
	Mode    uint32
	ModTime time.Time
	Dir     bool
}

type FileSystem interface {
	Stat(context.Context, string) (FileInfo, error)
	ReadFile(context.Context, string) ([]byte, error)
	Walk(context.Context, []string, func(FileInfo) error) error
}
