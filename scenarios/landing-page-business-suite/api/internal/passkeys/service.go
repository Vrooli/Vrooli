package passkeys

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type Service struct {
	RP *webauthn.WebAuthn
}

func NewService(config Config) (*Service, error) {
	rp, err := webauthn.New(&webauthn.Config{
		RPID: config.RPID, RPDisplayName: config.RPDisplayName, RPOrigins: config.RPOrigins,
		AttestationPreference:  protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{ResidentKey: protocol.ResidentKeyRequirementRequired, UserVerification: protocol.VerificationRequired},
	})
	if err != nil {
		return nil, fmt.Errorf("create WebAuthn relying party: %w", err)
	}
	return &Service{RP: rp}, nil
}

func (s *Service) BeginRegistration(user User) (options []byte, session *webauthn.SessionData, err error) {
	if s == nil || s.RP == nil {
		return nil, nil, fmt.Errorf("WebAuthn service unavailable")
	}
	creation, session, err := s.RP.BeginRegistration(user)
	if err != nil {
		return nil, nil, err
	}
	options, err = json.Marshal(creation)
	return options, session, err
}

func (s *Service) BeginAuthentication() (options []byte, session *webauthn.SessionData, err error) {
	if s == nil || s.RP == nil {
		return nil, nil, fmt.Errorf("WebAuthn service unavailable")
	}
	assertion, session, err := s.RP.BeginDiscoverableLogin()
	if err != nil {
		return nil, nil, err
	}
	options, err = json.Marshal(assertion)
	return options, session, err
}

func (s *Service) BeginUserAuthentication(user User) (options []byte, session *webauthn.SessionData, err error) {
	if s == nil || s.RP == nil {
		return nil, nil, fmt.Errorf("WebAuthn service unavailable")
	}
	assertion, session, err := s.RP.BeginLogin(user)
	if err != nil {
		return nil, nil, err
	}
	options, err = json.Marshal(assertion)
	return options, session, err
}

func (s *Service) FinishRegistration(user User, session webauthn.SessionData, credentialJSON []byte, origin string) (*webauthn.Credential, error) {
	if s == nil || s.RP == nil {
		return nil, fmt.Errorf("WebAuthn service unavailable")
	}
	request, err := http.NewRequest(http.MethodPost, origin+"/finish", bytes.NewReader([]byte(credentialJSON)))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	return s.RP.FinishRegistration(user, session, request)
}

func (s *Service) FinishLogin(user User, session webauthn.SessionData, credentialJSON []byte, origin string) (*webauthn.Credential, error) {
	if s == nil || s.RP == nil {
		return nil, fmt.Errorf("WebAuthn service unavailable")
	}
	request, err := http.NewRequest(http.MethodPost, origin+"/finish", bytes.NewReader([]byte(credentialJSON)))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	return s.RP.FinishLogin(user, session, request)
}

func (s *Service) FinishDiscoverableLogin(resolve webauthn.DiscoverableUserHandler, session webauthn.SessionData, credentialJSON, origin string) (webauthn.User, *webauthn.Credential, error) {
	if s == nil || s.RP == nil {
		return nil, nil, fmt.Errorf("WebAuthn service unavailable")
	}
	request, err := http.NewRequest(http.MethodPost, origin+"/finish", bytes.NewReader([]byte(credentialJSON)))
	if err != nil {
		return nil, nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	return s.RP.FinishPasskeyLogin(resolve, session, request)
}
