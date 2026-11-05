package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type businessAccountView struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	CreatedAt   string `json:"created_at"`
}

type businessAccountUsageView struct {
	BusinessAccountID string             `json:"business_account_id"`
	BillingPeriod     string             `json:"billing_period"`
	Tier              string             `json:"tier"`
	Limits            map[string]int64   `json:"limits"`
	Usage             map[string]int64   `json:"usage"`
	Reserved          map[string]int64   `json:"reserved"`
	Remaining         map[string]int64   `json:"remaining"`
	DisplayCredits    map[string]float64 `json:"display_credits"`
}

func (s *Server) businessAccountsHandler(w http.ResponseWriter, r *http.Request) {
	if !webConsoleSameOrigin(w, r) {
		return
	}
	var response struct {
		Accounts []businessAccountView `json:"accounts"`
	}
	if err := s.getLPBSJSON(r.Context(), "/api/v1/business-accounts", &response); err != nil {
		writeBusinessAccountProxyError(w, err)
		return
	}
	if response.Accounts == nil {
		response.Accounts = []businessAccountView{}
	}
	writeCredentialJSON(w, http.StatusOK, response)
}

func (s *Server) businessAccountUsageHandler(w http.ResponseWriter, r *http.Request) {
	if !webConsoleSameOrigin(w, r) {
		return
	}
	accountID := strings.TrimSpace(r.URL.Query().Get("business_account_id"))
	if accountID == "" || len(accountID) > 255 {
		http.Error(w, "business_account_id is required", http.StatusBadRequest)
		return
	}
	path := "/api/v1/usage/business-account-summary?business_account_id=" + url.QueryEscape(accountID)
	var response businessAccountUsageView
	if err := s.getLPBSJSON(r.Context(), path, &response); err != nil {
		writeBusinessAccountProxyError(w, err)
		return
	}
	if response.BusinessAccountID != accountID {
		http.Error(w, "account usage response did not match the selected account", http.StatusBadGateway)
		return
	}
	writeCredentialJSON(w, http.StatusOK, response)
}

func (s *Server) getLPBSJSON(ctx context.Context, path string, destination any) error {
	if s == nil || s.subscriptionResolver == nil {
		return errBusinessAccountAuthorityUnavailable
	}
	access, err := s.subscriptionResolver.Resolve(ctx)
	if err != nil {
		return fmt.Errorf("resolve linked LPBS session: %w", err)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(s.subscriptionResolver.LPBSBaseURL), "/")
	if baseURL == "" || strings.TrimSpace(access.AccessToken) == "" {
		return errBusinessAccountAuthorityUnavailable
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create LPBS account request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+access.AccessToken)
	request.Header.Set("Accept", "application/json")
	client := s.subscriptionResolver.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request LPBS account facts: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return lpbsAccountStatusError{status: response.StatusCode}
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(destination); err != nil {
		return fmt.Errorf("decode LPBS account facts: %w", err)
	}
	return nil
}

var errBusinessAccountAuthorityUnavailable = fmt.Errorf("business account authority unavailable")

type lpbsAccountStatusError struct{ status int }

func (e lpbsAccountStatusError) Error() string {
	return fmt.Sprintf("LPBS account response status %d", e.status)
}

func writeBusinessAccountProxyError(w http.ResponseWriter, err error) {
	status := http.StatusServiceUnavailable
	var upstream lpbsAccountStatusError
	if errors.As(err, &upstream) {
		switch upstream.status {
		case http.StatusUnauthorized:
			status = http.StatusUnauthorized
		case http.StatusForbidden:
			status = http.StatusForbidden
		default:
			status = http.StatusBadGateway
		}
	}
	http.Error(w, "business account facts are unavailable", status)
}
