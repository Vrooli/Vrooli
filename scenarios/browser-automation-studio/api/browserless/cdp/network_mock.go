package cdp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
)

// networkMockRule captures the normalized interception rule for a workflow node.
type networkMockRule struct {
	id             string
	urlPattern     string
	matcher        *regexp.Regexp
	patternIsRegex bool
	method         string
	mockType       string
	statusCode     int64
	headers        []*fetch.HeaderEntry
	headerMap      map[string]string
	body           []byte
	delay          time.Duration
	abortReason    network.ErrorReason
}

// ExecuteNetworkMock registers an interception rule with the active session.
func (s *Session) ExecuteNetworkMock(ctx context.Context, params runtime.InstructionParam) (*StepResult, error) {
	start := time.Now()
	debug := map[string]any{
		"pattern":  params.NetworkURLPattern,
		"mockType": params.NetworkMockType,
	}
	if params.NetworkMethod != "" {
		debug["method"] = params.NetworkMethod
	}
	if params.NetworkDelayMs > 0 {
		debug["delayMs"] = params.NetworkDelayMs
	}

	result := &StepResult{DebugContext: debug}

	rule, err := buildNetworkMockRule(params)
	if err != nil {
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if err := s.registerNetworkMockRule(rule); err != nil {
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	result.DebugContext["ruleId"] = rule.id
	if len(rule.headerMap) > 0 {
		result.DebugContext["headers"] = rule.headerMap
	}
	if len(rule.body) > 0 {
		result.DebugContext["bodyPreview"] = previewMockBody(rule.body)
	}

	// Capture current page context for traceability.
	_ = chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

func buildNetworkMockRule(params runtime.InstructionParam) (*networkMockRule, error) {
	pattern := strings.TrimSpace(params.NetworkURLPattern)
	if pattern == "" {
		return nil, fmt.Errorf("networkMock requires urlPattern")
	}

	matcher, isRegex, err := compileNetworkPattern(pattern)
	if err != nil {
		return nil, err
	}

	body, err := encodeNetworkMockBody(params.NetworkBody)
	if err != nil {
		return nil, err
	}

	headers := buildHeaderEntries(params.NetworkHeaders)
	headerMap := cloneHeaderMap(params.NetworkHeaders)
	method := strings.ToUpper(strings.TrimSpace(params.NetworkMethod))
	status := params.NetworkStatusCode
	if status <= 0 {
		status = 200
	}
	abortReason := network.ErrorReason(params.NetworkAbortReason)
	if abortReason == "" {
		abortReason = network.ErrorReasonFailed
	}

	return &networkMockRule{
		id:             uuid.NewString(),
		urlPattern:     pattern,
		matcher:        matcher,
		patternIsRegex: isRegex,
		method:         method,
		mockType:       params.NetworkMockType,
		statusCode:     int64(status),
		headers:        headers,
		headerMap:      headerMap,
		body:           body,
		delay:          time.Duration(params.NetworkDelayMs) * time.Millisecond,
		abortReason:    abortReason,
	}, nil
}

func (s *Session) registerNetworkMockRule(rule *networkMockRule) error {
	s.networkMockMu.Lock()
	s.networkMocks = append(s.networkMocks, rule)
	s.networkMockMu.Unlock()

	if err := s.syncNetworkInterceptionAcrossTargets(); err != nil {
		// Roll back when interception cannot be enabled so workflows fail fast.
		s.networkMockMu.Lock()
		if len(s.networkMocks) > 0 {
			s.networkMocks = s.networkMocks[:len(s.networkMocks)-1]
		}
		s.networkMockMu.Unlock()
		return fmt.Errorf("failed to enable network interception: %w", err)
	}

	if s.log != nil {
		s.log.WithField("pattern", rule.urlPattern).
			WithField("mockType", rule.mockType).
			Debug("registered network mock rule")
	}

	return nil
}

func (s *Session) hasNetworkMocks() bool {
	s.networkMockMu.RLock()
	defer s.networkMockMu.RUnlock()
	return len(s.networkMocks) > 0
}

func (s *Session) fetchPatternsSnapshot() ([]*fetch.RequestPattern, bool) {
	s.networkMockMu.RLock()
	defer s.networkMockMu.RUnlock()
	if len(s.networkMocks) == 0 {
		return nil, false
	}

	needsCatchAll := false
	for _, rule := range s.networkMocks {
		if rule != nil && rule.patternIsRegex {
			needsCatchAll = true
			break
		}
	}

	if needsCatchAll {
		return []*fetch.RequestPattern{{
			URLPattern:   "*",
			RequestStage: fetch.RequestStageRequest,
		}}, true
	}

	seen := make(map[string]struct{}, len(s.networkMocks))
	patterns := make([]*fetch.RequestPattern, 0, len(s.networkMocks))
	for _, rule := range s.networkMocks {
		if rule == nil || rule.urlPattern == "" {
			continue
		}
		if _, exists := seen[rule.urlPattern]; exists {
			continue
		}
		seen[rule.urlPattern] = struct{}{}
		patterns = append(patterns, &fetch.RequestPattern{
			URLPattern:   rule.urlPattern,
			RequestStage: fetch.RequestStageRequest,
		})
	}

	if len(patterns) == 0 {
		patterns = []*fetch.RequestPattern{{
			URLPattern:   "*",
			RequestStage: fetch.RequestStageRequest,
		}}
	}

	return patterns, true
}

func (s *Session) syncNetworkInterceptionAcrossTargets() error {
	patterns, hasRules := s.fetchPatternsSnapshot()
	if !hasRules {
		return nil
	}

	contexts := s.snapshotTargetContexts()
	var lastErr error
	for _, targetCtx := range contexts {
		if targetCtx == nil {
			continue
		}
		if err := s.applyFetchPatterns(targetCtx, patterns); err != nil {
			lastErr = err
			if s.log != nil {
				s.log.WithError(err).Warn("failed to sync network interception for target")
			}
		}
	}
	return lastErr
}

func (s *Session) ensureNetworkMocksForContext(ctx context.Context) {
	patterns, hasRules := s.fetchPatternsSnapshot()
	if !hasRules || ctx == nil {
		return
	}
	if err := s.applyFetchPatterns(ctx, patterns); err != nil && s.log != nil {
		s.log.WithError(err).Warn("failed to enable network mocks for new context")
	}
}

func (s *Session) applyFetchPatterns(ctx context.Context, patterns []*fetch.RequestPattern) error {
	return chromedp.Run(ctx, chromedp.ActionFunc(func(runCtx context.Context) error {
		if len(patterns) == 0 {
			return fetch.Disable().Do(runCtx)
		}
		req := fetch.Enable()
		req.Patterns = patterns
		req.HandleAuthRequests = false
		return req.Do(runCtx)
	}))
}

func (s *Session) processFetchIntercept(ctx context.Context, ev *fetch.EventRequestPaused) {
	if ev == nil {
		return
	}
	rule := s.matchNetworkMockRule(ev.Request.URL, ev.Request.Method)
	if rule == nil {
		s.continueNetworkRequest(ctx, ev.RequestID)
		return
	}
	s.applyNetworkMockRule(ctx, ev, rule)
}

func (s *Session) matchNetworkMockRule(url, method string) *networkMockRule {
	s.networkMockMu.RLock()
	defer s.networkMockMu.RUnlock()
	if len(s.networkMocks) == 0 {
		return nil
	}
	needle := strings.ToUpper(strings.TrimSpace(method))
	for i := len(s.networkMocks) - 1; i >= 0; i-- {
		rule := s.networkMocks[i]
		if rule == nil {
			continue
		}
		if rule.method != "" && rule.method != needle {
			continue
		}
		if rule.matcher != nil && rule.matcher.MatchString(url) {
			return rule
		}
	}
	return nil
}

func (s *Session) applyNetworkMockRule(ctx context.Context, ev *fetch.EventRequestPaused, rule *networkMockRule) {
	if rule == nil || ev == nil {
		return
	}

	if rule.delay > 0 {
		timer := time.NewTimer(rule.delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
	}

	switch rule.mockType {
	case "abort":
		s.failNetworkRequest(ctx, ev.RequestID, rule.abortReason)
	case "delay":
		s.continueNetworkRequest(ctx, ev.RequestID)
	default:
		s.fulfillNetworkRequest(ctx, ev.RequestID, rule)
	}

	if s.log != nil {
		s.log.WithField("pattern", rule.urlPattern).
			WithField("mockType", rule.mockType).
			WithField("url", ev.Request.URL).
			Debug("applied network mock")
	}
}

func (s *Session) continueNetworkRequest(ctx context.Context, requestID fetch.RequestID) {
	if ctx == nil {
		return
	}
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(runCtx context.Context) error {
		return fetch.ContinueRequest(requestID).Do(runCtx)
	})); err != nil && s.log != nil {
		s.log.WithError(err).Debug("failed to continue intercepted request")
	}
}

func (s *Session) failNetworkRequest(ctx context.Context, requestID fetch.RequestID, reason network.ErrorReason) {
	if ctx == nil {
		return
	}
	if reason == "" {
		reason = network.ErrorReasonFailed
	}
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(runCtx context.Context) error {
		return fetch.FailRequest(requestID, reason).Do(runCtx)
	})); err != nil && s.log != nil {
		s.log.WithError(err).Warn("failed to abort intercepted request")
	}
}

func (s *Session) fulfillNetworkRequest(ctx context.Context, requestID fetch.RequestID, rule *networkMockRule) {
	if ctx == nil {
		return
	}
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(runCtx context.Context) error {
		params := fetch.FulfillRequest(requestID, rule.statusCode)
		if len(rule.headers) > 0 {
			params = params.WithResponseHeaders(rule.headers)
		}
		if len(rule.body) > 0 {
			params = params.WithBody(base64.StdEncoding.EncodeToString(rule.body))
		}
		return params.Do(runCtx)
	})); err != nil && s.log != nil {
		s.log.WithError(err).Warn("failed to fulfill intercepted request")
	}
}

func compileNetworkPattern(pattern string) (*regexp.Regexp, bool, error) {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return nil, false, fmt.Errorf("urlPattern cannot be empty")
	}

	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "regex:") {
		expr := strings.TrimSpace(trimmed[6:])
		re, err := regexp.Compile(expr)
		if err != nil {
			return nil, false, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
		}
		return re, true, nil
	}

	expr := globToRegex(trimmed)
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, false, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}
	return re, false, nil
}

func globToRegex(pattern string) string {
	var builder strings.Builder
	builder.WriteString("^")
	for _, r := range pattern {
		switch r {
		case '*':
			builder.WriteString(".*")
		case '?':
			builder.WriteString(".")
		case '.', '+', '(', ')', '|', '{', '}', '^', '$', '[', ']', '\\':
			builder.WriteRune('\\')
			builder.WriteRune(r)
		default:
			builder.WriteRune(r)
		}
	}
	builder.WriteString("$")
	return builder.String()
}

func encodeNetworkMockBody(body any) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	switch typed := body.(type) {
	case string:
		return []byte(typed), nil
	case []byte:
		return typed, nil
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal mock body: %w", err)
		}
		return data, nil
	}
}

func buildHeaderEntries(headers map[string]string) []*fetch.HeaderEntry {
	if len(headers) == 0 {
		return nil
	}
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	entries := make([]*fetch.HeaderEntry, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, &fetch.HeaderEntry{Name: key, Value: headers[key]})
	}
	return entries
}

func cloneHeaderMap(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	clone := make(map[string]string, len(headers))
	for key, value := range headers {
		clone[key] = value
	}
	return clone
}

func previewMockBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	preview := string(body)
	if len(preview) > 160 {
		return preview[:160] + "…"
	}
	return preview
}
