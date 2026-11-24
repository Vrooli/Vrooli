package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/logx"
)

var stripeSecretPattern = regexp.MustCompile(`(?:rk|sk)_(?:test|live)_[A-Za-z0-9_-]+|whsec_[A-Za-z0-9_-]+`)
var stripeEventIDPattern = regexp.MustCompile(`evt_[A-Za-z0-9]+`)

type stripeTestListener struct {
	lines         chan stripeCLILine
	deliveries    chan stripeDeliveryReceipt
	done          chan error
	cancel        context.CancelFunc
	webhookSecret string
	homeDir       string
	proxy         *httptest.Server
}

type stripeDeliveryReceipt struct {
	eventID   string
	eventType string
	status    int
}

type stripeCLILine struct {
	text          string
	webhookSecret string
}

// startStripeTestListener gives the Stripe CLI a narrowly scoped copy of the
// authority-resolved test key through its documented environment interface.
// The key is never put in argv, a file, LPBS's process environment, or logs.
func startStripeTestListener(ctx context.Context, apiKey, forwardTo string) (*stripeTestListener, error) {
	return startStripeTestListenerForEvents(ctx, apiKey, forwardTo, "customer.updated")
}

func startStripeTestListenerForEvents(ctx context.Context, apiKey, forwardTo string, events ...string) (*stripeTestListener, error) {
	if !strings.HasPrefix(apiKey, "sk_test_") && !strings.HasPrefix(apiKey, "rk_test_") {
		return nil, fmt.Errorf("Stripe CLI listener requires an authority-resolved test-mode key")
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("Stripe CLI listener requires an explicit event allowlist")
	}
	allowed := map[string]bool{"customer.updated": true, "checkout.session.completed": true, "checkout.session.async_payment_succeeded": true, "invoice.paid": true, "invoice.payment_failed": true, "customer.subscription.created": true, "customer.subscription.updated": true, "customer.subscription.deleted": true, "payment_intent.payment_failed": true, "payment_intent.succeeded": true}
	for _, event := range events {
		if !allowed[strings.TrimSpace(event)] {
			return nil, fmt.Errorf("Stripe CLI listener event is not allowlisted")
		}
	}
	path, err := exec.LookPath("stripe")
	if err != nil {
		return nil, fmt.Errorf("Stripe CLI is unavailable")
	}
	homeDir, err := os.MkdirTemp("", "lpbs-stripe-cli-")
	if err != nil {
		return nil, fmt.Errorf("create isolated Stripe CLI config home")
	}
	listenerCtx, cancel := context.WithCancel(ctx)
	deliveries := make(chan stripeDeliveryReceipt, 16)
	proxy := stripeWebhookForwardProxy(listenerCtx, forwardTo, deliveries)
	cmd := exec.CommandContext(listenerCtx, path, "listen", "--events", strings.Join(events, ","), "--forward-to", proxy.URL+"/api/v1/webhooks/stripe")
	cmd.Env = stripeCLIEnvironment(apiKey, homeDir)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		proxy.Close()
		_ = os.RemoveAll(homeDir)
		return nil, fmt.Errorf("open Stripe CLI output")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		proxy.Close()
		_ = os.RemoveAll(homeDir)
		return nil, fmt.Errorf("open Stripe CLI error output")
	}
	listener := &stripeTestListener{lines: make(chan stripeCLILine, 128), deliveries: deliveries, done: make(chan error, 1), cancel: cancel, homeDir: homeDir, proxy: proxy}
	if err := cmd.Start(); err != nil {
		cancel()
		proxy.Close()
		_ = os.RemoveAll(homeDir)
		return nil, fmt.Errorf("start Stripe CLI listener")
	}
	go scanStripeCLIOutput(stdout, apiKey, listener.lines)
	go scanStripeCLIOutput(stderr, apiKey, listener.lines)
	go func() { listener.done <- cmd.Wait() }()

	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	for {
		select {
		case line := <-listener.lines:
			if line.webhookSecret != "" {
				listener.webhookSecret = line.webhookSecret
				return listener, nil
			}
		case err := <-listener.done:
			cancel()
			proxy.Close()
			_ = os.RemoveAll(homeDir)
			if err != nil {
				return nil, fmt.Errorf("Stripe CLI listener exited before readiness")
			}
			return nil, fmt.Errorf("Stripe CLI listener exited before readiness")
		case <-timer.C:
			cancel()
			proxy.Close()
			select {
			case <-listener.done:
			case <-time.After(3 * time.Second):
			}
			_ = os.RemoveAll(homeDir)
			return nil, fmt.Errorf("Stripe CLI listener did not become ready within 30 seconds")
		case <-ctx.Done():
			cancel()
			proxy.Close()
			select {
			case <-listener.done:
			case <-time.After(3 * time.Second):
			}
			_ = os.RemoveAll(homeDir)
			return nil, ctx.Err()
		}
	}
}

func (l *stripeTestListener) close() {
	if l == nil {
		return
	}
	l.cancel()
	select {
	case <-l.done:
	case <-time.After(3 * time.Second):
	}
	if l.proxy != nil {
		l.proxy.Close()
	}
	_ = os.RemoveAll(l.homeDir)
}

func (l *stripeTestListener) waitForDelivery(ctx context.Context) (string, string, error) {
	status, eventID, _, err := l.waitForEventDelivery(ctx, nil)
	return status, eventID, err
}

func (l *stripeTestListener) waitForEventDelivery(ctx context.Context, acceptedTypes map[string]bool) (string, string, string, error) {
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	for {
		select {
		case receipt := <-l.deliveries:
			if len(acceptedTypes) > 0 && !acceptedTypes[receipt.eventType] {
				continue
			}
			if receipt.status >= 200 && receipt.status < 300 {
				return "passed", receipt.eventID, receipt.eventType, nil
			}
			return "failed", receipt.eventID, receipt.eventType, fmt.Errorf("local Stripe webhook returned HTTP %d", receipt.status)
		case err := <-l.done:
			if err != nil {
				return "unknown", "", "", fmt.Errorf("Stripe CLI listener stopped before webhook receipt")
			}
			return "unknown", "", "", fmt.Errorf("Stripe CLI listener stopped before webhook receipt")
		case <-timer.C:
			return "unknown", "", "", fmt.Errorf("no successful Stripe webhook delivery was observed within 30 seconds")
		case <-ctx.Done():
			return "unknown", "", "", ctx.Err()
		}
	}
}

// stripeWebhookForwardProxy records only the event ID/type and HTTP status in
// memory, while forwarding the signed request unchanged to LPBS. The payload
// and signature are never logged or persisted by the proxy.
func stripeWebhookForwardProxy(ctx context.Context, target string, deliveries chan<- stripeDeliveryReceipt) *httptest.Server {
	client := &http.Client{Timeout: 15 * time.Second}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, incoming *http.Request) {
		body, err := io.ReadAll(io.LimitReader(incoming.Body, 2<<20))
		if err != nil {
			http.Error(w, "invalid webhook request body", http.StatusBadRequest)
			return
		}
		var event struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		}
		_ = json.Unmarshal(body, &event)
		request, err := http.NewRequestWithContext(ctx, incoming.Method, target, strings.NewReader(string(body)))
		if err != nil {
			http.Error(w, "cannot forward webhook request", http.StatusBadGateway)
			return
		}
		request.Header.Set("Content-Type", incoming.Header.Get("Content-Type"))
		request.Header.Set("Stripe-Signature", incoming.Header.Get("Stripe-Signature"))
		response, err := client.Do(request)
		if err != nil {
			http.Error(w, "LPBS webhook endpoint unavailable", http.StatusBadGateway)
			deliveries <- stripeDeliveryReceipt{eventID: event.ID, eventType: event.Type, status: http.StatusBadGateway}
			return
		}
		defer response.Body.Close()
		responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
		if readErr != nil {
			http.Error(w, "cannot read LPBS webhook response", http.StatusBadGateway)
			deliveries <- stripeDeliveryReceipt{eventID: event.ID, eventType: event.Type, status: http.StatusBadGateway}
			return
		}
		w.WriteHeader(response.StatusCode)
		_, _ = w.Write(responseBody)
		deliveries <- stripeDeliveryReceipt{eventID: event.ID, eventType: event.Type, status: response.StatusCode}
	}))
}

func triggerStripeTestEvent(ctx context.Context, apiKey, event string) error {
	path, err := exec.LookPath("stripe")
	if err != nil {
		return fmt.Errorf("Stripe CLI is unavailable")
	}
	homeDir, err := os.MkdirTemp("", "lpbs-stripe-cli-")
	if err != nil {
		return fmt.Errorf("create isolated Stripe CLI config home")
	}
	defer os.RemoveAll(homeDir)
	cmd := exec.CommandContext(ctx, path, "trigger", event)
	cmd.Env = stripeCLIEnvironment(apiKey, homeDir)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Stripe test event trigger failed")
	}
	return nil
}

func scanStripeCLIOutput(reader io.Reader, apiKey string, lines chan<- stripeCLILine) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 256*1024)
	for scanner.Scan() {
		raw := scanner.Text()
		secret := stripeSecretPattern.FindString(raw)
		if !strings.HasPrefix(secret, "whsec_") {
			secret = ""
		}
		line := redactStripeCLIOutput(raw, apiKey)
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.Contains(line, "--> ") || strings.Contains(line, "<-- [") || strings.Contains(line, "Ready!") {
			logx.Info("stripe_test_cli", map[string]interface{}{"line": line})
		}
		select {
		case lines <- stripeCLILine{text: line, webhookSecret: secret}:
		default:
		}
	}
}

func redactStripeCLIOutput(value, apiKey string) string {
	if apiKey != "" {
		value = strings.ReplaceAll(value, apiKey, "[redacted]")
	}
	return stripeSecretPattern.ReplaceAllString(value, "[redacted]")
}

func stripeCLIEnvironment(apiKey, homeDir string) []string {
	allowed := map[string]bool{"PATH": true, "HOME": true, "TMPDIR": true, "LANG": true, "LC_ALL": true, "NO_COLOR": true, "SSL_CERT_FILE": true, "SSL_CERT_DIR": true}
	env := make([]string, 0, 10)
	for _, pair := range os.Environ() {
		name, _, ok := strings.Cut(pair, "=")
		if ok && allowed[name] && name != "NO_COLOR" && name != "HOME" {
			env = append(env, pair)
		}
	}
	env = append(env, "HOME="+homeDir, "NO_COLOR=1", "STRIPE_API_KEY="+apiKey)
	return env
}
