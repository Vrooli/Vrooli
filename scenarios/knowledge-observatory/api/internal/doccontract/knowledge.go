package doccontract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path"
	"strings"
)

func FileSHA256(body []byte) string { sum := sha256.Sum256(body); return hex.EncodeToString(sum[:]) }

// KnowledgeMetadata preserves declarations. Discovery and ranking confer no authority.
func KnowledgeMetadata(doc Document, relative string) map[string]any {
	status := doc.KnowledgeStatus
	if status == "" {
		status = "unknown"
	}
	// HTML is plan supplementary material in this project. A misplaced artifact
	// needs promotion into an authoritative source, not a ranking-based upgrade.
	supplemental := strings.EqualFold(path.Ext(relative), ".html") || strings.EqualFold(path.Ext(relative), ".htm")
	for _, segment := range strings.Split(relative, "/") {
		if segment == "artifacts" || segment == "evidence" || segment == "reports" {
			supplemental = true
		}
	}
	if strings.EqualFold(path.Ext(relative), ".html") || strings.EqualFold(path.Ext(relative), ".htm") || (supplemental && doc.KnowledgeStatus == "") {
		status = "supplemental"
	}
	return map[string]any{
		"knowledge_status": status, "operating_systems": doc.OperatingSystems,
		"verified_operating_systems": doc.VerifiedOperatingSystems, "machine_scope": doc.MachineScope,
		"superseded_by": doc.SupersededBy, "aliases": doc.Aliases, "owner_skills": doc.OwnerSkills,
		"canonical_for": doc.CanonicalFor, "maturity": doc.Maturity,
	}
}

// ReadKnowledgeMetadata resolves the nearest matching manifest without loading
// the corpus or depending on a search index. Paths use repository-relative '/'.
func ReadKnowledgeMetadata(repoRoot, relative string) map[string]any {
	fallback := KnowledgeMetadata(Document{}, relative)
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fallback
	}
	defer root.Close()
	for dir := path.Dir(relative); ; dir = path.Dir(dir) {
		manifestPath := path.Join(dir, "docs/manifest.json")
		body, err := root.ReadFile(manifestPath)
		if err == nil {
			var manifest Manifest
			if json.Unmarshal(body, &manifest) == nil {
				for _, section := range manifest.Sections {
					for _, doc := range section.Documents {
						if path.Clean(path.Join(dir, NormalizeManifestPath(doc.Path))) == relative {
							meta := KnowledgeMetadata(doc, relative)
							meta["manifest_path"] = manifestPath
							meta["manifest_sha256"] = FileSHA256(body)
							return meta
						}
					}
				}
			}
		}
		if dir == "." || dir == "/" {
			break
		}
	}
	return fallback
}
