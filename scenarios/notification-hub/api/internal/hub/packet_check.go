package hub

import (
	"fmt"
	"net"
	"net/url"
	"path"
	"strings"

	"github.com/vrooli/api-core/operatordecision"
)

// The packet check: advice about how an ask reads to an operator who has none
// of the requester's context, and about explainer pages that will not work in
// the Hub's sealed frame. It is pure and never blocks an ask: every finding is
// a PacketWarning with severity "advice". The rules are documented in
// docs/concepts/DECISION_PACKETS.md ("Packet check", "HTML explainers"), and
// mirror the writing rules in skills/notification-hub/SKILL.md.
//
// The cover rules (internal labels, file paths, question length, big picture,
// reason) are api-core's operatordecision.CheckCover, the same rules every
// decision producer's tests assert, so the Hub and the producers read one set.

// Warning codes of the packet check.
const (
	WarnInternalIDInQuestion       = operatordecision.CoverInternalLabel
	WarnPathInQuestion             = operatordecision.CoverPath
	WarnLongQuestion               = operatordecision.CoverLongQuestion
	WarnMissingBigPicture          = operatordecision.CoverNoBigPicture
	WarnMissingRecommendReason     = operatordecision.CoverNoReason
	WarnLocalContextURL            = "local_context_url"
	WarnUnattachedPath             = "unattached_path"
	WarnExplainerNotHTML           = "explainer_not_html"
	WarnExplainerTooLarge          = "explainer_too_large"
	WarnExplainerNetwork           = "explainer_network"
	WarnExplainerNoViewport        = "explainer_no_viewport"
	WarnExplainerForm              = "explainer_form"
	WarnExplainerUnknownAttachment = "explainer_unknown_attachment"
)

// MaxExplainerBytes is the largest explainer page the Hub shows.
const MaxExplainerBytes = 2 << 20

// PacketAttachment is what the packet check needs to know about one attachment.
type PacketAttachment struct {
	Name string
	// SourcePath is the absolute path on the sender's host, "" when unknown.
	SourcePath string
}

// PacketExplainer is the explainer attachment and its page.
type PacketExplainer struct {
	// Field names the request field of the explainer, for example attachments[1].
	Field string
	Name  string
	// SourcePath and MediaType identify the page's format.
	SourcePath string
	MediaType  string
	// IsFolder is true when the explainer attachment is a folder.
	IsFolder bool
	// Size is the stored size in bytes.
	Size int64
	// HTML is the page, nil when the Hub could not read it. A page larger than
	// MaxExplainerBytes needs only its Size.
	HTML []byte
}

// CheckPacket returns the advice for a normalized ask: the cover text, the
// attachments and the explainer page (nil when the ask has none).
func CheckPacket(spec AskSpec, attachments []PacketAttachment, explainer *PacketExplainer) []PacketWarning {
	var warnings []PacketWarning
	warnings = append(warnings, checkCover(spec, attachments)...)
	var page *element
	if explainer != nil {
		attached := make(map[string]bool, len(attachments))
		for _, attachment := range attachments {
			attached[attachment.Name] = true
		}
		warnings = append(warnings, checkExplainer(*explainer, attached)...)
		if readablePage(*explainer) {
			page = parseElements(string(explainer.HTML))
			restates := checkRestates(spec, *explainer, page)
			warnings = append(warnings, restates...)
			warnings = append(warnings, checkPictures(*explainer, page, len(restates) > 0)...)
		}
	}
	warnings = append(warnings, checkCommandShown(spec, page)...)
	warnings = append(warnings, checkShape(explainer != nil)...)
	return append(warnings, checkPacketPath(spec)...)
}

func advice(code, field, message string) PacketWarning {
	return PacketWarning{Code: code, Field: field, Message: message, Severity: WarningAdvice}
}

func checkCover(spec AskSpec, attachments []PacketAttachment) []PacketWarning {
	cover := operatordecision.Facts{Question: spec.Question, BigPicture: spec.BigPicture, Recommended: spec.Recommended, RecommendationReason: spec.RecommendationReason}
	for _, option := range spec.Options {
		cover.Options = append(cover.Options, operatordecision.Option{Key: option.Key, Label: option.Label})
	}
	var warnings []PacketWarning
	for _, finding := range operatordecision.CheckCover(cover) {
		warnings = append(warnings, advice(finding.Code, finding.Field, finding.Message))
	}
	if isLocalContextURL(spec.ContextURL) {
		warnings = append(warnings, advice(WarnLocalContextURL, "context_url",
			"context_url points at this machine, which the operator's phone cannot open; attach the file instead"))
	}
	if paths := unattachedPaths(spec.TechnicalDetails, attachments); len(paths) > 0 {
		warnings = append(warnings, advice(WarnUnattachedPath, "technical_details",
			fmt.Sprintf("technical_details names a local file that is not attached (%s); the operator cannot open it from the Hub", operatordecision.QuoteList(paths))))
	}
	return warnings
}

// unattachedPaths returns the paths in text that no attachment's source path
// matches: the path itself, or a file inside an attached folder.
func unattachedPaths(text string, attachments []PacketAttachment) []string {
	var missing []string
	for _, candidate := range operatordecision.LocalPaths(text) {
		if !pathIsAttached(candidate, attachments) {
			missing = append(missing, candidate)
		}
	}
	return missing
}

func pathIsAttached(candidate string, attachments []PacketAttachment) bool {
	cleaned := path.Clean(candidate)
	for _, attachment := range attachments {
		source := strings.TrimRight(attachment.SourcePath, "/")
		if source == "" {
			continue
		}
		if cleaned == path.Clean(source) || strings.HasPrefix(cleaned, path.Clean(source)+"/") {
			return true
		}
	}
	return false
}

// --- context url -----------------------------------------------------------

// isLocalContextURL reports a context_url the operator's phone cannot open: a
// path on the requester's machine or an address on its loopback interface.
func isLocalContextURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "file:") || strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "~") || strings.HasPrefix(raw, ".") {
		return true
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || host == "0.0.0.0" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
