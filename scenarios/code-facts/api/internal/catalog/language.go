package catalog

import (
	"path/filepath"
	"strings"
)

type Capabilities struct {
	Catalog     bool
	Lexical     bool
	Declaration bool
	Semantic    bool
	Graph       bool
	Proof       bool
}

type Language struct {
	Name         string
	Extensions   []string
	Capabilities Capabilities
}

var languageRegistry = []Language{
	{Name: "go", Extensions: []string{".go"}, Capabilities: Capabilities{true, true, true, true, true, true}},
	{Name: "typescript", Extensions: []string{".ts", ".tsx"}, Capabilities: Capabilities{true, true, true, true, true, true}},
	{Name: "javascript", Extensions: []string{".js", ".jsx", ".mjs", ".cjs"}, Capabilities: Capabilities{true, true, true, true, false, false}},
	{Name: "protobuf", Extensions: []string{".proto"}, Capabilities: Capabilities{true, true, true, true, true, true}},
	{Name: "python", Extensions: []string{".py"}, Capabilities: Capabilities{true, true, false, true, false, false}},
	{Name: "rust", Extensions: []string{".rs"}, Capabilities: Capabilities{true, true, false, true, false, false}},
	{Name: "shell", Extensions: []string{".sh", ".bash", ".bats"}, Capabilities: Capabilities{true, true, true, true, true, false}},
	{Name: "sql", Extensions: []string{".sql"}, Capabilities: Capabilities{true, true, true, true, false, false}},
	{Name: "json", Extensions: []string{".json", ".jsonc"}, Capabilities: Capabilities{true, true, false, false, false, false}},
	{Name: "yaml", Extensions: []string{".yaml", ".yml"}, Capabilities: Capabilities{true, true, false, false, false, false}},
	{Name: "markdown", Extensions: []string{".md", ".mdx"}, Capabilities: Capabilities{true, true, false, false, false, false}},
}

func Languages() []Language {
	out := make([]Language, len(languageRegistry))
	copy(out, languageRegistry)
	return out
}

func LanguageForPath(path string) (Language, bool) {
	extension := strings.ToLower(filepath.Ext(path))
	for _, language := range languageRegistry {
		for _, candidate := range language.Extensions {
			if extension == candidate {
				return language, true
			}
		}
	}
	return Language{Name: "unknown", Capabilities: Capabilities{Catalog: true}}, false
}

type Classification struct {
	Language   Language
	Role       Role
	Scope      string
	Authority  string
	Owner      string
	Searchable bool
}

func Classify(path string, prefix []byte) Classification {
	path = canonicalPath(path)
	language, _ := LanguageForPath(path)
	role := classifyRole(path, prefix, language.Name)
	classification := Classification{
		Language:   language,
		Role:       role,
		Scope:      scopeForPath(path),
		Authority:  authorityForRole(role),
		Owner:      ownerForPath(path),
		Searchable: role == RoleImplementation || role == RoleContract,
	}
	return classification
}

func classifyRole(path string, prefix []byte, language string) Role {
	lower := strings.ToLower("/" + path + "/")
	base := strings.ToLower(filepath.Base(path))
	if strings.Contains(lower, "/packages/proto/.verify-") || strings.Contains(lower, "/.vrooli/cache/") || strings.Contains(lower, "/.vrooli/.tmp-") {
		return RoleTransient
	}
	// Provider-owned evaluation corpora repeat their golden queries and expected
	// answers verbatim. Indexing them as implementation evidence leaks the test
	// set into retrieval and produces meaningless perfect scores.
	if strings.Contains(lower, "/.vrooli/search.json/") {
		return RoleFixture
	}
	generatedHeader := strings.Contains(strings.ToLower(string(prefix)), "code generated") || strings.Contains(strings.ToLower(string(prefix)), "generated file")
	if generatedHeader || strings.Contains(lower, "/packages/proto/gen/") {
		return RoleGeneratedAlias
	}
	if strings.Contains(lower, "/packages/proto/schemas/") {
		return RoleContract
	}
	if containsPathSegment(lower, ".git", "node_modules", "vendor", "dist", "build", ".next", ".turbo", ".cache", "coverage", "tmp", "temp") {
		return RoleTransient
	}
	if strings.Contains(lower, "/testdata/") || strings.Contains(lower, "/fixtures/") || strings.Contains(lower, "/fixture/") || strings.Contains(base, ".fixture.") {
		return RoleFixture
	}
	if strings.HasSuffix(base, "_test.go") || strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") || strings.Contains(lower, "/tests/") || strings.Contains(lower, "/test/") {
		return RoleTest
	}
	if language == "protobuf" || strings.Contains(lower, "/schemas/") {
		return RoleContract
	}
	if language == "markdown" || strings.Contains(lower, "/docs/") || strings.HasPrefix(base, "readme") {
		return RoleDocumentation
	}
	return RoleImplementation
}

func containsPathSegment(path string, segments ...string) bool {
	for _, segment := range segments {
		if strings.Contains(path, "/"+segment+"/") {
			return true
		}
	}
	return false
}

func scopeForPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 2 {
		switch parts[0] {
		case "scenarios":
			return "scenario:" + parts[1]
		case "packages":
			return "package:" + parts[1]
		case "resources":
			return "resource:" + parts[1]
		case "cmd":
			return "control-plane"
		case "internal":
			return "control-plane"
		}
	}
	return "repository"
}

func ownerForPath(path string) string {
	scope := scopeForPath(path)
	if scope == "repository" {
		return "vrooli"
	}
	return scope
}

func authorityForRole(role Role) string {
	switch role {
	case RoleImplementation, RoleContract:
		return "authoritative"
	case RoleGeneratedAlias:
		return "derived_alias"
	case RoleTest, RoleFixture:
		return "supporting"
	case RoleDocumentation:
		return "explanatory"
	default:
		return "excluded"
	}
}
