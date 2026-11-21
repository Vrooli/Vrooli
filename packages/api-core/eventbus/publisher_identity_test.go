package eventbus

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventpublisher"
	"github.com/vrooli/api-core/servicetoken"
)

// bus is a vrooli-events that records the Authorization header of each publish.
type bus struct {
	server *httptest.Server
	auth   []string
}

func newBus(t *testing.T) *bus {
	t.Helper()
	b := &bus{}
	b.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		b.auth = append(b.auth, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(b.server.Close)
	return b
}

func publish(t *testing.T, b *bus, source string) {
	t.Helper()
	err := Client{BaseURL: b.server.URL}.PublishDomainEvent(context.Background(), DomainEvent{Source: source, EventType: "x.y.notification.v1", Payload: map[string]any{"n": 1}})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
}

// useTokens installs a provider whose token client exchanges through exchange.
func useTokens(t *testing.T, credential PublisherCredentials, exchange servicetoken.Exchanger) {
	t.Helper()
	clock := time.Now()
	newPublisherTokens = func(source string, cred servicetoken.CredentialSource) *servicetoken.Tokens {
		return eventpublisher.NewTokens(source, cred, servicetoken.WithExchanger(exchange), servicetoken.WithClock(func() time.Time { return clock }))
	}
	UsePublisherCredentials(credential)
	t.Cleanup(func() {
		UsePublisherCredentials(nil)
		newPublisherTokens = func(source string, c servicetoken.CredentialSource) *servicetoken.Tokens {
			return eventpublisher.NewTokens(source, c)
		}
		publishers.now = time.Now
	})
}

func exchanged(source string) servicetoken.Exchanged {
	return servicetoken.Exchanged{
		AccessToken: "tok-" + source, ExpiresAt: time.Now().Add(10 * time.Minute),
		Subject: eventpublisher.Principal(source), Audience: eventpublisher.Audience, Scopes: []string{eventpublisher.Scope},
	}
}

func TestWithoutAProviderAnEventIsPublishedAsBefore(t *testing.T) {
	UsePublisherCredentials(nil)
	b := newBus(t)
	publish(t, b, "storage-manager")
	if len(b.auth) != 1 || b.auth[0] != "" {
		t.Fatalf("auth = %q, want one publish with no Authorization", b.auth)
	}
}

func TestAnIssuedCredentialMakesEachSourcePresentItsOwnPrincipal(t *testing.T) {
	var exchanges atomic.Int32
	useTokens(t, func(_ context.Context, source string) (string, error) { return "cred-" + source, nil },
		func(_ context.Context, credential string) (servicetoken.Exchanged, error) {
			exchanges.Add(1)
			return exchanged(credential[len("cred-"):]), nil
		})
	b := newBus(t)
	publish(t, b, "storage-manager")
	publish(t, b, "storage-manager")
	publish(t, b, "agent-manager")
	want := []string{"Bearer tok-storage-manager", "Bearer tok-storage-manager", "Bearer tok-agent-manager"}
	if len(b.auth) != 3 || b.auth[0] != want[0] || b.auth[1] != want[1] || b.auth[2] != want[2] {
		t.Fatalf("auth = %q, want %q", b.auth, want)
	}
	if exchanges.Load() != 2 {
		t.Fatalf("exchanges = %d, want one per source (the token is cached)", exchanges.Load())
	}
}

// Adopting an identity must never lose an event: a missing credential, a refusing
// authenticator and a token for the wrong principal all publish unauthenticated.
func TestAnEventIsStillPublishedWhenNoIdentityCanBeHad(t *testing.T) {
	cases := map[string]struct {
		credential PublisherCredentials
		exchange   servicetoken.Exchanger
	}{
		"credential not issued": {
			func(context.Context, string) (string, error) { return "", errors.New("no entry") },
			func(context.Context, string) (servicetoken.Exchanged, error) { return servicetoken.Exchanged{}, nil },
		},
		"authenticator down": {
			func(context.Context, string) (string, error) { return "cred", nil },
			func(context.Context, string) (servicetoken.Exchanged, error) {
				return servicetoken.Exchanged{}, errors.New("connection refused")
			},
		},
		"token for another principal": {
			func(context.Context, string) (string, error) { return "cred", nil },
			func(context.Context, string) (servicetoken.Exchanged, error) { return exchanged("agent-manager"), nil },
		},
	}
	for name, c := range cases {
		useTokens(t, c.credential, c.exchange)
		b := newBus(t)
		publish(t, b, "storage-manager")
		if len(b.auth) != 1 || b.auth[0] != "" {
			t.Fatalf("%s: auth = %q, want one publish with no Authorization", name, b.auth)
		}
	}
}

// A failing authenticator is asked again after a minute, not on every event.
func TestAFailedExchangeIsNotRetriedForAMinute(t *testing.T) {
	var exchanges atomic.Int32
	useTokens(t, func(context.Context, string) (string, error) { return "cred", nil },
		func(context.Context, string) (servicetoken.Exchanged, error) {
			exchanges.Add(1)
			return servicetoken.Exchanged{}, errors.New("down")
		})
	now := time.Now()
	publishers.now = func() time.Time { return now }
	b := newBus(t)
	for range 5 {
		publish(t, b, "storage-manager")
	}
	if exchanges.Load() != 1 {
		t.Fatalf("exchanges = %d, want 1 inside the retry window", exchanges.Load())
	}
	now = now.Add(publisherRetry + time.Second)
	publish(t, b, "storage-manager")
	if exchanges.Load() != 2 {
		t.Fatalf("exchanges = %d, want a retry after the window", exchanges.Load())
	}
}

// The operator issues the credential while the scenario runs. Events published
// before that are unauthenticated, and no restart is needed afterwards: the first
// publish once the retry window has passed presents the identity.
func TestACredentialIssuedWhileTheScenarioRunsIsPresentedWithinAMinute(t *testing.T) {
	var issued atomic.Bool
	useTokens(t, func(_ context.Context, source string) (string, error) {
		if !issued.Load() {
			return "", errors.New("no such entry")
		}
		return "cred-" + source, nil
	}, func(_ context.Context, credential string) (servicetoken.Exchanged, error) {
		return exchanged(credential[len("cred-"):]), nil
	})
	now := time.Now()
	publishers.now = func() time.Time { return now }
	b := newBus(t)

	publish(t, b, "storage-manager")
	issued.Store(true)
	publish(t, b, "storage-manager")
	if b.auth[0] != "" || b.auth[1] != "" {
		t.Fatalf("inside the retry window the source still publishes without an identity: %q", b.auth)
	}
	now = now.Add(publisherRetry + time.Second)
	publish(t, b, "storage-manager")
	publish(t, b, "storage-manager")
	if b.auth[2] != "Bearer tok-storage-manager" || b.auth[3] != "Bearer tok-storage-manager" {
		t.Fatalf("after the retry window the issued credential is presented: %q", b.auth)
	}
}
