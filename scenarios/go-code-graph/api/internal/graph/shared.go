// Package graph preserves Go Code Graph's API-facing extraction names while
// delegating the parser, normalization, cache and concurrency implementation
// to API Core's shared Go graph substrate.
package graph

import (
	"github.com/vrooli/api-core/gograph"
	"github.com/vrooli/api-core/schedule"
	"golang.org/x/tools/go/packages"
)

type (
	ExtractionProfile   = gograph.ExtractionProfile
	InformationOmission = gograph.InformationOmission
	Language            = gograph.Language
	NodeKind            = gograph.NodeKind
	EdgeKind            = gograph.EdgeKind
	FileNode            = gograph.FileNode
	Package             = gograph.Package
	SymbolNode          = gograph.SymbolNode
	ImportEdge          = gograph.ImportEdge
	Node                = gograph.Node
	Edge                = gograph.Edge
	WarningKind         = gograph.WarningKind
	Warning             = gograph.Warning
	Graph               = gograph.Graph
	ExtractInput        = gograph.ExtractInput
	ExtractErrorKind    = gograph.ExtractErrorKind
	ExtractError        = gograph.ExtractError
	LoadOptions         = gograph.LoadOptions
	PackagesLoader      = gograph.PackagesLoader
	PackagesLoaderImpl  = gograph.PackagesLoaderImpl
	Service             = gograph.Service
	ExtractionStats     = gograph.ExtractionStats
	PathMutex           = gograph.PathMutex
	ExtractionCache     = gograph.ExtractionCache
	FileExtractionCache = gograph.FileExtractionCache
)

const (
	ExtractionProfileFull            = gograph.ExtractionProfileFull
	ExtractionProfileSemantic        = gograph.ExtractionProfileSemantic
	ExtractionProfileStructural      = gograph.ExtractionProfileStructural
	LanguageGo                       = gograph.LanguageGo
	NodeKindFile                     = gograph.NodeKindFile
	NodeKindPackage                  = gograph.NodeKindPackage
	NodeKindModule                   = gograph.NodeKindModule
	NodeKindType                     = gograph.NodeKindType
	NodeKindFunc                     = gograph.NodeKindFunc
	NodeKindVar                      = gograph.NodeKindVar
	NodeKindConst                    = gograph.NodeKindConst
	NodeKindInterface                = gograph.NodeKindInterface
	NodeKindMethod                   = gograph.NodeKindMethod
	NodeKindImportSpec               = gograph.NodeKindImportSpec
	NodeKindReference                = gograph.NodeKindReference
	NodeKindCall                     = gograph.NodeKindCall
	NodeKindTypeUsage                = gograph.NodeKindTypeUsage
	NodeKindRouteRegistration        = gograph.NodeKindRouteRegistration
	EdgeKindImport                   = gograph.EdgeKindImport
	EdgeKindIntraPackageRef          = gograph.EdgeKindIntraPackageRef
	WarningKindParseError            = gograph.WarningKindParseError
	WarningKindUnresolvedImport      = gograph.WarningKindUnresolvedImport
	WarningKindTypeCheckFailure      = gograph.WarningKindTypeCheckFailure
	WarningKindUnresolvedSymbol      = gograph.WarningKindUnresolvedSymbol
	ExtractErrorNoGoMod              = gograph.ExtractErrorNoGoMod
	ExtractErrorMultipleGoMod        = gograph.ExtractErrorMultipleGoMod
	ExtractErrorWorkspaceUnsupported = gograph.ExtractErrorWorkspaceUnsupported
	ExtractErrorPathUnreadable       = gograph.ExtractErrorPathUnreadable
	ExtractErrorInvalidInput         = gograph.ExtractErrorInvalidInput
	ExtractErrorInternal             = gograph.ExtractErrorInternal
	DefaultPathMutexCapacity         = gograph.DefaultPathMutexCapacity
	DefaultMaxConcurrentExtracts     = gograph.DefaultMaxConcurrentExtracts
)

func NewPackagesLoader() PackagesLoader { return gograph.NewPackagesLoader() }
func Normalize(pkgs []*packages.Package, root string) (Graph, []Warning) {
	return gograph.Normalize(pkgs, root)
}
func GraphHash(g Graph) string     { return gograph.GraphHash(g) }
func CanonicalJSON(g Graph) []byte { return gograph.CanonicalJSON(g) }
func NewPathMutex() *PathMutex     { return gograph.NewPathMutex() }
func NewPathMutexWithCapacity(capacity int) *PathMutex {
	return gograph.NewPathMutexWithCapacity(capacity)
}

func NewFileExtractionCache(dir string) (*FileExtractionCache, error) {
	return gograph.NewFileExtractionCache(dir)
}

func NewFileExtractionCacheWithLimit(dir string, maxBytes int64) (*FileExtractionCache, error) {
	return gograph.NewFileExtractionCacheWithLimit(dir, maxBytes)
}

func NewService(loader PackagesLoader, mutex *PathMutex) *Service {
	return gograph.NewService(loader, mutex)
}

func NewServiceWithMaxConcurrent(loader PackagesLoader, mutex *PathMutex, max int) *Service {
	return gograph.NewServiceWithMaxConcurrent(loader, mutex, max)
}

func NewServiceWithMaxConcurrentAndClock(loader PackagesLoader, mutex *PathMutex, max int, clock schedule.Clock) *Service {
	return gograph.NewServiceWithMaxConcurrentAndClock(loader, mutex, max, clock)
}

func NewServiceWithCache(loader PackagesLoader, mutex *PathMutex, max int, cache ExtractionCache) *Service {
	return gograph.NewServiceWithCache(loader, mutex, max, cache)
}

func NewServiceWithCacheAndEnvironment(loader PackagesLoader, mutex *PathMutex, max int, cache ExtractionCache, fingerprint string) *Service {
	return gograph.NewServiceWithCacheAndEnvironment(loader, mutex, max, cache, fingerprint)
}

func NewServiceWithCacheAndEnvironmentAndClock(loader PackagesLoader, mutex *PathMutex, max int, cache ExtractionCache, fingerprint string, clock schedule.Clock) *Service {
	return gograph.NewServiceWithCacheAndEnvironmentAndClock(loader, mutex, max, cache, fingerprint, clock)
}
