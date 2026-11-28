package operatordecision

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Facts is the domain-neutral description of an operator decision: what the
// operator is asked, the menu of answers, which is recommended, whether acting is
// reversible, and the correlation the resolution carries back verbatim. It is the
// producer side of the notification-hub ask contract, ported from
// goalhome.Decision (its Validate + RequestPayload). The hub reads these fields
// to render the Ask; nothing scenario-specific leaks into the hub.
//
// Facts.Payload() produces the `decision_requested` event payload map; a producer
// wraps it with BuildEvent and publishes it via a Publisher.
type Facts struct {
	// Subject is the short notification title (e.g. "Run abc123 needs review").
	Subject string
	// Question is the full prompt shown to the operator.
	Question string
	// Options is the answer menu; >=2 required. Key is what the handler receives,
	// Label what the operator sees.
	Options []Option
	// Recommended is the option key the producer suggests; optional, and when set
	// it must be one of Options. Leave it empty when there is no honest
	// preference: the operator then sees the options unranked.
	Recommended string
	// RecommendationReason optionally explains the recommendation.
	RecommendationReason string
	// DefaultAnswer, when set, is applied by the hub at the deadline with no answer.
	// It is allowed ONLY when Reversible is true and must be one of Options.
	DefaultAnswer string
	// Reversible declares that acting on the decision can be undone; it is the gate
	// for allowing a silent DefaultAnswer.
	Reversible bool
	// Urgency is "", "low", "normal", "high" or "critical". Empty takes the Hub's
	// default for a decision request, which is normal, the same as the CLI's.
	// Set "high" when work is blocked on the answer right now, and "low" when
	// it is not blocking anything.
	Urgency string
	// Note is optional free text carried to the operator.
	Note string
	// BigPicture explains the decision's purpose and practical consequences.
	BigPicture string
	// TechnicalDetails explains mechanism, evidence, risks and known limits.
	TechnicalDetails string
	// ContextURL optionally deep-links the operator to a review surface. A producer
	// resolves this (e.g. from its web-console base) BEFORE building Facts.
	ContextURL string
	// Deadline, when set, bounds how long the hub waits before applying the default.
	Deadline time.Time
	// Correlation is returned verbatim on notification_hub.ask.resolved.v1, so the
	// resolver can find the durable record the ask belongs to.
	Correlation map[string]string
	// SubjectKey names the situation this decision is about, for example
	// "backup-failure:<destination-id>". With a key, the hub keeps at most ONE
	// open decision per (source, event type, subject key):
	//   - a re-send whose answer menu is unchanged (options, recommended, default,
	//     reversible, urgency) updates the open decision in place, without a new
	//     push; its text and Correlation become the new revision's;
	//   - a re-send with a different answer menu replaces the open decision: the
	//     old one closes as superseded and the new one is delivered;
	//   - the producer closes it with a withdrawal (see Withdrawal) when the
	//     situation clears.
	// A decision the operator dismissed is not re-opened by a re-send with the
	// same terms. Leave it empty for a one-off decision that never changes.
	SubjectKey string
	// ReassertWithin, when positive, tells the hub to close the open decision if
	// the producer has not re-sent it for this long: each re-send is the "still
	// needed" signal. Set it only when the producer re-sends on a cadence (and
	// then set AssertedAt on each re-send). Requires SubjectKey.
	ReassertWithin time.Duration
	// AcceptsContextRequests declares that the producer will answer the operator's
	// "I need more context" question: it receives
	// notification_hub.ask.context_requested.v1 (see ContextResponder) and re-sends
	// the decision under the same SubjectKey with the answer in its text. The hub
	// offers the button only when this is set, so the operator is never left
	// waiting on a sender that cannot reply. Requires SubjectKey.
	AcceptsContextRequests bool
	// Explainer is the decision's page: one self-contained HTML document the Hub
	// shows above the answer buttons, the way an agent's `ask --explainer` page
	// is shown. Build it with Page (page.go) or write your own; it must load
	// nothing from the network (embed images as data: URIs). The question, big
	// picture and options still carry the whole decision, because the push and
	// the decision list never show the page. At most MaxExplainerBytes.
	Explainer string
	// TemplateName and TemplateData identify a Hub-owned page template and its
	// producer supplied JSON data. They are the data driven page contract: the
	// producer does not send markup. Set both fields together.
	TemplateName string
	TemplateData map[string]any
	// Packet is the absolute path of the decision's ready-to-run packet: the
	// brief (work/<ID>/BRIEF.md, see docs/agent-system/LEVELS.md) that says what
	// executes on each answer, so the agent that receives the answer starts a
	// worker from it and re-derives nothing. The Hub shows it to that agent on
	// the ask and in the resolution; the operator never needs it.
	Packet string
	// Questions is the decision's answer form, shown under the options (see
	// Question). A page's Groups block fills the multi_choice question it names;
	// the answers come back on DecisionResolution.Responses.
	Questions []Question
	// AssertedAt is when the producer last observed that the decision is still
	// needed. The hub uses it only to order events for one subject: a request
	// asserted before the subject's last withdrawal is stale and is ignored. A
	// Raiser fills it with the generation's RaisedAt when it is zero.
	AssertedAt time.Time
}

// SubjectKeyRule is the hub's subject-key rule: 1-128 letters, digits, '.', '_',
// ':', '/', '@' or '-', starting with a letter or digit.
var SubjectKeyRule = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/@-]{0,127}$`)

// Reassert window bounds the hub accepts.
const (
	MinReassertWithin = time.Minute
	MaxReassertWithin = 30 * 24 * time.Hour
)

// Option is one answer in a decision's menu. Key is the stable machine key the
// resolution carries; Label is the human-facing text.
type Option struct {
	Key   string
	Label string
}

// CorrelationRaisedByRunID is the well-known Facts.Correlation key a producer
// sets to the agent-manager run id of the agent that raised the decision, so the
// resolution can be delivered back to that exact run automatically. The producer
// obtains the id from the shared agent-provenance layer
// (provenance.FromContext(ctx) → the verified RunID), which attributes a run id
// to ANY coding agent — including ones agent-manager did not create — by
// verifying the agent's identity token against agent-manager. On resolution,
// agent-manager delivers the operator's answer to that run by its current state
// (a parked run is woken; a running interactive or terminal run is continued),
// so the tie-back does NOT require the agent to have parked. A producer that
// does not set it simply gets no tie-back.
const CorrelationRaisedByRunID = "raised_by_run_id"

// CorrelationWebConsoleSessionID is the STABLE delivery handle for a
// web-console-hosted agent. Unlike CorrelationRaisedByRunID (an agent-manager
// run id reached via the ephemeral identity token, which expires on a
// long-lived session), a web-console session id does not expire for the life of
// the session. When set, agent-manager delivers the answer straight to that
// session via its existing web-console client — used as the fallback when no
// usable run id is present (e.g. a multi-day interactive session whose env
// token has aged out). The agent supplies its own WC_WEB_CONSOLE_SESSION_ID.
const CorrelationWebConsoleSessionID = "web_console_session_id"

// CorrelationAgentAwaitKey is an OPTIONAL companion to CorrelationRaisedByRunID.
// When an agent chose to park (zero tokens while waiting) it parks on
// agent-manager producer "operator-decision" with this value as the await key —
// conventionally the decision's own correlation id. agent-manager uses it to
// confirm a parked run is parked on THIS decision before waking it (so a run
// parked on unrelated work is never disturbed). Parking is optional; without it
// the run is still reached via CorrelationRaisedByRunID while it runs.
const CorrelationAgentAwaitKey = "agent_await_key"

// Facts limits mirror the notification-hub ask contract, so a decision this
// package accepts is never refused by the hub. They match goalhome's Decision
// limits.
const (
	MaxQuestion                   = 600
	MaxReason                     = 300
	MaxOptions                    = 8
	MaxOptionLabel                = 200
	MaxBigPictureCodePoints       = 4_000
	MaxTechnicalDetailsCodePoints = 12_000
	// MaxExplainerBytes keeps a decision event, page included, under the 1 MiB
	// request limit of vrooli-events and of the Hub's event intake. It bounds the
	// page as JSON encodes it (each '<', '>' and '&' becomes six bytes).
	MaxExplainerBytes = 720 << 10
	// MaxTemplateDataBytes bounds producer supplied JSON while leaving room for
	// the Hub's decision envelope and event metadata in its 1 MiB intake limit.
	MaxTemplateDataBytes = 256 << 10
	// MaxPacketPath bounds Facts.Packet, as the Hub does.
	MaxPacketPath = 1024
)

var (
	factsKeyRule = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)
	factsUrgency = map[string]bool{"": true, "low": true, "normal": true, "high": true, "critical": true}
)

// Validate checks Facts against the notification-hub ask contract: a non-empty
// bounded question, 2-8 options with unique valid keys, a recommended option (if
// any) that exists, and a default that is allowed only for a reversible decision and is one
// of the options. It ports goalhome.Decision.Validate, minus the decision-id rule
// (correlation here is generic, not a goal-home-scoped id).
func (f Facts) Validate() error {
	switch {
	case strings.TrimSpace(f.Question) == "" || len(f.Question) > MaxQuestion:
		return fmt.Errorf("the question is required and at most %d bytes", MaxQuestion)
	case len(f.Options) < 2 || len(f.Options) > MaxOptions:
		return fmt.Errorf("a decision needs 2-%d options", MaxOptions)
	case len(f.RecommendationReason) > MaxReason:
		return fmt.Errorf("the recommendation reason is at most %d bytes", MaxReason)
	case !utf8.ValidString(f.BigPicture) || utf8.RuneCountInString(f.BigPicture) > MaxBigPictureCodePoints:
		return fmt.Errorf("big_picture must be valid UTF-8 and at most %d Unicode code points", MaxBigPictureCodePoints)
	case !utf8.ValidString(f.TechnicalDetails) || utf8.RuneCountInString(f.TechnicalDetails) > MaxTechnicalDetailsCodePoints:
		return fmt.Errorf("technical_details must be valid UTF-8 and at most %d Unicode code points", MaxTechnicalDetailsCodePoints)
	case !utf8.ValidString(f.Explainer) || EncodedLen(f.Explainer) > MaxExplainerBytes:
		return fmt.Errorf("the explainer page must be valid UTF-8 and at most %d bytes; put large images in fewer, smaller files", MaxExplainerBytes)
	case (f.TemplateName == "") != (f.TemplateData == nil):
		return fmt.Errorf("template_name and template_data must be set together")
	case len(f.TemplateName) > 80:
		return fmt.Errorf("template_name exceeds 80 bytes")
	case f.Packet != "" && PacketPathProblem(f.Packet) != "":
		return fmt.Errorf("packet %s", PacketPathProblem(f.Packet))
	case !factsUrgency[f.Urgency]:
		return fmt.Errorf("urgency must be low, normal, high or critical")
	}
	if f.TemplateData != nil {
		encoded, err := json.Marshal(f.TemplateData)
		if err != nil {
			return fmt.Errorf("template_data must be JSON encodable: %w", err)
		}
		if len(encoded) > MaxTemplateDataBytes {
			return fmt.Errorf("template_data exceeds %d bytes", MaxTemplateDataBytes)
		}
	}
	seen := map[string]bool{}
	for _, option := range f.Options {
		if !factsKeyRule.MatchString(option.Key) {
			return fmt.Errorf("option key %q must be 1-64 letters, digits, '.', '_', ':' or '-'", option.Key)
		}
		if option.Key == "other" {
			return fmt.Errorf("option key %q is reserved for the recipient's free-form answer", option.Key)
		}
		if seen[option.Key] {
			return fmt.Errorf("option key %q is repeated", option.Key)
		}
		if len(option.Label) > MaxOptionLabel {
			return fmt.Errorf("option %q label exceeds %d bytes", option.Key, MaxOptionLabel)
		}
		seen[option.Key] = true
	}
	if f.Recommended != "" && !seen[f.Recommended] {
		return fmt.Errorf("the recommended option %q is not one of the options", f.Recommended)
	}
	if f.DefaultAnswer != "" {
		if !seen[f.DefaultAnswer] {
			return fmt.Errorf("the default %q is not one of the options", f.DefaultAnswer)
		}
		if !f.Reversible {
			return fmt.Errorf("a default is allowed only for a reversible decision")
		}
	}
	if err := validateContextURL(f.ContextURL); err != nil {
		return err
	}
	if f.SubjectKey != "" && !SubjectKeyRule.MatchString(f.SubjectKey) {
		return fmt.Errorf("subject_key %q must be 1-128 letters, digits, '.', '_', ':', '/', '@' or '-'", f.SubjectKey)
	}
	if err := validateQuestions(f.Questions); err != nil {
		return err
	}
	if f.AcceptsContextRequests && f.SubjectKey == "" {
		return fmt.Errorf("accepts_context_requests needs a subject_key: the answer is a re-send under the same key")
	}
	if f.ReassertWithin != 0 {
		if f.SubjectKey == "" {
			return fmt.Errorf("reassert_within needs a subject_key")
		}
		if f.ReassertWithin < MinReassertWithin || f.ReassertWithin > MaxReassertWithin {
			return fmt.Errorf("reassert_within must be between %s and %s", MinReassertWithin, MaxReassertWithin)
		}
	}
	return nil
}

// PacketPathProblem says what is wrong with a packet path, or "" when it is a
// usable one: absolute, one line, valid UTF-8, at most MaxPacketPath bytes. The
// Hub applies the same rule to every ask.
func PacketPathProblem(path string) string {
	switch {
	case !utf8.ValidString(path) || len(path) > MaxPacketPath:
		return fmt.Sprintf("must be valid UTF-8 and at most %d bytes", MaxPacketPath)
	case !strings.HasPrefix(path, "/"):
		return "must be an absolute path (the brief the executing agent reads, for example /home/me/.vrooli/research/topics/t/work/ID/BRIEF.md)"
	case strings.ContainsAny(path, "\x00\n\r"):
		return "must be one line"
	}
	return ""
}

// validateContextURL refuses a context_url whose scheme could execute or embed
// code when rendered as an operator-clickable href. Only an empty value, an
// http(s) absolute URL, or a root-relative path ("/...") is allowed; a
// javascript:, data:, vbscript:, file:, etc. URL is rejected — otherwise a
// producer-supplied link flows unmodified to the inbox and becomes an XSS /
// phishing vector.
func validateContextURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	// A root-relative path is safe (same-origin, no scheme). Reject the
	// scheme-relative "//host" form — it inherits the page scheme and points
	// off-origin, which is the phishing case a bare "/path" avoids.
	if strings.HasPrefix(raw, "/") {
		if strings.HasPrefix(raw, "//") {
			return fmt.Errorf("context_url %q must be http(s) or a root-relative path, not a scheme-relative URL", raw)
		}
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("context_url %q is not a valid URL: %w", raw, err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("context_url %q must be empty, http(s), or a root-relative path (scheme %q is not allowed)", raw, parsed.Scheme)
	}
}

// Label returns an option's label, or the key when the option has none. It
// mirrors goalhome.Decision.Label.
func (f Facts) Label(key string) string {
	for _, option := range f.Options {
		if option.Key == key && option.Label != "" {
			return option.Label
		}
	}
	return key
}

// Payload is the decision_requested event payload: facts only, the shape
// notification-hub's decision intake reads. Correlation comes back verbatim on
// notification_hub.ask.resolved.v1. It ports goalhome.Decision.RequestPayload;
// the Subject is a field here rather than a separate argument.
func (f Facts) Payload() map[string]any {
	options := make([]any, 0, len(f.Options))
	for _, option := range f.Options {
		options = append(options, map[string]any{"key": option.Key, "label": f.Label(option.Key)})
	}
	payload := map[string]any{
		"subject":     f.Subject,
		"question":    f.Question,
		"options":     options,
		"recommended": f.Recommended,
		"reversible":  f.Reversible,
		"urgency":     f.Urgency,
	}
	// recommendation_reason, default_answer, context_url and note are omitted when
	// empty so a decision that carries none of them (a run approval, an
	// authorization ask) emits the same bytes it did before this substrate existed
	// — no silent default_answer and no empty context_url key. A producer that sets
	// them gets them verbatim. notification-hub tolerates either absence or an empty
	// string, so a producer that WANTS the always-present shape (the effort decision
	// renderer in goalhome) keeps building its own payload.
	if f.RecommendationReason != "" {
		payload["recommendation_reason"] = f.RecommendationReason
	}
	if f.DefaultAnswer != "" {
		payload["default_answer"] = f.DefaultAnswer
	}
	if f.ContextURL != "" {
		payload["context_url"] = f.ContextURL
	}
	if f.Note != "" {
		payload["note"] = f.Note
	}
	if f.BigPicture != "" {
		payload["big_picture"] = f.BigPicture
	}
	if f.TechnicalDetails != "" {
		payload["technical_details"] = f.TechnicalDetails
	}
	if !f.Deadline.IsZero() {
		payload["deadline"] = f.Deadline.UTC().Format(time.RFC3339)
	}
	// subject_key, reassert_within_seconds and asserted_at are omitted when unset
	// so a producer without a subject emits the bytes it always did.
	if f.SubjectKey != "" {
		payload["subject_key"] = f.SubjectKey
	}
	if f.ReassertWithin > 0 {
		payload["reassert_within_seconds"] = int64(f.ReassertWithin / time.Second)
	}
	if f.AcceptsContextRequests {
		payload["accepts_context_requests"] = true
	}
	if f.Explainer != "" {
		payload["explainer_html"] = f.Explainer
	}
	if f.TemplateName != "" {
		payload["template_name"] = f.TemplateName
		payload["template_data"] = f.TemplateData
	}
	if f.Packet != "" {
		payload["packet"] = f.Packet
	}
	if len(f.Questions) > 0 {
		payload["questions"] = questionsPayload(f.Questions)
	}
	if !f.AssertedAt.IsZero() {
		payload["asserted_at"] = f.AssertedAt.UTC().Format(time.RFC3339Nano)
	}
	if len(f.Correlation) > 0 {
		values := make(map[string]any, len(f.Correlation))
		for key, value := range f.Correlation {
			values[key] = value
		}
		payload["correlation"] = values
	}
	return payload
}
