package browsercapability

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"
)

var ErrTabConsentDenied = errors.New("selected-tab consent is not valid for this browser lease")

const (
	maxTabConsentLifetime = 5 * time.Minute
	maxTabConsentRecords  = 4096
)

// TabConsentIssue describes the one tab selected through the installed browser
// bridge. Run and grant identity are supplied by Agent Manager's verified
// request context, never by request JSON.
type TabConsentIssue struct {
	RunID        string
	GrantReceipt string
	Profile      PermissionProfileSpec
	SessionID    string
	LeaseID      string
	OwnerID      string
	ProfileID    string
	TabID        string
	URL          string
	PairingNonce string
	ExpiresAt    time.Time
}

// TabConsentBinding is checked on every attached-tab action. The browser
// owner remains responsible for current URL scope; this authority binds the
// receipt to the run, session lease, profile and selected tab.
type TabConsentBinding struct {
	RunID        string
	GrantReceipt string
	Profile      PermissionProfileSpec
	SessionID    string
	LeaseID      string
	OwnerID      string
	ProfileID    string
	TabID        string
	CurrentURL   string
}

type tabConsentRecord struct {
	issue TabConsentIssue
	value string
}

// TabConsentAuthority is the Agent Manager owner for short-lived selected-tab
// leases. It keeps the receipt in the existing browser permission authority;
// BAS retains only the opaque receipt and cannot mint or broaden one. Process
// restart drops receipts, which intentionally fails closed and requires a new
// extension pairing.
type TabConsentAuthority struct {
	mu             sync.Mutex
	records        map[string]tabConsentRecord
	pairingByNonce map[string]string
	now            func() time.Time
}

func NewTabConsentAuthority() *TabConsentAuthority {
	return &TabConsentAuthority{records: make(map[string]tabConsentRecord), pairingByNonce: make(map[string]string), now: func() time.Time { return time.Now().UTC() }}
}

func (a *TabConsentAuthority) Issue(issue TabConsentIssue) (string, error) {
	if a == nil {
		return "", ErrTabConsentDenied
	}
	issue.RunID = strings.TrimSpace(issue.RunID)
	issue.GrantReceipt = strings.TrimSpace(issue.GrantReceipt)
	issue.SessionID = strings.TrimSpace(issue.SessionID)
	issue.LeaseID = strings.TrimSpace(issue.LeaseID)
	issue.OwnerID = strings.TrimSpace(issue.OwnerID)
	issue.ProfileID = strings.TrimSpace(issue.ProfileID)
	issue.TabID = strings.TrimSpace(issue.TabID)
	issue.URL = strings.TrimSpace(issue.URL)
	issue.PairingNonce = strings.TrimSpace(issue.PairingNonce)
	if issue.RunID == "" || issue.GrantReceipt == "" || issue.SessionID == "" || issue.LeaseID == "" || issue.OwnerID == "" || issue.ProfileID == "" || issue.TabID == "" || issue.PairingNonce == "" {
		return "", fmt.Errorf("%w: incomplete selected-tab binding", ErrTabConsentDenied)
	}
	if issue.Profile.ProfileMode != SessionProfileUserAttached || issue.Profile.ProfileID != issue.ProfileID || (!issue.Profile.View && !issue.Profile.Control) || issue.Profile.Level == ProfileUnrestricted {
		return "", fmt.Errorf("%w: run profile does not authorize selected-tab attachment", ErrTabConsentDenied)
	}
	if err := validateInitialURL(issue.URL, issue.Profile.AllowedOrigins); err != nil {
		return "", fmt.Errorf("%w: selected tab is outside the granted origin scope", ErrTabConsentDenied)
	}
	now := a.now().UTC()
	issue.ExpiresAt = issue.ExpiresAt.UTC()
	if issue.ExpiresAt.IsZero() || !issue.ExpiresAt.After(now) || issue.ExpiresAt.After(now.Add(maxTabConsentLifetime)) {
		return "", fmt.Errorf("%w: selected-tab receipt lifetime is invalid", ErrTabConsentDenied)
	}
	issue.Profile = clonePermissionProfile(issue.Profile)
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("create selected-tab receipt: %w", err)
	}
	receipt := base64.RawURLEncoding.EncodeToString(raw[:])
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pruneLocked(now)
	if len(a.records) >= maxTabConsentRecords {
		return "", errors.New("selected-tab consent authority is at capacity")
	}
	if a.pairingByNonce == nil {
		a.pairingByNonce = make(map[string]string)
	}
	pairingKey := issue.RunID + "\x00" + issue.PairingNonce
	if _, exists := a.pairingByNonce[pairingKey]; exists {
		return "", fmt.Errorf("%w: pairing nonce was already consumed", ErrTabConsentDenied)
	}
	a.records[receipt] = tabConsentRecord{issue: issue, value: receipt}
	a.pairingByNonce[pairingKey] = receipt
	return receipt, nil
}

func (a *TabConsentAuthority) Verify(receipt string, binding TabConsentBinding) error {
	if a == nil {
		return ErrTabConsentDenied
	}
	receipt = strings.TrimSpace(receipt)
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now().UTC()
	a.pruneLocked(now)
	record, ok := a.records[receipt]
	if !ok || record.value != receipt {
		return ErrTabConsentDenied
	}
	issue := record.issue
	if issue.RunID != strings.TrimSpace(binding.RunID) || issue.GrantReceipt != strings.TrimSpace(binding.GrantReceipt) ||
		issue.SessionID != strings.TrimSpace(binding.SessionID) || issue.LeaseID != strings.TrimSpace(binding.LeaseID) ||
		issue.OwnerID != strings.TrimSpace(binding.OwnerID) || issue.ProfileID != strings.TrimSpace(binding.ProfileID) ||
		issue.TabID != strings.TrimSpace(binding.TabID) || !reflect.DeepEqual(issue.Profile, binding.Profile) ||
		!sameOriginAllowed(binding.CurrentURL, binding.Profile.AllowedOrigins) {
		return ErrTabConsentDenied
	}
	return nil
}

func (a *TabConsentAuthority) Revoke(receipt, runID string) error {
	if a == nil {
		return ErrTabConsentDenied
	}
	return a.RevokeSession(receipt, runID, "")
}

func (a *TabConsentAuthority) RevokeSession(receipt, runID, sessionID string) error {
	if a == nil {
		return ErrTabConsentDenied
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	record, ok := a.records[strings.TrimSpace(receipt)]
	if !ok || record.issue.RunID != strings.TrimSpace(runID) || (sessionID != "" && record.issue.SessionID != strings.TrimSpace(sessionID)) {
		return ErrTabConsentDenied
	}
	delete(a.records, strings.TrimSpace(receipt))
	delete(a.pairingByNonce, record.issue.RunID+"\x00"+record.issue.PairingNonce)
	return nil
}

func (a *TabConsentAuthority) pruneLocked(now time.Time) {
	for receipt, record := range a.records {
		if !record.issue.ExpiresAt.After(now) {
			delete(a.records, receipt)
			delete(a.pairingByNonce, record.issue.RunID+"\x00"+record.issue.PairingNonce)
		}
	}
}

func sameOriginAllowed(raw string, origins []string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	return validateInitialURL(raw, origins) == nil
}

func clonePermissionProfile(profile PermissionProfileSpec) PermissionProfileSpec {
	profile.AllowedOrigins = append([]string(nil), profile.AllowedOrigins...)
	profile.CredentialRefs = append([]string(nil), profile.CredentialRefs...)
	profile.ExtensionIDs = append([]string(nil), profile.ExtensionIDs...)
	profile.RequireApprovalFor = append([]string(nil), profile.RequireApprovalFor...)
	if profile.IdentityProfile != nil {
		identity := *profile.IdentityProfile
		profile.IdentityProfile = &identity
	}
	return profile
}
