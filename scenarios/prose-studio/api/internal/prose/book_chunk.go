package prose

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/textmetrics"
)

const bookSectionChunkVersion = "book-section-chunks/v2"

type BookSectionPartPlan struct {
	Index       int
	Purpose     string
	TargetWords int
	MinWords    int
	MaxWords    int
}

var (
	speciesAssertionPattern  = regexp.MustCompile(`(?i)\b(?:identified as|identification\s*(?:is|:)|this\s+(?:bird|plant|species|sound)\s+(?:is|must be)\s+(?:a|an))\b`)
	wellnessClaimPattern     = regexp.MustCompile(`(?i)\b(?:cures?|heals?|treats?|prevents?|reduces?)\b.{0,60}\b(?:anxiety|stress|depression|pain|illness|health|wellness)\b`)
	concreteTimePattern      = regexp.MustCompile(`(?i)\b(?:\d{1,2}:\d{2}(?:\s?[ap]m)?|\d{1,2}(?:\s?(?:a\.m\.|p\.m\.|am|pm))|\d+\s*[- ]\s*(?:seconds?|minutes?|hours?|days?|weeks?|months?|years?)|(?:one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty|thirty|forty|fifty|sixty|seventy|eighty|ninety)(?:[- ](?:one|two|three|four|five|six|seven|eight|nine))?\s*[- ]\s*(?:seconds?|minutes?|hours?|days?|weeks?|months?|years?)|(?:noon|midnight|dawn|dusk|sunrise|sunset))\b`)
	namedIntersectionPattern = regexp.MustCompile(`\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+){0,2}\s+(?:and|&)\s+[A-Z][a-z]+(?:\s+[A-Z][a-z]+){0,2}\b`)
	namedStreetPattern       = regexp.MustCompile(`\b(?:at|on|near|from|of)\s+[A-Z][a-z]+(?:\s+[A-Z][a-z]+){0,2}\s+(?:Street|St\.?|Avenue|Ave\.?|Road|Rd\.?|Boulevard|Blvd\.?|Drive|Dr\.?|Lane|Ln\.?|Way|Square|Plaza|Park|Station|Bridge)\b`)
	concreteSourcePattern    = regexp.MustCompile(`(?i)\b(?:hvac|ventilation|fan|siren|car|cars|vehicle|vehicles|truck|trucks|bus|buses|train|trains|engine|engines|tire|tires|footsteps?|bird|birds|chirp|chirps|door|doors|horn|horns|traffic light|storefront sign|cooling unit|cooling fan)\b`)
	evidenceNormalizePattern = regexp.MustCompile(`[^a-z0-9]+`)
)

func bookSectionPartPlans(section Section, sectionTarget, minWords int, followingIntents []string) []BookSectionPartPlan {
	count := maxInt(1, int(math.Ceil(float64(minWords)/80)))
	target := int(math.Ceil(float64(sectionTarget) / float64(count)))
	minPart := maxInt(30, target-45)
	maxPart := minInt(150, target+45)
	var purposes []string
	switch section.Kind {
	case "chapter":
		purposes = []string{
			"Open with a precise concept or reader question grounded in the intent. Use an observation only when an exact approved fact exists; otherwise do not invent one. Do not stage the reader, prescribe the practice, name an unapproved place or source, or preview later material.",
			"Explain only the first named practice or concept in the exact intent. Add useful detail that follows the opening; do not restart the scene or restate the opening observation.",
			"Advance to a different practice step or contrast explicitly named by the intent. Do not repeat the first step; introduce new reader actions and their observable distinction.",
			"Develop the declared exercise or field-card use. Explain what the reader records, while leaving unknown values blank instead of inventing examples.",
			"Clarify the boundary between direct observation and inference. Do not repeat earlier instructions or supply unsupported sources, places, or times.",
			"Close with one new implication or next action that fulfills this slot. Do not summarize or reintroduce the opening scene.",
		}
	case "front_matter":
		purposes = []string{
			"Orient the reader to this exact front-matter item's purpose and the book's promise.",
			"Explain how the reader should use this item before entering the chapters.",
			"Set the item's limits and give a clear transition into the declared book structure.",
		}
	default:
		purposes = []string{
			"Frame this exact back-matter item's role in supporting the book's reader promise.",
			"Develop the item's distinct reference material without repeating chapter instruction.",
			"Explain how to use or revisit this item and close the declared book structure.",
		}
	}
	plans := make([]BookSectionPartPlan, count)
	for i := range plans {
		purpose := purposes[i%len(purposes)]
		if i >= len(purposes) {
			purpose = fmt.Sprintf("Add one distinct, intent-grounded nuance to this section without repeating earlier parts: %s", purpose)
		}
		if section.Kind == "chapter" && i == count-1 && len(followingIntents) > 0 {
			purpose = "Synthesize this chapter's unique contribution and leave a clear bridge to the next declared slot, " + followingIntents[0] + ", without recapping prior parts."
		}
		plans[i] = BookSectionPartPlan{Index: i + 1, Purpose: purpose, TargetWords: target, MinWords: minPart, MaxWords: maxPart}
	}
	return plans
}

func bookSectionPartPrompt(brief *BookBrief, section Section, plan BookSectionPartPlan, plans []BookSectionPartPlan, previousPartText string, fullMin, fullMax int) string {
	priorPurposes := make([]string, 0, plan.Index-1)
	for _, earlier := range plans {
		if earlier.Index >= plan.Index {
			break
		}
		priorPurposes = append(priorPurposes, earlier.Purpose)
	}
	parts := []string{
		brief.PromptText(),
		fmt.Sprintf("Current manuscript slot: %d of %d. Kind: %s. Exact declared intent: %s.", section.Position+1, len(brief.FrontMatterIntent)+len(brief.ChapterIntent)+len(brief.BackMatterIntent), section.Kind, section.Intent),
		"Slot summary: " + section.Summary,
		fmt.Sprintf("Structured section part %d of %d. Distinct purpose: %s", plan.Index, len(plans), plan.Purpose),
	}
	if len(priorPurposes) > 0 {
		parts = append(parts, "Earlier parts already have these distinct purposes; continue after them without repeating their setup, observations, examples, or actions: "+strings.Join(priorPurposes, " | "))
	}
	if strings.TrimSpace(previousPartText) != "" {
		parts = append(parts, "Continuation context from the immediately preceding part (already-written text; do not repeat or paraphrase it):\n"+strings.TrimSpace(previousPartText))
	}
	parts = append(parts,
		fmt.Sprintf("Write one complete prose part of approximately %d words, between %d and %d words. Return prose only; no heading, plan, summary, filler, or commentary. Continue forward from the immediately preceding part; never restart the scene, repeat the same setup/action, or repeat prior material. Stay within this exact slot and purpose. The combined section must later pass its unchanged %d–%d word gate.", plan.TargetWords, plan.MinWords, plan.MaxWords, fullMin, fullMax),
	)
	return strings.Join(parts, "\n\n")
}

func validateBookChunkCandidate(brief *BookBrief, plans []BookSectionPartPlan, candidates []Candidate) (string, []SectionPartEvidence, error) {
	if len(plans) == 0 || len(candidates) != len(plans) {
		return "", nil, fmt.Errorf("chunk assembly requires exactly %d planned parts; got %d", len(plans), len(candidates))
	}
	parts := make([]SectionPartEvidence, len(candidates))
	var combined strings.Builder
	for i, candidate := range candidates {
		plan := plans[i]
		if !candidate.Eligibility.Eligible {
			return "", nil, fmt.Errorf("part %d candidate is ineligible: %s", plan.Index, candidateReason(&candidate))
		}
		text := strings.TrimSpace(candidate.Text)
		metrics := textmetrics.Analyze(text, nil)
		if metrics.WordCount < plan.MinWords || metrics.WordCount > plan.MaxWords {
			return "", nil, fmt.Errorf("part %d word_count %d outside its declared %d..%d band", plan.Index, metrics.WordCount, plan.MinWords, plan.MaxWords)
		}
		if metrics.DistinctN["n1"] < 0.3 || metrics.SelfRepetition > 0.7 {
			return "", nil, fmt.Errorf("part %d rejected as repetitive or filler-like", plan.Index)
		}
		for j := 0; j < i; j++ {
			if textmetrics.SectionNovelty(text, candidates[j].Text) < 0.25 {
				return "", nil, fmt.Errorf("part %d repeats the substance of part %d", plan.Index, plans[j].Index)
			}
		}
		if violation := bookBriefViolation(brief, text); violation != "" {
			return "", nil, fmt.Errorf("part %d violates the brief constraint: %s", plan.Index, violation)
		}
		if i > 0 && repeatsSetup(text, candidates[:i]) {
			return "", nil, fmt.Errorf("part %d repeats prior scene setup or staging", plan.Index)
		}
		if candidate.ID == "" || candidate.Provenance.Provider == "" || candidate.Provenance.ResolvedModelRef == "" || candidate.Provenance.Locality != "local_only" || candidate.Provenance.OutputTokens <= 0 || candidate.Provenance.InputTokens <= 0 || !isSHA256(candidate.Provenance.PromptSHA256) {
			return "", nil, fmt.Errorf("part %d is missing complete local generation provenance", plan.Index)
		}
		if combined.Len() > 0 {
			combined.WriteString("\n\n")
		}
		combined.WriteString(text)
		parts[i] = SectionPartEvidence{
			Index: plan.Index, Purpose: plan.Purpose, CandidateID: candidate.ID,
			Provider: candidate.Provenance.Provider, Model: candidate.Provenance.ResolvedModelRef,
			Locality: candidate.Provenance.Locality, WordCount: metrics.WordCount,
			InputTokens: candidate.Provenance.InputTokens, OutputTokens: candidate.Provenance.OutputTokens,
			PromptSHA256: candidate.Provenance.PromptSHA256, OutputSHA256: hash([]byte(text)),
		}
	}
	return combined.String(), parts, nil
}

func assembleBookSectionCandidate(brief *BookBrief, section Section, plans []BookSectionPartPlan, parts []Candidate, constraints Constraints, selection *SelectionContext, artifacts []string) (Candidate, error) {
	combined, evidence, err := validateBookChunkCandidate(brief, plans, parts)
	if err != nil {
		return Candidate{}, err
	}
	metrics := textmetrics.Analyze(combined, nil)
	eligibility := gate(metrics, constraints, combined, gateInputs{Selection: selection, Artifacts: artifacts})
	if !eligibility.Eligible {
		return Candidate{}, fmt.Errorf("assembled candidate is ineligible: %s", eligibility.Reason)
	}
	partIDs := make([]string, len(parts))
	promptHashes := make([]string, len(parts))
	var inputTokens, outputTokens int
	var costMicros int64
	provider, model := parts[0].Provenance.Provider, parts[0].Provenance.ResolvedModelRef
	for i, part := range parts {
		partIDs[i] = part.ID
		promptHashes[i] = part.Provenance.PromptSHA256
		inputTokens += part.Provenance.InputTokens
		outputTokens += part.Provenance.OutputTokens
		costMicros += part.Provenance.CostMicros
		if provider != part.Provenance.Provider {
			provider = "mixed"
		}
		if model != part.Provenance.ResolvedModelRef {
			model = "mixed"
		}
	}
	provenance := parts[0].Provenance
	provenance.Strategy = "book_section_chunked"
	provenance.Provider = provider
	provenance.ResolvedModelRef = model
	provenance.PromptSHA256 = hash([]byte(strings.Join(promptHashes, "\n")))
	provenance.InputTokens = inputTokens
	provenance.OutputTokens = outputTokens
	provenance.CostMicros = costMicros
	provenance.ContextSnapshot = &ContextSnapshot{EstimatedTokens: estimateContextTokens(combined)}
	id := uuid.NewString()
	roundID := uuid.NewString()
	return Candidate{
		ID: id, RoundID: roundID, DerivedFrom: partIDs, Text: combined,
		SetIndex: 0, Measurements: metrics,
		SetMeasurements: textmetrics.SetMetrics{CandidateCount: 1, Basis: "composed from persisted local section parts"},
		Provenance:      provenance, Eligibility: eligibility,
		SectionAssembly: &SectionAssembly{SchemaVersion: bookSectionChunkVersion, Kind: section.Kind, Intent: section.Intent, MinWords: constraints.MinWords, MaxWords: constraints.MaxWords, WordCount: metrics.WordCount, Parts: evidence},
	}, nil
}

func bookBriefViolation(brief *BookBrief, text string) string {
	if brief == nil {
		return "missing book brief"
	}
	policy := strings.ToLower(brief.ReaderPromise + " " + brief.AdditionalDirection)
	if (strings.Contains(policy, "species-identification") || strings.Contains(policy, "species identification") || strings.Contains(policy, "species facts")) && speciesAssertionPattern.MatchString(text) {
		return "unsupported species/source identification is prohibited"
	}
	if (strings.Contains(policy, "wellness promise") || strings.Contains(policy, "wellness promises") || strings.Contains(policy, "health promise")) && wellnessClaimPattern.MatchString(text) {
		return "wellness or health promises are prohibited"
	}
	facts := []string(nil)
	if brief.EvidenceBoundary != nil {
		facts = brief.EvidenceBoundary.ApprovedFacts
	}
	if claim := unsupportedConcreteClaim(text, facts); claim != "" {
		return claim
	}
	return ""
}

func unsupportedConcreteClaim(text string, approvedFacts []string) string {
	for _, bounds := range concreteTimePattern.FindAllStringIndex(text, -1) {
		if !supportedByApprovedFactSentence(text, bounds[0], bounds[1], approvedFacts) {
			match := text[bounds[0]:bounds[1]]
			return "unsupported concrete time detail: " + match
		}
	}
	for _, pattern := range []*regexp.Regexp{namedIntersectionPattern, namedStreetPattern} {
		for _, bounds := range pattern.FindAllStringIndex(text, -1) {
			if !supportedByApprovedFactSentence(text, bounds[0], bounds[1], approvedFacts) {
				match := text[bounds[0]:bounds[1]]
				return "unsupported named location detail: " + match
			}
		}
	}
	for _, bounds := range concreteSourcePattern.FindAllStringIndex(text, -1) {
		if !supportedByApprovedFactSentence(text, bounds[0], bounds[1], approvedFacts) {
			match := text[bounds[0]:bounds[1]]
			return "unsupported concrete source example or identity: " + match
		}
	}
	return ""
}

func supportedByApprovedFactSentence(text string, start, end int, approvedFacts []string) bool {
	left := strings.LastIndexAny(text[:start], ".!?\n") + 1
	right := len(text)
	if offset := strings.IndexAny(text[end:], ".!?\n"); offset >= 0 {
		right = end + offset
	}
	claimSentence := normalizeEvidenceText(text[left:right])
	if claimSentence == "" {
		return false
	}
	for _, fact := range approvedFacts {
		if claimSentence == normalizeEvidenceText(fact) {
			return true
		}
	}
	return false
}

func normalizeEvidenceText(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(evidenceNormalizePattern.ReplaceAllString(value, " "))), " ")
}

var setupActionPattern = regexp.MustCompile(`(?i)\b(?:stand still|close your eyes|take a breath|settle|hold still|sit down|open your eyes|look around|begin by|start by|return to|walk to|stop at|listen first)\b`)

func repeatsSetup(text string, previous []Candidate) bool {
	current := openingSetupActions(text)
	if len(current) == 0 {
		return false
	}
	for _, prior := range previous {
		matches := 0
		for action := range current {
			if openingSetupActions(prior.Text)[action] {
				matches++
			}
		}
		if matches >= 1 {
			return true
		}
	}
	return false
}

func openingSetupActions(text string) map[string]bool {
	sentences := strings.SplitN(strings.TrimSpace(text), ".", 3)
	opening := strings.Join(sentences[:minInt(2, len(sentences))], ".")
	actions := map[string]bool{}
	for _, action := range setupActionPattern.FindAllString(opening, -1) {
		actions[strings.ToLower(action)] = true
	}
	return actions
}

func lastSentences(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	parts := regexp.MustCompile(`[.!?]+`).Split(strings.TrimSpace(text), -1)
	if len(parts) > limit {
		parts = parts[len(parts)-limit:]
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return strings.Join(parts, ". ")
}

func (s *Service) produceBookSectionChunked(ctx context.Context, doc Document, profile Profile, section Section, planned OutlineSection, prior []Section, followingIntents []string, totalSections int) (Section, error) {
	if doc.BookBrief == nil || profile.GatewayRole != "write.default" || strings.TrimSpace(profile.Locality) != "local_only" {
		return Section{}, ErrPrivateRouteNotLocal
	}
	target := planned.TargetWords
	if target <= 0 {
		target = defaultTargetSectionWords
	}
	fullMin, fullMax := sectionWordBand(profile, target)
	profile.Constraints.MinWords = fullMin
	profile.Constraints.MaxWords = fullMax
	profile.Constraints.RequiredFormat = sectionRequiredFormat(profile)
	profile.Constraints.MinSectionNovelty = profile.Coherence.MinSectionNovelty
	profile.SelectionPolicy = "take_first"
	profile.Sampler.Kind = samplerDirect
	profile.Sampler.K = 1
	profile.Sampler.MaxOutputTokens = sectionOutputCap(profile, fullMax)
	profile.Budget.MaxOutputTokens = profile.Sampler.MaxOutputTokens
	selection := &SelectionContext{PriorText: s.committedSectionTexts(ctx, prior), TargetWords: target}
	plans := bookSectionPartPlans(section, target, fullMin, followingIntents)
	parts := make([]Candidate, 0, len(plans))
	var sessionID string
	for _, plan := range plans {
		partProfile := profile
		partProfile.Constraints.MinWords = plan.MinWords
		partProfile.Constraints.MaxWords = plan.MaxWords
		partProfile.Constraints.MinSectionNovelty = 0
		partProfile.Composition.TargetSectionWords = plan.TargetWords
		partProfile.Sampler.Kind = samplerDirect
		partProfile.Sampler.K = 1
		partProfile.Sampler.MaxOutputTokens = sectionOutputCap(partProfile, plan.MaxWords)
		partProfile.Budget.MaxOutputTokens = partProfile.Sampler.MaxOutputTokens
		previousPartText := ""
		if len(parts) > 0 {
			previousPartText = lastSentences(parts[len(parts)-1].Text, 2)
		}
		query := bookSectionPartPrompt(doc.BookBrief, section, plan, plans, previousPartText, fullMin, fullMax)
		generated, err := s.generateWithRetry(ctx, GenerateRequest{
			ProfileKey: profile.Key, Query: query, SessionID: sessionID, IncludeCandidates: true, Artifacts: doc.Artifacts,
		}, partProfile, fmt.Sprintf("section %d part %d", section.Position, plan.Index))
		if err != nil {
			return Section{}, err
		}
		sessionID = generated.Session.ID
		candidate := eligibleCandidate(generated)
		if candidate == nil {
			return Section{}, fmt.Errorf("section %d part %d has no eligible candidate: %s", section.Position, plan.Index, candidateReasonWithMeasurements(firstCandidate(generated)))
		}
		parts = append(parts, *candidate)
	}
	assembledCandidate, err := assembleBookSectionCandidate(doc.BookBrief, section, plans, parts, profile.Constraints, selection, doc.Artifacts)
	if err != nil {
		return Section{}, fmt.Errorf("section %d chunk validation failed: %w", section.Position, err)
	}
	partRuns := make([]BookGenerationRun, len(parts))
	for i := range parts {
		e := assembledCandidate.SectionAssembly.Parts[i]
		partRuns[i] = BookGenerationRun{
			Date: s.now().UTC().Format(time.RFC3339), Model: e.Model, Provider: e.Provider,
			PromptSHA256: e.PromptSHA256, OutputSHA256: e.OutputSHA256,
			SectionID: section.ID, CandidateID: e.CandidateID, PartIndex: e.Index,
			Purpose: e.Purpose, WordCount: e.WordCount, InputTokens: e.InputTokens, OutputTokens: e.OutputTokens,
		}
	}
	assemblyPromptHash := assembledCandidate.Provenance.PromptSHA256
	assemblyID := assembledCandidate.ID
	assemblyRoundID := assembledCandidate.RoundID
	model, provider := assembledCandidate.Provenance.ResolvedModelRef, assembledCandidate.Provenance.Provider
	round := Round{ID: assemblyRoundID, SessionID: sessionID, CompositionMode: bookSectionChunkVersion, CandidateIDs: []string{assemblyID}, Strategy: Sampler{Kind: "composed", K: 1}}
	if err := s.saveJSON(ctx, "prose_rounds", round.ID, round); err != nil {
		return Section{}, err
	}
	if err := s.saveJSON(ctx, "prose_candidates", assembledCandidate.ID, assembledCandidate); err != nil {
		return Section{}, err
	}
	var session Session
	if err := s.loadJSON(ctx, "prose_sessions", sessionID, &session); err != nil {
		return Section{}, err
	}
	session.RoundIDs = append(session.RoundIDs, round.ID)
	if err := s.saveJSON(ctx, "prose_sessions", session.ID, session); err != nil {
		return Section{}, err
	}
	section.SessionID = sessionID
	section.PartGenerationRuns = partRuns
	section.GenerationRun = &BookGenerationRun{
		Date: s.now().UTC().Format(time.RFC3339), Model: model, Provider: provider,
		PromptSHA256: assemblyPromptHash, OutputSHA256: hash([]byte(assembledCandidate.Text)),
		SectionID: section.ID, CandidateID: assemblyID, Purpose: "canonical assembled manuscript section",
		WordCount: assembledCandidate.SectionAssembly.WordCount, InputTokens: assembledCandidate.Provenance.InputTokens, OutputTokens: assembledCandidate.Provenance.OutputTokens,
	}
	if err := s.commitSectionCandidate(ctx, section.ID, assemblyID, section); err != nil {
		return Section{}, err
	}
	section.CommittedCandidateID = assemblyID
	section.Context.FullTextSectionRefs = nil
	section.Context.SummarizedSectionRefs = nil
	section.Context.SectionSummaries = nil
	section.Context.EstimatedTokens = 0
	if err := s.saveSection(ctx, section); err != nil {
		return Section{}, err
	}
	return section, nil
}
