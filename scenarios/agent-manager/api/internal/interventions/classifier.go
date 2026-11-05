package interventions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	// DefaultRole walks Winnow, hosted Jev and Clef before local Laya and the
	// classify.fast fallback, so a better decision model is used as soon as it
	// is pulled or its key is provisioned. Background classification favours
	// accuracy over latency.
	DefaultRole = "decide.accurate"
	// MaxBatch is the most states sent in one DecideBatch call.
	MaxBatch = 32
	// DefaultConcurrency bounds DecideBatch calls in flight; the decision
	// models share one GPU with every other local inference.
	DefaultConcurrency = 2
	// defaultTimeout sits above decide.accurate's 120 s role timeout so the
	// gateway reports the reason, while a stalled gateway cannot hang a pass.
	defaultTimeout = 150 * time.Second
	// operatorChars and agentChars bound the state. Winnow-E4B reads the whole
	// corpus unclipped at these limits (0 of 659 train states overflowed; the
	// longest corpus texts are 2,000 and 1,500 chars); Laya's 512-token window
	// needed 1,200/300. The opening of a correction and the end of the agent's
	// last turn carry the signal.
	operatorChars = 2000
	agentChars    = 1000
	// shrinkScale is applied once to a state the primary decision model could
	// not fit, so it is answered by a decision model rather than the fallback.
	shrinkScale = 0.6
)

// Status of one operator message's classification.
type Status string

const (
	StatusClassified   Status = "classified"
	StatusUnclassified Status = "unclassified" // the gateway could not answer; retried later, never read as "no friction"
	StatusInitial      Status = "initial"      // the run's opening message: no agent work preceded it
	StatusDuplicate    Status = "duplicate"    // a replay of a message already recorded (same time and text: a resumed Claude sub-agent file copies earlier entries)
)

// Message is one operator message and the agent text before it.
type Message struct {
	EventID            string
	OperatorText       string
	PrecedingAgentText string
}

// Answer is the decision for one message.
type Answer struct {
	Status        Status
	Category      Category
	Probabilities map[Category]float64 // empty when the method reports no distribution
	Method        string               // DECISION_MODEL or LLM_ENUM
	Provider      string
	Model         string
	ResourceRole  string
	Reason        string // why Status is unclassified
}

// Probability returns the answer's probability for its category, or nil when
// the answering method reports no distribution.
func (a Answer) Probability() *float64 {
	p, ok := a.Probabilities[a.Category]
	if !ok {
		return nil
	}
	return &p
}

// DecideClient is the generated ai-gateway inference client surface used here.
type DecideClient interface {
	DecideBatch(context.Context, *connect.Request[inferencev1.DecideBatchRequest]) (*connect.Response[inferencev1.DecideBatchResponse], error)
}

// Classifier answers the category question for operator messages through
// ai-gateway's DecideBatch. It never names a provider or model: the role's
// candidate walk does.
type Classifier struct {
	Client      DecideClient
	ResolveURL  func(context.Context) (string, error)
	HTTPClient  connect.HTTPClient
	Role        string
	Concurrency int

	mu       sync.Mutex
	resolved DecideClient
}

// NewClassifier resolves ai-gateway through service discovery on first use, so
// an absent gateway degrades classification instead of failing startup.
func NewClassifier() *Classifier {
	return &Classifier{
		HTTPClient: &http.Client{Timeout: defaultTimeout},
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		},
	}
}

// ErrGatewayUnavailable wraps every failure to reach ai-gateway or get any
// answer from it; the answers it accompanies are unclassified.
var ErrGatewayUnavailable = errors.New("ai-gateway decision capability unavailable")

// Classify answers every message, in order. A message the gateway did not
// answer is StatusUnclassified with a reason; the error is non-nil when any
// batch failed outright, so a caller can back off.
func (c *Classifier) Classify(ctx context.Context, messages []Message) ([]Answer, error) {
	answers := make([]Answer, len(messages))
	client, err := c.resolve(ctx)
	if err != nil {
		for i := range answers {
			answers[i] = unclassified(err.Error())
		}
		return answers, fmt.Errorf("%w: %v", ErrGatewayUnavailable, err)
	}
	return answers, c.runBatches(len(messages), MaxBatch, func(start, end int) error {
		return c.classifyBatch(ctx, client, messages[start:end], answers[start:end])
	})
}

// runBatches calls fn for each size-sized range of n items, at most
// Concurrency at a time, and returns the first error.
func (c *Classifier) runBatches(n, size int, fn func(start, end int) error) error {
	concurrency := c.Concurrency
	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
		slots    = make(chan struct{}, concurrency)
	)
	for start := 0; start < n; start += size {
		end := min(start+size, n)
		wg.Add(1)
		slots <- struct{}{}
		go func(start, end int) {
			defer func() { <-slots; wg.Done() }()
			if err := fn(start, end); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(start, end)
	}
	wg.Wait()
	return firstErr
}

// classifyBatch fills out with one answer per message. States the primary
// decision model could not fit are retried once, shrunk.
func (c *Classifier) classifyBatch(ctx context.Context, client DecideClient, messages []Message, out []Answer) error {
	results, err := c.decide(ctx, client, messages, 1)
	if err != nil {
		for i := range out {
			out[i] = unclassified(err.Error())
		}
		return err
	}
	var shrink []int
	for i, result := range results {
		out[i] = answerFrom(result)
		if overflowed(result) {
			shrink = append(shrink, i)
		}
	}
	if len(shrink) == 0 {
		return nil
	}
	retry := make([]Message, len(shrink))
	for j, i := range shrink {
		retry[j] = messages[i]
	}
	retried, err := c.decide(ctx, client, retry, shrinkScale)
	if err != nil {
		return nil // the first answers stand
	}
	for j, i := range shrink {
		if answer := answerFrom(retried[j]); answer.Status == StatusClassified {
			out[i] = answer
		}
	}
	return nil
}

func (c *Classifier) decide(ctx context.Context, client DecideClient, messages []Message, scale float64) ([]*inferencev1.DecideResponse, error) {
	states := make([]string, len(messages))
	for i, message := range messages {
		states[i] = stateJSON(message, scale)
	}
	// Message text never leaves the machine: the local-only profile keeps the
	// role's walk to local decision models, as ClassifySignals does.
	return c.decideStates(ctx, client, questions(), sharedv1.Profile_PROFILE_LOCAL_ONLY, states)
}

// decideStates sends one DecideBatch call: the same questions about each state.
func (c *Classifier) decideStates(ctx context.Context, client DecideClient, qs []*inferencev1.DecisionQuestion, profile sharedv1.Profile, states []string) ([]*inferencev1.DecideResponse, error) {
	items := make([]*inferencev1.DecideBatchItem, len(states))
	for i, state := range states {
		items[i] = &inferencev1.DecideBatchItem{StateJson: state}
	}
	role := c.Role
	if role == "" {
		role = DefaultRole
	}
	response, err := client.DecideBatch(ctx, connect.NewRequest(&inferencev1.DecideBatchRequest{Role: role, Questions: qs, Items: items, Profile: profile}))
	if err != nil {
		return nil, fmt.Errorf("%w: decide-batch: %v", ErrGatewayUnavailable, err)
	}
	results := response.Msg.GetResults()
	if len(results) != len(states) {
		return nil, fmt.Errorf("%w: decide-batch returned %d results for %d states", ErrGatewayUnavailable, len(results), len(states))
	}
	return results, nil
}

// stateJSON is the structured state: the operator message's opening and the
// end of the agent's preceding text, each bounded and scaled.
func stateJSON(message Message, scale float64) string {
	body, _ := json.Marshal(struct {
		PrecedingAgentText string `json:"preceding_agent_text"`
		OperatorMessage    string `json:"operator_message"`
	}{tail(message.PrecedingAgentText, int(agentChars*scale)), head(message.OperatorText, int(operatorChars*scale))})
	return string(body)
}

func head(text string, limit int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}

func tail(text string, limit int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[len(runes)-limit:])
}

// overflowed reports a state that no candidate could read whole, or that only
// the fallback answered because a decision model could not fit it.
func overflowed(result *inferencev1.DecideResponse) bool {
	if result.GetError().GetCode() == inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_CONTEXT_OVERFLOW {
		return true
	}
	if result.GetMethod() != inferencev1.DecisionMethod_DECISION_METHOD_LLM_ENUM {
		return false
	}
	for _, candidate := range result.GetCandidates() {
		if candidate.GetReason() == "context_overflow" {
			return true
		}
	}
	return false
}

func answerFrom(result *inferencev1.DecideResponse) Answer {
	if e := result.GetError(); e != nil && e.GetCode() != inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNSPECIFIED {
		return unclassified(strings.ToLower(strings.TrimPrefix(e.GetCode().String(), "INFERENCE_ERROR_CODE_")) + ": " + e.GetMessage())
	}
	var choice *inferencev1.DecisionAnswer
	for _, answer := range result.GetAnswers() {
		if answer.GetQuestionId() == questionID {
			choice = answer
		}
	}
	category := Category(choice.GetChoice())
	if _, known := spec(category); !known {
		return unclassified(fmt.Sprintf("invalid_answer: %q is not a category", choice.GetChoice()))
	}
	answer := Answer{
		Status: StatusClassified, Category: category,
		Method:   strings.TrimPrefix(result.GetMethod().String(), "DECISION_METHOD_"),
		Provider: result.GetProvider(), Model: result.GetModel(), ResourceRole: result.GetResourceRole(),
	}
	if len(choice.GetProbabilities()) > 0 {
		answer.Probabilities = make(map[Category]float64, len(choice.GetProbabilities()))
		for _, p := range choice.GetProbabilities() {
			answer.Probabilities[Category(p.GetOption())] = p.GetProbability()
		}
	}
	return answer
}

func unclassified(reason string) Answer {
	return Answer{Status: StatusUnclassified, Reason: reason}
}

// resolve builds the client once the gateway resolves; a failed resolution is
// retried on the next call, so a gateway that starts later is picked up.
func (c *Classifier) resolve(ctx context.Context) (DecideClient, error) {
	if c.Client != nil {
		return c.Client, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.resolved != nil {
		return c.resolved, nil
	}
	if c.ResolveURL == nil {
		return nil, errors.New("ai-gateway URL resolver is unset")
	}
	base, err := c.ResolveURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve ai-gateway: %w", err)
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return nil, errors.New("ai-gateway URL is empty")
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	c.resolved = inferenceconnect.NewInferenceServiceClient(httpClient, base)
	return c.resolved, nil
}
