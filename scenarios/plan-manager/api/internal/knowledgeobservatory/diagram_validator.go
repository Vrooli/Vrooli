package knowledgeobservatory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

const (
	procedure      = "/knowledge_observatory.v1.KnowledgeObservatoryService/ValidateMarkdownDiagrams"
	defaultTimeout = 3 * time.Second
)

type Client struct {
	resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	httpClient *http.Client
	timeout    time.Duration
}
type Request struct {
	Content string
	Source  string
}
type Finding struct {
	Code, Message string
	Line          int
}
type Result struct {
	Findings   []Finding
	Unverified bool
}

// Adapter mirrors clihealth.Adapter: the transport package stays independent of
// authoring while callers map their own request/result types at the boundary.
type Adapter[DomainRequest any, DomainResult any] struct {
	client     Client
	toRequest  func(DomainRequest) Request
	fromResult func(Result) DomainResult
}

func NewClient() Client {
	return Client{resolver: discovery.NewResolver(discovery.ResolverConfig{}), timeout: defaultTimeout}
}

func NewAdapter[DomainRequest any, DomainResult any](toRequest func(DomainRequest) Request, fromResult func(Result) DomainResult) Adapter[DomainRequest, DomainResult] {
	return Adapter[DomainRequest, DomainResult]{client: NewClient(), toRequest: toRequest, fromResult: fromResult}
}

func (a Adapter[DomainRequest, DomainResult]) ValidateMarkdownDiagrams(ctx context.Context, request DomainRequest) (DomainResult, error) {
	result, err := a.client.ValidateMarkdownDiagrams(ctx, a.toRequest(request))
	if err != nil {
		var zero DomainResult
		return zero, err
	}
	return a.fromResult(result), nil
}

func (c Client) ValidateMarkdownDiagrams(ctx context.Context, input Request) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	base, err := c.resolver.ResolveScenarioURLDefault(ctx, "knowledge-observatory")
	if err != nil {
		return Result{}, fmt.Errorf("resolve knowledge-observatory URL: %w", err)
	}
	body, err := json.Marshal(map[string]string{"content": input.Content, "sourceLabel": input.Source})
	if err != nil {
		return Result{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+procedure, bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	client := c.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var decoded struct {
		Findings []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Line    int    `json:"line"`
		} `json:"findings"`
		Unverified bool `json:"unverified"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return Result{}, err
	}
	out := Result{Unverified: decoded.Unverified}
	for _, f := range decoded.Findings {
		out.Findings = append(out.Findings, Finding{Code: f.Code, Message: f.Message, Line: f.Line})
	}
	return out, nil
}
