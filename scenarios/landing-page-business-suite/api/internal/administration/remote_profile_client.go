package administration

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

const (
	remoteAdminLoginProcedure   = "/landing_page_business_suite.v1.AdminAuthService/Login"
	remoteAdminLogoutProcedure  = "/landing_page_business_suite.v1.AdminAuthService/Logout"
	remoteAdminSessionProcedure = "/landing_page_business_suite.v1.AdminAuthService/Session"
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

func (s *RemoteProfileService) BuildRemoteURL(apiBase string, pathValue string, query map[string]string) (string, error) {
	return s.buildRemoteURL(apiBase, pathValue, query)
}

func (s *RemoteProfileService) RemoteLogin(ctx context.Context, apiBase string, email string, password string, metadata RemoteProfileSessionMetadata) (string, string, *time.Time, error) {
	return s.remoteLogin(ctx, apiBase, email, password, metadata)
}

func (s *RemoteProfileService) RemoteSessionCheck(ctx context.Context, apiBase string, sessionValue string) (bool, error) {
	return s.remoteSessionCheck(ctx, apiBase, sessionValue)
}

func (s *RemoteProfileService) RemoteLogout(ctx context.Context, apiBase string, sessionValue string) error {
	return s.remoteLogout(ctx, apiBase, sessionValue)
}

func (s *RemoteProfileService) DoJSONRequest(ctx context.Context, method, urlValue string, body []byte, cookies []*http.Cookie) (*http.Response, []byte, error) {
	return s.doJSONRequest(ctx, method, urlValue, body, cookies)
}

func ClassifyRemoteError(err error) *RemoteProfileError { return classifyRemoteError(err) }

func (s *RemoteProfileService) remoteLogin(ctx context.Context, apiBase string, email string, password string, metadata RemoteProfileSessionMetadata) (string, string, *time.Time, error) {
	// #nosec G117 -- this password is intentionally sent only to a configured remote
	// admin login endpoint; production profile validation requires an HTTPS API base.
	payload, err := json.Marshal(remoteProfileLoginRequest{Email: email, Password: password})
	if err != nil {
		return "", "", nil, err
	}
	urlValue, err := remoteConnectURL(apiBase, remoteAdminLoginProcedure)
	if err != nil {
		return "", "", nil, err
	}
	headers := map[string]string{
		"User-Agent": BuildRemoteProfileSessionUserAgent(metadata),
	}
	resp, body, err := s.doConnectRequest(ctx, urlValue, payload, nil, headers)
	if err != nil {
		return "", "", nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := extractRemoteErrorMessage(body)
		return "", "", nil, &RemoteProfileError{
			Status:    mapRemoteStatus(resp.StatusCode),
			ErrorType: inferRemoteErrorType(resp.StatusCode),
			Message:   message,
		}
	}

	cookie := findCookie(resp.Cookies(), remoteProfileCookieName)
	if cookie == nil || cookie.Value == "" {
		return "", "", nil, &RemoteProfileError{
			Status:    http.StatusBadGateway,
			ErrorType: apiErrorTypeServerError,
			Message:   "Remote login did not return a session cookie",
		}
	}
	var sessionResp remoteProfileLoginResponse
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		return "", "", nil, err
	}
	if !sessionResp.Authenticated {
		return "", "", nil, &RemoteProfileError{
			Status:    http.StatusUnauthorized,
			ErrorType: apiErrorTypeUnauthorized,
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
			ErrorType: apiErrorTypeUnauthorized,
			Message:   "Remote session verification failed",
		}
	}

	expiresAt := deriveCookieExpiry(cookie, s.nowTime())
	return cookie.Value, normalizeRemoteProfileSessionID(sessionResp.SessionID), expiresAt, nil
}

func (s *RemoteProfileService) nowTime() time.Time {
	if s == nil || s.Now == nil {
		return time.Now()
	}
	return s.Now()
}

func (s *RemoteProfileService) remoteSessionCheck(ctx context.Context, apiBase string, sessionValue string) (bool, error) {
	urlValue, err := remoteConnectURL(apiBase, remoteAdminSessionProcedure)
	if err != nil {
		return false, err
	}
	cookies := []*http.Cookie{remoteProfileSessionCookie(sessionValue)}
	resp, body, err := s.doConnectRequest(ctx, urlValue, []byte(`{}`), cookies, nil)
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
			ErrorType: inferRemoteErrorType(resp.StatusCode),
			Message:   message,
		}
	}
	var sessionResp remoteProfileLoginResponse
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		return false, err
	}
	return sessionResp.Authenticated, nil
}

func (s *RemoteProfileService) remoteLogout(ctx context.Context, apiBase string, sessionValue string) error {
	urlValue, err := remoteConnectURL(apiBase, remoteAdminLogoutProcedure)
	if err != nil {
		return err
	}
	cookies := []*http.Cookie{remoteProfileSessionCookie(sessionValue)}
	resp, _, err := s.doConnectRequest(ctx, urlValue, []byte(`{}`), cookies, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &RemoteProfileError{
			Status:    mapRemoteStatus(resp.StatusCode),
			ErrorType: inferRemoteErrorType(resp.StatusCode),
			Message:   "Remote logout failed",
		}
	}
	return nil
}

func remoteConnectURL(apiBase, procedure string) (string, error) {
	parsed, err := url.Parse(apiBase)
	if err != nil {
		return "", err
	}
	parsed.Path = strings.TrimSuffix(strings.TrimSuffix(parsed.Path, "/"), "/api/v1") + procedure
	parsed.RawQuery = ""
	return parsed.String(), nil
}

func (s *RemoteProfileService) doConnectRequest(ctx context.Context, urlValue string, body []byte, cookies []*http.Cookie, headers map[string]string) (*http.Response, []byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, urlValue, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Connect-Protocol-Version", "1")
	for key, value := range headers {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			request.Header.Set(key, value)
		}
	}
	for _, cookie := range cookies {
		if cookie != nil {
			request.AddCookie(cookie)
		}
	}
	response, err := s.HTTPClient.Do(request)
	if err != nil {
		return nil, nil, classifyRemoteError(err)
	}
	defer response.Body.Close()
	responseBody, err := readLimitedBody(response.Body, remoteProfileProxyResponseMax)
	if err != nil {
		return response, nil, err
	}
	return response, responseBody, nil
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
			ErrorType: apiErrorTypeUnauthorized,
			Message:   "Remote session expired. Please log in again.",
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &RemoteProfileError{
			Status:    mapRemoteStatus(resp.StatusCode),
			ErrorType: inferRemoteErrorType(resp.StatusCode),
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
			ErrorType: apiErrorTypeUnauthorized,
			Message:   "Remote session expired. Please log in again.",
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &RemoteProfileError{
			Status:    mapRemoteStatus(resp.StatusCode),
			ErrorType: inferRemoteErrorType(resp.StatusCode),
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
	resp, err := s.HTTPClient.Do(req)
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
			ErrorType: apiErrorTypeTimeout,
			Message:   "Remote request timed out",
		}
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return &RemoteProfileError{
				Status:    http.StatusGatewayTimeout,
				ErrorType: apiErrorTypeTimeout,
				Message:   "Remote request timed out",
			}
		}
	}
	return &RemoteProfileError{
		Status:    http.StatusBadGateway,
		ErrorType: apiErrorTypeNetwork,
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
	var apiErr remoteProfileAPIErrorResponse
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

func ExtractRemoteErrorMessage(body []byte) string { return extractRemoteErrorMessage(body) }
func MapRemoteStatus(status int) int               { return mapRemoteStatus(status) }

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
