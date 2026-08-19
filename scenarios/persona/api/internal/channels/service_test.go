package channels

import (
	"context"
	"errors"
	"testing"
	"time"

	"persona/internal/journal"
	"persona/internal/personas"
)

type channelPersonaFunc func(context.Context, string) (personas.Persona, error)

func (f channelPersonaFunc) Get(ctx context.Context, id string) (personas.Persona, error) {
	return f(ctx, id)
}

type channelJournal struct{ entries []journal.Entry }

func (j *channelJournal) Append(_ context.Context, entry journal.Entry) (journal.Entry, error) {
	j.entries = append(j.entries, entry)
	return entry, nil
}

func (j *channelJournal) List(context.Context, string, int) ([]journal.Entry, error) {
	return j.entries, nil
}

type channelSource struct {
	code Code
	err  error
}

func (s channelSource) Retrieve(context.Context, Channel, string) (Code, error) {
	return s.code, s.err
}

func (s channelSource) Send(_ context.Context, c Channel, in MessageInput) (Message, error) {
	if in.Recipient == "" || in.Body == "" {
		return Message{}, ErrInvalidMessage
	}
	return Message{ID: "message-1", FromAddress: c.Address}, nil
}

func channelPersona() personas.Persona {
	return personas.Persona{ID: "persona-1", Kind: personas.KindPersonal, Status: personas.StatusActive}
}

func TestAllAdaptersShareOneCodeAndMessageContract(t *testing.T) {
	// [REQ:PSN-P0-006] email, SMS, and device adapters use the same typed seams.
	source := channelSource{code: Code{Value: "123456", ExpiresAt: time.Now().Add(time.Minute)}}
	adapters := []Adapter{EmailAdapter{Source: source}, SMSAdapter{Source: source}, DeviceAdapter{Source: source}}
	for _, adapter := range adapters {
		code, err := adapter.Retrieve(context.Background(), Channel{ID: "channel-1", PersonaID: "persona-1", Address: "controlled@example.test"}, "signup")
		if err != nil || code.Value != "123456" || code.ExpiresAt.IsZero() {
			t.Fatalf("%s Retrieve() = %#v, %v", adapter.Name(), code, err)
		}
		message, err := adapter.Send(context.Background(), Channel{ID: "channel-1", PersonaID: "persona-1", Address: "controlled@example.test"}, MessageInput{Recipient: "counterparty@example.test", Body: "verification"})
		if err != nil || message.ID == "" || message.FromAddress != "controlled@example.test" {
			t.Fatalf("%s Send() = %#v, %v", adapter.Name(), message, err)
		}
	}
}

func TestControlledChannelSendsAndReadsOnlyThroughItsBinding(t *testing.T) {
	// [REQ:PSN-P0-005] the persona's controlled address is the only send/read route.
	source := channelSource{code: Code{Value: "654321", ExpiresAt: time.Now().Add(time.Minute)}}
	repo := &FakeRepository{
		CreateFunc: func(_ context.Context, c Channel) (Channel, error) { c.ID = "channel-1"; return c, nil },
		ListFunc:   func(context.Context, string) ([]Channel, error) { return nil, nil },
		GetFunc: func(_ context.Context, id string) (Channel, error) {
			if id != "channel-1" {
				return Channel{}, ErrChannelNotFound
			}
			return Channel{ID: id, PersonaID: "persona-1", Kind: KindEmail, Address: "persona@example.test", Adapter: "email", Enabled: true}, nil
		},
	}
	j := &channelJournal{}
	service := NewService(repo, channelPersonaFunc(func(context.Context, string) (personas.Persona, error) { return channelPersona(), nil }), Registry{"email": EmailAdapter{Source: source}}, j, nil)
	channel, err := service.Bind(context.Background(), ChannelInput{PersonaID: "persona-1", Kind: KindEmail, Address: "persona@example.test", CredentialRef: "secrets://persona/email", Adapter: "email"})
	if err != nil || channel.CredentialRef != "secrets://persona/email" {
		t.Fatalf("Bind() = %#v, %v", channel, err)
	}
	message, err := service.SendMessage(context.Background(), "persona-1", "channel-1", MessageInput{Recipient: "counterparty@example.test", Body: "hello"})
	if err != nil || message.FromAddress != "persona@example.test" {
		t.Fatalf("SendMessage() = %#v, %v", message, err)
	}
	code, err := service.RetrieveCode(context.Background(), "persona-1", "channel-1", "signup")
	if err != nil || code.Value != "654321" {
		t.Fatalf("RetrieveCode() = %#v, %v", code, err)
	}
	_, err = service.SendMessage(context.Background(), "other-persona", "channel-1", MessageInput{Recipient: "counterparty@example.test", Body: "no"})
	if !errors.Is(err, ErrChannelOwnership) {
		t.Fatalf("cross-persona send error = %v", err)
	}
	if len(j.entries) < 3 || j.entries[len(j.entries)-1].Outcome != "refused" {
		t.Fatalf("channel journal = %#v", j.entries)
	}
}

func TestUnavailableAdapterDoesNotFallback(t *testing.T) {
	// [REQ:PSN-P0-006] an unavailable route is a named refusal, never another persona's route.
	repo := FakeRepository{GetFunc: func(context.Context, string) (Channel, error) {
		return Channel{ID: "channel-1", PersonaID: "persona-1", Adapter: "email", Enabled: true}, nil
	}}
	service := NewService(repo, channelPersonaFunc(func(context.Context, string) (personas.Persona, error) { return channelPersona(), nil }), Registry{"email": EmailAdapter{Source: NewUnavailableSource("email")}}, nil, nil)
	_, err := service.RetrieveCode(context.Background(), "persona-1", "channel-1", "signup")
	if !errors.Is(err, ErrAdapterUnavailable) {
		t.Fatalf("unavailable adapter error = %v", err)
	}
}

func TestChannelHealthSurfacesUnreachableOTPRoute(t *testing.T) {
	// [REQ:PSN-P1-005] an unreachable registered OTP route is visible before enrolment starts.
	service := NewService(FakeRepository{
		ListFunc: func(context.Context, string) ([]Channel, error) {
			return []Channel{{ID: "channel-1", PersonaID: "persona-1", Adapter: "email", Enabled: true}}, nil
		},
	}, channelPersonaFunc(func(context.Context, string) (personas.Persona, error) { return channelPersona(), nil }), Registry{"email": EmailAdapter{Source: NewUnavailableSource("email")}}, nil, nil)
	findings, err := service.CheckHealth(context.Background(), "persona-1")
	if err != nil || len(findings) != 1 || findings[0].Code != "otp_route_unreachable" || !findings[0].Blocking {
		t.Fatalf("channel health = %#v, %v", findings, err)
	}
}
