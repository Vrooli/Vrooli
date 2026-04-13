package repocontractmeta

import "path/filepath"

const (
	ProjectConfigDir           = ".vrooli"
	ContractFilename           = "repo-contract.json"
	ContractSchemaRef          = "schemas/repo-contract.schema.json"
	AdoptionExceptionsFilename = "repo-contract-adoption-exceptions.json"
	SchemaDir                  = "schemas"
	SchemaFilename             = "repo-contract.schema.json"
	CommonSchemaFilename       = "common.schema.json"
	ValidationScriptFilename   = "validate-repo-contract.py"
	DocsPath                   = "docs/repo-contract.md"
	MiniBundleProfile          = "mini_vrooli_bundle"
	InfoManifestFilename       = "info-manifest.json"
	DefaultContractVersion     = "1.0.0"
)

func ContractPath(root string) string {
	return filepath.Join(root, ProjectConfigDir, ContractFilename)
}

func AdoptionExceptionsPath(root string) string {
	return filepath.Join(root, ProjectConfigDir, AdoptionExceptionsFilename)
}

func SchemaPath(root string) string {
	return filepath.Join(root, ProjectConfigDir, SchemaDir, SchemaFilename)
}

func CommonSchemaPath(root string) string {
	return filepath.Join(root, ProjectConfigDir, SchemaDir, CommonSchemaFilename)
}

func ValidationScriptPath(root string) string {
	return filepath.Join(root, ProjectConfigDir, SchemaDir, ValidationScriptFilename)
}

func InfoManifestPath(root string) string {
	return filepath.Join(root, ProjectConfigDir, InfoManifestFilename)
}
