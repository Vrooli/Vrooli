package setup

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"compute-manager/internal/module"
	"compute-manager/internal/provider"
	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/operatordecision"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/setup"
	setupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/setup/setup_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const decisionKind = "provider-setup"

type TokenResolver func(context.Context, string) (string, error)
type CredentialConfiguredResolver func(context.Context, string) (bool, error)

type service struct {
	setupconnect.UnimplementedSetupServiceHandler
	repo       Repository
	resolve    TokenResolver
	configured CredentialConfiguredResolver
	publisher  operatordecision.Publisher
	client     *http.Client
	now        func() time.Time
}

func NewModule(database db, resolve TokenResolver, publisher operatordecision.Publisher) module.Module {
	return NewModuleWithCredentialStatus(database, resolve, nil, publisher)
}

func NewModuleWithCredentialStatus(database db, resolve TokenResolver, configured CredentialConfiguredResolver, publisher operatordecision.Publisher) module.Module {
	s := &service{repo: Repository{DB: database}, resolve: resolve, configured: configured, publisher: publisher, client: &http.Client{Timeout: 20 * time.Second}, now: time.Now}
	path, handler := setupconnect.NewSetupServiceHandler(s)
	return module.Module{Name: "setup", Mount: func(r *mux.Router) { r.PathPrefix(path).Handler(handler) }, Endpoints: Endpoints}
}

func (s *service) ListProviders(context.Context, *connect.Request[setupv1.ListProvidersRequest]) (*connect.Response[setupv1.ListProvidersResponse], error) {
	response := &setupv1.ListProvidersResponse{}
	for _, item := range Catalogue() {
		response.Providers = append(response.Providers, providerProto(item))
	}
	return connect.NewResponse(response), nil
}

func (s *service) BeginSetupStep(ctx context.Context, req *connect.Request[setupv1.BeginSetupStepRequest]) (*connect.Response[setupv1.BeginSetupStepResponse], error) {
	item, step := findStep(req.Msg.GetProviderId(), req.Msg.GetStepId())
	if item == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("provider setup step not found"))
	}
	if err := s.requirePreviousSteps(ctx, *item, step.ID); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	prior, err := s.repo.Step(ctx, item.ID, step.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if prior.State == "complete" {
		return connect.NewResponse(&setupv1.BeginSetupStepResponse{AskId: prior.AskID, State: prior.State}), nil
	}
	if prior.AskID == "" || prior.State != "pending" {
		askID, err := randomID()
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		prior, err = s.repo.StartStep(ctx, StepRecord{ProviderID: item.ID, StepID: step.ID, AskID: askID, EventTime: s.now().UTC(), State: "pending", UpdatedAt: s.now().UTC()})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if s.publisher == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("Notification Hub decision delivery is unavailable"))
	}
	facts := stepFacts(*item, *step)
	event := operatordecision.BuildEvent("compute-manager", decisionKind, prior.EventTime, facts)
	if err := s.publisher.PublishDomainEvent(ctx, event); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("could not deliver the setup decision to Notification Hub"))
	}
	return connect.NewResponse(&setupv1.BeginSetupStepResponse{AskId: prior.AskID, State: prior.State}), nil
}

func (s *service) GetReadiness(ctx context.Context, req *connect.Request[setupv1.GetReadinessRequest]) (*connect.Response[setupv1.GetReadinessResponse], error) {
	item := findProvider(req.Msg.GetProviderId())
	if item == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("provider not found"))
	}
	ready, err := s.readiness(ctx, *item)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&setupv1.GetReadinessResponse{Readiness: ready}), nil
}

func (s *service) VerifyProvider(ctx context.Context, req *connect.Request[setupv1.VerifyProviderRequest]) (*connect.Response[setupv1.VerifyProviderResponse], error) {
	item := findProvider(req.Msg.GetProviderId())
	if item == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("provider not found"))
	}
	if item.ID == "akash" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Akash signer verification and chain probes are not available; credential entry alone cannot enable them"))
	}
	if s.resolve == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("credential authority is unavailable"))
	}
	secret, err := s.resolve(ctx, item.CredentialField)
	if err != nil || secret == "" {
		ready, _ := s.readiness(ctx, *item)
		return connect.NewResponse(&setupv1.VerifyProviderResponse{Readiness: ready}), nil
	}
	base, vendor := "https://api.hetzner.cloud", "Hetzner"
	if item.ID == "digitalocean" {
		base, vendor = "https://api.digitalocean.com", "DigitalOcean"
	}
	all := true
	for _, probe := range item.Probes {
		var output map[string]any
		err := provider.DoJSONRequest(ctx, func(context.Context) (string, error) { return secret, nil }, s.client, base, base, vendor, http.MethodGet, probe.Path, nil, &output)
		state := Authenticated
		verified := err == nil
		detail := "Provider API accepted the read-only request."
		if err != nil {
			all = false
			detail = "Probe unknown; the provider API did not verify this permission."
		}
		if probe.ID == "servers" || probe.ID == "droplets" {
			state = AccountUsable
		}
		e := Evidence{ID: "probe:" + probe.ID, State: state, Verified: verified, Detail: detail, CheckedAt: s.now().UTC(), SourceURL: base + probe.Path}
		if putErr := s.repo.PutEvidence(ctx, item.ID, e); putErr != nil {
			return nil, connect.NewError(connect.CodeInternal, putErr)
		}
	}
	if all && req.Msg.GetRegion() != "" && req.Msg.GetSize() != "" {
		fact, priceErr := FetchPriceFact(ctx, s.client, secret, item.ID, req.Msg.GetRegion(), req.Msg.GetSize(), true, s.now())
		if priceErr == nil {
			if err := s.repo.PutPrice(ctx, fact); err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			e := Evidence{ID: "price:" + fact.Region + ":" + fact.Size, State: OfferAvailable, Verified: true, Detail: "Fresh provider quote facts are available.", CheckedAt: fact.ObservedAt, SourceURL: fact.SourceURL, SourceVersion: fact.SourceVersion}
			if err := s.repo.PutEvidence(ctx, item.ID, e); err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		} else {
			e := Evidence{ID: "price:" + req.Msg.GetRegion() + ":" + req.Msg.GetSize(), State: OfferAvailable, Verified: false, Detail: "Provider price remains unknown; no quote will be used.", CheckedAt: s.now().UTC()}
			if err := s.repo.PutEvidence(ctx, item.ID, e); err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		}
	}
	ready, err := s.readiness(ctx, *item)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&setupv1.VerifyProviderResponse{Readiness: ready}), nil
}

func (s *service) readiness(ctx context.Context, item Provider) (*setupv1.ProviderReadiness, error) {
	steps, err := s.repo.Steps(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	evidence, err := s.repo.Evidence(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	prices, err := s.repo.Prices(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	byStep := map[string]StepRecord{}
	var active StepRecord
	for _, step := range steps {
		byStep[step.StepID] = step
		if step.State == "pending" {
			active = step
		}
	}
	current := NeedsConsoleSteps
	credentialPresent := false
	credentialStatus := "missing"
	credentialStatusKnown := false
	if s.configured != nil {
		configured, statusErr := s.configured(ctx, item.CredentialField)
		credentialStatusKnown = statusErr == nil
		credentialPresent = statusErr == nil && configured
		if statusErr != nil {
			credentialStatus = "status_unavailable"
		} else if configured {
			credentialStatus = "configured"
		}
	}
	consoleComplete := len(steps) >= len(item.ConsoleSteps)
	for _, step := range item.ConsoleSteps {
		if byStep[step.ID].State != "complete" {
			consoleComplete = false
		}
	}
	if item.ID == "akash" {
		current = CredentialNeeded
		if !credentialStatusKnown {
			credentialStatus = "status_unavailable"
		} else if credentialPresent {
			credentialStatus = "ready_for_authorized_verification"
		} else if active.StepID == "wallet" && active.State == "pending" {
			credentialStatus = "awaiting_secure_entry"
		} else {
			credentialStatus = "not_configured"
		}
	} else if consoleComplete {
		current = CredentialNeeded
		if credentialPresent {
			current = Authenticated
			verified := map[State]bool{}
			for _, e := range evidence {
				if e.Verified && (e.State != AccountUsable || (!e.CheckedAt.After(s.now()) && s.now().Sub(e.CheckedAt) <= 5*time.Minute)) {
					verified[e.State] = true
				}
			}
			freshOffer := false
			for _, fact := range prices {
				if s.now().Before(fact.ExpiresAt) && fact.SourceURL != "" && fact.SourceVersion != "" {
					freshOffer = true
					break
				}
			}
			if verified[AccountUsable] {
				current = AccountUsable
			}
			if freshOffer {
				current = OfferAvailable
			}
			if verified[PaymentValid] && current == OfferAvailable {
				current = PaymentValid
			}
		}
	}
	if !strings.HasPrefix(item.ID, "akash") {
		if credentialPresent {
			credentialStatus = "configured"
		} else if !credentialStatusKnown {
			credentialStatus = "missing"
		}
	}
	out := &setupv1.ProviderReadiness{ProviderId: item.ID, Current: string(current), CredentialStatus: credentialStatus, PurchaseEnabled: false}
	if active.AskID != "" {
		out.AskId = active.AskID
		out.ActiveStep = active.StepID
	}
	for _, step := range steps {
		if step.Answer != "" {
			out.LastAnswer = step.Answer
		}
	}
	for _, e := range evidence {
		out.Evidence = append(out.Evidence, &setupv1.ReadinessEvidence{Id: e.ID, State: string(e.State), Verified: e.Verified, Detail: e.Detail, CheckedAt: timestamppb.New(e.CheckedAt), SourceUrl: e.SourceURL, SourceVersion: e.SourceVersion})
	}
	for _, fact := range prices {
		if fact.SourceURL == "" || fact.SourceVersion == "" {
			continue
		}
		out.Prices = append(out.Prices, &setupv1.PriceFact{Provider: fact.Provider, OfferId: fact.OfferID, Region: fact.Region, Size: fact.Size, Currency: fact.Currency, ServerMinorPerMonth: fact.ServerMinor, Ipv4MinorPerMonth: fact.IPv4Minor, TaxBasis: fact.TaxBasis, SourceUrl: fact.SourceURL, SourceVersion: fact.SourceVersion, ObservedAt: timestamppb.New(fact.ObservedAt), ExpiresAt: timestamppb.New(fact.ExpiresAt)})
	}
	return out, nil
}

func (s *service) requirePreviousSteps(ctx context.Context, item Provider, stepID string) error {
	for _, step := range item.ConsoleSteps {
		if step.ID == stepID {
			return nil
		}
		record, err := s.repo.Step(ctx, item.ID, step.ID)
		if err != nil {
			return err
		}
		if record.State != "complete" {
			return fmt.Errorf("complete the %s provider setup decision first", step.Title)
		}
	}
	return errors.New("provider setup step is invalid")
}

func stepFacts(item Provider, step ConsoleStep) operatordecision.Facts {
	second := "Not yet — I need to finish this on the provider website"
	return operatordecision.Facts{Subject: item.Name + " setup", Question: step.Decision, Options: []operatordecision.Option{{Key: "complete", Label: "Yes, done"}, {Key: "not_yet", Label: second}}, Urgency: "normal", BigPicture: "Compute Manager is recording provider setup progress so later compute actions can be guided from your phone.", TechnicalDetails: "Open " + step.URL + " to complete this provider website step. This decision records only your confirmation. It does not store payment details, create a server, or grant purchase approval.", ContextURL: "/embedded/compute-manager/setup", SubjectKey: "compute-manager:provider-setup:" + item.ID + ":" + step.ID, Correlation: map[string]string{"provider_id": item.ID, "step_id": step.ID}}
}

func findProvider(id string) *Provider {
	for i := range catalogue {
		if catalogue[i].ID == id {
			return &catalogue[i]
		}
	}
	return nil
}
func findStep(providerID, stepID string) (*Provider, *ConsoleStep) {
	item := findProvider(providerID)
	if item == nil {
		return nil, nil
	}
	for i := range item.ConsoleSteps {
		if item.ConsoleSteps[i].ID == stepID {
			return item, &item.ConsoleSteps[i]
		}
	}
	return item, nil
}
func providerProto(item Provider) *setupv1.Provider {
	out := &setupv1.Provider{Id: item.ID, Name: item.Name, ConsoleUrl: item.ConsoleURL, CredentialLogicalId: item.CredentialLogicalID, CredentialField: item.CredentialField, CredentialLabel: item.CredentialLabel, CredentialScopes: item.CredentialScopes, BillingFacts: item.BillingFacts}
	for _, s := range item.ConsoleSteps {
		out.ConsoleSteps = append(out.ConsoleSteps, &setupv1.ConsoleStep{Id: s.ID, Title: s.Title, Instructions: s.Instructions, Url: s.URL})
	}
	for _, p := range item.Probes {
		out.Probes = append(out.Probes, &setupv1.Probe{Id: p.ID, Method: p.Method, Path: p.Path, Permission: p.Permission, ReadOnly: p.ReadOnly})
	}
	for _, offer := range item.Offers {
		out.Offers = append(out.Offers, &setupv1.Offer{Id: offer.ID, Label: offer.Label, Cpu: offer.CPU, Memory: offer.Memory, Regions: offer.Regions})
	}
	return out
}
func randomID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// ResolutionEffect records only the Hub's read-back-confirmed menu answer;
// free-form notes never enter readiness state.
type ResolutionEffect struct {
	Repo Repository
	Now  func() time.Time
}

func (r ResolutionEffect) Handles(event string) bool {
	return event == operatordecision.DecisionRequestedEventType("compute-manager", decisionKind)
}
func (r ResolutionEffect) Apply(ctx context.Context, res operatordecision.DecisionResolution) (bool, error) {
	providerID, stepID := res.Correlation["provider_id"], res.Correlation["step_id"]
	if providerID == "" || stepID == "" {
		return false, nil
	}
	record, err := r.Repo.Step(ctx, providerID, stepID)
	if err != nil {
		return true, err
	}
	if record.AskID == "" || record.State != "pending" {
		return true, nil
	}
	answer := strings.TrimSpace(res.Answer)
	state := "needs_action"
	verified := false
	if (res.State == operatordecision.StateAnswered || res.State == operatordecision.StateDefaulted) && answer == "complete" {
		state = "complete"
		verified = true
	}
	if res.State == operatordecision.StateWithdrawn || res.State == operatordecision.StateDismissed || res.State == operatordecision.StateExpired {
		state = "closed"
	}
	now := time.Now().UTC()
	if r.Now != nil {
		now = r.Now().UTC()
	}
	record.State, record.Answer, record.UpdatedAt = state, answer, now
	if err := r.Repo.ResolveStep(ctx, record); err != nil {
		return true, err
	}
	if state != "closed" {
		detail := "The operator has not completed the provider website step."
		if verified {
			detail = "The operator confirmed this provider website step is complete."
		}
		e := Evidence{ID: "console:" + stepID, State: NeedsConsoleSteps, Verified: verified, Detail: detail, CheckedAt: now}
		if stepID == "payment" {
			e.State = PaymentValid
		}
		if err := r.Repo.PutEvidence(ctx, providerID, e); err != nil {
			return true, err
		}
	}
	return true, nil
}

var _ operatordecision.DomainEffect = ResolutionEffect{}
