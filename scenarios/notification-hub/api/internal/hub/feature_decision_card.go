package hub

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
)

const (
	WarnFeatureCardInvalid          = "feature_card_invalid"
	WarnFeatureCardIncomplete       = "feature_card_incomplete"
	featureCardReadLimit      int64 = 4 << 20
)

var (
	featureCardImage = regexp.MustCompile(`^data:image/(png|jpeg|webp);base64,([A-Za-z0-9+/=]+)$`)
	featureCardID    = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{0,127}$`)
	featureBuildID   = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

type featureCard struct {
	Schema    string `json:"schema"`
	ID        string `json:"id"`
	FeatureID string `json:"feature_id"`
	Title     string `json:"title"`
	Location  struct {
		Screen    string   `json:"screen"`
		Taps      []string `json:"taps"`
		Component string   `json:"component"`
		Source    string   `json:"source"`
	} `json:"location"`
	Summary  string `json:"summary"`
	Evidence []struct {
		Alt   string `json:"alt"`
		Image string `json:"image"`
	} `json:"evidence"`
	Options []struct {
		Key    string `json:"key"`
		Label  string `json:"label"`
		Effect string `json:"effect"`
	} `json:"options"`
	Recommendation       string `json:"recommendation"`
	RecommendationReason string `json:"recommendation_reason"`
	SubjectKey           string `json:"subject_key"`
	Status               string `json:"status"`
	Answer               string `json:"answer"`
	Terms                []struct {
		Word    string `json:"word"`
		Meaning string `json:"meaning"`
	} `json:"terms"`
	Finding struct {
		Version     string `json:"version"`
		BeforeBuild string `json:"before_build"`
		AfterBuild  string `json:"after_build"`
		Kind        string `json:"kind"`
		Component   string `json:"component"`
		Source      string `json:"source"`
	} `json:"finding"`
}

// checkFeatureDecisionCards validates attached feature decision cards at the
// Hub packet boundary. Ordinary JSON evidence is ignored. V1 cards remain
// readable but receive an explicit warning because they do not bind an answer
// to the exact finding and served build pair; v2 carries that binding.
func checkFeatureDecisionCards(ctx context.Context, spec AskSpec, explainer *PacketExplainer, read BlobReader) []PacketWarning {
	var warnings []PacketWarning
	for index, attachment := range spec.Attachments {
		if attachment.Kind != AttachmentFile || !(strings.HasSuffix(strings.ToLower(attachment.Name), ".json") || strings.EqualFold(attachment.MediaType, "application/json")) {
			continue
		}
		data, err := read(ctx, attachment.SHA256, featureCardReadLimit)
		if err != nil {
			if err == ErrBlobTooLarge {
				warnings = append(warnings, advice(WarnFeatureCardInvalid, fmt.Sprintf("attachments[%d]", index), "the feature decision card exceeds 4 MiB; reduce embedded evidence so the Hub can check the card"))
			}
			continue
		}
		var header struct {
			Schema string `json:"schema"`
		}
		if json.Unmarshal(data, &header) != nil || !strings.HasPrefix(header.Schema, "research-feature-decision/") {
			continue
		}
		field := fmt.Sprintf("attachments[%d]", index)
		var card featureCard
		if err := json.Unmarshal(data, &card); err != nil {
			warnings = append(warnings, advice(WarnFeatureCardInvalid, field, "the feature decision card is not valid JSON; attach the complete card file"))
			continue
		}
		var rawCard map[string]json.RawMessage
		_ = json.Unmarshal(data, &rawCard)
		terms, hasTerms := rawCard["terms"]
		if !hasTerms || strings.TrimSpace(string(terms)) == "null" || !strings.HasPrefix(strings.TrimSpace(string(terms)), "[") {
			warnings = append(warnings, advice(WarnFeatureCardInvalid, field, "the feature decision card must include a terms list, using [] when there are no technical words to define"))
		}
		if problems := validateFeatureCard(card); len(problems) > 0 {
			warnings = append(warnings, advice(WarnFeatureCardInvalid, field, "the feature decision card is incomplete: "+strings.Join(problems, "; ")+"."))
		}
		if card.Schema == "research-feature-decision/v1" {
			warnings = append(warnings, advice(WarnFeatureCardIncomplete, field, "this card does not bind the answer to the exact before/after build and finding version; use research-feature-decision/v2 with finding evidence"))
		} else if card.Schema == "research-feature-decision/v2" {
			if problems := validateFeatureCardPage(card, explainer); len(problems) > 0 {
				warnings = append(warnings, advice(WarnFeatureCardIncomplete, field, "the Hub page does not show the complete feature decision: "+strings.Join(problems, "; ")+"."))
			}
			if problems := validateFeatureCardAsk(card, spec); len(problems) > 0 {
				warnings = append(warnings, advice(WarnFeatureCardIncomplete, field, "the Hub answer is not bound to the feature finding: "+strings.Join(problems, "; ")+"."))
			}
		}
	}
	return warnings
}

func featureCardSubjectKey(card featureCard) string {
	sum := sha256.Sum256([]byte(card.ID + "\x00" + card.FeatureID + "\x00" + card.Finding.Version))
	return "feature-loss:" + hex.EncodeToString(sum[:])
}

func validateFeatureCardAsk(card featureCard, spec AskSpec) []string {
	var out []string
	if card.Status != "pending" {
		out = append(out, "only a pending feature decision card can be sent for an answer")
	}
	if spec.SubjectKey != featureCardSubjectKey(card) {
		out = append(out, "subject_key must bind this exact card and finding version")
	}
	if len(spec.Options) != 2 {
		out = append(out, "ask must offer exactly keep and drop as its answers")
	} else {
		options := map[string]string{}
		for _, option := range spec.Options {
			options[option.Key] = option.Label
		}
		cardOptions := map[string]string{}
		for _, option := range card.Options {
			cardOptions[option.Key] = option.Label
		}
		if options["keep"] == "" || options["drop"] == "" || options["keep"] != cardOptions["keep"] || options["drop"] != cardOptions["drop"] {
			out = append(out, "ask answer labels must match the card's keep and drop choices")
		}
	}
	if spec.Recommended != card.Recommendation {
		out = append(out, "ask recommendation must match the card")
	}
	if strings.TrimSpace(spec.RecommendationReason) != strings.TrimSpace(card.RecommendationReason) {
		out = append(out, "ask must carry the card's recommendation reason")
	}
	if !strings.Contains(strings.ToLower(spec.Question), strings.ToLower(card.Title)) {
		out = append(out, "ask question must name the feature shown on the card")
	}
	return out
}

// validateFeatureCardPage binds the phone-rendered page to the checked card.
// The producer supplies the explainer HTML; the Hub verifies it shows the
// exact finding, both answer actions and the same embedded evidence.
func validateFeatureCardPage(card featureCard, explainer *PacketExplainer) []string {
	var out []string
	if explainer == nil || len(explainer.HTML) == 0 {
		return []string{"attach the rendered decision page as the ask explainer"}
	}
	page := string(explainer.HTML)
	var root map[string]string
	answers := map[string]bool{}
	images := map[string]int{}
	for _, match := range tagPattern.FindAllStringSubmatch(page, -1) {
		attrs := attributes(match[2])
		if _, ok := attrs["data-research-feature-card"]; ok {
			root = attrs
		}
		if strings.EqualFold(match[1], "button") && attrs["data-od-answer"] != "" {
			answers[attrs["data-od-answer"]] = true
		}
		if strings.EqualFold(match[1], "img") && attrs["src"] != "" {
			images[attrs["src"]]++
		}
	}
	if root == nil {
		return []string{"page needs a data-research-feature-card root with the card identity"}
	}
	for key, value := range map[string]string{
		"data-research-feature-card": "v2", "data-card-id": card.ID,
		"data-feature-id": card.FeatureID, "data-finding-version": card.Finding.Version,
		"data-before-build": card.Finding.BeforeBuild, "data-after-build": card.Finding.AfterBuild,
	} {
		if root[key] != value {
			out = append(out, key+" must match the checked card")
		}
	}
	if !answers["keep"] || !answers["drop"] {
		out = append(out, "page must offer both keep and drop buttons that prefill the Hub answer")
	}
	for _, evidence := range card.Evidence {
		if images[evidence.Image] == 0 {
			out = append(out, "page must show every image embedded in the checked card")
		}
	}
	visible := strings.ToLower(html.UnescapeString(tagPattern.ReplaceAllString(page, " ")))
	for _, text := range []string{card.Title, card.Summary, card.Recommendation, card.RecommendationReason} {
		if !strings.Contains(visible, strings.ToLower(text)) {
			out = append(out, "page must show the title, summary and recommendation reason")
			break
		}
	}
	for _, option := range card.Options {
		if !strings.Contains(visible, strings.ToLower(option.Label)) || !strings.Contains(visible, strings.ToLower(option.Effect)) {
			out = append(out, "page must explain both choices and what each one does")
			break
		}
	}
	for _, term := range card.Terms {
		if !strings.Contains(visible, strings.ToLower(term.Word)) || !strings.Contains(visible, strings.ToLower(term.Meaning)) {
			out = append(out, "page must define each technical term in plain language")
			break
		}
	}
	return out
}

func validateFeatureCard(card featureCard) []string {
	var out []string
	add := func(ok bool, message string) {
		if !ok {
			out = append(out, message)
		}
	}
	add(card.Schema == "research-feature-decision/v1" || card.Schema == "research-feature-decision/v2", "schema must be research-feature-decision/v1 or v2")
	add(featureCardID.MatchString(card.ID), "id must use uppercase letters, digits, underscores or hyphens")
	add(strings.TrimSpace(card.FeatureID) != "", "feature_id is required")
	add(strings.TrimSpace(card.Title) != "", "title is required")
	add(strings.TrimSpace(card.Location.Screen) != "", "location.screen must name the screen")
	add(len(card.Location.Taps) > 0, "location.taps must list the taps to reach the feature")
	for _, tap := range card.Location.Taps {
		add(strings.TrimSpace(tap) != "", "location.taps cannot contain blank steps")
	}
	add(strings.TrimSpace(card.Summary) != "", "summary must be a plain-language sentence")
	words := strings.Fields(card.Summary)
	add(len(words) <= 28, "summary must use 28 words or fewer")
	trimmedSummary := strings.TrimSpace(card.Summary)
	add(regexp.MustCompile(`[.!?]$`).MatchString(trimmedSummary), "summary must end with a sentence mark")
	add(len(regexp.MustCompile(`[.!?](?:\s|$)`).FindAllStringIndex(trimmedSummary, -1)) <= 1, "summary must be one sentence")
	if len(card.Evidence) == 0 {
		out = append(out, "at least one embedded before/after or component/crawl image is required")
	}
	for index, evidence := range card.Evidence {
		add(strings.TrimSpace(evidence.Alt) != "", fmt.Sprintf("evidence %d needs plain-language alt text", index+1))
		match := featureCardImage.FindStringSubmatch(evidence.Image)
		if match == nil {
			out = append(out, fmt.Sprintf("evidence %d must embed a PNG, JPEG or WebP image", index+1))
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(match[2])
		if err != nil || len(decoded) == 0 || len(decoded) > 1<<20 {
			out = append(out, fmt.Sprintf("evidence %d image must decode to 1 byte through 1 MiB", index+1))
			continue
		}
		valid := (match[1] == "png" && len(decoded) >= 8 && string(decoded[:8]) == "\x89PNG\r\n\x1a\n") ||
			(match[1] == "jpeg" && len(decoded) >= 3 && string(decoded[:3]) == "\xff\xd8\xff") ||
			(match[1] == "webp" && len(decoded) >= 12 && string(decoded[:4]) == "RIFF" && string(decoded[8:12]) == "WEBP")
		add(valid, fmt.Sprintf("evidence %d image bytes do not match its type", index+1))
	}
	keys := map[string]bool{}
	for _, option := range card.Options {
		keys[option.Key] = true
	}
	add(len(card.Options) == 2 && keys["keep"] && keys["drop"], "options must be exactly keep and drop")
	for _, option := range card.Options {
		add(strings.TrimSpace(option.Label) != "", "both choices need plain-language labels")
		add(strings.TrimSpace(option.Effect) != "", "both choices must explain what will happen")
		if option.Key == "keep" {
			add(containsOneOf(option.Effect, "restore", "retain", "keep"), "keep must say the feature will be restored or retained")
		}
		if option.Key == "drop" {
			add(containsOneOf(option.Effect, "delete", "remove"), "drop must say the code will be deleted or removed")
		}
	}
	add(card.Recommendation == "keep" || card.Recommendation == "drop", "recommendation must be keep or drop")
	add(strings.TrimSpace(card.RecommendationReason) != "", "recommendation_reason must explain the recommended choice")
	add(card.Status == "pending" || card.Status == "resolved", "status must be pending or resolved")
	add(card.Status != "pending" || card.Answer == "", "a pending card cannot claim an answer")
	add(card.Status != "resolved" || card.Answer == "keep" || card.Answer == "drop", "a resolved card must record keep or drop")
	for _, term := range card.Terms {
		add(strings.TrimSpace(term.Word) != "" && strings.TrimSpace(term.Meaning) != "", "each technical term needs a plain-language meaning")
	}
	if card.Schema == "research-feature-decision/v2" {
		add(strings.TrimSpace(card.Location.Component+card.Location.Source) != "" || strings.TrimSpace(card.Finding.Component+card.Finding.Source) != "", "location must name the component or source when available")
		add(featureBuildID.MatchString(card.Finding.Version), "finding.version must be a full sha256 identity")
		add(featureBuildID.MatchString(card.Finding.BeforeBuild) && featureBuildID.MatchString(card.Finding.AfterBuild), "finding must bind full measured before_build and after_build identities")
		add(card.Finding.BeforeBuild != card.Finding.AfterBuild, "before_build and after_build must differ")
		add(strings.TrimSpace(card.Finding.Kind) != "", "finding.kind must name the served feature or component type")
		add(card.SubjectKey == featureCardSubjectKey(card), "subject_key must bind this card and finding version")
	}
	return out
}

func containsOneOf(value string, words ...string) bool {
	value = strings.ToLower(value)
	for _, word := range words {
		if strings.Contains(value, word) {
			return true
		}
	}
	return false
}
