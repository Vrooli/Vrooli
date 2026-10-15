// Package finiteinstallation loads protected role-owned finite startup data.
// Loading never enrolls identities, publishes listeners, captures witnesses or starts units.
package finiteinstallation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/vrooli/api-core/effortauthority"
	ca "github.com/vrooli/vrooli/packages/credential-authority-go"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
)

const ProtectedRoot = "/etc/vrooli/finite-startup"
const TLSField = "finite-startup-tls-v1"

type Bundle struct {
	// DevelopmentSetup is complete nonsecret owner-selected setup, not an approval.
	// Only disabled acceptance startup may compose it. Its owner validates every byte.
	DevelopmentSetup json.RawMessage                                               `json:"developmentSetup,omitempty"`
	Version          int                                                           `json:"version"`
	Role             string                                                        `json:"role"`
	Installation     effortauthority.Installation                                  `json:"installation"`
	Endpoints        map[string]string                                             `json:"endpoints"`
	ServerIdentity   string                                                        `json:"serverIdentity"`
	ClientIdentity   string                                                        `json:"clientIdentity"`
	ClientPin        string                                                        `json:"clientPin"`
	Manifest         isolation.Manifest                                            `json:"manifest"`
	Continuing       isolation.ProbeSpec                                           `json:"continuing"`
	ProfileMappings  map[string]string                                             `json:"profileMappings"`
	CommissionTarget CommissionTarget                                              `json:"commissionTarget"`
	ProfileGrants    map[string]map[string]effortauthority.ProfileContractGrant    `json:"profileGrants"`
	ScopeGrants      map[string]map[string]effortauthority.AcceptanceContractGrant `json:"scopeGrants"`
	PurposeIdentity  string                                                        `json:"purposeIdentity"`
	PurposeHandle    string                                                        `json:"purposeHandle"`
	ExistingJWTKID   string                                                        `json:"existingJWTKID"`
	BrokerPeers      map[string]map[string]effortauthority.BrokerGrant             `json:"brokerPeers"`
	UnitPeers        map[string]isolation.UnitPeerRole                             `json:"unitPeers"`
}

type CommissionTarget struct {
	WorkShape string `json:"workShape"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
}

type Plan struct{ bundle Bundle }

// Load uses one fixed role path and the actual current executable build digest.
// It does not accept an environment-selected authority path or credential field.
func Load(role string) (*Plan, error) {
	build, e := executingBuild()
	if e != nil {
		return nil, e
	}
	return load(ProtectedRoot, role, build, 0)
}

func load(root, role, build string, owner uint32) (*Plan, error) {
	if role != "am" && role != "pm" && role != "issuer" && role != "acceptance" {
		return nil, effortauthority.ErrRefused
	}
	raw, e := readProtected(root, role+".json", owner)
	if e != nil {
		return nil, e
	}
	return parseBundle(raw, role, build)
}

func parseBundle(raw []byte, role, build string) (*Plan, error) {
	return parseBundleAt(raw, role, build, time.Now())
}
func parseBundleAt(raw []byte, role, build string, now time.Time) (*Plan, error) {
	if exactJSON(raw) != nil {
		return nil, effortauthority.ErrRefused
	}
	var b Bundle
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&b) != nil || d.Decode(new(any)) != io.EOF || b.Version != 1 || b.Role != role || b.Installation.Builds[role] != build || !validHash(build) || (role != "acceptance" && !validHash(b.ClientPin)) {
		return nil, effortauthority.ErrRefused
	}
	descriptor := b.Installation
	descriptor.Enabled = true
	frozen, e := descriptor.Freeze()
	if e != nil {
		return nil, e
	}
	if (b.Installation.Enabled && now.Before(frozen.Policy.NotBefore)) || !now.Before(frozen.Policy.Deadline) {
		return nil, effortauthority.ErrRefused
	}
	frozen.Enabled = b.Installation.Enabled
	b.Installation = frozen
	if _, e = ca.ParseIdentity(b.ServerIdentity); e != nil {
		return nil, effortauthority.ErrRefused
	}
	if role != "acceptance" {
		if _, e = ca.ParseIdentity(b.ClientIdentity); e != nil {
			return nil, effortauthority.ErrRefused
		}
	}
	if len(b.ProfileMappings) != len(b.Installation.Policy.Profiles) {
		return nil, effortauthority.ErrRefused
	}
	for logical := range b.Installation.Policy.Profiles {
		if b.ProfileMappings[logical] == "" {
			return nil, effortauthority.ErrRefused
		}
	}
	if len(b.DevelopmentSetup) > 0 && (role != "acceptance" || b.Installation.Enabled || bytes.Equal(bytes.TrimSpace(b.DevelopmentSetup), []byte("null"))) {
		return nil, effortauthority.ErrRefused
	}
	if role == "acceptance" && (b.Installation.Enabled || (b.CommissionTarget.WorkShape != "plan-backed" && b.CommissionTarget.WorkShape != "bounded-authority")) {
		return nil, effortauthority.ErrRefused
	}
	for _, r := range []string{"am", "pm", "issuer", "acceptance"} {
		u, e := url.Parse(b.Endpoints[r])
		if e != nil || u.Scheme != "https" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return nil, effortauthority.ErrRefused
		}
		host, port, e := net.SplitHostPort(u.Host)
		if e != nil || net.ParseIP(host) == nil || !validPort(port) {
			return nil, effortauthority.ErrRefused
		}
	}
	if !validHash(b.Installation.ServerPins["acceptance"]) || !validHash(b.Installation.Builds["acceptance"]) || len(b.Endpoints) != 4 {
		return nil, effortauthority.ErrRefused
	}
	if b.Installation.Enabled {
		if !b.Manifest.Enabled || b.Manifest.Validate() != nil || b.Continuing.Purpose != isolation.ContinuingWitness || effortauthority.Digest(b.Continuing.Manifest) != effortauthority.Digest(b.Manifest) {
			return nil, effortauthority.ErrRefused
		}
	} else if b.Manifest.Enabled || b.Continuing.Purpose != "" {
		return nil, effortauthority.ErrRefused
	}
	return &Plan{bundle: b}, nil
}
func validHash(s string) bool { raw, e := hex.DecodeString(s); return e == nil && len(raw) == 32 }

func (p *Plan) Snapshot() (Bundle, error) {
	if p == nil {
		return Bundle{}, effortauthority.ErrRefused
	}
	raw, e := json.Marshal(p.bundle)
	if e != nil {
		return Bundle{}, e
	}
	var b Bundle
	e = json.Unmarshal(raw, &b)
	return b, e
}

// AcquireTLS resolves only an existing purpose record through the concrete
// credential authority. No provider override, refresh, fallback or write exists.
func (p *Plan) AcquireTLS() (*tls.Config, *tls.Config, error) {
	if p.checkWindow() != nil {
		return nil, nil, effortauthority.ErrRefused
	}
	authority, e := ca.Default()
	if e != nil {
		return nil, nil, e
	}
	serverID, e := ca.ParseIdentity(p.bundle.ServerIdentity)
	if e != nil {
		return nil, nil, e
	}
	clientID, e := ca.ParseIdentity(p.bundle.ClientIdentity)
	if e != nil {
		return nil, nil, e
	}
	serverRaw, e := authority.Resolve(serverID, TLSField)
	if e != nil {
		return nil, nil, e
	}
	clientRaw, e := authority.Resolve(clientID, TLSField)
	if e != nil {
		return nil, nil, e
	}
	server, e := p.parseTLS([]byte(serverRaw), time.Now(), "server", p.bundle.Installation.ServerPins[p.bundle.Role])
	if e != nil {
		return nil, nil, e
	}
	client, e := p.parseTLS([]byte(clientRaw), time.Now(), "client", p.bundle.ClientPin)
	if e != nil {
		return nil, nil, e
	}
	return server, client, nil
}

func (p *Plan) parseTLS(raw []byte, now time.Time, usage, expectedPin string) (*tls.Config, error) {
	if p == nil || exactJSON(raw) != nil {
		return nil, effortauthority.ErrRefused
	}
	var record struct {
		Usage        string `json:"usage"`
		Role         string `json:"role"`
		PolicyDigest string `json:"policyDigest"`
		Certificate  string `json:"certificate"`
		PrivateKey   string `json:"privateKey"`
		CA           string `json:"ca"`
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if len(raw) > 262144 || d.Decode(&record) != nil || d.Decode(new(any)) != io.EOF || record.Usage != usage || record.Role != p.bundle.Role || record.PolicyDigest != effortauthority.Digest(p.bundle.Installation.Policy) {
		return nil, effortauthority.ErrRefused
	}
	cert, e := tls.X509KeyPair([]byte(record.Certificate), []byte(record.PrivateKey))
	if e != nil || len(cert.Certificate) == 0 {
		return nil, effortauthority.ErrRefused
	}
	leaf, e := x509.ParseCertificate(cert.Certificate[0])
	if e != nil || now.Before(leaf.NotBefore) || !now.Before(leaf.NotAfter) {
		return nil, effortauthority.ErrRefused
	}
	hash := sha256.Sum256(leaf.Raw)
	pin := hex.EncodeToString(hash[:])
	if pin != expectedPin {
		return nil, effortauthority.ErrRefused
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(record.CA)) {
		return nil, effortauthority.ErrRefused
	}
	server := &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert}
	client := &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}, RootCAs: pool.Clone()}
	if usage == "server" {
		return server, nil
	}
	return client, nil
}

// CaptureContinuing is an explicit owner live read after the protected descriptor
// has passed. It creates no service and never reconstructs a witness from JSON.
func (p *Plan) CaptureContinuing(ctx context.Context) (*isolation.Witness, error) {
	if p.checkWindow() != nil || !p.bundle.Installation.Enabled {
		return nil, effortauthority.ErrRefused
	}
	witness, e := isolation.CaptureWitness(ctx, p.bundle.Continuing)
	if e != nil {
		return nil, e
	}
	if witness.CheckPolicy(p.bundle.Installation.Policy) != nil {
		return nil, effortauthority.ErrRefused
	}
	return witness, nil
}

func validPort(s string) bool {
	n, e := strconv.Atoi(s)
	return e == nil && n >= 1024 && n <= 65535 && strconv.Itoa(n) == s
}

func (p *Plan) checkWindow() error {
	if p == nil {
		return effortauthority.ErrRefused
	}
	now := time.Now()
	policy := p.bundle.Installation.Policy
	if !now.Before(policy.Deadline) || (p.bundle.Installation.Enabled && now.Before(policy.NotBefore)) {
		return effortauthority.ErrRefused
	}
	return nil
}
