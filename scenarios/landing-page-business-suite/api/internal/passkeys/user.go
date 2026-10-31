package passkeys

import "github.com/go-webauthn/webauthn/webauthn"

// User adapts LPBS's existing user identity to the WebAuthn relying-party
// contract. The opaque handle is the authorization key; email is display-only.
type User struct {
	ID          string
	Handle      []byte
	Email       string
	Credentials []webauthn.Credential
}

func (u User) WebAuthnID() []byte                         { return u.Handle }
func (u User) WebAuthnName() string                       { return u.Email }
func (u User) WebAuthnDisplayName() string                { return u.Email }
func (u User) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }
