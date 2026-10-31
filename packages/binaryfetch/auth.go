package binaryfetch

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// BearerAuthorization is transient transport material, never a manifest value.
// Its token is intentionally absent from JSON and diagnostic formatting.
type BearerAuthorization struct{ token string }

func NewBearerAuthorization(token string) *BearerAuthorization {
	return &BearerAuthorization{token: token}
}
func (*BearerAuthorization) String() string     { return "[redacted bearer authorization]" }
func (a *BearerAuthorization) GoString() string { return a.String() }

func downloadResponse(ctx context.Context, rawURL string, auth *BearerAuthorization) (*http.Response, error) {
	if auth != nil {
		if err := (AcquisitionAuth{Scheme: "bearer", LogicalID: "transport", Field: "token"}).ValidateURL(rawURL); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("binaryfetch: build download request: %w", err)
	}
	client := HTTPClient
	if auth != nil {
		if strings.TrimSpace(auth.token) == "" || strings.ContainsAny(auth.token, "\r\n") {
			return nil, errors.New("binaryfetch: bearer credential is empty or invalid")
		}
		origin := req.URL
		copyClient := *client
		priorRedirect := copyClient.CheckRedirect
		copyClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
			// Always remove Go's inherited header first: its default allows subdomains.
			next.Header.Del("Authorization")
			if priorRedirect != nil {
				if err := priorRedirect(next, via); err != nil {
					return err
				}
			} else if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			next.Header.Del("Authorization")
			if sameHTTPSOrigin(next.URL, origin) {
				next.Header.Set("Authorization", "Bearer "+auth.token)
			}
			return nil
		}
		client = &copyClient
		req.Header.Set("Authorization", "Bearer "+auth.token)
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, safeDownloadError(auth, err)
	}
	return response, nil
}

func sameHTTPSOrigin(a, b *url.URL) bool {
	port := func(u *url.URL) string {
		if p := u.Port(); p != "" {
			return p
		}
		return "443"
	}
	return a.Scheme == "https" && b.Scheme == "https" && strings.EqualFold(a.Hostname(), b.Hostname()) && port(a) == port(b) && a.User == nil
}

func safeDownloadError(auth *BearerAuthorization, err error) error {
	if auth == nil {
		return err
	}
	// Transport/redirect/body errors may echo headers or signed redirect URLs.
	// Preserve cancellation identity; do not expose arbitrary remote error text.
	for _, sentinel := range []error{context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, sentinel) {
			return fmt.Errorf("binaryfetch: authenticated download: %w", sentinel)
		}
	}
	return fmt.Errorf("binaryfetch: authenticated download failed (%T)", err)
}
