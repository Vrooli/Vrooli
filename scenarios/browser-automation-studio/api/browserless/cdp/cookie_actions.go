package cdp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
)

func (s *Session) ExecuteSetCookie(ctx context.Context, params runtime.InstructionParam) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{DebugContext: map[string]interface{}{
		"name":     params.CookieName,
		"url":      params.CookieURL,
		"domain":   params.CookieDomain,
		"path":     params.CookiePath,
		"sameSite": params.CookieSameSite,
	}}

	timeout := params.TimeoutMs
	if timeout <= 0 {
		timeout = 30000
	}

	timeoutCtx, cancel := context.WithTimeout(s.ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	cmd := network.SetCookie(params.CookieName, params.CookieValue)
	if strings.TrimSpace(params.CookieURL) != "" {
		cmd = cmd.WithURL(params.CookieURL)
	} else {
		cmd = cmd.WithDomain(params.CookieDomain)
		path := params.CookiePath
		if strings.TrimSpace(path) == "" {
			path = "/"
		}
		cmd = cmd.WithPath(path)
	}

	if params.CookieSecure != nil {
		cmd = cmd.WithSecure(*params.CookieSecure)
	}
	if params.CookieHTTPOnly != nil {
		cmd = cmd.WithHTTPOnly(*params.CookieHTTPOnly)
	}
	if sameSite := cookieSameSiteParam(params.CookieSameSite); sameSite != "" {
		cmd = cmd.WithSameSite(sameSite)
	}

	if expires, err := computeCookieExpiration(params.CookieExpiresAt, params.CookieTTLSeconds); err != nil {
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	} else if expires != nil {
		cmd = cmd.WithExpires(expires)
		result.DebugContext["expiresAt"] = time.Time(*expires).Format(time.RFC3339)
	}

	if err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(runCtx context.Context) error {
		return cmd.Do(runCtx)
	})); err != nil {
		result.Error = fmt.Sprintf("setCookie failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

func (s *Session) ExecuteGetCookie(ctx context.Context, params runtime.InstructionParam) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{DebugContext: map[string]interface{}{
		"name":   params.CookieName,
		"url":    params.CookieURL,
		"domain": params.CookieDomain,
		"format": params.CookieResultFormat,
	}}

	timeout := params.TimeoutMs
	if timeout <= 0 {
		timeout = 30000
	}

	timeoutCtx, cancel := context.WithTimeout(s.ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	cmd := network.GetCookies()
	if trimmed := strings.TrimSpace(params.CookieURL); trimmed != "" {
		cmd = cmd.WithURLs([]string{trimmed})
	}

	var cookies []*network.Cookie
	if err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(runCtx context.Context) error {
		fetched, err := cmd.Do(runCtx)
		if err != nil {
			return err
		}
		cookies = fetched
		return nil
	})); err != nil {
		result.Error = fmt.Sprintf("getCookie failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	target := findMatchingCookie(cookies, params.CookieName, params.CookieDomain)
	if target == nil {
		err := fmt.Errorf("cookie %s not found", params.CookieName)
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	payload := map[string]interface{}{
		"name":     target.Name,
		"value":    target.Value,
		"domain":   target.Domain,
		"path":     target.Path,
		"secure":   target.Secure,
		"httpOnly": target.HTTPOnly,
		"expires":  target.Expires,
		"sameSite": target.SameSite,
	}
	result.DebugContext["cookie"] = payload

	if strings.ToLower(params.CookieResultFormat) == "object" {
		result.ExtractedData = payload
	} else {
		result.ExtractedData = target.Value
	}

	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

func (s *Session) ExecuteClearCookie(ctx context.Context, params runtime.InstructionParam) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{DebugContext: map[string]interface{}{
		"name":     params.CookieName,
		"url":      params.CookieURL,
		"domain":   params.CookieDomain,
		"path":     params.CookiePath,
		"clearAll": params.CookieClearAll,
	}}

	timeout := params.TimeoutMs
	if timeout <= 0 {
		timeout = 30000
	}

	timeoutCtx, cancel := context.WithTimeout(s.ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	var actionErr error
	if params.CookieClearAll {
		actionErr = chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(runCtx context.Context) error {
			return network.ClearBrowserCookies().Do(runCtx)
		}))
	} else {
		actionErr = chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(runCtx context.Context) error {
			cmd := network.DeleteCookies(params.CookieName)
			if strings.TrimSpace(params.CookieURL) != "" {
				cmd = cmd.WithURL(params.CookieURL)
			}
			if strings.TrimSpace(params.CookieDomain) != "" {
				cmd = cmd.WithDomain(params.CookieDomain)
			}
			if strings.TrimSpace(params.CookiePath) != "" {
				cmd = cmd.WithPath(params.CookiePath)
			}
			return cmd.Do(runCtx)
		}))
	}

	if actionErr != nil {
		result.Error = fmt.Sprintf("clearCookie failed: %v", actionErr)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, actionErr
	}

	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

func cookieSameSiteParam(raw string) network.CookieSameSite {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "strict":
		return network.CookieSameSiteStrict
	case "none":
		return network.CookieSameSiteNone
	case "lax":
		return network.CookieSameSiteLax
	default:
		return ""
	}
}

func computeCookieExpiration(raw string, ttlSeconds int) (*cdp.TimeSinceEpoch, error) {
	if ttlSeconds > 0 {
		ts := cdp.TimeSinceEpoch(time.Now().Add(time.Duration(ttlSeconds) * time.Second))
		return &ts, nil
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if numeric, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		if len(trimmed) > 10 {
			ts := cdp.TimeSinceEpoch(time.UnixMilli(numeric))
			return &ts, nil
		}
		ts := cdp.TimeSinceEpoch(time.Unix(numeric, 0))
		return &ts, nil
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		ts := cdp.TimeSinceEpoch(parsed)
		return &ts, nil
	}
	if parsed, err := time.Parse(time.RFC1123, trimmed); err == nil {
		ts := cdp.TimeSinceEpoch(parsed)
		return &ts, nil
	}
	return nil, fmt.Errorf("invalid cookie expiration %q", raw)
}

func findMatchingCookie(cookies []*network.Cookie, name, domain string) *network.Cookie {
	normalizedDomain := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(domain)), ".")
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		if cookie.Name != name {
			continue
		}
		if normalizedDomain == "" {
			return cookie
		}
		candidateDomain := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(cookie.Domain)), ".")
		if candidateDomain == normalizedDomain || strings.HasSuffix(candidateDomain, "."+normalizedDomain) {
			return cookie
		}
	}
	return nil
}
