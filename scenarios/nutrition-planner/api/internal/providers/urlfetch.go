package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrUnsupportedScheme  = errors.New("url scheme is not supported")
	ErrBlockedDestination = errors.New("url destination is not allowed")
	ErrResponseTooLarge   = errors.New("url response exceeds configured limit")
)

type LookupIP func(context.Context, string) ([]net.IP, error)

type URLFetchResult struct {
	URL       string
	FinalURL  string
	MediaType string
	Body      []byte
	Truncated bool
	FetchedAt time.Time
	Redirects int
}

type URLFetcher struct {
	client   *http.Client
	lookup   LookupIP
	maxBytes int64
	maxHops  int
	timeout  time.Duration
}

func NewURLFetcher(client *http.Client, lookup LookupIP, maxBytes int64, maxHops int, timeout time.Duration) *URLFetcher {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	if lookup == nil {
		lookup = func(ctx context.Context, host string) ([]net.IP, error) {
			return net.DefaultResolver.LookupIP(ctx, "ip", host)
		}
	}
	if maxBytes <= 0 {
		maxBytes = 2 * 1024 * 1024
	}
	if maxHops <= 0 {
		maxHops = 5
	}
	if client.Timeout <= 0 {
		client.Timeout = timeout
	}
	return &URLFetcher{client: client, lookup: lookup, maxBytes: maxBytes, maxHops: maxHops, timeout: timeout}
}

func (f *URLFetcher) Fetch(ctx context.Context, rawURL string) (URLFetchResult, error) {
	if err := f.validate(ctx, rawURL); err != nil {
		return URLFetchResult{}, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()
	hops := 0
	client := *f.client
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		hops++
		if hops > f.maxHops {
			return errors.New("url redirect limit exceeded")
		}
		return f.validate(requestCtx, req.URL.String())
	}
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return URLFetchResult{}, err
	}
	response, err := client.Do(request)
	if err != nil {
		return URLFetchResult{}, err
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, f.maxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return URLFetchResult{}, err
	}
	if int64(len(body)) > f.maxBytes {
		return URLFetchResult{}, ErrResponseTooLarge
	}
	return URLFetchResult{URL: rawURL, FinalURL: response.Request.URL.String(), MediaType: response.Header.Get("Content-Type"), Body: body, FetchedAt: time.Now().UTC(), Redirects: hops}, nil
}

func (f *URLFetcher) validate(ctx context.Context, rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.User != nil || parsed.Hostname() == "" {
		return ErrBlockedDestination
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrUnsupportedScheme
	}
	host := parsed.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if blockedIP(ip) {
			return ErrBlockedDestination
		}
		return nil
	}
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") || strings.HasSuffix(strings.ToLower(host), ".local") {
		return ErrBlockedDestination
	}
	ips, err := f.lookup(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve url host %q: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("resolve url host %q: no addresses", host)
	}
	for _, ip := range ips {
		if blockedIP(ip) {
			return ErrBlockedDestination
		}
	}
	return nil
}

func blockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}
