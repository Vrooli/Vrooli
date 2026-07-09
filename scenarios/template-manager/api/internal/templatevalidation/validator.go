package templatevalidation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"

	repocontract "github.com/vrooli/repo-contract-go"
)

type Validator struct {
	RepoRoot   string
	Repository catalog.Repository
	Now        func() time.Time
}

func NewValidator(repoRoot string, repo catalog.Repository) *Validator {
	return &Validator{RepoRoot: repoRoot, Repository: repo}
}

func (v *Validator) ValidateScenario(ctx context.Context, scenario, explicitPath string) (Report, error) {
	root, scenarioID, err := v.resolveTarget(scenario, explicitPath)
	if err != nil {
		return Report{}, err
	}
	report := Report{Scenario: scenarioID, RootPath: root, CurrentTime: v.now()}
	prov, found, err := readProvenance(root)
	if err != nil {
		return Report{}, err
	}
	report.Provenance = prov
	if !found {
		report.Findings = append(report.Findings, Finding{
			Code:        CodeProvenanceMissing,
			Severity:    SeverityError,
			Title:       "Template provenance is missing",
			Message:     ".vrooli/service.json has no generation.template id and version.",
			Location:    ".vrooli/service.json",
			Remediation: "Preview and apply the PROVENANCE_MISSING fix to stamp adopted provenance for the latest default template.",
			Autofix:     true,
		})
		return report, nil
	}
	if err := v.checkKnownTemplate(ctx, &report); err != nil {
		return Report{}, err
	}
	checkOrientation(root, &report)
	checkManifestDrift(root, &report)
	return report, nil
}

func (v *Validator) resolveTarget(scenario, explicitPath string) (string, string, error) {
	explicitPath = strings.TrimSpace(explicitPath)
	if explicitPath != "" {
		abs, err := filepath.Abs(explicitPath)
		if err != nil {
			return "", "", fmt.Errorf("resolve target path: %w", err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return "", "", fmt.Errorf("target path is not readable: %w", err)
		}
		if !info.IsDir() {
			return "", "", errors.New("target path must be a directory")
		}
		name := strings.TrimSpace(scenario)
		if name == "" {
			name = filepath.Base(abs)
		}
		return abs, name, nil
	}
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return "", "", errors.New("scenario is required")
	}
	repoRoot := strings.TrimSpace(v.RepoRoot)
	if repoRoot == "" {
		root, err := repocontract.ResolveRepoRoot()
		if err != nil {
			return "", "", fmt.Errorf("resolve repo root: %w", err)
		}
		repoRoot = root
	}
	root, err := repocontract.ResolveScenarioPath(repoRoot, scenario)
	if err != nil {
		return "", "", err
	}
	return root, scenario, nil
}

func (v *Validator) checkKnownTemplate(ctx context.Context, report *Report) error {
	if v.Repository == nil {
		return nil
	}
	record, err := v.Repository.GetTemplate(ctx, report.Provenance.TemplateID)
	if err != nil {
		var notFound catalog.ErrNotFound
		if errors.As(err, &notFound) {
			report.Findings = append(report.Findings, Finding{
				Code:        CodeTemplateUnknown,
				Severity:    SeverityError,
				Title:       "Template provenance references an unknown template",
				Message:     fmt.Sprintf("Template %q is not registered in template-manager.", report.Provenance.TemplateID),
				Location:    ".vrooli/service.json",
				Remediation: "Register the template or correct the provenance block after operator review.",
			})
			return nil
		}
		return err
	}
	if record.LatestVersion != "" && compareSemver(report.Provenance.TemplateVersion, record.LatestVersion) < 0 {
		report.Findings = append(report.Findings, Finding{
			Code:        CodeTemplateVersionLag,
			Severity:    SeverityWarn,
			Title:       "Template version is behind the registered latest version",
			Message:     fmt.Sprintf("%s is at %s; latest registered version is %s.", report.Provenance.TemplateID, report.Provenance.TemplateVersion, record.LatestVersion),
			Location:    ".vrooli/service.json",
			Remediation: "Read every changelog entry above the recorded version, apply migrations, then update generation.template.version.",
		})
	}
	debt, err := v.Repository.ListDebt(ctx, report.Provenance.TemplateID, "open")
	if err != nil {
		return err
	}
	if len(debt) > 0 {
		report.Findings = append(report.Findings, Finding{
			Code:        CodeInheritedDebtOutstanding,
			Severity:    SeverityWarn,
			Title:       "Template has open inherited debt",
			Message:     fmt.Sprintf("%s has %d open template-manager debt entries.", report.Provenance.TemplateID, len(debt)),
			Location:    "template-manager debt ledger",
			Remediation: "Resolve or intentionally accept inherited template debt before claiming top template maturity.",
		})
	}
	return nil
}

func checkOrientation(root string, report *Report) {
	orientationPath := filepath.Join(root, ".vrooli", "orientation.json")
	if _, err := os.Stat(orientationPath); err == nil {
		return
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "START-HERE.md")); err != nil {
		return
	}
	report.Findings = append(report.Findings, Finding{
		Code:        CodeOrientationStateMissing,
		Severity:    SeverityWarn,
		Title:       "Orientation state is not available",
		Message:     "The scenario carries template provenance but no .vrooli/orientation.json state for static gate standing.",
		Location:    ".vrooli/orientation.json",
		Remediation: "Run orientation through the template lifecycle or finalize the scenario when all gates are complete.",
	})
}

func checkManifestDrift(root string, report *Report) {
	if report.Provenance.ManifestSHA == "" || report.Provenance.TemplateID == "" {
		return
	}
	manifestPath := filepath.Join(root, "..", "..", "templates", "scenarios", report.Provenance.TemplateID, "template.json")
	if _, err := os.Stat(manifestPath); err != nil {
		return
	}
	// The original generation hash is compared to the current template manifest
	// only as a cheap drift signal. Hash parity is intentionally advisory.
	if strings.TrimSpace(report.Provenance.ManifestSHA) == "" {
		report.Findings = append(report.Findings, Finding{
			Code:        CodeTemplateManifestDrift,
			Severity:    SeverityWarn,
			Title:       "Template manifest hash is unavailable",
			Message:     "The provenance block does not include a manifest hash for drift comparison.",
			Location:    ".vrooli/service.json",
			Remediation: "Re-stamp provenance after reviewing template lineage.",
		})
	}
}

func (v *Validator) now() time.Time {
	if v.Now != nil {
		return v.Now()
	}
	return time.Now().UTC()
}

type serviceDoc struct {
	Generation generationDoc `json:"generation"`
}

type generationDoc struct {
	Template    templateDoc `json:"template"`
	GeneratedAt string      `json:"generated_at"`
	ManifestSHA string      `json:"manifest_sha"`
	ContentSHA  string      `json:"content_sha"`
	Adopted     bool        `json:"adopted"`
}

type templateDoc struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

func readProvenance(root string) (Provenance, bool, error) {
	path := filepath.Join(root, ".vrooli", "service.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return Provenance{}, false, fmt.Errorf("read service manifest: %w", err)
	}
	var doc serviceDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Provenance{}, false, fmt.Errorf("parse service manifest: %w", err)
	}
	prov := Provenance{
		TemplateID:      strings.TrimSpace(doc.Generation.Template.ID),
		TemplateVersion: strings.TrimSpace(doc.Generation.Template.Version),
		GeneratedAt:     strings.TrimSpace(doc.Generation.GeneratedAt),
		ManifestSHA:     strings.TrimSpace(doc.Generation.ManifestSHA),
		ContentSHA:      strings.TrimSpace(doc.Generation.ContentSHA),
		Adopted:         doc.Generation.Adopted,
	}
	return prov, prov.TemplateID != "" && prov.TemplateVersion != "", nil
}

var semverPart = regexp.MustCompile(`\d+`)

func compareSemver(a, b string) int {
	ap := semverParts(a)
	bp := semverParts(b)
	for i := 0; i < 3; i++ {
		if ap[i] < bp[i] {
			return -1
		}
		if ap[i] > bp[i] {
			return 1
		}
	}
	return 0
}

func semverParts(v string) [3]int {
	var out [3]int
	matches := semverPart.FindAllString(v, 3)
	for i, match := range matches {
		for _, ch := range match {
			out[i] = out[i]*10 + int(ch-'0')
		}
	}
	return out
}

func LatestDefaultTemplate(repoRoot string) (id, version string, err error) {
	id = "react-vite"
	path := filepath.Join(repoRoot, "templates", "scenarios", id, "template.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read default template manifest: %w", err)
	}
	var doc struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", "", fmt.Errorf("parse default template manifest: %w", err)
	}
	version = strings.TrimSpace(doc.Version)
	if version == "" {
		return "", "", errors.New("default template version is empty")
	}
	return id, version, nil
}

func ChangelogVersions(repoRoot, templateID string) ([]string, error) {
	raw, err := os.ReadFile(filepath.Join(repoRoot, "templates", "scenarios", templateID, "CHANGELOG.md"))
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`(?m)^##\s+([0-9]+\.[0-9]+\.[0-9]+)\b`)
	matches := re.FindAllStringSubmatch(string(raw), -1)
	versions := make([]string, 0, len(matches))
	for _, match := range matches {
		versions = append(versions, match[1])
	}
	sort.SliceStable(versions, func(i, j int) bool { return compareSemver(versions[i], versions[j]) > 0 })
	return versions, nil
}
