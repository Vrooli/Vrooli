package bridge

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
)

const IngressPath = "/api/bridge/local-work-events"

// SitesAdmissionHeader is consumed by private Sites dispatch. It is neither
// visitor identity nor application/source authorization; the HMAC is required too.
const SitesAdmissionHeader = "OAI-Sites-Authorization"

var sitesBearer = regexp.MustCompile(`^Bearer [A-Za-z0-9._~+/-]+=*$`)

// SitesAdmissionProvider supplies already-approved private Sites admission for
// exactly destination (the fixed origin and ingress path), under WakeBoundary.
// Implementations must bind custody, expiry, rotation and revocation to that
// destination. No provider or real credential is installed by this package.
// The returned value is the complete Bearer header, never logged or persisted.
type SitesAdmissionProvider interface {
	SitesAuthorization(context.Context, string) (string, error)
}

// Signer is dedicated to this one bridge and approved destination. Implementors
// sign HMAC-SHA256 messages under a dedicated bridge key. No secret is read,
// created, persisted, or taken from a request/event by this package.
type Signer interface {
	SignWake(context.Context, []byte) (keyID string, mac []byte, err error)
}

type Forwarder struct {
	sitesAdmission SitesAdmissionProvider
	origin         string
	signer         Signer
	boundary       WakeBoundary
	client         *http.Client
	now            func() time.Time
}

// NewForwarder performs no I/O. Construction alone never creates subscriptions
// or starts a dispatcher. Without NewSitesForwarder, delivery is disabled.
// The fixed HTTPS origin comes from approved setup,
// never event data. Production startup deliberately leaves this uninstalled.
func NewForwarder(origin string, signer Signer, boundary WakeBoundary) (*Forwarder, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" ||
		u.Opaque != "" || u.ForceQuery || u.RawPath != "" || origin != u.String() || signer == nil || boundary == nil {
		return nil, ErrWakeRejected
	}
	host := strings.ToLower(u.Hostname())
	if net.ParseIP(host) != nil || host == "localhost" || !strings.Contains(host, ".") || strings.HasSuffix(host, ".localhost") {
		return nil, ErrWakeRejected
	}
	// Explicit transport never inherits HTTP(S)_PROXY, cookies, client auth,
	// environment credentials, or a caller-supplied redirector. TLS uses the
	// operating system trust roots.
	transport := &http.Transport{
		Proxy: nil, DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 5 * time.Second, MaxResponseHeaderBytes: 8192,
		DisableKeepAlives: true, DisableCompression: true,
	}
	return &Forwarder{
		origin: origin, signer: signer, boundary: boundary, now: time.Now,
		client: &http.Client{
			Transport: transport, Timeout: 8 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

// NewSitesForwarder adds the separate private Sites admission boundary. It
// performs no I/O, retrieves no credentials and activates no subscription.
// Both legacy notification and terminal delivery require this provider plus
// the dedicated application signer; neither credential substitutes for the other.
func NewSitesForwarder(origin string, signer Signer, boundary WakeBoundary, admission SitesAdmissionProvider) (*Forwarder, error) {
	if admission == nil {
		return nil, ErrWakeRejected
	}
	f, err := NewForwarder(origin, signer, boundary)
	if err != nil {
		return nil, err
	}
	f.sitesAdmission = admission
	return f, nil
}

// Deliver accepts only the stored, canonical typed projection and exact reserved
// subscription. Retries use identical bytes/batch identity but a fresh signature
// timestamp. Unknown outcomes remain pending for idempotent cloud admission.
func (f *Forwarder) Deliver(ctx context.Context, sub subscription.Subscription, queued subscription.QueuedDelivery) error {
	if !ValidSubscription(sub) || queued.SubscriptionID != sub.ID {
		return ErrWakeRejected
	}
	return f.within(ctx, func(inner context.Context, p Policy) error {
		wake, err := decodeWake(queued.PayloadJSON, p, queued.EventID)
		if err != nil {
			return err
		}
		if wake.Events[0].EventType != sub.EventPattern {
			return ErrWakeRejected
		}
		if f.signer == nil || f.client == nil || f.sitesAdmission == nil {
			return ErrWakeDisabled
		}
		destination := f.origin + IngressPath
		authorization, err := f.sitesAdmission.SitesAuthorization(inner, destination)
		if err != nil || len(authorization) > 4096 || !sitesBearer.MatchString(authorization) || inner.Err() != nil {
			return ErrWakeRejected
		}
		stamp := strconv.FormatInt(f.now().UTC().Unix(), 10)
		message := []byte("flywheel.local-work-http.v1\n" + stamp + "\nPOST\n" + f.origin + IngressPath + "\n" + queued.PayloadJSON)
		key, mac, err := f.signer.SignWake(inner, message)
		if err != nil || !opaque.MatchString(key) || len(mac) != 32 || inner.Err() != nil {
			return ErrWakeRejected
		}
		req, err := http.NewRequestWithContext(inner, http.MethodPost, f.origin+IngressPath, strings.NewReader(queued.PayloadJSON))
		if err != nil {
			return ErrWakeRejected
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(SitesAdmissionHeader, authorization)
		req.Header.Set("X-Flywheel-Bridge-Key-Id", key)
		req.Header.Set("X-Flywheel-Bridge-Timestamp", stamp)
		req.Header.Set("X-Flywheel-Bridge-Signature", "v1,"+base64.StdEncoding.EncodeToString(mac))
		req.Header.Set("Idempotency-Key", wake.BatchID)
		response, err := f.client.Do(req)
		if err != nil {
			return errors.New("local-work transport failed")
		}
		defer response.Body.Close()
		reply, err := io.ReadAll(io.LimitReader(response.Body, 4097))
		if err != nil || len(reply) > 4096 || response.StatusCode < 200 || response.StatusCode >= 300 {
			return errors.New("local-work response rejected")
		}
		// Responses are bounded and discarded, never logged or interpreted as commands.
		return nil
	})
}
