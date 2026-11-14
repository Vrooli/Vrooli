// Package catalog owns corpus identity, roles, and generation state.
package catalog

import (
	"context"
	"time"
)

type Role string

const (
	RoleImplementation Role = "implementation"
	RoleContract       Role = "contract"
	RoleGeneratedAlias Role = "generated_alias"
	RoleTest           Role = "test"
	RoleFixture        Role = "fixture"
	RoleDocumentation  Role = "documentation"
	RoleTransient      Role = "transient"
)

type Generation struct {
	ID               string
	State            string
	Policy           string
	SourceDigest     string
	DescriptorDigest string
	Failure          string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

const (
	GenerationShadow  = "shadow"
	GenerationActive  = "active"
	GenerationRetired = "retired"
	GenerationFailed  = "failed"
)

type SourceFile struct {
	ID         string
	Generation string
	Path       string
	Language   string
	Role       Role
	Scope      string
	Authority  string
	Owner      string
	Hash       string
	Size       int64
	ModTime    time.Time
	Searchable bool
}

type Page struct {
	Files     []SourceFile
	NextToken string
}

type Repository interface {
	BeginGeneration(context.Context, Generation) error
	UpsertFiles(context.Context, string, []SourceFile) error
	DeleteFiles(context.Context, string, []string) error
	CompleteGeneration(context.Context, string, string, string) error
	FailGeneration(context.Context, string, string) error
	PageFiles(context.Context, string, string, int) (Page, error)
	Activate(context.Context, string) error
	Rollback(context.Context, string) error
	Active(context.Context) (Generation, error)
}

type FileIterator interface {
	Next(context.Context) (SourceFile, bool, error)
	Close() error
}

type Discoverer interface {
	Open(context.Context) (FileIterator, error)
}

type Clock interface{ Now() time.Time }
