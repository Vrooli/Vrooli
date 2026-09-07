package main

import (
	"fmt"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// authoritySeedCustodian adapts the control-plane credential authority to the
// MFA store. Account IDs never become credential paths; they are normalized
// into a namespaced logical identity, while each enrollment slot remains one
// authority field.
type authoritySeedCustodian struct {
	authority *credentialauthority.Authority
}

func newAuthoritySeedCustodian(authority *credentialauthority.Authority) (*authoritySeedCustodian, error) {
	if authority == nil {
		return nil, fmt.Errorf("credential authority is required")
	}
	return &authoritySeedCustodian{authority: authority}, nil
}

func (c *authoritySeedCustodian) address(userID, slot string) (credentialauthority.Identity, string, error) {
	userID = strings.TrimSpace(strings.ToLower(userID))
	slot = strings.TrimSpace(slot)
	if userID == "" || slot == "" || strings.ContainsAny(slot, "/\\") {
		return "", "", fmt.Errorf("invalid MFA seed custody address")
	}
	identity, err := credentialauthority.ParseIdentity("scenario-authenticator/mfa/" + userID)
	if err != nil {
		return "", "", err
	}
	return identity, slot, nil
}

func (c *authoritySeedCustodian) Put(userID, slot, secret string) error {
	identity, field, err := c.address(userID, slot)
	if err != nil {
		return err
	}
	return c.authority.Put(identity, field, secret)
}

func (c *authoritySeedCustodian) Resolve(userID, slot string) (string, error) {
	identity, field, err := c.address(userID, slot)
	if err != nil {
		return "", err
	}
	return c.authority.Require(identity, field)
}

func (c *authoritySeedCustodian) Delete(userID, slot string) error {
	identity, field, err := c.address(userID, slot)
	if err != nil {
		return err
	}
	return c.authority.Delete(identity, field)
}
