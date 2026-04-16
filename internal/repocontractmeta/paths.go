package repocontractmeta

import "path/filepath"

const (
	ProjectConfigDir         = ".vrooli"
	ServiceManifestFilename  = "service.json"
	ServiceManifestPathname  = ProjectConfigDir + "/" + ServiceManifestFilename
	ResourceManifestFilename = "resource.json"
	ContractFilename         = "repo-contract.json"
	ContractSchemaRef        = "schemas/repo-contract.schema.json"
	SchemaDir                = "schemas"
	SchemaFilename           = "repo-contract.schema.json"
	CommonSchemaFilename     = "common.schema.json"
	DocsPath                 = "docs/repo-contract.md"
	MiniBundleProfile        = "mini_vrooli_bundle"
	DefaultContractVersion   = "1.0.0"
)

func ContractPath(root string) string {
	return filepath.Join(root, ProjectConfigDir, ContractFilename)
}

func ProjectServiceManifestPath(root string) string {
	return filepath.Join(root, ProjectConfigDir, ServiceManifestFilename)
}

func SchemaPath(root string) string {
	return filepath.Join(root, ProjectConfigDir, SchemaDir, SchemaFilename)
}

func CommonSchemaPath(root string) string {
	return filepath.Join(root, ProjectConfigDir, SchemaDir, CommonSchemaFilename)
}

func ResourceManifestPath(root, name string) string {
	return filepath.Join(root, "resources", name, ResourceManifestFilename)
}
