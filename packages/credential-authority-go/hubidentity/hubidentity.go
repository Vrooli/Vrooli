// Package hubidentity gives a scenario the service identities it needs at
// Notification Hub, from the credentials the operator issues with
// `scenario-authenticator service-principals sync --apply`:
//
//   - the ask-reader (api-core askreader), with which a decision consumer reads an
//     ask back before it applies an operator's answer;
//   - the sender (api-core hubsender), with which it sends the operator a notice
//     as itself.
//
// Every constructor is safe before the operator has issued anything and after: the
// credential is read when it is used, never when the value is built, so a scenario
// needs no restart once it is issued. Until then the ask reader reports
// askreader.ErrNotIssued, which the resolution webhook counts and logs as an
// unverified resolution, and the notice sender reports servicetoken.ErrNotIssued.
package hubidentity

import (
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/hubsender"
	"github.com/vrooli/api-core/operatordecision"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// NewAskReader reads asks back from the Hub as source's ask-reader principal.
func NewAskReader(source string) *askreader.Reader {
	return askreader.NewReader(source, AskReaderCredential(source))
}

// NewAskReaderTokens is the ask-reader token client of source, for a scenario that
// presents the token itself.
func NewAskReaderTokens(source string) *askreader.Tokens {
	return askreader.NewTokens(source, AskReaderCredential(source))
}

// AskReaderCredential reads source's ask-reader credential.
func AskReaderCredential(source string) askreader.CredentialSource {
	return credentialauthority.ServiceCredential(askreader.CredentialIdentity(source), askreader.CredentialField)
}

// NewNoticeSender sends notices to the operator as source's sender principal.
func NewNoticeSender(source string) *hubsender.Client {
	return hubsender.NewClient(source, credentialauthority.ServiceCredential(hubsender.CredentialIdentity(source), hubsender.CredentialField))
}

// NewRefusalNotifier is how source tells the operator that a resolution of one of
// its asks was refused. Build one per webhook; it keeps what it has already
// reported. A scenario does not call this itself: WebhookConfig wires it.
func NewRefusalNotifier(source string) *operatordecision.RefusalNotifier {
	return operatordecision.NewRefusalNotifier(operatordecision.RefusalNoticeConfig{Source: source, Sender: NewNoticeSender(source)})
}

// WebhookConfig completes a decision consumer's resolution webhook with its Hub
// identities: the ask reader that confirms every answer with the Hub before it is
// applied, and the notifier that tells the operator when one is refused. cfg names
// the scenario (Source, required) and its Secret and Effects; a Reader or OnRefused
// the caller already set is kept, which is how a test substitutes a Hub.
//
// Every scenario builds its webhook as
// operatordecision.NewWebhook(hubidentity.WebhookConfig(cfg)), so none can ship
// without the confirmation or the notice; TestEveryDecisionWebhookIsWiredToTheHub
// enforces it.
func WebhookConfig(cfg operatordecision.WebhookConfig) operatordecision.WebhookConfig {
	if cfg.Source == "" {
		panic("hubidentity.WebhookConfig: the scenario's Source is required")
	}
	if cfg.Reader == nil {
		cfg.Reader = NewAskReader(cfg.Source)
	}
	if cfg.OnRefused == nil {
		cfg.OnRefused = NewRefusalNotifier(cfg.Source).Notify
	}
	return cfg
}
