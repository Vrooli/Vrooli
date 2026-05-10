package doctemplates

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"knowledge-observatory/internal/doccontract"
)

const DefaultTemplateID = "react-vite"

var (
	ErrScenarioPathRequired = errors.New("scenario path is required")
	ErrTemplateNotFound     = errors.New("scenario template manifest not found")
	ErrDocumentNotFound     = errors.New("template document not found")
)

type Resolver struct {
	RepoRoot string
}

type Source struct {
	TemplateID           string
	TemplateManifestPath string
	ScenarioManifestPath string
	ScenarioManifestUsed bool
	ProvenanceFallback   bool
}

type Resolved struct {
	Contract         *doccontract.ResolvedContract
	ContractFindings []doccontract.Finding
	Source           Source
}

func NewResolverFromScenariosRoot(scenariosRoot string) Resolver {
	return Resolver{RepoRoot: resolveRepoRoot(filepath.Dir(filepath.Clean(scenariosRoot)))}
}

func (r Resolver) ResolveScenario(scenarioPath string) (*Resolved, error) {
	scenarioPath = strings.TrimSpace(scenarioPath)
	if scenarioPath == "" {
		return nil, ErrScenarioPathRequired
	}
	templateID, fallback := r.templateIDForScenario(scenarioPath)
	repoRoot := resolveRepoRoot(r.RepoRoot)
	templateManifestPath := filepath.Join(repoRoot, "templates", "scenarios", templateID, "docs", "manifest.json")
	if _, err := os.Stat(templateManifestPath); err != nil {
		return nil, ErrTemplateNotFound
	}

	manifestPath := filepath.Join(scenarioPath, "docs", "manifest.json")
	scenarioManifestUsed := true
	if _, err := os.Stat(manifestPath); err != nil {
		manifestPath = templateManifestPath
		scenarioManifestUsed = false
	}
	manifest, err := doccontract.LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	contract, findings := doccontract.Resolve(manifest, manifestPath)
	if contract != nil && contract.TemplateID == "" {
		contract.TemplateID = templateID
	}
	return &Resolved{
		Contract:         contract,
		ContractFindings: findings,
		Source: Source{
			TemplateID:           templateID,
			TemplateManifestPath: filepath.ToSlash(templateManifestPath),
			ScenarioManifestPath: filepath.ToSlash(manifestPath),
			ScenarioManifestUsed: scenarioManifestUsed,
			ProvenanceFallback:   fallback,
		},
	}, nil
}

func (r Resolver) ResolveTemplate(templateID string) (*doccontract.ResolvedContract, []doccontract.Finding, error) {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		templateID = DefaultTemplateID
	}
	repoRoot := resolveRepoRoot(r.RepoRoot)
	manifestPath := filepath.Join(repoRoot, "templates", "scenarios", templateID, "docs", "manifest.json")
	manifest, err := doccontract.LoadManifest(manifestPath)
	if err != nil {
		return nil, nil, err
	}
	contract, findings := doccontract.Resolve(manifest, manifestPath)
	if contract != nil && contract.TemplateID == "" {
		contract.TemplateID = templateID
	}
	return contract, findings, nil
}

func (r Resolver) TemplateContent(templateID, id string) (doccontract.Document, string, error) {
	contract, findings, err := r.ResolveTemplate(templateID)
	if err != nil {
		return doccontract.Document{}, "", err
	}
	if doccontract.HasError(findings) {
		return doccontract.Document{}, "", doccontract.ErrorFromFindings(findings)
	}
	doc, ok := contract.ResolveIdentifier(id)
	if !ok {
		return doccontract.Document{}, "", ErrDocumentNotFound
	}
	templateRoot := filepath.Join(resolveRepoRoot(r.RepoRoot), "templates", "scenarios", contract.TemplateID)
	content, err := os.ReadFile(filepath.Join(templateRoot, filepath.FromSlash(doc.ScenarioPath)))
	if err != nil {
		if fallback, fallbackErr := r.designTemplateContent(templateRoot, doc); fallbackErr == nil {
			return doc, fallback, nil
		}
		return doccontract.Document{}, "", err
	}
	return doc, string(content), nil
}

func (r Resolver) designTemplateContent(templateRoot string, doc doccontract.Document) (string, error) {
	if doc.ScenarioPath != "DESIGN.md" {
		return "", os.ErrNotExist
	}
	type templateConfig struct {
		Design struct {
			Default string `json:"default"`
		} `json:"design"`
	}
	data, err := os.ReadFile(filepath.Join(templateRoot, "template.json"))
	if err != nil {
		return "", err
	}
	var cfg templateConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", err
	}
	designID := strings.TrimSpace(cfg.Design.Default)
	if designID == "" {
		return "", os.ErrNotExist
	}
	content, err := os.ReadFile(filepath.Join(resolveRepoRoot(r.RepoRoot), "templates", "design", designID, "DESIGN.md"))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func resolveRepoRoot(candidate string) string {
	candidate = strings.TrimSpace(candidate)
	if hasDefaultTemplate(candidate) {
		return candidate
	}
	if cwd, err := os.Getwd(); err == nil {
		for dir := cwd; dir != "" && dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			if hasDefaultTemplate(dir) {
				return dir
			}
		}
	}
	return candidate
}

func hasDefaultTemplate(root string) bool {
	if root == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(root, "templates", "scenarios", DefaultTemplateID, "docs", "manifest.json"))
	return err == nil && !info.IsDir()
}

func (r Resolver) templateIDForScenario(scenarioPath string) (string, bool) {
	type serviceConfig struct {
		Generation struct {
			Template struct {
				ID string `json:"id"`
			} `json:"template"`
		} `json:"generation"`
	}
	data, err := os.ReadFile(filepath.Join(scenarioPath, ".vrooli", "service.json"))
	if err != nil {
		return DefaultTemplateID, true
	}
	var cfg serviceConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultTemplateID, true
	}
	id := strings.TrimSpace(cfg.Generation.Template.ID)
	if id == "" {
		return DefaultTemplateID, true
	}
	return id, false
}
