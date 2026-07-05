package authoring

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"experience-manager/internal/reconcile"
	specRender "experience-manager/internal/render"
	"experience-manager/internal/spec"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
)

type Service struct {
	Repo     Repository
	Evidence reconcile.EvidenceRepository
	RepoRoot string
	Now      func() time.Time
}

type Diff struct {
	Path   string
	Action string
	Before string
	After  string
}

type Preview struct {
	Session Session
	Diffs   []Diff
	Report  spec.Report
}

type Suggestion struct {
	ElementID      string
	TestID         string
	Role           string
	AccessibleName string
	Source         string
}

type (
	RenderResult  = specRender.Result
	VariantResult = specRender.VariantResult
)

type VariantPromotion struct {
	Scenario string
	PageID   string
	Variant  VariantResult
	Diffs    []Diff
	Report   spec.Report
}

func (s Service) StartSession(ctx context.Context, scenario, path string) (Session, error) {
	target, err := s.resolveTarget(scenario, path)
	if err != nil {
		return Session{}, err
	}
	now := s.now()
	session := Session{
		ID:         newID("expauth"),
		Scenario:   scenarioName(scenario, target),
		TargetPath: target,
		Status:     "open",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.repo().SaveSession(ctx, session); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s Service) SubmitPage(ctx context.Context, sessionID string, form *contractv1.PageForm) (Session, PageDraft, error) {
	session, err := s.repo().GetSession(ctx, sessionID)
	if err != nil {
		return Session{}, PageDraft{}, err
	}
	page, status := pageFromForm(form)
	if page.Page.ID == "" {
		return Session{}, PageDraft{}, fmt.Errorf("page id is required")
	}
	data, err := json.MarshalIndent(page, "", "  ")
	if err != nil {
		return Session{}, PageDraft{}, fmt.Errorf("encode page %q: %w", page.Page.ID, err)
	}
	data = append(data, '\n')
	now := s.now()
	draft := PageDraft{
		SessionID: session.ID,
		PageID:    page.Page.ID,
		Path:      "pages/" + page.Page.ID + ".json",
		Title:     page.Page.Title,
		Status:    status,
		JSON:      string(data),
		UpdatedAt: now,
	}
	if err := s.repo().SavePage(ctx, draft); err != nil {
		return Session{}, PageDraft{}, err
	}
	session.UpdatedAt = now
	if err := s.repo().SaveSession(ctx, session); err != nil {
		return Session{}, PageDraft{}, err
	}
	return session, draft, nil
}

func (s Service) Preview(ctx context.Context, sessionID string) (Preview, error) {
	session, pages, err := s.sessionPages(ctx, sessionID)
	if err != nil {
		return Preview{}, err
	}
	tmp, err := os.MkdirTemp("", "experience-authoring-*")
	if err != nil {
		return Preview{}, err
	}
	defer os.RemoveAll(tmp)
	if err := copyScenarioForPreview(session.TargetPath, tmp); err != nil {
		return Preview{}, err
	}
	diffs, err := applyDrafts(tmp, pages, true)
	if err != nil {
		return Preview{}, err
	}
	report, err := spec.ParseScenario(tmp)
	if err != nil {
		return Preview{}, err
	}
	report.Scenario = session.Scenario
	report.TargetPath = session.TargetPath
	return Preview{Session: session, Diffs: diffs, Report: report}, nil
}

func (s Service) Apply(ctx context.Context, sessionID string) (Preview, error) {
	preview, err := s.Preview(ctx, sessionID)
	if err != nil {
		return Preview{}, err
	}
	for _, finding := range preview.Report.Findings {
		if finding.Severity == spec.SeverityError {
			return preview, fmt.Errorf("preview has contract findings; apply refused")
		}
	}
	_, pages, err := s.sessionPages(ctx, sessionID)
	if err != nil {
		return Preview{}, err
	}
	diffs, err := applyDrafts(preview.Session.TargetPath, pages, true)
	if err != nil {
		return Preview{}, err
	}
	report, err := spec.ParseScenario(preview.Session.TargetPath)
	if err != nil {
		return Preview{}, err
	}
	report.Scenario = preview.Session.Scenario
	preview.Diffs = diffs
	preview.Report = report
	return preview, nil
}

func (s Service) Discard(ctx context.Context, sessionID string) error {
	return s.repo().DeleteSession(ctx, sessionID)
}

func (s Service) ListSpec(_ context.Context, scenario, path string) (spec.Report, error) {
	target, err := s.resolveTarget(scenario, path)
	if err != nil {
		return spec.Report{}, err
	}
	report, err := spec.ParseScenario(target)
	if err != nil {
		return report, err
	}
	if scenario != "" {
		report.Scenario = scenario
	}
	return report, nil
}

func (s Service) ShowPage(ctx context.Context, scenario, path, pageID string) (string, error) {
	report, err := s.ListSpec(ctx, scenario, path)
	if err != nil {
		return "", err
	}
	if report.Spec == nil {
		return "", fmt.Errorf("scenario %q has no parsed experience spec", report.Scenario)
	}
	ref, ok := pageRef(report.Spec.Index.Pages, pageID)
	if !ok {
		return "", fmt.Errorf("page %q not found", pageID)
	}
	data, err := os.ReadFile(filepath.Join(report.Spec.ExperienceDir, filepath.FromSlash(ref.Path)))
	if err != nil {
		return "", fmt.Errorf("read page %q: %w", pageID, err)
	}
	return string(data), nil
}

func (s Service) SuggestBindings(ctx context.Context, scenario, path, pageID string, limit int) ([]Suggestion, error) {
	report, err := s.ListSpec(ctx, scenario, path)
	if err != nil {
		return nil, err
	}
	if report.Spec == nil {
		return nil, fmt.Errorf("scenario %q has no parsed experience spec", report.Scenario)
	}
	page, ok := report.Spec.Pages[pageID]
	if !ok {
		return nil, fmt.Errorf("page %q not found", pageID)
	}
	var out []Suggestion
	for _, el := range page.Elements {
		b := page.Bindings.Elements[el.ID]
		if b.TestID == "" && b.Selector == "" {
			continue
		}
		out = append(out, Suggestion{
			ElementID:      el.ID,
			TestID:         b.TestID,
			Role:           el.Role,
			AccessibleName: el.Name,
			Source:         "spec",
		})
	}
	if s.Evidence != nil {
		evidence, err := s.Evidence.ListEvidence(ctx, reconcile.EvidenceFilter{Scenario: report.Scenario, PageID: pageID, Limit: 20})
		if err != nil {
			return nil, err
		}
		for _, ev := range evidence {
			role, name := evidenceRoleName(ev.AXNodeJSON)
			if role == "" && name == "" {
				continue
			}
			out = append(out, Suggestion{Role: role, AccessibleName: name, Source: "evidence:" + ev.ClaimID})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ElementID == out[j].ElementID {
			return out[i].Source < out[j].Source
		}
		return out[i].ElementID < out[j].ElementID
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s Service) RenderSpec(_ context.Context, scenario, path, pageID, mode string) (RenderResult, error) {
	target, err := s.resolveTarget(scenario, path)
	if err != nil {
		return RenderResult{}, err
	}
	return specRender.Render(specRender.Request{
		ScenarioDir: target,
		Scenario:    scenarioName(scenario, target),
		PageID:      pageID,
		Mode:        mode,
	})
}

func (s Service) CompareVariants(_ context.Context, scenario, path, pageID, mode string, variants []*contractv1.SpecVariant) (specRender.CompareResult, error) {
	target, err := s.resolveTarget(scenario, path)
	if err != nil {
		return specRender.CompareResult{}, err
	}
	renderVariants, err := renderVariantsFromProto(pageID, variants)
	if err != nil {
		return specRender.CompareResult{}, err
	}
	return specRender.Compare(specRender.CompareRequest{
		ScenarioDir: target,
		Scenario:    scenarioName(scenario, target),
		PageID:      pageID,
		Mode:        mode,
		Variants:    renderVariants,
	})
}

func (s Service) PromoteVariant(_ context.Context, scenario, path, pageID string, variant *contractv1.SpecVariant) (VariantPromotion, error) {
	target, err := s.resolveTarget(scenario, path)
	if err != nil {
		return VariantPromotion{}, err
	}
	if strings.TrimSpace(pageID) == "" {
		return VariantPromotion{}, fmt.Errorf("page is required")
	}
	draft, renderVariant, err := pageDraftFromVariant(pageID, variant)
	if err != nil {
		return VariantPromotion{}, err
	}
	tmp, err := os.MkdirTemp("", "experience-variant-*")
	if err != nil {
		return VariantPromotion{}, err
	}
	defer os.RemoveAll(tmp)
	if err := copyScenarioForPreview(target, tmp); err != nil {
		return VariantPromotion{}, err
	}
	if _, err := applyDrafts(tmp, []PageDraft{draft}, true); err != nil {
		return VariantPromotion{}, err
	}
	previewReport, err := spec.ParseScenario(tmp)
	if err != nil {
		return VariantPromotion{}, err
	}
	for _, finding := range previewReport.Findings {
		if finding.Severity == spec.SeverityError {
			previewReport.Scenario = scenarioName(scenario, target)
			previewReport.TargetPath = target
			return VariantPromotion{Scenario: previewReport.Scenario, PageID: pageID, Variant: renderVariant, Report: previewReport}, fmt.Errorf("promoted variant has contract findings; apply refused")
		}
	}
	diffs, err := applyDrafts(target, []PageDraft{draft}, true)
	if err != nil {
		return VariantPromotion{}, err
	}
	report, err := spec.ParseScenario(target)
	if err != nil {
		return VariantPromotion{}, err
	}
	report.Scenario = scenarioName(scenario, target)
	report.TargetPath = target
	renderVariant.HTML = specRender.RenderPage(report.Scenario, mustPage(report, pageID))
	return VariantPromotion{
		Scenario: report.Scenario,
		PageID:   pageID,
		Variant:  renderVariant,
		Diffs:    diffs,
		Report:   report,
	}, nil
}

func (s Service) sessionPages(ctx context.Context, sessionID string) (Session, []PageDraft, error) {
	session, err := s.repo().GetSession(ctx, sessionID)
	if err != nil {
		return Session{}, nil, err
	}
	pages, err := s.repo().ListPages(ctx, sessionID)
	if err != nil {
		return Session{}, nil, err
	}
	if len(pages) == 0 {
		return Session{}, nil, fmt.Errorf("authoring session %q has no page drafts", sessionID)
	}
	return session, pages, nil
}

func (s Service) repo() Repository {
	if s.Repo == nil {
		panic("authoring.Service requires Repository")
	}
	return s.Repo
}

func (s Service) now() string {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	return now().UTC().Format(time.RFC3339Nano)
}

func (s Service) resolveTarget(scenario, path string) (string, error) {
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("target path %q: %w", path, err)
		}
		return filepath.Clean(path), nil
	}
	if scenario == "" {
		return "", fmt.Errorf("scenario is required")
	}
	root := s.RepoRoot
	if root == "" {
		root = "."
	}
	target := filepath.Join(root, "scenarios", scenario)
	if _, err := os.Stat(target); err != nil {
		return "", fmt.Errorf("scenario %q not found under %s: %w", scenario, filepath.Join(root, "scenarios"), err)
	}
	return target, nil
}

func scenarioName(scenario, target string) string {
	if scenario != "" {
		return scenario
	}
	return filepath.Base(target)
}

func pageFromForm(form *contractv1.PageForm) (spec.PageDocument, string) {
	if form == nil {
		form = &contractv1.PageForm{}
	}
	status := strings.TrimSpace(form.GetStatus())
	if status == "" {
		status = "draft"
	}
	page := spec.PageDocument{
		Kind:          "experience-page",
		SchemaVersion: "1.0.0",
		Page: spec.PageIdentity{
			ID:      strings.TrimSpace(form.GetId()),
			Title:   strings.TrimSpace(form.GetTitle()),
			Routes:  cleanStrings(form.GetRoutes()),
			Purpose: strings.TrimSpace(form.GetPurpose()),
			PRDRefs: cleanStrings(form.GetPrdRefs()),
		},
		Bindings: spec.Bindings{Elements: map[string]spec.Binding{}},
	}
	for _, p := range form.GetPriorities() {
		page.Priorities = append(page.Priorities, spec.Priority{Statement: strings.TrimSpace(p.GetStatement()), Notes: strings.TrimSpace(p.GetNotes())})
	}
	for _, st := range form.GetStates() {
		page.States = append(page.States, spec.State{ID: strings.TrimSpace(st.GetId()), Description: strings.TrimSpace(st.GetDescription())})
	}
	for _, el := range form.GetElements() {
		page.Elements = append(page.Elements, spec.Element{ID: strings.TrimSpace(el.GetId()), Role: strings.TrimSpace(el.GetRole()), Name: strings.TrimSpace(el.GetName()), Description: strings.TrimSpace(el.GetDescription())})
	}
	for _, cl := range form.GetClaims() {
		page.Claims = append(page.Claims, spec.Claim{
			ID:        strings.TrimSpace(cl.GetId()),
			Type:      strings.TrimSpace(cl.GetType()),
			Statement: strings.TrimSpace(cl.GetStatement()),
			Tier:      strings.TrimSpace(cl.GetTier()),
			Elements:  cleanStrings(cl.GetElements()),
			States:    cleanStrings(cl.GetStates()),
			Viewports: cleanStrings(cl.GetViewports()),
			Locales:   cleanStrings(cl.GetLocales()),
			Rationale: strings.TrimSpace(cl.GetRationale()),
		})
	}
	for _, b := range form.GetBindings() {
		page.Bindings.Elements[strings.TrimSpace(b.GetElementId())] = spec.Binding{TestID: strings.TrimSpace(b.GetTestid()), Selector: strings.TrimSpace(b.GetSelector()), Note: strings.TrimSpace(b.GetNote())}
	}
	for _, r := range form.GetSketchRegions() {
		page.Sketch.Regions = append(page.Sketch.Regions, spec.SketchRegion{ID: strings.TrimSpace(r.GetId()), Elements: cleanStrings(r.GetElements())})
	}
	return page, status
}

func renderVariantsFromProto(pageID string, variants []*contractv1.SpecVariant) ([]specRender.Variant, error) {
	out := make([]specRender.Variant, 0, len(variants))
	for _, item := range variants {
		page, _, err := pageDocumentFromVariant(pageID, item)
		if err != nil {
			return nil, err
		}
		out = append(out, specRender.Variant{
			ID:    strings.TrimSpace(item.GetId()),
			Title: strings.TrimSpace(item.GetTitle()),
			Page:  page,
		})
	}
	return out, nil
}

func pageDraftFromVariant(pageID string, variant *contractv1.SpecVariant) (PageDraft, specRender.VariantResult, error) {
	page, status, err := pageDocumentFromVariant(pageID, variant)
	if err != nil {
		return PageDraft{}, specRender.VariantResult{}, err
	}
	data, err := json.MarshalIndent(page, "", "  ")
	if err != nil {
		return PageDraft{}, specRender.VariantResult{}, fmt.Errorf("encode promoted page %q: %w", page.Page.ID, err)
	}
	data = append(data, '\n')
	title := strings.TrimSpace(variant.GetTitle())
	if title == "" {
		title = page.Page.Title
	}
	rendered := specRender.VariantResult{
		ID:    strings.TrimSpace(variant.GetId()),
		Title: title,
	}
	if rendered.ID == "" {
		rendered.ID = page.Page.ID
	}
	return PageDraft{
		PageID: page.Page.ID,
		Path:   "pages/" + page.Page.ID + ".json",
		Title:  page.Page.Title,
		Status: status,
		JSON:   string(data),
	}, rendered, nil
}

func pageDocumentFromVariant(pageID string, variant *contractv1.SpecVariant) (spec.PageDocument, string, error) {
	if variant == nil || variant.GetPage() == nil {
		return spec.PageDocument{}, "", fmt.Errorf("variant page form is required")
	}
	page, status := pageFromForm(variant.GetPage())
	if page.Page.ID == "" {
		page.Page.ID = strings.TrimSpace(pageID)
	}
	if page.Page.ID == "" {
		return spec.PageDocument{}, "", fmt.Errorf("variant page id is required")
	}
	if strings.TrimSpace(pageID) != "" && page.Page.ID != strings.TrimSpace(pageID) {
		return spec.PageDocument{}, "", fmt.Errorf("variant page id %q must match target page %q", page.Page.ID, strings.TrimSpace(pageID))
	}
	return page, status, nil
}

func cleanStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func copyScenarioForPreview(src, dst string) error {
	for _, rel := range []string{"experience", "PRD.md", "DESIGN.md"} {
		from := filepath.Join(src, rel)
		if _, err := os.Stat(from); err != nil {
			continue
		}
		if err := copyPath(from, filepath.Join(dst, rel)); err != nil {
			return err
		}
	}
	return nil
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, 0o755); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyPath(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func applyDrafts(target string, pages []PageDraft, write bool) ([]Diff, error) {
	expDir := filepath.Join(target, "experience")
	if err := os.MkdirAll(filepath.Join(expDir, "pages"), 0o755); err != nil {
		return nil, err
	}
	var diffs []Diff
	for _, page := range pages {
		rel := filepath.ToSlash(filepath.Join("experience", page.Path))
		path := filepath.Join(expDir, filepath.FromSlash(page.Path))
		before := readOptional(path)
		diffs = append(diffs, Diff{Path: rel, Action: diffAction(before), Before: before, After: page.JSON})
		if write {
			if err := os.WriteFile(path, []byte(page.JSON), 0o644); err != nil {
				return nil, err
			}
		}
	}
	indexPath := filepath.Join(expDir, "index.json")
	before := readOptional(indexPath)
	after, err := updatedIndex(before, pages)
	if err != nil {
		return nil, err
	}
	diffs = append(diffs, Diff{Path: "experience/index.json", Action: diffAction(before), Before: before, After: after})
	if write {
		if err := os.WriteFile(indexPath, []byte(after), 0o644); err != nil {
			return nil, err
		}
	}
	return diffs, nil
}

func updatedIndex(current string, pages []PageDraft) (string, error) {
	var doc map[string]any
	if strings.TrimSpace(current) == "" {
		doc = map[string]any{
			"kind":          "experience-index",
			"schemaVersion": "1.0.0",
			"contract": map[string]any{
				"kind":   "scenario-experience",
				"schema": "scenario-experience-spec/v1",
			},
			"pages":    []any{},
			"journeys": []any{},
		}
	} else if err := json.Unmarshal([]byte(current), &doc); err != nil {
		return "", fmt.Errorf("parse experience/index.json: %w", err)
	}
	byID := map[string]map[string]any{}
	var ordered []string
	for _, item := range listOfMaps(doc["pages"]) {
		id, _ := item["id"].(string)
		if id == "" {
			continue
		}
		byID[id] = item
		ordered = append(ordered, id)
	}
	for _, page := range pages {
		if _, ok := byID[page.PageID]; !ok {
			ordered = append(ordered, page.PageID)
		}
		byID[page.PageID] = map[string]any{"id": page.PageID, "path": page.Path, "title": page.Title, "status": page.Status}
	}
	var outPages []any
	for _, id := range ordered {
		outPages = append(outPages, byID[id])
	}
	doc["pages"] = outPages
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return string(append(data, '\n')), nil
}

func listOfMaps(value any) []map[string]any {
	items, _ := value.([]any)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if ok {
			out = append(out, m)
		}
	}
	return out
}

func readOptional(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func diffAction(before string) string {
	if before == "" {
		return "create"
	}
	return "update"
}

func mustPage(report spec.Report, pageID string) spec.PageDocument {
	if report.Spec == nil {
		return spec.PageDocument{}
	}
	return report.Spec.Pages[pageID]
}

func pageRef(refs []spec.DocumentRef, pageID string) (spec.DocumentRef, bool) {
	for _, ref := range refs {
		if ref.ID == pageID {
			return ref, true
		}
	}
	return spec.DocumentRef{}, false
}

func evidenceRoleName(raw string) (string, string) {
	var node map[string]any
	if err := json.Unmarshal([]byte(raw), &node); err != nil {
		return "", ""
	}
	role, _ := node["role"].(string)
	name, _ := node["name"].(string)
	return role, name
}

func newID(prefix string) string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(buf[:])
}
