package repocontract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Load reads and semantically validates a repo-contract file.
func Load(path string) (*Contract, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, &Error{Kind: ErrInvalidInput, Message: "contract path is required"}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &Error{Kind: ErrNotFound, Message: "read contract", Details: path, Err: err}
	}

	var doc contractDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, &Error{Kind: ErrInvalidContract, Message: "decode contract JSON", Details: path, Err: err}
	}

	if err := validateContractDoc(doc); err != nil {
		return nil, err
	}

	return &Contract{doc: deepCopyContractDoc(doc)}, nil
}

// LoadDefault reads the canonical contract from the provided repo root.
func LoadDefault(repoRoot string) (*Contract, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return nil, &Error{Kind: ErrInvalidInput, Message: "repo root is required"}
	}
	return Load(filepath.Join(filepath.Clean(repoRoot), filepath.FromSlash(defaultContractRelPath)))
}

func validateContractDoc(doc contractDoc) error {
	if doc.Schema != "schemas/repo-contract.schema.json" {
		return &Error{Kind: ErrInvalidContract, Message: "unexpected $schema", Details: doc.Schema}
	}
	if err := validateVersion(doc.Version); err != nil {
		return err
	}
	if doc.Platform.Mode != "cross_platform_go_native" {
		return &Error{Kind: ErrInvalidContract, Message: "unexpected platform mode", Details: doc.Platform.Mode}
	}
	if doc.Platform.LegacyProjectBashSupported {
		return &Error{Kind: ErrInvalidContract, Message: "legacy project bash support must be false"}
	}

	if err := validateSlashPaths("root.required_dirs", doc.Root.Markers.RequiredDirs); err != nil {
		return err
	}
	if err := validateSlashPaths("root.required_files", doc.Root.Markers.RequiredFiles); err != nil {
		return err
	}

	layoutPaths := map[string]string{
		"layout.project_config_dir": doc.Layout.ProjectConfigDir,
		"layout.scenario_dir":       doc.Layout.ScenarioDir,
		"layout.resource_dir":       doc.Layout.ResourceDir,
		"layout.package_dir":        doc.Layout.PackageDir,
		"layout.command_dir":        doc.Layout.CommandDir,
		"layout.internal_dir":       doc.Layout.InternalDir,
		"layout.docs_dir":           doc.Layout.DocsDir,
		"resource.manifest":         doc.Resource.Manifest,
	}
	for field, value := range layoutPaths {
		if err := validateSlashPath(field, value); err != nil {
			return err
		}
	}

	if err := validateSlashPaths("scenario.required_files", doc.Scenario.RequiredFiles); err != nil {
		return err
	}
	for key, value := range doc.Scenario.WellKnownPaths {
		if err := validateSlashPath("scenario.well_known_paths."+key, value); err != nil {
			return err
		}
	}
	for key, value := range doc.Resource.WellKnownPaths {
		if err := validateSlashPath("resource.well_known_paths."+key, value); err != nil {
			return err
		}
	}

	if doc.Globs.Syntax != "doublestar" {
		return &Error{Kind: ErrInvalidContract, Message: "unsupported glob syntax", Details: doc.Globs.Syntax}
	}
	if !doc.Globs.RootRelative || !doc.Globs.CaseSensitive || doc.Globs.AllowAbsolute || doc.Globs.PathFormat != "slash_normalized" {
		return &Error{Kind: ErrInvalidContract, Message: "unexpected glob policy"}
	}

	for key, value := range doc.Environment.Variables {
		if err := validateEnvVarName("environment.variables."+key, value); err != nil {
			return err
		}
	}

	if len(doc.Sandbox.FullRepoScopes) == 0 {
		return &Error{Kind: ErrInvalidContract, Message: "sandbox.full_repo_scopes must not be empty"}
	}
	if !slicesContain(doc.Sandbox.FullRepoScopes, "") || !slicesContain(doc.Sandbox.FullRepoScopes, ".") || !slicesContain(doc.Sandbox.FullRepoScopes, "/") {
		return &Error{Kind: ErrInvalidContract, Message: "sandbox.full_repo_scopes must include empty, dot, and slash scopes"}
	}
	if err := validateSlashPath("sandbox.scenario_scope_prefix", strings.TrimSuffix(doc.Sandbox.ScenarioScopePrefix, "/")); err != nil {
		return err
	}

	if len(doc.Profiles) == 0 {
		return &Error{Kind: ErrInvalidContract, Message: "profiles must not be empty"}
	}
	for name, profile := range doc.Profiles {
		if strings.TrimSpace(name) == "" {
			return &Error{Kind: ErrInvalidContract, Message: "profile name must not be empty"}
		}
		if strings.TrimSpace(profile.Description) == "" {
			return &Error{Kind: ErrInvalidContract, Message: "profile description must not be empty", Details: name}
		}
		if err := validateSlashPaths("profiles."+name+".include", profile.Include); err != nil {
			return err
		}
		if err := validateSlashPaths("profiles."+name+".optional_include", profile.OptionalInclude); err != nil {
			return err
		}
		if err := validateSlashPaths("profiles."+name+".exclude", profile.Exclude); err != nil {
			return err
		}
	}

	return nil
}

func validateVersion(version string) error {
	parts := strings.Split(strings.TrimSpace(version), ".")
	if len(parts) != 3 {
		return &Error{Kind: ErrInvalidContract, Message: "contract version must be semantic", Details: version}
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return &Error{Kind: ErrInvalidContract, Message: "contract major version must be numeric", Details: version, Err: err}
	}
	if major != supportedMajorVersion {
		return &Error{Kind: ErrUnsupportedVersion, Message: "unsupported contract major version", Details: version}
	}
	for _, part := range parts[1:] {
		if _, err := strconv.Atoi(part); err != nil {
			return &Error{Kind: ErrInvalidContract, Message: "contract version must be semantic", Details: version, Err: err}
		}
	}
	return nil
}

func deepCopyContractDoc(doc contractDoc) contractDoc {
	out := doc
	out.Root.Markers.RequiredDirs = slicesClone(doc.Root.Markers.RequiredDirs)
	out.Root.Markers.RequiredFiles = slicesClone(doc.Root.Markers.RequiredFiles)
	out.Scenario.RequiredFiles = slicesClone(doc.Scenario.RequiredFiles)
	out.Scenario.WellKnownPaths = cloneStringMap(doc.Scenario.WellKnownPaths)
	out.Resource.WellKnownPaths = cloneStringMap(doc.Resource.WellKnownPaths)
	out.Environment.Variables = cloneStringMap(doc.Environment.Variables)
	out.Sandbox.FullRepoScopes = slicesClone(doc.Sandbox.FullRepoScopes)
	out.Profiles = make(map[string]Profile, len(doc.Profiles))
	for name, profile := range doc.Profiles {
		out.Profiles[name] = Profile{
			Description:     profile.Description,
			Parameters:      slicesClone(profile.Parameters),
			Include:         slicesClone(profile.Include),
			OptionalInclude: slicesClone(profile.OptionalInclude),
			Exclude:         slicesClone(profile.Exclude),
		}
	}
	return out
}

func slicesClone[T any](in []T) []T {
	if len(in) == 0 {
		return nil
	}
	out := make([]T, len(in))
	copy(out, in)
	return out
}

func slicesContain(in []string, want string) bool {
	for _, item := range in {
		if item == want {
			return true
		}
	}
	return false
}

func validateEnvVarName(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return &Error{Kind: ErrInvalidContract, Message: field + " must not be empty"}
	}
	for i, r := range value {
		if i == 0 {
			if r < 'A' || r > 'Z' {
				return &Error{Kind: ErrInvalidContract, Message: field + " must start with A-Z", Details: value}
			}
			continue
		}
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return &Error{Kind: ErrInvalidContract, Message: field + " must use [A-Z0-9_]", Details: value}
		}
	}
	return nil
}

func validateSlashPaths(field string, values []string) error {
	seen := map[string]struct{}{}
	for _, value := range values {
		if err := validateSlashPath(field, value); err != nil {
			return err
		}
		if _, ok := seen[value]; ok {
			return &Error{Kind: ErrInvalidContract, Message: field + " contains duplicate path", Details: value}
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateSlashPath(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return &Error{Kind: ErrInvalidContract, Message: field + " must not be empty"}
	}
	if strings.Contains(value, "\\") {
		return &Error{Kind: ErrInvalidContract, Message: field + " must use slash-normalized paths", Details: value}
	}
	if isAbsolutePathLike(value) {
		return &Error{Kind: ErrInvalidContract, Message: field + " must be repository-relative", Details: value}
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return &Error{Kind: ErrInvalidContract, Message: field + " must not contain parent traversal", Details: value}
		}
	}
	return nil
}

func cleanIdentifier(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", &Error{Kind: ErrInvalidInput, Message: "identifier is required"}
	}
	if strings.Contains(value, "..") || strings.ContainsAny(value, `/\`) {
		return "", &Error{Kind: ErrInvalidInput, Message: "identifier must not contain path traversal", Details: value}
	}
	return value, nil
}

func unexpectedFieldError(field string, err error) error {
	return &Error{Kind: ErrInvalidContract, Message: fmt.Sprintf("invalid %s", field), Err: err}
}
