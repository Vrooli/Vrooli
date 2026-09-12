// Package render turns parsed experience specs into deterministic workshop artifacts.
package render

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"experience-manager/internal/spec"
	corestorage "github.com/vrooli/api-core/storage"
)

const (
	ModeWireframe = "wireframe"
	ModeImage     = "image"
)

type Request struct {
	ScenarioDir string
	Scenario    string
	PageID      string
	Mode        string
}

type Result struct {
	Scenario       string
	PageID         string
	Mode           string
	HTML           string
	ArtifactPath   string
	DegradedReason string
}

type Variant struct {
	ID    string
	Title string
	Page  spec.PageDocument
}

type VariantResult struct {
	ID    string
	Title string
	HTML  string
}

type CompareRequest struct {
	ScenarioDir string
	Scenario    string
	PageID      string
	Mode        string
	Variants    []Variant
}

type CompareResult struct {
	Scenario       string
	PageID         string
	Mode           string
	HTML           string
	ArtifactPath   string
	DegradedReason string
	Variants       []VariantResult
}

func Render(req Request) (Result, error) {
	resultMode, degraded, err := normalizeMode(req.Mode)
	if err != nil {
		return Result{}, err
	}
	report, err := spec.ParseScenario(req.ScenarioDir)
	if err != nil {
		return Result{}, err
	}
	if req.Scenario != "" {
		report.Scenario = req.Scenario
	}
	if report.Spec == nil {
		return Result{}, fmt.Errorf("scenario %q has no parsed experience spec", report.Scenario)
	}
	page, ok := report.Spec.Pages[req.PageID]
	if !ok {
		return Result{}, fmt.Errorf("page %q not found", req.PageID)
	}
	htmlDoc := RenderPage(report.Scenario, page)
	artifact, err := writeArtifact(req.ScenarioDir, report.Scenario, page.Page.ID, resultMode, htmlDoc)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Scenario:       report.Scenario,
		PageID:         page.Page.ID,
		Mode:           resultMode,
		HTML:           htmlDoc,
		ArtifactPath:   artifact,
		DegradedReason: degraded,
	}, nil
}

func Compare(req CompareRequest) (CompareResult, error) {
	mode, degraded, err := normalizeMode(req.Mode)
	if err != nil {
		return CompareResult{}, err
	}
	if len(req.Variants) == 0 {
		return CompareResult{}, fmt.Errorf("at least one variant is required")
	}
	report, err := spec.ParseScenario(req.ScenarioDir)
	if err != nil {
		return CompareResult{}, err
	}
	if req.Scenario != "" {
		report.Scenario = req.Scenario
	}
	if report.Spec == nil {
		return CompareResult{}, fmt.Errorf("scenario %q has no parsed experience spec", report.Scenario)
	}
	if _, ok := report.Spec.Pages[req.PageID]; !ok {
		return CompareResult{}, fmt.Errorf("page %q not found", req.PageID)
	}
	var variants []VariantResult
	for _, variant := range req.Variants {
		id := strings.TrimSpace(variant.ID)
		if id == "" {
			id = variant.Page.Page.ID
		}
		if id == "" {
			return CompareResult{}, fmt.Errorf("variant id is required")
		}
		title := strings.TrimSpace(variant.Title)
		if title == "" {
			title = variant.Page.Page.Title
		}
		variants = append(variants, VariantResult{
			ID:    id,
			Title: title,
			HTML:  RenderPage(report.Scenario, variant.Page),
		})
	}
	htmlDoc := RenderVariantComparison(report.Scenario, req.PageID, variants)
	artifact, err := writeArtifact(req.ScenarioDir, report.Scenario, req.PageID+".variants", mode, htmlDoc)
	if err != nil {
		return CompareResult{}, err
	}
	return CompareResult{
		Scenario:       report.Scenario,
		PageID:         req.PageID,
		Mode:           mode,
		HTML:           htmlDoc,
		ArtifactPath:   artifact,
		DegradedReason: degraded,
		Variants:       variants,
	}, nil
}

func RenderPage(scenario string, page spec.PageDocument) string {
	var buf bytes.Buffer
	buf.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n")
	buf.WriteString("<meta charset=\"utf-8\">\n")
	buf.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	buf.WriteString("<title>")
	buf.WriteString(esc(scenario + " / " + page.Page.Title))
	buf.WriteString(" wireframe</title>\n")
	buf.WriteString("<style>")
	buf.WriteString(css())
	buf.WriteString("</style>\n</head>\n<body>\n")
	buf.WriteString("<main class=\"wireframe\" data-scenario=\"")
	buf.WriteString(esc(scenario))
	buf.WriteString("\" data-page=\"")
	buf.WriteString(esc(page.Page.ID))
	buf.WriteString("\">\n")
	writeHeader(&buf, page)
	writePriorities(&buf, page)
	writeRegions(&buf, page)
	writeClaims(&buf, page)
	buf.WriteString("</main>\n</body>\n</html>\n")
	return buf.String()
}

func RenderVariantComparison(scenario, pageID string, variants []VariantResult) string {
	var buf bytes.Buffer
	buf.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n")
	buf.WriteString("<meta charset=\"utf-8\">\n")
	buf.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	buf.WriteString("<title>")
	buf.WriteString(esc(scenario + " / " + pageID + " variants"))
	buf.WriteString("</title>\n<style>")
	buf.WriteString(css())
	buf.WriteString(variantCSS())
	buf.WriteString("</style>\n</head>\n<body>\n")
	buf.WriteString("<main class=\"variant-compare\" data-scenario=\"")
	buf.WriteString(esc(scenario))
	buf.WriteString("\" data-page=\"")
	buf.WriteString(esc(pageID))
	buf.WriteString("\">\n<header class=\"hero\"><p class=\"eyebrow\">variant compare</p><h1>")
	buf.WriteString(esc(pageID))
	buf.WriteString("</h1><p>Side-by-side deterministic workshop renderings.</p></header>\n")
	buf.WriteString("<section class=\"variant-grid\" aria-label=\"Rendered variants\">\n")
	for _, variant := range variants {
		buf.WriteString("<article class=\"variant\" data-variant=\"")
		buf.WriteString(esc(variant.ID))
		buf.WriteString("\"><h2>")
		buf.WriteString(esc(variant.Title))
		buf.WriteString("</h2><iframe title=\"")
		buf.WriteString(esc(variant.Title))
		buf.WriteString("\" sandbox srcdoc=\"")
		buf.WriteString(html.EscapeString(variant.HTML))
		buf.WriteString("\"></iframe></article>\n")
	}
	buf.WriteString("</section>\n</main>\n</body>\n</html>\n")
	return buf.String()
}

func normalizeMode(raw string) (string, string, error) {
	mode := strings.TrimSpace(raw)
	if mode == "" {
		mode = ModeWireframe
	}
	if mode != ModeWireframe && mode != ModeImage {
		return "", "", fmt.Errorf("unsupported render mode %q", mode)
	}
	if mode == ModeImage {
		return ModeWireframe, "image-tools unavailable; rendered deterministic wireframe only", nil
	}
	return mode, "", nil
}

func writeHeader(buf *bytes.Buffer, page spec.PageDocument) {
	buf.WriteString("<header class=\"hero\"><p class=\"eyebrow\">")
	buf.WriteString(esc(page.Page.ID))
	buf.WriteString("</p><h1>")
	buf.WriteString(esc(page.Page.Title))
	buf.WriteString("</h1><p>")
	buf.WriteString(esc(page.Page.Purpose))
	buf.WriteString("</p><nav aria-label=\"Routes\">")
	for _, route := range page.Page.Routes {
		buf.WriteString("<span>")
		buf.WriteString(esc(route))
		buf.WriteString("</span>")
	}
	buf.WriteString("</nav></header>\n")
}

func writePriorities(buf *bytes.Buffer, page spec.PageDocument) {
	buf.WriteString("<section class=\"priorities\" aria-label=\"Communication priorities\">\n")
	for i, priority := range page.Priorities {
		buf.WriteString("<article><strong>")
		buf.WriteString(fmt.Sprintf("P%d", i+1))
		buf.WriteString("</strong><p>")
		buf.WriteString(esc(priority.Statement))
		buf.WriteString("</p></article>\n")
	}
	buf.WriteString("</section>\n")
}

func writeRegions(buf *bytes.Buffer, page spec.PageDocument) {
	regions := page.Sketch.Regions
	if len(regions) == 0 {
		regions = []spec.SketchRegion{{ID: "declared-elements", Elements: elementIDs(page.Elements)}}
	}
	buf.WriteString("<section class=\"grid\" aria-label=\"Wireframe regions\">\n")
	for _, region := range regions {
		buf.WriteString("<article class=\"region\" data-region=\"")
		buf.WriteString(esc(region.ID))
		buf.WriteString("\"><h2>")
		buf.WriteString(esc(region.ID))
		buf.WriteString("</h2>\n")
		for _, elementID := range region.Elements {
			el, ok := elementByID(page.Elements, elementID)
			if !ok {
				continue
			}
			binding := page.Bindings.Elements[elementID]
			buf.WriteString("<div class=\"element\" data-element=\"")
			buf.WriteString(esc(el.ID))
			buf.WriteString("\"><span class=\"role\">")
			buf.WriteString(esc(el.Role))
			buf.WriteString("</span><strong>")
			buf.WriteString(esc(el.Name))
			buf.WriteString("</strong><p>")
			buf.WriteString(esc(el.Description))
			buf.WriteString("</p><code>")
			buf.WriteString(esc(bindingLabel(binding)))
			buf.WriteString("</code></div>\n")
		}
		buf.WriteString("</article>\n")
	}
	buf.WriteString("</section>\n")
}

func writeClaims(buf *bytes.Buffer, page spec.PageDocument) {
	claims := append([]spec.Claim(nil), page.Claims...)
	sort.SliceStable(claims, func(i, j int) bool { return claims[i].ID < claims[j].ID })
	buf.WriteString("<section class=\"claims\" aria-label=\"Claim annotations\">\n")
	for _, claim := range claims {
		buf.WriteString("<article class=\"claim ")
		buf.WriteString(esc(claim.Tier))
		buf.WriteString("\"><header><strong>")
		buf.WriteString(esc(claim.ID))
		buf.WriteString("</strong><span>")
		buf.WriteString(esc(claim.Type + " / " + claim.Tier))
		buf.WriteString("</span></header><p>")
		buf.WriteString(esc(claim.Statement))
		buf.WriteString("</p><footer>")
		buf.WriteString(esc(strings.Join(claim.Elements, ", ")))
		buf.WriteString("</footer></article>\n")
	}
	buf.WriteString("</section>\n")
}

func writeArtifact(scenarioDir, scenario, pageID, mode, htmlDoc string) (string, error) {
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{AppID: "vrooli", Profile: corestorage.ProfileAuto})
	if err != nil {
		return "", fmt.Errorf("create wireframe storage resolver: %w", err)
	}
	dir, err := resolver.EnsureArtifactDir(corestorage.Options{ScenarioID: "experience-manager"}, corestorage.ArtifactRef{
		Owner: "experience-manager", Domain: "wireframes", Class: corestorage.ClassCache, Segments: []string{scenario},
	}, 0o755)
	if err != nil {
		return "", fmt.Errorf("create wireframe artifact directory: %w", err)
	}
	path := filepath.Join(dir, pageID+"."+mode+".html")
	if err := os.WriteFile(path, []byte(htmlDoc), 0o644); err != nil {
		return "", fmt.Errorf("write wireframe artifact: %w", err)
	}
	return path, nil
}

func css() string {
	return `:root{color-scheme:dark;font-family:Inter,Arial,sans-serif;color:#f8fafc;background:#020617}body{margin:0;background:#020617}.wireframe,.variant-compare{background:#020617;color:#f8fafc}.wireframe{max-width:1180px;margin:0 auto;padding:32px}.wireframe .hero,.variant-compare .hero{border-bottom:2px solid #334155;padding-bottom:20px}.wireframe .eyebrow,.variant-compare .eyebrow{font-size:12px;text-transform:uppercase;letter-spacing:0;color:#38bdf8}.wireframe h1,.variant-compare h1{font-size:32px;line-height:1.1;margin:0 0 12px}.wireframe nav{display:flex;gap:8px;flex-wrap:wrap}.wireframe nav span,.wireframe .role{border:1px solid #475569;border-radius:6px;padding:3px 7px;background:#0f172a;color:#cbd5e1}.wireframe .priorities{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:12px;margin:20px 0}.wireframe .priorities article,.wireframe .region,.wireframe .claim,.variant{background:#0f172a;border:2px solid #334155;border-radius:6px;padding:14px}.wireframe .grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:16px}.wireframe .element{border:1px dashed #475569;border-radius:6px;padding:12px;margin:10px 0;background:#111c31}.wireframe .element strong{display:block;margin-top:8px}.wireframe .element code{display:inline-block;margin-top:8px;color:#fbbf24}.wireframe .claims{display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:12px;margin-top:20px}.wireframe .claim header{display:flex;justify-content:space-between;gap:12px}.wireframe .claim.machine{border-color:#22c55e}.wireframe .claim.manual{border-color:#f59e0b}.wireframe .claim.aspirational{border-color:#818cf8}`
}

func variantCSS() string {
	return `.variant-compare{max-width:1440px;margin:0 auto;padding:32px}.variant-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(420px,1fr));gap:16px;margin-top:20px}.variant h2{font-size:18px;margin:0 0 12px}.variant iframe{width:100%;height:720px;border:1px solid #334155;border-radius:6px;background:#020617}`
}

func elementIDs(elements []spec.Element) []string {
	out := make([]string, 0, len(elements))
	for _, el := range elements {
		out = append(out, el.ID)
	}
	return out
}

func elementByID(elements []spec.Element, id string) (spec.Element, bool) {
	for _, el := range elements {
		if el.ID == id {
			return el, true
		}
	}
	return spec.Element{}, false
}

func bindingLabel(binding spec.Binding) string {
	switch {
	case binding.TestID != "":
		return "data-testid=" + binding.TestID
	case binding.Selector != "":
		return binding.Selector
	default:
		return "unbound"
	}
}

func esc(value string) string {
	return html.EscapeString(value)
}
