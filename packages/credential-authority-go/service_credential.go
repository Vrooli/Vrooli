package credentialauthority

import "context"

// ServiceCredential reads the credential scenario-authenticator issued for a
// service principal: the credential authority entry at address (the identity the
// issuing command prints) and field. It is the CredentialSource of the api-core
// service-principal clients (askreader, hubsender, eventpublisher).
//
// The authority is asked each time the value is needed. A credential the operator
// issues while the scenario runs is found by the next call, and a missing one is
// not remembered, so nothing needs a restart after
// `scenario-authenticator service-principals sync --apply`. The value is returned
// only to the caller exchanging it for a token; nothing here logs it.
func ServiceCredential(address, field string) func(context.Context) (string, error) {
	return ServiceCredentialFrom(Default, address, field)
}

// ServiceCredentialFrom is ServiceCredential over an authority the caller opens,
// for a store other than the host default.
func ServiceCredentialFrom(open func() (*Authority, error), address, field string) func(context.Context) (string, error) {
	return func(context.Context) (string, error) {
		authority, err := open()
		if err != nil {
			return "", err
		}
		identity, err := ParseIdentity(address)
		if err != nil {
			return "", err
		}
		return authority.Require(identity, field)
	}
}
