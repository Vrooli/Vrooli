package askreader

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/provenance"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
)

// Ask is the Hub's own record of an ask, as the scenario that raised it reads it
// back. It carries facts only: what the Hub says is authoritative, and a
// scenario applies an operator's answer from here, never from an event body that
// anyone able to publish an event could have written.
type Ask struct {
	ID string
	// Question is what the Hub asked the operator.
	Question string
	// State is pending, escalated, answered, defaulted, expired, withdrawn or dismissed.
	State                    string
	Answer                   string
	AnswerLabel              string
	Note                     string
	AnsweredBy               string
	AnsweredByIdentityKind   string
	AnsweredByIdentitySource string
	AnsweredAt               string
	ResolvedAt               string
	Reason                   string
	Late                     bool
	// Source is the scenario that raised the ask; SourceEventType the event it
	// raised it with; RequesterRef that event's id; SubjectKey its situation key.
	Source            string
	SourceEventType   string
	RequesterRef      string
	SubjectKey        string
	SupersededByAskID string
	// Correlation is the producer's own map, as it raised the ask.
	Correlation map[string]string
	// ContextRequests are the operator's questions about the ask.
	ContextRequests []ContextRequest
	// Responses are the operator's answers to the ask's form questions
	// (operatordecision.Facts.Questions); AnswerSummary is the whole answer as
	// one text.
	Responses     []Response
	AnswerSummary string
	// Recurrence is the Hub's exact source-and-subject history. Count includes
	// this ask; previous answers exclude it.
	Recurrence SubjectRecurrence
}

// SubjectRecurrence is the Hub's history for one exact source and subject key.
type SubjectRecurrence struct {
	Count           int
	Dates           []string
	PreviousAnswers []HistoricalAnswer
}

// HistoricalAnswer is a prior answer on the same source and subject key.
type HistoricalAnswer struct {
	Answer     string
	Note       string
	AnsweredAt string
}

// Response is the operator's answer to one form question of the ask.
type Response struct {
	QuestionID string           `json:"question_id"`
	Selected   []string         `json:"selected,omitempty"`
	Text       string           `json:"text,omitempty"`
	Sections   []SectionVerdict `json:"sections,omitempty"`
}

// SectionVerdict is the operator's verdict on one section of a review.
type SectionVerdict struct {
	SectionKey string `json:"section_key"`
	Verdict    string `json:"verdict"`
	Note       string `json:"note,omitempty"`
}

// ContextRequest is one operator question about an open ask.
type ContextRequest struct {
	ID          string
	Question    string
	RequestedAt string
}

var (
	// ErrAskNotFound means the Hub holds no such ask for this reader: it does not
	// exist, or its source is another scenario's. The Hub reports both the same way.
	ErrAskNotFound = errors.New("the Hub has no such ask for this reader")
	// ErrHubUnavailable means the ask could not be read now: the Hub or the
	// authenticator is unreachable or refused the credential. Reading again later
	// may succeed; nothing about the ask is known.
	ErrHubUnavailable = errors.New("the Hub could not be read")
)

// BaseURL resolves where Notification Hub's API is.
type BaseURL func(context.Context) (string, error)

// Reader reads asks at Notification Hub as the source's ask-reader principal.
type Reader struct {
	tokens       *Tokens
	tokenOptions []Option
	baseURL      BaseURL
	http         *http.Client
	testMode     bool
}

// ReaderOption customizes a Reader.
type ReaderOption func(*Reader)

// WithTokenOptions customizes the reader's token client, for example to replace
// the exchange at scenario-authenticator.
func WithTokenOptions(options ...Option) ReaderOption {
	return func(r *Reader) { r.tokenOptions = append(r.tokenOptions, options...) }
}

// WithBaseURL replaces discovery of the Hub, for a fixed address.
func WithBaseURL(base BaseURL) ReaderOption { return func(r *Reader) { r.baseURL = base } }

// WithHTTPClient replaces the HTTP client.
func WithHTTPClient(client *http.Client) ReaderOption { return func(r *Reader) { r.http = client } }

// WithTestMode opts this reader into the Hub's isolated test-pool routing.
// It is effective only when the Hub has an active test lease; ordinary Hub
// requests ignore the header. Production readers must leave it disabled.
func WithTestMode() ReaderOption { return func(r *Reader) { r.testMode = true } }

// NewReader builds the reader for the scenario that raised asks as source, with
// the credential the operator issues for its ask-reader principal. The Hub is
// found through scenario discovery unless WithBaseURL says otherwise.
//
// The credential is read when an ask is read, never when the reader is built,
// and a missing one is not remembered: a scenario that started before the
// operator issued the credential reads its next ask verified, with no restart.
func NewReader(source string, credential CredentialSource, options ...ReaderOption) *Reader {
	r := &Reader{
		baseURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "notification-hub")
		},
		http: &http.Client{Timeout: 10 * time.Second},
	}
	for _, option := range options {
		option(r)
	}
	r.tokens = NewTokens(source, credential, r.tokenOptions...)
	return r
}

// ReadAsk reads one ask. ErrNotIssued (see Tokens) means the scenario has no
// ask-reader credential yet, the operator's one command to issue it is named in
// the error; ErrAskNotFound means the Hub has no such ask for this scenario;
// ErrHubUnavailable means try again later.
func (r *Reader) ReadAsk(ctx context.Context, askID string) (Ask, error) {
	askID = strings.TrimSpace(askID)
	if r == nil || r.tokens == nil || askID == "" {
		return Ask{}, fmt.Errorf("%w: no ask to read", ErrAskNotFound)
	}
	return r.readAsk(ctx, &conversationv1.GetAskRequest{AskId: askID}, askID, "")
}

// ReadAskBySubjectKey reads the current ask for an exact event-sourced subject
// raised by this reader's source. The Hub enforces that source boundary.
func (r *Reader) ReadAskBySubjectKey(ctx context.Context, subjectKey string) (Ask, error) {
	subjectKey = strings.TrimSpace(subjectKey)
	if r == nil || r.tokens == nil || subjectKey == "" {
		return Ask{}, fmt.Errorf("%w: no subject key to read", ErrAskNotFound)
	}
	return r.readAsk(ctx, &conversationv1.GetAskRequest{SubjectKey: subjectKey}, "", subjectKey)
}

func (r *Reader) readAsk(ctx context.Context, request *conversationv1.GetAskRequest, expectedID, expectedSubjectKey string) (Ask, error) {
	// The read is the scenario's own, under its ask-reader token. When it runs
	// inside a request an agent run made, that run's identity must not ride
	// along: the Hub resolves an agent identity first and would then show only
	// that run's own asks.
	ctx = provenance.WithoutCallerIdentity(ctx)
	token, err := r.tokens.Token(ctx)
	switch {
	case errors.Is(err, ErrNotIssued):
		return Ask{}, err
	case err != nil:
		return Ask{}, fmt.Errorf("%w: %w", ErrHubUnavailable, err)
	}
	base, err := r.baseURL(ctx)
	if err != nil || strings.TrimSpace(base) == "" {
		return Ask{}, fmt.Errorf("%w: find notification-hub: %v", ErrHubUnavailable, err)
	}
	client := conversationconnect.NewConversationsServiceClient(r.http, strings.TrimRight(base, "/"))
	req := connect.NewRequest(request)
	req.Header().Set("Authorization", "Bearer "+token)
	if r.testMode {
		req.Header().Set("X-Vrooli-Test-Mode", "1")
	}
	response, err := client.GetAsk(ctx, req)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return Ask{}, fmt.Errorf("%w: %v", ErrAskNotFound, err)
		}
		return Ask{}, fmt.Errorf("%w: %v", ErrHubUnavailable, err)
	}
	ask := response.Msg.GetAsk()
	if expectedID != "" && ask.GetId() != expectedID {
		return Ask{}, fmt.Errorf("%w: the Hub answered for ask %q, not %q", ErrAskNotFound, ask.GetId(), expectedID)
	}
	if expectedSubjectKey != "" && ask.GetSubjectKey() != expectedSubjectKey {
		return Ask{}, fmt.Errorf("%w: the Hub answered for subject %q, not the requested subject", ErrAskNotFound, ask.GetSubjectKey())
	}
	return AskFromProto(ask), nil
}

// AskFromProto reads the Hub's wire form of an ask.
func AskFromProto(ask *conversationv1.Ask) Ask {
	out := Ask{
		ID: ask.GetId(), Question: ask.GetQuestion(), State: ask.GetState(), Answer: ask.GetAnswer(), AnswerLabel: ask.GetAnswerLabel(), Note: ask.GetNote(),
		AnsweredBy: ask.GetAnsweredBy(), AnsweredByIdentityKind: ask.GetAnsweredByIdentityKind(), AnsweredByIdentitySource: ask.GetAnsweredByIdentitySource(),
		AnsweredAt: ask.GetAnsweredAt(), ResolvedAt: ask.GetResolvedAt(), Reason: ask.GetReason(), Late: ask.GetLate(),
		Source: ask.GetSource(), SourceEventType: ask.GetSourceEventType(), RequesterRef: ask.GetRequesterRef(), SubjectKey: ask.GetSubjectKey(),
		SupersededByAskID: ask.GetSupersededByAskId(), Correlation: maps.Clone(ask.GetCorrelation()),
	}
	out.AnswerSummary = ask.GetAnswerSummary()
	if recurrence := ask.GetRecurrence(); recurrence != nil {
		out.Recurrence.Count = int(recurrence.GetCount())
		out.Recurrence.Dates = slices.Clone(recurrence.GetDates())
		for _, previous := range recurrence.GetPreviousAnswers() {
			out.Recurrence.PreviousAnswers = append(out.Recurrence.PreviousAnswers, HistoricalAnswer{Answer: previous.GetAnswer(), Note: previous.GetNote(), AnsweredAt: previous.GetAnsweredAt()})
		}
	}
	for _, response := range ask.GetResponses() {
		read := Response{QuestionID: response.GetQuestionId(), Selected: slices.Clone(response.GetSelected()), Text: response.GetText()}
		for _, verdict := range response.GetSections() {
			read.Sections = append(read.Sections, SectionVerdict{SectionKey: verdict.GetSectionKey(), Verdict: verdict.GetVerdict(), Note: verdict.GetNote()})
		}
		out.Responses = append(out.Responses, read)
	}
	for _, request := range ask.GetContextRequests() {
		out.ContextRequests = append(out.ContextRequests, ContextRequest{ID: request.GetId(), Question: request.GetQuestion(), RequestedAt: request.GetRequestedAt()})
	}
	return out
}
