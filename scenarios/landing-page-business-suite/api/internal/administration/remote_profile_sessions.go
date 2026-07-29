package administration

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
)

func (s *RemoteProfileService) SessionLinks(ctx context.Context, id int64) (*RemoteProfileSessionLinks, error) {
	rec, err := s.getRecordByID(ctx, id)
	if err != nil {
		return nil, err
	}

	links := &RemoteProfileSessionLinks{
		ProfileID:             rec.ID,
		ProfileTag:            rec.Tag,
		ConnectorID:           remoteProfileConnectorID(rec),
		LocalHasSession:       rec.EncryptedSession.Valid && strings.TrimSpace(rec.EncryptedSession.String) != "",
		LocalStatus:           rec.Status,
		LocalSessionExpiresAt: nullTimeValue(rec.SessionExpiresAt),
		RemoteSessionID:       nullStringValue(rec.RemoteSessionID),
		RemoteSessions:        []IncomingRemoteProfileSession{},
	}
	if !links.LocalHasSession {
		return links, nil
	}

	sessionValue, err := s.decrypt(rec.EncryptedSession.String)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(sessionValue) == "" {
		return links, nil
	}

	sessions, err := s.listIncomingRemoteSessions(ctx, rec.APIBase, sessionValue, remoteProfileConnectorID(rec))
	if err != nil {
		var remoteErr *RemoteProfileError
		if errors.As(err, &remoteErr) && remoteErr.Status == http.StatusUnauthorized {
			_ = s.clearSession(ctx, id, remoteProfileStatusExpired)
		}
		return nil, err
	}
	links.RemoteSessions = sessions
	return links, nil
}

func (s *RemoteProfileService) RevokeRemoteSessions(ctx context.Context, id int64) (*RemoteProfileSessionLinks, error) {
	rec, err := s.getRecordByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !rec.EncryptedSession.Valid || strings.TrimSpace(rec.EncryptedSession.String) == "" {
		return nil, ErrRemoteProfileSessionMissing
	}

	sessionValue, err := s.decrypt(rec.EncryptedSession.String)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(sessionValue) == "" {
		return nil, ErrRemoteProfileSessionMissing
	}

	sessions, err := s.listIncomingRemoteSessions(ctx, rec.APIBase, sessionValue, remoteProfileConnectorID(rec))
	if err != nil {
		return nil, err
	}
	for _, session := range sessions {
		if revokeErr := s.revokeIncomingRemoteSession(ctx, rec.APIBase, sessionValue, session.SessionID); revokeErr != nil {
			return nil, revokeErr
		}
	}

	if err := s.clearSession(ctx, id, remoteProfileStatusExpired); err != nil {
		return nil, err
	}
	return s.SessionLinks(ctx, id)
}

func (s *RemoteProfileService) Proxy(ctx context.Context, id int64, req RemoteProfileProxyRequest) (*RemoteProxyResponse, error) {
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		return nil, &RemoteProfileError{Status: http.StatusBadRequest, ErrorType: apiErrorTypeValidation, Message: "method is required"}
	}
	allowedMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true}
	if !allowedMethods[method] {
		return nil, &RemoteProfileError{Status: http.StatusBadRequest, ErrorType: apiErrorTypeValidation, Message: "unsupported method"}
	}

	pathValue, err := normalizeRemoteProxyPath(req.Path)
	if err != nil {
		return nil, &RemoteProfileError{Status: http.StatusBadRequest, ErrorType: apiErrorTypeValidation, Message: err.Error()}
	}
	if !isAllowedRemoteProxyPath(pathValue) {
		return nil, ErrRemoteProfileDisallowedPath
	}

	rec, err := s.getRecordByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !rec.EncryptedSession.Valid || rec.EncryptedSession.String == "" {
		return nil, ErrRemoteProfileSessionMissing
	}
	sessionValue, err := s.decrypt(rec.EncryptedSession.String)
	if err != nil {
		return nil, err
	}
	if sessionValue == "" {
		return nil, ErrRemoteProfileSessionMissing
	}

	remoteURL, err := s.buildRemoteURL(rec.APIBase, pathValue, req.Query)
	if err != nil {
		return nil, &RemoteProfileError{Status: http.StatusBadRequest, ErrorType: apiErrorTypeValidation, Message: err.Error()}
	}

	var body io.Reader
	if len(req.Body) > 0 {
		body = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, remoteURL, body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	for key, value := range req.Headers {
		keyLower := strings.ToLower(strings.TrimSpace(key))
		if keyLower == "" || !remoteProfileProxyAllowedHeaders[keyLower] {
			continue
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		httpReq.Header.Set(key, value)
	}
	if len(req.Body) > 0 && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	httpReq.AddCookie(remoteProfileSessionCookie(sessionValue))

	resp, err := s.HTTPClient.Do(httpReq)
	if err != nil {
		_ = s.updateStatus(ctx, id, remoteProfileStatusError)
		return nil, classifyRemoteError(err)
	}
	defer resp.Body.Close()

	bodyBytes, readErr := readLimitedBody(resp.Body, remoteProfileProxyResponseMax)
	if readErr != nil {
		_ = s.updateStatus(ctx, id, remoteProfileStatusError)
		return nil, readErr
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_ = s.clearSession(ctx, id, remoteProfileStatusExpired)
	} else if resp.StatusCode >= 500 {
		_ = s.updateStatus(ctx, id, remoteProfileStatusError)
	} else {
		_ = s.updateStatus(ctx, id, remoteProfileStatusActive)
	}

	contentType := resp.Header.Get("Content-Type")
	return &RemoteProxyResponse{
		StatusCode:  resp.StatusCode,
		Body:        bodyBytes,
		ContentType: contentType,
	}, nil
}
