package prose

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/textmetrics"
)

const bookChapterWordTarget = 550
const bookMatterWordTarget = 250

func (s *Service) loadBookProfileVersion(ctx context.Context, ref string) (Profile, error) {
	separator := strings.LastIndex(ref, "@")
	if separator <= 0 || separator == len(ref)-1 {
		return Profile{}, errors.New("book source export requires a pinned section profile version")
	}
	key := ref[:separator]
	version, err := strconv.Atoi(ref[separator+1:])
	if err != nil || version <= 0 {
		return Profile{}, errors.New("book source export requires a valid pinned section profile version")
	}
	var raw string
	if err := s.db.QueryRowContext(ctx, `SELECT payload FROM prose_records WHERE kind='profile' AND record_key=? AND version=?`, key, version).Scan(&raw); err != nil {
		return Profile{}, fmt.Errorf("book source export cannot resolve pinned section profile %s: %w", ref, err)
	}
	var profile Profile
	if err := json.Unmarshal([]byte(raw), &profile); err != nil {
		return Profile{}, fmt.Errorf("book source export cannot decode pinned section profile %s: %w", ref, err)
	}
	if profile.Key != key || profile.Version != version {
		return Profile{}, errors.New("book source export profile version record does not match its pinned reference")
	}
	return profile, nil
}

func (s *Service) validateStoredBookSource(ctx context.Context, doc Document, candidates map[string]Candidate) error {
	if doc.BookBrief == nil {
		return ErrInvalidBookBrief
	}
	if len(doc.Sections) == 0 {
		return errors.New("book source export requires at least one committed section")
	}
	needsChunkValidation := doc.BookBrief.ProductionMetadata.GeneratedWithAI
	for _, section := range doc.Sections {
		candidate := candidates[section.CommittedCandidateID]
		if candidate.SectionAssembly != nil || len(section.PartGenerationRuns) > 0 {
			needsChunkValidation = true
		}
	}
	if !needsChunkValidation {
		if err := validateBookSourceOrder(doc, false, defaultTargetSectionWords); err != nil {
			return err
		}
		return nil // Explicit supported path for preauthored, non-AI source sections.
	}
	if doc.BookBrief.ProductionMetadata.GeneratedWithAI && doc.Status != "assembled" {
		return errors.New("book source export requires an assembled AI manuscript")
	}
	profile, err := s.loadBookProfileVersion(ctx, doc.SectionProfileVersion)
	if err != nil {
		return err
	}
	if profile.GatewayRole != "write.default" || strings.TrimSpace(profile.Locality) != "local_only" {
		return ErrPrivateRouteNotLocal
	}
	profileChapterMin, profileChapterMax := sectionWordBand(profile, bookChapterWordTarget)
	if profile.Composition.TargetSectionWords != bookChapterWordTarget || profileChapterMin != 440 || profileChapterMax != 660 {
		return errors.New("book source export pinned profile does not declare the unchanged 440-660 chapter band")
	}
	if err := validateBookSourceOrder(doc, true, profile.Composition.TargetSectionWords); err != nil {
		return err
	}

	assembled := make([]string, len(doc.Sections))
	var totalWords, totalInput, totalOutput int
	var totalCost int64
	var providers, models []string
	for i, section := range doc.Sections {
		persistedSection, err := s.loadSection(ctx, section.ID)
		if err != nil {
			return fmt.Errorf("book source export missing durable section %s: %w", section.ID, err)
		}
		if !sameSectionBinding(section, persistedSection) {
			return fmt.Errorf("book source export document and durable section binding differ at position %d", i)
		}
		candidate, ok := candidates[section.CommittedCandidateID]
		if !ok {
			return fmt.Errorf("book source export missing canonical candidate at position %d", i)
		}
		if err := s.validateStoredBookSection(ctx, doc, section, candidate, profile, i); err != nil {
			return fmt.Errorf("book source export section %d quality/provenance rejected: %w", i+1, err)
		}
		assembled[i] = candidate.Text
		metrics := textmetrics.Analyze(candidate.Text, nil)
		totalWords += metrics.WordCount
		totalInput += candidate.Provenance.InputTokens
		totalOutput += candidate.Provenance.OutputTokens
		totalCost += candidate.Provenance.CostMicros
		providers = appendUnique(providers, candidate.Provenance.Provider)
		models = appendUnique(models, candidate.Provenance.ResolvedModelRef)
	}
	assembledText := strings.Join(assembled, "\n\n")
	if doc.AssembledText != assembledText || doc.Provenance.SectionCount != len(doc.Sections) || doc.Provenance.WordCount != totalWords || doc.Provenance.InputTokens != totalInput || doc.Provenance.OutputTokens != totalOutput || doc.Provenance.TotalCostMicros != totalCost || !sameStrings(doc.Provenance.Providers, providers) || !sameStrings(doc.Provenance.Models, models) {
		return errors.New("book source export document aggregate text or generation provenance is stale or mismatched")
	}
	return nil
}

func validateBookSourceOrder(doc Document, strictTargets bool, chapterTarget int) error {
	if doc.BookBrief == nil || doc.BookBrief.ProductionMetadata == nil {
		return ErrInvalidBookBrief
	}
	expected := make([]OutlineSection, 0, len(doc.BookBrief.FrontMatterIntent)+len(doc.BookBrief.ChapterIntent)+len(doc.BookBrief.BackMatterIntent))
	for _, intent := range doc.BookBrief.FrontMatterIntent {
		expected = append(expected, OutlineSection{Kind: "front_matter", Intent: intent, TargetWords: bookMatterWordTarget})
	}
	for _, intent := range doc.BookBrief.ChapterIntent {
		expected = append(expected, OutlineSection{Kind: "chapter", Intent: intent, TargetWords: chapterTarget})
	}
	for _, intent := range doc.BookBrief.BackMatterIntent {
		expected = append(expected, OutlineSection{Kind: "back_matter", Intent: intent, TargetWords: bookMatterWordTarget})
	}
	if len(doc.Sections) != len(expected) || len(doc.SectionIDs) != len(expected) {
		return errors.New("book source export section count/order does not match brief intents")
	}
	if len(doc.Outline) != len(expected) {
		return errors.New("book source export outline count does not match brief intents")
	}
	for i, wanted := range expected {
		section := doc.Sections[i]
		outline := doc.Outline[i]
		badTarget := section.TargetWords != wanted.TargetWords
		if !strictTargets && section.TargetWords == 0 {
			badTarget = false
		}
		if section.ID == "" || doc.SectionIDs[i] != section.ID || section.DocumentID != doc.ID || section.Position != i || section.Intent != wanted.Intent || section.Kind != wanted.Kind || badTarget {
			return fmt.Errorf("book source export section order/kind/intent drift at position %d: kind=%q intent=%q target=%d position=%d section_id=%q listed_id=%q document_id=%q; want kind=%q intent=%q target=%d", i, section.Kind, section.Intent, section.TargetWords, section.Position, section.ID, doc.SectionIDs[i], section.DocumentID, wanted.Kind, wanted.Intent, wanted.TargetWords)
		}
		badOutlineTarget := outline.TargetWords != wanted.TargetWords
		if !strictTargets && outline.TargetWords == 0 {
			badOutlineTarget = false
		}
		if outline.Kind != wanted.Kind || outline.Intent != wanted.Intent || badOutlineTarget {
			return fmt.Errorf("book source export outline order/kind/intent drift at position %d", i)
		}
	}
	return nil
}

func sameSectionBinding(docSection, stored Section) bool {
	generationRunEqual := docSection.GenerationRun == nil && stored.GenerationRun == nil
	if docSection.GenerationRun != nil && stored.GenerationRun != nil {
		generationRunEqual = *docSection.GenerationRun == *stored.GenerationRun
	}
	return docSection.ID == stored.ID && docSection.DocumentID == stored.DocumentID &&
		docSection.Position == stored.Position && docSection.Intent == stored.Intent && docSection.Summary == stored.Summary &&
		docSection.TargetWords == stored.TargetWords && docSection.ProfileKey == stored.ProfileKey &&
		docSection.SessionID == stored.SessionID && docSection.CommittedCandidateID == stored.CommittedCandidateID &&
		generationRunEqual &&
		sameBookRuns(docSection.PartGenerationRuns, stored.PartGenerationRuns) && docSection.Kind == stored.Kind
}

func sameBookRuns(a, b []BookGenerationRun) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (s *Service) validateStoredBookSection(ctx context.Context, doc Document, section Section, assembly Candidate, profile Profile, position int) error {
	if assembly.ID != section.CommittedCandidateID || !assembly.Committed || assembly.ID == "" || assembly.RoundID == "" {
		return errors.New("canonical assembled candidate is not the committed section candidate")
	}
	if assembly.SectionAssembly == nil {
		return errors.New("canonical candidate is missing its chunk assembly contract")
	}
	if assembly.Provenance.GatewayRole != "write.default" || assembly.Provenance.Locality != "local_only" ||
		assembly.Provenance.ProfileVersion != doc.SectionProfileVersion || assembly.Provenance.Provider == "" || assembly.Provenance.ResolvedModelRef == "" ||
		assembly.Provenance.Strategy != "book_section_chunked" || !assembly.Eligibility.Eligible {
		return errors.New("canonical candidate route/profile/eligibility provenance is incomplete or mismatched")
	}
	minWords, maxWords := sectionWordBand(profile, section.TargetWords)
	if section.Kind == "chapter" && (section.TargetWords != bookChapterWordTarget || minWords != 440 || maxWords != 660) {
		return errors.New("chapter does not use the exact 440-660 word gate")
	}
	if section.Kind != "chapter" && (section.TargetWords != bookMatterWordTarget || minWords != 200 || maxWords != 300) {
		return errors.New("matter does not use the declared 200-300 word band")
	}
	assemblyMeta := assembly.SectionAssembly
	if assemblyMeta.Kind != section.Kind || assemblyMeta.Intent != section.Intent || assemblyMeta.MinWords != minWords || assemblyMeta.MaxWords != maxWords {
		return errors.New("canonical assembly kind/intent/word bounds are stale or mismatched")
	}
	following := []string(nil)
	if position+1 < len(doc.Outline) {
		following = []string{doc.Outline[position+1].Intent}
	}
	plans := bookSectionPartPlans(section, section.TargetWords, minWords, following)
	if len(assemblyMeta.Parts) != len(plans) || len(assembly.DerivedFrom) != len(plans) || len(section.PartGenerationRuns) != len(plans) {
		return errors.New("canonical assembly has incomplete part evidence or generation runs")
	}
	parts := make([]Candidate, len(plans))
	partIDs := make([]string, len(plans))
	promptHashes := make([]string, len(plans))
	var inputTokens, outputTokens int
	var partRunsLatest time.Time
	provider, model := "", ""
	for i := range plans {
		evidence := assemblyMeta.Parts[i]
		run := section.PartGenerationRuns[i]
		partID := assembly.DerivedFrom[i]
		if partID == "" || evidence.Index != i+1 || evidence.CandidateID != partID ||
			run.PartIndex != i+1 || run.CandidateID != partID || run.SectionID != section.ID {
			return fmt.Errorf("part %d identity or order drift", i+1)
		}
		part, err := s.loadCandidate(ctx, partID)
		if err != nil {
			return fmt.Errorf("part %d candidate is missing: %w", i+1, err)
		}
		if part.ID != partID || part.RoundID == "" || part.SectionAssembly != nil || len(part.DerivedFrom) > 0 || !part.Eligibility.Eligible {
			return fmt.Errorf("part %d candidate identity, lineage, or eligibility is invalid", i+1)
		}
		if part.Provenance.GatewayRole != "write.default" || part.Provenance.Locality != "local_only" || part.Provenance.ProfileVersion != doc.SectionProfileVersion ||
			part.Provenance.Provider == "" || part.Provenance.ResolvedModelRef == "" || part.Provenance.Strategy != samplerDirect || part.Provenance.StrategyParameters.Kind != samplerDirect || part.Provenance.StrategyParameters.K != 1 ||
			!isSHA256(part.Provenance.PromptSHA256) || part.Provenance.InputTokens <= 0 || part.Provenance.OutputTokens <= 0 || part.Provenance.MaxOutputTokensEffective <= 0 || part.Provenance.OutputTokens > part.Provenance.MaxOutputTokensEffective {
			return fmt.Errorf("part %d route/model/token/prompt provenance is incomplete", i+1)
		}
		if provider == "" {
			provider, model = part.Provenance.Provider, part.Provenance.ResolvedModelRef
		} else {
			if provider != part.Provenance.Provider {
				provider = "mixed"
			}
			if model != part.Provenance.ResolvedModelRef {
				model = "mixed"
			}
		}
		if _, err := validateBookRunDate(run, doc.CreatedAt, s.now()); err != nil {
			return fmt.Errorf("part %d generation date is invalid: %w", i+1, err)
		} else if parsed, _ := time.Parse(time.RFC3339, run.Date); parsed.After(partRunsLatest) {
			partRunsLatest = parsed
		}
		metrics := textmetrics.Analyze(strings.TrimSpace(part.Text), nil)
		if metrics.WordCount != evidence.WordCount || metrics.WordCount != run.WordCount || run.InputTokens != part.Provenance.InputTokens || run.OutputTokens != part.Provenance.OutputTokens ||
			evidence.Provider != part.Provenance.Provider || evidence.Model != part.Provenance.ResolvedModelRef || evidence.Locality != part.Provenance.Locality ||
			evidence.InputTokens != part.Provenance.InputTokens || evidence.OutputTokens != part.Provenance.OutputTokens || evidence.PromptSHA256 != part.Provenance.PromptSHA256 ||
			!isSHA256(evidence.OutputSHA256) || evidence.OutputSHA256 != hash([]byte(strings.TrimSpace(part.Text))) || run.PromptSHA256 != part.Provenance.PromptSHA256 || run.OutputSHA256 != evidence.OutputSHA256 ||
			run.Provider != part.Provenance.Provider || run.Model != part.Provenance.ResolvedModelRef {
			return fmt.Errorf("part %d text/hash/metrics/generation run does not match its stored candidate", i+1)
		}
		if err := s.validateCandidateBinding(ctx, part, section.SessionID, profile.Key); err != nil {
			return fmt.Errorf("part %d candidate lineage is unbound: %w", i+1, err)
		}
		parts[i] = part
		partIDs[i] = partID
		promptHashes[i] = part.Provenance.PromptSHA256
		inputTokens += part.Provenance.InputTokens
		outputTokens += part.Provenance.OutputTokens
	}
	cleanParts := make([]Candidate, len(parts))
	for i, part := range parts {
		cleanParts[i] = part
		cleanParts[i].Eligibility = Eligibility{Eligible: true}
	}
	combined, expectedEvidence, err := validateBookChunkCandidate(doc.BookBrief, plans, cleanParts)
	if err != nil {
		return fmt.Errorf("stored part text fails quality boundary: %w", err)
	}
	if combined != assembly.Text {
		return errors.New("canonical section text is not exactly derived from its ordered part texts")
	}
	if assemblyMeta.SchemaVersion != bookSectionChunkVersion {
		return errors.New("canonical candidate uses a stale chunk assembly contract")
	}
	for i := range plans {
		if assemblyMeta.Parts[i].Purpose != plans[i].Purpose || section.PartGenerationRuns[i].Purpose != plans[i].Purpose {
			return fmt.Errorf("part %d purpose differs from the current ordered section plan", i+1)
		}
	}
	for i := range expectedEvidence {
		if expectedEvidence[i] != assemblyMeta.Parts[i] {
			return fmt.Errorf("part %d assembly metadata does not match recomputed evidence", i+1)
		}
	}
	fullMetrics := textmetrics.Analyze(assembly.Text, nil)
	constraints := profile.Constraints
	constraints.MinWords, constraints.MaxWords = minWords, maxWords
	constraints.RequiredFormat = sectionRequiredFormat(profile)
	fullGate := gate(fullMetrics, constraints, assembly.Text, gateInputs{})
	if !fullGate.Eligible || assemblyMeta.WordCount != fullMetrics.WordCount || candidateWordCount(assembly) != fullMetrics.WordCount {
		return fmt.Errorf("canonical section fails the unchanged full word/quality gate: %s", fullGate.Reason)
	}
	if violation := bookBriefViolation(doc.BookBrief, assembly.Text); violation != "" {
		return fmt.Errorf("canonical section violates the evidence boundary: %s", violation)
	}
	sectionRun := section.GenerationRun
	if sectionRun == nil {
		return errors.New("canonical section generation run is missing")
	}
	assemblyPromptHash := hash([]byte(strings.Join(promptHashes, "\n")))
	if _, err := validateBookRunDate(*sectionRun, doc.CreatedAt, s.now()); err != nil {
		return fmt.Errorf("canonical section generation date is invalid: %w", err)
	}
	sectionDate, _ := time.Parse(time.RFC3339, sectionRun.Date)
	if sectionDate.Before(partRunsLatest) {
		return errors.New("canonical section generation run predates a part generation run")
	}
	if sectionRun.SectionID != section.ID || sectionRun.CandidateID != assembly.ID || sectionRun.Purpose != "canonical assembled manuscript section" ||
		sectionRun.WordCount != fullMetrics.WordCount || sectionRun.Provider != provider || sectionRun.Model != model || sectionRun.InputTokens != inputTokens || sectionRun.OutputTokens != outputTokens ||
		sectionRun.PromptSHA256 != assemblyPromptHash || sectionRun.OutputSHA256 != hash([]byte(assembly.Text)) ||
		assembly.Provenance.PromptSHA256 != assemblyPromptHash || assembly.Provenance.Provider != provider || assembly.Provenance.ResolvedModelRef != model || assembly.Provenance.InputTokens != inputTokens || assembly.Provenance.OutputTokens != outputTokens {
		return errors.New("canonical generation provenance is incomplete, stale, or mismatched")
	}
	if err := s.validateCandidateBinding(ctx, assembly, section.SessionID, profile.Key); err != nil {
		return fmt.Errorf("canonical candidate lineage is unbound: %w", err)
	}
	return nil
}

func validateBookRunDate(run BookGenerationRun, documentCreatedAt, now time.Time) (time.Time, error) {
	if run.Date == "" {
		return time.Time{}, errors.New("generation date is missing")
	}
	date, err := time.Parse(time.RFC3339, run.Date)
	if err != nil {
		return time.Time{}, errors.New("generation date is not RFC3339")
	}
	if documentCreatedAt.IsZero() || date.Before(documentCreatedAt.Truncate(time.Second)) || date.After(now.Add(5*time.Minute)) {
		return time.Time{}, errors.New("generation date is outside the document's durable lifetime")
	}
	return date, nil
}

func (s *Service) validateCandidateBinding(ctx context.Context, candidate Candidate, sectionSessionID, profileKey string) error {
	var round Round
	if err := s.loadJSON(ctx, "prose_rounds", candidate.RoundID, &round); err != nil {
		return err
	}
	if round.ID != candidate.RoundID || round.SessionID != sectionSessionID || !containsString(round.CandidateIDs, candidate.ID) {
		return errors.New("candidate round does not bind the candidate to the section session")
	}
	if candidate.SectionAssembly != nil && round.CompositionMode != bookSectionChunkVersion {
		return errors.New("canonical candidate round has a stale composition version")
	}
	if candidate.SectionAssembly == nil && (round.CompositionMode != "" || round.Strategy.Kind != samplerDirect || round.Strategy.K != 1) {
		return errors.New("part candidate round has an unexpected strategy")
	}
	var session Session
	if err := s.loadJSON(ctx, "prose_sessions", sectionSessionID, &session); err != nil {
		return err
	}
	if session.ID != sectionSessionID || session.ProfileKey != profileKey || !containsString(session.RoundIDs, round.ID) {
		return errors.New("candidate session does not bind its round to the pinned profile")
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (s *Service) loadCandidate(ctx context.Context, id string) (Candidate, error) {
	var candidate Candidate
	if err := s.loadJSON(ctx, "prose_candidates", id, &candidate); err != nil {
		return Candidate{}, err
	}
	return candidate, nil
}

func (s *Service) loadSection(ctx context.Context, id string) (Section, error) {
	var section Section
	if err := s.loadJSON(ctx, "prose_sections", id, &section); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Section{}, fmt.Errorf("section %s has no durable record", id)
		}
		return Section{}, err
	}
	return section, nil
}
