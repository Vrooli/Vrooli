package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (s *RemoteProfileService) buildRemoteURL(apiBase string, pathValue string, query map[string]string) (string, error) {
	parsed, err := url.Parse(apiBase)
	if err != nil {
		return "", err
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + pathValue
	values := url.Values{}
	for key, value := range query {
		if strings.TrimSpace(key) == "" {
			continue
		}
		values.Set(key, value)
	}
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

func (s *RemoteProfileService) remoteLogin(ctx context.Context, apiBase string, email string, password string, metadata RemoteProfileSessionMetadata) (string, string, *time.Time, error) {
	// #nosec G117 -- this password is intentionally sent only to a configured remote
	// admin login endpoint; production profile validation requires an HTTPS API base.
	payload, err := json.Marshal(LoginRequest{Email: email, Password: password})
	if err != nil {
		return "", "", nil, err
	}
	urlValue := strings.TrimSuffix(apiBase, "/") + "/admin/login"
	headers := map[string]string{
		"User-Agent": buildRemoteProfileSessionUserAgent(metadata),
	}
	resp, body, err := s.doJSONRequestWithHeaders(ctx, http.MethodPost, urlValue, payload, nil, headers)
	if err != nil {
		return "", "", nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := extractRemoteErrorMessage(body)
		return "", "", nil, &RemoteProfileError{
			Status:    mapRemoteStatus(resp.StatusCode),
			ErrorType: inferErrorType(resp.StatusCode),
			Message:   message,
		}
	}

	cookie := findCookie(resp.Cookies(), remoteProfileCookieName)
	if cookie == nil || cookie.Value == "" {
		return "", "", nil, &RemoteProfileError{
			Status:    http.StatusBadGateway,
			ErrorType: ApiErrorTypeServerError,
			Message:   "Remote login did not return a session cookie",
		}
	}
	var sessionResp LoginResponse
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		return "", "", nil, err
	}
	if !sessionResp.Authenticated {
		return "", "", nil, &RemoteProfileError{
			Status:    http.StatusUnauthorized,
			ErrorType: ApiErrorTypeUnauthorized,
			Message:   "Remote login failed",
		}
	}

	authenticated, err := s.remoteSessionCheck(ctx, apiBase, cookie.Value)
	if err != nil {
		return "", "", nil, err
	}
	if !authenticated {
		return "", "", nil, &RemoteProfileError{
			Status:    http.StatusUnauthorized,
			ErrorType: ApiErrorTypeUnauthorized,
			Message:   "Remote session verification failed",
		}
	}

	expiresAt := deriveCookieExpiry(cookie, s.nowTime())
	return cookie.Value, normalizeRemoteProfileSessionID(sessionResp.SessionID), expiresAt, nil
}

func (s *RemoteProfileService) nowTime() time.Time {
	if s == nil || s.now == nil {
		return time.Now()
	}
	return s.now()
}

func (s *RemoteProfileService) remoteSessionCheck(ctx context.Context, apiBase string, sessionValue string) (bool, error) {
	urlValue := strings.TrimSuffix(apiBase, "/") + "/admin/session"
	cookies := []*http.Cookie{remoteProfileSessionCookie(sessionValue)}
	resp, body, err := s.doJSONRequest(ctx, http.MethodGet, urlValue, nil, cookies)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := extractRemoteErrorMessage(body)
		return false, &RemoteProfileError{
			Status:    mapRemoteStatus(resp.StatusCode),
			ErrorType: inferErrorType(resp.StatusCode),
			Message:   message,
		}
	}
	var sessionResp LoginResponse
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		return false, err
	}
	return sessionResp.Authenticated, nil
}

func (s *RemoteProfileService) remoteLogout(ctx context.Context, apiBase string, sessionValue string) error {
	urlValue := strings.TrimSuffix(apiBase, "/") + "/admin/logout"
	cookies := []*http.Cookie{remoteProfileSessionCookie(sessionValue)}
	resp, _, err := s.doJSONRequest(ctx, http.MethodPost, urlValue, nil, cookies)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &RemoteProfileError{
			Status:    mapRemoteStatus(resp.StatusCode),
			ErrorType: inferErrorType(resp.StatusCode),
			Message:   "Remote logout failed",
		}
	}
	return nil
}

func (s *RemoteProfileService) listIncomingRemoteSessions(ctx context.Context, apiBase string, sessionValue string, connectorID string) ([]IncomingRemoteProfileSession, error) {
	query := url.Values{}
	if trimmed := strings.TrimSpace(connectorID); trimmed != "" {
		query.Set("connector_id", trimmed)
	}
	urlValue := strings.TrimSuffix(apiBase, "/") + "/admin/remote-profile-sessions"
	if encoded := query.Encode(); encoded != "" {
		urlValue += "?" + encoded
	}

	cookies := []*http.Cookie{remoteProfileSessionCookie(sessionValue)}
	resp, body, err := s.doJSONRequest(ctx, http.MethodGet, urlValue, nil, cookies)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &RemoteProfileError{
			Status:    http.StatusUnauthorized,
			ErrorType: ApiErrorTypeUnauthorized,
			Message:   "Remote session expired. Please log in again.",
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &RemoteProfileError{
			Status:    mapRemoteStatus(resp.StatusCode),
			ErrorType: inferErrorType(resp.StatusCode),
			Message:   extractRemoteErrorMessage(body),
		}
	}

	var payload struct {
		Sessions []IncomingRemoteProfileSession `json:"sessions"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.Sessions == nil {
		payload.Sessions = []IncomingRemoteProfileSession{}
	}
	return payload.Sessions, nil
}

func (s *RemoteProfileService) revokeIncomingRemoteSession(ctx context.Context, apiBase string, sessionValue string, sessionID string) error {
	urlValue := strings.TrimSuffix(apiBase, "/") + "/admin/remote-profile-sessions/" + url.PathEscape(strings.TrimSpace(sessionID))
	cookies := []*http.Cookie{remoteProfileSessionCookie(sessionValue)}
	resp, body, err := s.doJSONRequest(ctx, http.MethodDelete, urlValue, nil, cookies)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &RemoteProfileError{
			Status:    http.StatusUnauthorized,
			ErrorType: ApiErrorTypeUnauthorized,
			Message:   "Remote session expired. Please log in again.",
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &RemoteProfileError{
			Status:    mapRemoteStatus(resp.StatusCode),
			ErrorType: inferErrorType(resp.StatusCode),
			Message:   extractRemoteErrorMessage(body),
		}
	}
	return nil
}

func (s *RemoteProfileService) doJSONRequest(ctx context.Context, method, urlValue string, body []byte, cookies []*http.Cookie) (*http.Response, []byte, error) {
	return s.doJSONRequestWithHeaders(ctx, method, urlValue, body, cookies, nil)
}

func (s *RemoteProfileService) doJSONRequestWithHeaders(ctx context.Context, method, urlValue string, body []byte, cookies []*http.Cookie, headers map[string]string) (*http.Response, []byte, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, urlValue, reader)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}
	for _, cookie := range cookies {
		if cookie != nil {
			req.AddCookie(cookie)
		}
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, classifyRemoteError(err)
	}
	defer resp.Body.Close()
	bodyBytes, err := readLimitedBody(resp.Body, remoteProfileProxyResponseMax)
	if err != nil {
		return resp, nil, err
	}
	return resp, bodyBytes, nil
}

func classifyRemoteError(err error) *RemoteProfileError {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &RemoteProfileError{
			Status:    http.StatusGatewayTimeout,
			ErrorType: ApiErrorTypeTimeout,
			Message:   "Remote request timed out",
		}
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return &RemoteProfileError{
				Status:    http.StatusGatewayTimeout,
				ErrorType: ApiErrorTypeTimeout,
				Message:   "Remote request timed out",
			}
		}
	}
	return &RemoteProfileError{
		Status:    http.StatusBadGateway,
		ErrorType: ApiErrorTypeNetwork,
		Message:   "Remote connection failed",
	}
}

func mapRemoteStatus(status int) int {
	if status >= 500 {
		return http.StatusBadGateway
	}
	return status
}

func extractRemoteErrorMessage(body []byte) string {
	if len(body) == 0 {
		return "Remote request failed"
	}
	var apiErr ApiErrorResponse
	if err := json.Unmarshal(body, &apiErr); err == nil {
		if strings.TrimSpace(apiErr.Error) != "" {
			return apiErr.Error
		}
	}
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		return "Remote request failed"
	}
	return msg
}

func deriveCookieExpiry(cookie *http.Cookie, now time.Time) *time.Time {
	if cookie == nil {
		return nil
	}
	if !cookie.Expires.IsZero() {
		expiry := cookie.Expires.UTC()
		return &expiry
	}
	if cookie.MaxAge > 0 {
		expiry := now.Add(time.Duration(cookie.MaxAge) * time.Second).UTC()
		return &expiry
	}
	return nil
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie != nil && cookie.Name == name {
			return cookie
		}
	}
	return nil
}
