package channels

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/schedule"

	"persona/internal/journal"
	"persona/internal/personas"
)

var (
	ErrMissingPersona       = errors.New("persona_id is required")
	ErrMissingChannel       = errors.New("channel_id is required")
	ErrInvalidChannel       = errors.New("channel kind and address are required")
	ErrInvalidMessage       = errors.New("recipient and body are required")
	ErrChannelExists        = errors.New("persona already has a controlled channel")
	ErrChannelNotFound      = errors.New("channel not found")
	ErrAdapterUnavailable   = errors.New("requested code adapter is unavailable")
	ErrSenderUnavailable    = errors.New("controlled channel sender is unavailable")
	ErrAdapterNotRegistered = errors.New("requested code adapter is not registered")
	ErrChannelOwnership     = errors.New("channel does not belong to this persona")
)

type Kind string

const (
	KindEmail  Kind = "email"
	KindSMS    Kind = "sms"
	KindDevice Kind = "device"
)

type Channel struct {
	ID            string
	PersonaID     string
	Kind          Kind
	Address       string
	CredentialRef string
	Adapter       string
	Enabled       bool
	CreatedAt     time.Time
}

type Code struct {
	Value     string
	ExpiresAt time.Time
	Adapter   string
}

type Message struct {
	ID          string
	FromAddress string
}

type Adapter interface {
	Name() string
	Check(context.Context, Channel) error
	Send(context.Context, Channel, MessageInput) (Message, error)
	Retrieve(context.Context, Channel, string) (Code, error)
}

type AdapterRegistry interface{ Adapter(string) (Adapter, bool) }

type Registry map[string]Adapter

func (r Registry) Adapter(name string) (Adapter, bool) { a, ok := r[name]; return a, ok }

type CodeSource interface {
	Retrieve(context.Context, Channel, string) (Code, error)
}

type HealthSource interface {
	Check(context.Context, Channel) error
}

type MessageInput struct {
	Recipient string
	Subject   string
	Body      string
}

type MessageSource interface {
	Send(context.Context, Channel, MessageInput) (Message, error)
}

type EmailAdapter struct{ Source CodeSource }

func (a EmailAdapter) Name() string { return "email" }
func (a EmailAdapter) Check(ctx context.Context, c Channel) error {
	return sourceCheck(a.Source, ctx, c)
}

func (a EmailAdapter) Retrieve(ctx context.Context, c Channel, purpose string) (Code, error) {
	return sourceRetrieve(a.Source, ctx, c, purpose)
}

func (a EmailAdapter) Send(ctx context.Context, c Channel, in MessageInput) (Message, error) {
	return sourceSend(a.Source, ctx, c, in)
}

type SMSAdapter struct{ Source CodeSource }

func (a SMSAdapter) Name() string                               { return "sms" }
func (a SMSAdapter) Check(ctx context.Context, c Channel) error { return sourceCheck(a.Source, ctx, c) }
func (a SMSAdapter) Retrieve(ctx context.Context, c Channel, purpose string) (Code, error) {
	return sourceRetrieve(a.Source, ctx, c, purpose)
}

func (a SMSAdapter) Send(ctx context.Context, c Channel, in MessageInput) (Message, error) {
	return sourceSend(a.Source, ctx, c, in)
}

type DeviceAdapter struct{ Source CodeSource }

func (a DeviceAdapter) Name() string { return "device" }
func (a DeviceAdapter) Check(ctx context.Context, c Channel) error {
	return sourceCheck(a.Source, ctx, c)
}

func (a DeviceAdapter) Retrieve(ctx context.Context, c Channel, purpose string) (Code, error) {
	return sourceRetrieve(a.Source, ctx, c, purpose)
}

func (a DeviceAdapter) Send(ctx context.Context, c Channel, in MessageInput) (Message, error) {
	return sourceSend(a.Source, ctx, c, in)
}

type unavailableSource struct{ name string }

func (s unavailableSource) Check(context.Context, Channel) error {
	return fmt.Errorf("%w: %s", ErrAdapterUnavailable, s.name)
}

func (s unavailableSource) Retrieve(context.Context, Channel, string) (Code, error) {
	return Code{}, fmt.Errorf("%w: %s", ErrAdapterUnavailable, s.name)
}
func NewUnavailableSource(name string) CodeSource { return unavailableSource{name: name} }

type StaticSource struct {
	Code Code
	Err  error
}

func (s StaticSource) Retrieve(context.Context, Channel, string) (Code, error) { return s.Code, s.Err }
func (s StaticSource) Check(context.Context, Channel) error                    { return s.Err }

func sourceCheck(source CodeSource, ctx context.Context, channel Channel) error {
	if source == nil {
		return ErrAdapterUnavailable
	}
	if checker, ok := source.(HealthSource); ok {
		return checker.Check(ctx, channel)
	}
	return nil
}

func sourceRetrieve(source CodeSource, ctx context.Context, channel Channel, purpose string) (Code, error) {
	if source == nil {
		return Code{}, ErrAdapterUnavailable
	}
	code, err := source.Retrieve(ctx, channel, purpose)
	if err != nil {
		return Code{}, err
	}
	if strings.TrimSpace(code.Value) == "" || code.ExpiresAt.IsZero() {
		return Code{}, errors.New("adapter returned an incomplete one-time code")
	}
	return code, nil
}

func sourceSend(source CodeSource, ctx context.Context, channel Channel, in MessageInput) (Message, error) {
	if strings.TrimSpace(in.Recipient) == "" || strings.TrimSpace(in.Body) == "" {
		return Message{}, ErrInvalidMessage
	}
	sender, ok := source.(MessageSource)
	if !ok {
		return Message{}, ErrSenderUnavailable
	}
	message, err := sender.Send(ctx, channel, in)
	if err != nil {
		return Message{}, err
	}
	if strings.TrimSpace(message.ID) == "" {
		return Message{}, errors.New("adapter returned no message id")
	}
	if strings.TrimSpace(message.FromAddress) == "" {
		message.FromAddress = channel.Address
	}
	return message, nil
}

type PersonaResolver interface {
	Get(context.Context, string) (personas.Persona, error)
}

type Service interface {
	Bind(context.Context, ChannelInput) (Channel, error)
	List(context.Context, string) ([]Channel, error)
	CheckHealth(context.Context, string) ([]personas.HealthFinding, error)
	SendMessage(context.Context, string, string, MessageInput) (Message, error)
	RetrieveCode(context.Context, string, string, string) (Code, error)
}

type ChannelInput struct {
	PersonaID     string
	Kind          Kind
	Address       string
	CredentialRef string
	Adapter       string
}

type service struct {
	repo     Repository
	personas PersonaResolver
	adapters AdapterRegistry
	journal  journal.Service
	clock    schedule.Clock
}

func NewService(repo Repository, personaResolver PersonaResolver, adapters AdapterRegistry, actionJournal journal.Service, clock schedule.Clock) Service {
	if clock == nil {
		clock = schedule.System()
	}
	return &service{repo: repo, personas: personaResolver, adapters: adapters, journal: actionJournal, clock: clock}
}

var _ Service = (*service)(nil)

func (s *service) Bind(ctx context.Context, in ChannelInput) (Channel, error) {
	if strings.TrimSpace(in.PersonaID) == "" {
		return Channel{}, ErrMissingPersona
	}
	if in.Kind != KindEmail && in.Kind != KindSMS && in.Kind != KindDevice || strings.TrimSpace(in.Address) == "" {
		return Channel{}, ErrInvalidChannel
	}
	if _, err := s.personas.Get(ctx, in.PersonaID); err != nil {
		return Channel{}, err
	}
	channels, err := s.repo.List(ctx, in.PersonaID)
	if err != nil {
		return Channel{}, err
	}
	for _, existing := range channels {
		if existing.Enabled {
			return Channel{}, ErrChannelExists
		}
	}
	adapter := strings.TrimSpace(in.Adapter)
	if adapter == "" {
		adapter = string(in.Kind)
	}
	if _, ok := s.adapters.Adapter(adapter); !ok {
		return Channel{}, ErrAdapterNotRegistered
	}
	channel, err := s.repo.Create(ctx, Channel{PersonaID: in.PersonaID, Kind: in.Kind, Address: strings.TrimSpace(in.Address), CredentialRef: strings.TrimSpace(in.CredentialRef), Adapter: adapter, Enabled: true})
	if err == nil {
		s.record(ctx, channel, "channel_bound", map[string]string{"adapter": adapter}, "granted")
	}
	return channel, err
}

func (s *service) List(ctx context.Context, personaID string) ([]Channel, error) {
	if strings.TrimSpace(personaID) == "" {
		return nil, ErrMissingPersona
	}
	return s.repo.List(ctx, personaID)
}

func (s *service) CheckHealth(ctx context.Context, personaID string) ([]personas.HealthFinding, error) {
	if strings.TrimSpace(personaID) == "" {
		return nil, ErrMissingPersona
	}
	channels, err := s.repo.List(ctx, personaID)
	if err != nil {
		return nil, err
	}
	findings := make([]personas.HealthFinding, 0)
	for _, channel := range channels {
		adapter, ok := s.adapters.Adapter(channel.Adapter)
		if !ok {
			findings = append(findings, personas.HealthFinding{Code: "otp_route_unreachable", Message: "OTP adapter " + channel.Adapter + " is not registered.", Blocking: true})
			continue
		}
		if err := adapter.Check(ctx, channel); err != nil {
			findings = append(findings, personas.HealthFinding{Code: "otp_route_unreachable", Message: err.Error(), Blocking: true})
		}
	}
	return findings, nil
}

func (s *service) SendMessage(ctx context.Context, personaID, channelID string, in MessageInput) (Message, error) {
	if strings.TrimSpace(personaID) == "" {
		return Message{}, ErrMissingPersona
	}
	if strings.TrimSpace(channelID) == "" {
		return Message{}, ErrMissingChannel
	}
	if strings.TrimSpace(in.Recipient) == "" || strings.TrimSpace(in.Body) == "" {
		return Message{}, ErrInvalidMessage
	}
	channel, err := s.repo.Get(ctx, channelID)
	if err != nil {
		return Message{}, err
	}
	if channel.PersonaID != personaID {
		s.record(ctx, channel, "message_send_refused", map[string]string{"constraint": "channel_ownership"}, "refused")
		return Message{}, ErrChannelOwnership
	}
	adapter, ok := s.adapters.Adapter(channel.Adapter)
	if !ok {
		s.record(ctx, channel, "message_send_refused", map[string]string{"constraint": "adapter_not_registered"}, "refused")
		return Message{}, ErrAdapterNotRegistered
	}
	message, err := adapter.Send(ctx, channel, in)
	if err != nil {
		s.record(ctx, channel, "message_send_refused", map[string]string{"constraint": "sender_unavailable"}, "refused")
		return Message{}, err
	}
	s.record(ctx, channel, "message_sent", map[string]string{"message_id": message.ID, "recipient": in.Recipient}, "granted")
	return message, nil
}

func (s *service) RetrieveCode(ctx context.Context, personaID, channelID, purpose string) (Code, error) {
	if strings.TrimSpace(personaID) == "" {
		return Code{}, ErrMissingPersona
	}
	if strings.TrimSpace(channelID) == "" {
		return Code{}, ErrMissingChannel
	}
	channel, err := s.repo.Get(ctx, channelID)
	if err != nil {
		return Code{}, err
	}
	if channel.PersonaID != personaID {
		s.record(ctx, Channel{PersonaID: personaID, ID: channelID}, "code_retrieval_refused", map[string]string{"constraint": "channel_ownership"}, "refused")
		return Code{}, ErrChannelOwnership
	}
	adapter, ok := s.adapters.Adapter(channel.Adapter)
	if !ok {
		s.record(ctx, channel, "code_retrieval_refused", map[string]string{"constraint": "adapter_not_registered"}, "refused")
		return Code{}, ErrAdapterNotRegistered
	}
	code, err := adapter.Retrieve(ctx, channel, purpose)
	if err != nil {
		s.record(ctx, channel, "code_retrieval_refused", map[string]string{"constraint": "adapter_unavailable"}, "refused")
		return Code{}, err
	}
	s.record(ctx, channel, "code_retrieved", map[string]string{"adapter": adapter.Name(), "expires_at": code.ExpiresAt.UTC().Format(time.RFC3339Nano)}, "granted")
	return code, nil
}

func (s *service) record(ctx context.Context, channel Channel, verb string, details map[string]string, outcome string) {
	if s.journal == nil {
		return
	}
	entry := journal.Entry{PersonaID: channel.PersonaID, Actor: "agent", Verb: verb, Outcome: outcome, Details: details}
	verified := provenance.FromContext(ctx)
	if verified.IsVerifiedAgent() {
		entry.RunID = verified.RunID
		entry.AuthorisingHuman = verified.Subject
	}
	_, _ = s.journal.Append(ctx, entry)
}

// Schema returns the controlled-channel SQL contribution.
func Schema() string { return schemaSQL }
