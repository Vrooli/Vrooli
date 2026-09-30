package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"secrets-manager-api/internal/vault"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
)

var (
	errVaultLocked                   = errors.New("vault is locked")
	errVaultKeyMissing               = errors.New("vault key is unavailable; recovery is required")
	errItemConflict                  = errors.New("item revision conflict")
	errDecisionConflict              = errors.New("access request decision conflict")
	errAssuranceRequired             = errors.New("fresh action-bound assurance is required")
	errAccessDenied                  = errors.New("access denied")
	errOwnerAuthMissing              = errors.New("owner authentication is not configured")
	errWorkspaceEnrolled             = errors.New("workspace is already enrolled")
	errMemberExists                  = errors.New("workspace member already exists")
	errMachineExists                 = errors.New("machine principal already exists")
	errCredentialExecutorUnsupported = errors.New("credential executor capability is unsupported")
	errBrowserAuthorizationRequired  = errors.New("browser navigation requires new authorization")
	errBrowserDocumentMismatch       = errors.New("browser document binding changed")
	errSSHDestinationDenied          = errors.New("ssh destination or principal is not authorized")
	errRecoveryEpochConflict         = errors.New("recovery epoch changed; reload recovery state")
)

var supportedVaultItemTypes = map[string]bool{
	"login": true, "password": true, "api_credential": true,
	"secure_note": true, "totp": true, "ssh": true,
}

// VaultItem is metadata safe to return to ordinary clients. Secret fields are
// encrypted in the authority and are returned only by Reveal.
type VaultItem struct {
	ID        string    `json:"id"`
	VaultID   string    `json:"vault_id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Username  string    `json:"username,omitempty"`
	URI       string    `json:"uri,omitempty"`
	Folder    string    `json:"folder,omitempty"`
	Tags      []string  `json:"tags"`
	Favorite  bool      `json:"favorite"`
	Trashed   bool      `json:"trashed"`
	Revision  int       `json:"revision"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type vaultRecord struct {
	Meta       VaultItem
	Workspace  string
	Ciphertext string
	Status     string
}

type grantRecord struct {
	ID            string    `json:"id"`
	WorkspaceID   string    `json:"workspace_id"`
	VaultID       string    `json:"vault_id"`
	ItemID        string    `json:"item_id,omitempty"`
	ParentGrantID string    `json:"parent_grant_id,omitempty"`
	PrincipalType string    `json:"principal_type"`
	PrincipalID   string    `json:"principal_id"`
	SelectorMode  string    `json:"selector_mode"`
	Members       []string  `json:"members"`
	Operations    []string  `json:"operations"`
	Target        string    `json:"target,omitempty"`
	ExpiresAt     time.Time `json:"expires_at"`
	Status        string    `json:"status"`
	CreatedBy     string    `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

type grantAccessExplanation struct {
	GrantID          string `json:"grant_id"`
	WorkspaceID      string `json:"workspace_id"`
	Actor            string `json:"actor"`
	ItemID           string `json:"item_id,omitempty"`
	Operation        string `json:"operation"`
	Decision         string `json:"decision"`
	Reason           string `json:"reason"`
	SelectorMode     string `json:"selector_mode"`
	FutureMembers    bool   `json:"future_members"`
	RawReadPermitted bool   `json:"raw_read_permitted"`
}

type accessRequest struct {
	ID             string     `json:"id"`
	WorkspaceID    string     `json:"workspace_id"`
	GrantID        string     `json:"grant_id"`
	ItemID         string     `json:"item_id"`
	Operation      string     `json:"operation"`
	RequestDigest  string     `json:"request_digest"`
	Status         string     `json:"status"`
	RequestedBy    string     `json:"requested_by"`
	DecidedBy      string     `json:"decided_by,omitempty"`
	DecisionReason string     `json:"decision_reason,omitempty"`
	ExpiresAt      time.Time  `json:"expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
	DecidedAt      *time.Time `json:"decided_at,omitempty"`
}

type auditEvent struct {
	ID              string    `json:"id"`
	WorkspaceID     string    `json:"workspace_id"`
	ActorID         string    `json:"actor_id"`
	Action          string    `json:"action"`
	ItemID          string    `json:"item_id,omitempty"`
	RequestID       string    `json:"request_id,omitempty"`
	CorrelationID   string    `json:"correlation_id,omitempty"`
	EventKey        string    `json:"event_key,omitempty"`
	Outcome         string    `json:"outcome"`
	Decision        string    `json:"decision,omitempty"`
	Destination     string    `json:"destination,omitempty"`
	Detail          string    `json:"detail,omitempty"`
	IntegrityStatus string    `json:"integrity_status"`
	IntegrityHash   string    `json:"integrity_hash,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// PasswordManager owns the management metadata and delegates secret custody
// to one encrypted payload per item. The memory mode is deliberately limited
// to API unit tests and database-less health checks; a configured database
// never silently falls back to it.
type PasswordManager struct {
	db  *database.RoutedDB
	key []byte

	mu              sync.RWMutex
	memory          bool
	vaults          map[string]vaultRecord
	items           map[string]vaultRecord
	grants          map[string]grantRecord
	requests        map[string]accessRequest
	requestWake     chan struct{}
	assurance       map[string]assuranceRecord
	idempotent      map[string]string
	events          []auditEvent
	auditKeys       map[string]struct{}
	brokers         map[string]brokerSession
	exports         map[string]exportHandle
	sources         map[string]sourceRecord
	bindings        map[string]sourceBinding
	history         map[string][]itemHistoryRecord
	enrollments     map[string]workspaceEnrollment
	ownerTokens     map[string]ownerTokenRecord
	members         map[string]workspaceMember
	machines        map[string]machinePrincipalRecord
	nativeHosts     map[string]nativeHostEnrollment
	browsers        map[string]browserSession
	sshSigners      map[string]sshSignerSession
	recoveryEpochs  map[string]uint64
	browserExec     trustedBrowserExecutor
	credentialUseMu sync.Mutex
}

type workspaceEnrollment struct {
	WorkspaceID, Status, EnrolledBy string
	EnrolledAt                      time.Time
}

type workspaceMember struct {
	ID, WorkspaceID, PrincipalID, Role, Status string
	CreatedAt, RevokedAt                       time.Time
}

type workspaceMemberInput struct {
	PrincipalID string `json:"principal_id"`
	Role        string `json:"role"`
}

type ownerTokenRecord struct {
	WorkspaceID, PrincipalID, Role, Status string
}

type ownerAuthContext struct {
	WorkspaceID, PrincipalID, Role string
	RunID                          string
	AuthenticatedAt                time.Time
}

type machinePrincipalRecord struct {
	ID, WorkspaceID, MachineID, Label, TokenDigest, Status, CreatedBy string
	CreatedAt, RevokedAt                                              time.Time
}

type machinePrincipalInput struct {
	MachineID string `json:"machine_id"`
	Label     string `json:"label"`
}

type ownerAuthContextKey struct{}

type nativeHostEnrollment struct {
	WorkspaceID, ExtensionID, Origin, Status, EnrolledBy string
	EnrolledAt, RevokedAt                                time.Time
}

type itemHistoryRecord struct {
	ID, ItemID, ChangedBy, Ciphertext string
	Version                           int
	ChangedAt                         time.Time
}

type assuranceRecord struct {
	Workspace string
	Actor     string
	Operation string
	Request   string
	Expires   time.Time
	Consumed  bool
}

type createGrantInput struct {
	VaultID       string    `json:"vault_id"`
	ItemID        string    `json:"item_id"`
	ParentGrantID string    `json:"parent_grant_id"`
	PrincipalType string    `json:"principal_type"`
	PrincipalID   string    `json:"principal_id"`
	SelectorMode  string    `json:"selector_mode"`
	Members       []string  `json:"members"`
	Operations    []string  `json:"operations"`
	Target        string    `json:"target"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type createAccessRequestInput struct {
	GrantID        string `json:"grant_id"`
	ItemID         string `json:"item_id"`
	Operation      string `json:"operation"`
	DurationSec    int    `json:"duration_seconds"`
	IdempotencyKey string `json:"-"`
}

var supportedGrantOperations = map[string]bool{
	"use": true, "reveal": true, "export": true, "inject": true, "sign": true,
}

func newPasswordManager(db *database.RoutedDB) *PasswordManager {
	key, err := configuredVaultKey()
	if err != nil && db != nil {
		return &PasswordManager{db: db}
	}
	if err != nil {
		key = []byte("01234567890123456789012345678901")
	}
	manager := &PasswordManager{
		db: db, key: key, memory: db == nil,
		vaults: make(map[string]vaultRecord), items: make(map[string]vaultRecord),
		grants: make(map[string]grantRecord), requests: make(map[string]accessRequest), requestWake: make(chan struct{}, 1),
		assurance: make(map[string]assuranceRecord), idempotent: make(map[string]string), auditKeys: make(map[string]struct{}), brokers: make(map[string]brokerSession), exports: make(map[string]exportHandle), sources: make(map[string]sourceRecord), bindings: make(map[string]sourceBinding), history: make(map[string][]itemHistoryRecord), enrollments: make(map[string]workspaceEnrollment), ownerTokens: make(map[string]ownerTokenRecord), members: make(map[string]workspaceMember), machines: make(map[string]machinePrincipalRecord), nativeHosts: make(map[string]nativeHostEnrollment), browsers: make(map[string]browserSession), sshSigners: make(map[string]sshSignerSession), recoveryEpochs: make(map[string]uint64),
	}
	if manager.memory {
		manager.vaults["personal"] = vaultRecord{Meta: VaultItem{VaultID: "personal"}, Workspace: "local", Status: "unlocked"}
	}
	return manager
}

func configuredVaultKey() ([]byte, error) {
	raw, err := resolveSecretsManagerCredential("SECRETS_MANAGER_VAULT_KEY", "vrooli/secrets-manager/vault", "encryption-key")
	if err != nil {
		return nil, fmt.Errorf("resolve vault encryption key: %w", err)
	}
	if raw == "" {
		return nil, errVaultKeyMissing
	}
	if key, err := base64.RawStdEncoding.DecodeString(raw); err == nil && len(key) == 32 {
		return key, nil
	}
	if key, err := hex.DecodeString(raw); err == nil && len(key) == 32 {
		return key, nil
	}
	return nil, fmt.Errorf("%w: expected 32-byte base64 or hex key", errVaultKeyMissing)
}

func getEnv(name string) string {
	// Kept behind a function so tests can replace the constructor without
	// mutating process-global configuration.
	return lookupEnv(name)
}

var lookupEnv = func(name string) string { return getenv(name) }

// getenv is separated into a tiny function to keep this file easy to test in
// package main without shadowing os.Getenv in existing scenario tests.
func getenv(name string) string { return os.Getenv(name) }

func validateItemInput(itemType, name string, fields map[string]string) error {
	if !supportedVaultItemTypes[itemType] {
		return fmt.Errorf("unsupported item type %q", itemType)
	}
	if strings.TrimSpace(name) == "" || len([]rune(name)) > 200 {
		return errors.New("item name is required and must be at most 200 characters")
	}
	if len(fields) > 64 {
		return errors.New("item has too many fields")
	}
	total := 0
	for key, value := range fields {
		if strings.TrimSpace(key) == "" || len([]rune(key)) > 100 {
			return errors.New("item field names must be non-empty and bounded")
		}
		total += len(value)
	}
	if total > 256*1024 {
		return errors.New("encrypted item payload exceeds 256 KiB")
	}
	if uri := strings.TrimSpace(fields["uri"]); uri != "" {
		parsed, err := url.Parse(uri)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
			return errors.New("uri must be an absolute origin without embedded credentials")
		}
	}
	return nil
}

func (p *PasswordManager) cipher() (cipher.AEAD, error) {
	if len(p.key) != 32 {
		return nil, errVaultKeyMissing
	}
	block, err := aes.NewCipher(p.key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (p *PasswordManager) encrypt(vaultID, itemID string, fields map[string]string) (string, error) {
	box, err := p.cipher()
	if err != nil {
		return "", err
	}
	plain, err := json.Marshal(fields)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, box.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	aad := []byte("vrooli/password-manager/v1/" + vaultID + "/" + itemID)
	sealed := box.Seal(nil, nonce, plain, aad)
	return base64.RawStdEncoding.EncodeToString(append(nonce, sealed...)), nil
}

func (p *PasswordManager) decrypt(vaultID, itemID, encoded string) (map[string]string, error) {
	box, err := p.cipher()
	if err != nil {
		return nil, err
	}
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil || len(raw) < box.NonceSize() {
		return nil, errors.New("encrypted item envelope is invalid")
	}
	aad := []byte("vrooli/password-manager/v1/" + vaultID + "/" + itemID)
	plain, err := box.Open(nil, raw[:box.NonceSize()], raw[box.NonceSize():], aad)
	if err != nil {
		return nil, errors.New("encrypted item envelope failed authentication")
	}
	var fields map[string]string
	if err := json.Unmarshal(plain, &fields); err != nil {
		return nil, errors.New("encrypted item payload is invalid")
	}
	return fields, nil
}

func (p *PasswordManager) ensureVault(ctx context.Context, workspace, vaultID string) (vaultRecord, error) {
	workspace = strings.TrimSpace(workspace)
	vaultID = strings.TrimSpace(vaultID)
	if workspace == "" || vaultID == "" {
		return vaultRecord{}, errors.New("workspace and vault are required")
	}
	if p.memory {
		p.mu.Lock()
		defer p.mu.Unlock()
		v, ok := p.vaults[vaultID]
		if !ok {
			v = vaultRecord{Meta: VaultItem{VaultID: vaultID}, Workspace: workspace, Status: "locked"}
			p.vaults[vaultID] = v
		}
		if v.Workspace != workspace {
			return vaultRecord{}, errAccessDenied
		}
		return v, nil
	}
	var v vaultRecord
	var created, updated string
	err := p.db.QueryRowContext(ctx, `SELECT id, workspace_id, name, status, created_at, updated_at FROM pm_vaults WHERE id = $1 AND workspace_id = $2`, vaultID, workspace).Scan(&v.Meta.VaultID, &v.Workspace, &v.Meta.Name, &v.Status, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		// Vault IDs are globally unique. A request for an existing vault in a
		// different workspace must fail closed instead of attempting an insert
		// that would leak a uniqueness error or create a cross-tenant alias.
		var existingWorkspace string
		if lookupErr := p.db.QueryRowContext(ctx, `SELECT workspace_id FROM pm_vaults WHERE id=$1`, vaultID).Scan(&existingWorkspace); lookupErr == nil && existingWorkspace != workspace {
			return vaultRecord{}, errAccessDenied
		}
		_, err = p.db.ExecContext(ctx, `INSERT INTO pm_vaults(id, workspace_id, name, status) VALUES($1,$2,$3,'locked')`, vaultID, workspace, vaultID)
		if err != nil {
			return vaultRecord{}, err
		}
		v = vaultRecord{Meta: VaultItem{VaultID: vaultID, Name: vaultID}, Workspace: workspace, Status: "locked"}
		return v, nil
	}
	if err != nil {
		return vaultRecord{}, err
	}
	v.Meta.CreatedAt = parseTime(created)
	v.Meta.UpdatedAt = parseTime(updated)
	return v, nil
}

func parseTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag != "" && !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}
	sort.Strings(result)
	return result
}

type createItemInput struct {
	Type     string            `json:"type"`
	Name     string            `json:"name"`
	Username string            `json:"username"`
	URI      string            `json:"uri"`
	Folder   string            `json:"folder"`
	Tags     []string          `json:"tags"`
	Favorite bool              `json:"favorite"`
	Fields   map[string]string `json:"fields"`
}

func (p *PasswordManager) createItem(ctx context.Context, workspace, vaultID, actor, idem string, input createItemInput) (VaultItem, error) {
	if input.Fields == nil {
		input.Fields = map[string]string{}
	}
	input.Fields["username"] = input.Username
	input.Fields["uri"] = input.URI
	if err := validateItemInput(input.Type, input.Name, input.Fields); err != nil {
		return VaultItem{}, err
	}
	v, err := p.ensureVault(ctx, workspace, vaultID)
	if err != nil {
		return VaultItem{}, err
	}
	if v.Status == "locked" {
		return VaultItem{}, errVaultLocked
	}
	if idem != "" {
		if len(idem) > 200 {
			return VaultItem{}, errors.New("Idempotency-Key must be at most 200 characters")
		}
		p.mu.RLock()
		known := p.idempotent[workspace+":"+idem]
		p.mu.RUnlock()
		if known != "" {
			return p.metadata(ctx, workspace, vaultID, known)
		}
		if !p.memory {
			var knownID string
			err := p.db.QueryRowContext(ctx, `SELECT item_id FROM pm_idempotency WHERE workspace_id=$1 AND idempotency_key=$2`, workspace, idem).Scan(&knownID)
			if err == nil {
				return p.metadata(ctx, workspace, vaultID, knownID)
			}
			if err != sql.ErrNoRows {
				return VaultItem{}, err
			}
		}
	}
	id := uuid.NewString()
	sealed, err := p.encrypt(vaultID, id, input.Fields)
	if err != nil {
		return VaultItem{}, err
	}
	now := time.Now().UTC()
	item := VaultItem{ID: id, VaultID: vaultID, Type: input.Type, Name: input.Name, Username: input.Username, URI: input.URI, Folder: input.Folder, Tags: normalizeTags(input.Tags), Favorite: input.Favorite, Revision: 1, CreatedAt: now, UpdatedAt: now}
	record := vaultRecord{Meta: item, Workspace: workspace, Ciphertext: sealed}
	if p.memory {
		p.mu.Lock()
		p.items[id] = record
		if idem != "" {
			p.idempotent[workspace+":"+idem] = id
		}
		p.mu.Unlock()
		p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "item.create", ItemID: id, Outcome: "success"})
		return item, nil
	}
	tags, _ := json.Marshal(item.Tags)
	if idem == "" {
		_, err = p.db.ExecContext(ctx, `INSERT INTO pm_items(id,vault_id,workspace_id,item_type,name,username,uri,folder,tags,favorite,trashed,version,encrypted_payload) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,FALSE,1,$11)`, id, vaultID, workspace, item.Type, item.Name, item.Username, item.URI, item.Folder, string(tags), item.Favorite, sealed)
		if err != nil {
			return VaultItem{}, err
		}
	} else {
		tx, txErr := p.db.BeginTx(ctx, nil)
		if txErr != nil {
			return VaultItem{}, txErr
		}
		defer func() { _ = tx.Rollback() }()
		result, txErr := tx.ExecContext(ctx, `INSERT INTO pm_idempotency(workspace_id,idempotency_key,item_id) VALUES($1,$2,$3) ON CONFLICT (workspace_id,idempotency_key) DO NOTHING`, workspace, idem, id)
		if txErr != nil {
			return VaultItem{}, txErr
		}
		if count, _ := result.RowsAffected(); count == 0 {
			var knownID string
			if txErr = tx.QueryRowContext(ctx, `SELECT item_id FROM pm_idempotency WHERE workspace_id=$1 AND idempotency_key=$2`, workspace, idem).Scan(&knownID); txErr != nil {
				return VaultItem{}, txErr
			}
			_ = tx.Rollback()
			return p.metadata(ctx, workspace, vaultID, knownID)
		}
		if _, txErr = tx.ExecContext(ctx, `INSERT INTO pm_items(id,vault_id,workspace_id,item_type,name,username,uri,folder,tags,favorite,trashed,version,encrypted_payload) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,FALSE,1,$11)`, id, vaultID, workspace, item.Type, item.Name, item.Username, item.URI, item.Folder, string(tags), item.Favorite, sealed); txErr != nil {
			return VaultItem{}, txErr
		}
		if txErr = tx.Commit(); txErr != nil {
			return VaultItem{}, txErr
		}
	}
	p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "item.create", ItemID: id, Outcome: "success"})
	return item, nil
}

func (p *PasswordManager) recordAudit(ctx context.Context, event auditEvent) {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	event.Detail = RedactSensitiveText(event.Detail)
	if event.EventKey == "" {
		event.EventKey = event.ID
	}
	if event.CorrelationID == "" {
		event.CorrelationID = event.EventKey
	}
	event.IntegrityHash = auditIntegrityHash(event)
	event.IntegrityStatus = "verified"
	if p.memory {
		p.mu.Lock()
		key := event.WorkspaceID + "\x00" + event.EventKey
		if _, exists := p.auditKeys[key]; exists {
			p.mu.Unlock()
			return
		}
		p.auditKeys[key] = struct{}{}
		p.events = append(p.events, event)
		if len(p.events) > 200 {
			p.events = p.events[len(p.events)-200:]
		}
		p.mu.Unlock()
		return
	}
	_, _ = p.db.ExecContext(ctx, `INSERT INTO pm_audit_events(id,workspace_id,actor_id,action,item_id,request_id,correlation_id,event_key,outcome,decision,destination,detail,integrity_hash,integrity_status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT (workspace_id,event_key) DO NOTHING`, event.ID, event.WorkspaceID, event.ActorID, event.Action, event.ItemID, event.RequestID, event.CorrelationID, event.EventKey, event.Outcome, event.Decision, event.Destination, event.Detail, event.IntegrityHash, event.IntegrityStatus)
}

func auditIntegrityHash(event auditEvent) string {
	canonical := strings.Join([]string{event.ID, event.WorkspaceID, event.ActorID, event.Action, event.ItemID, event.RequestID, event.CorrelationID, event.EventKey, event.Outcome, event.Decision, event.Destination, event.Detail, event.CreatedAt.UTC().Format(time.RFC3339Nano)}, "\x00")
	digest := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(digest[:])
}

func verifyAuditIntegrity(event *auditEvent) {
	if event.IntegrityHash == "" {
		event.IntegrityStatus = "unknown"
		return
	}
	if subtle.ConstantTimeCompare([]byte(event.IntegrityHash), []byte(auditIntegrityHash(*event))) == 1 {
		event.IntegrityStatus = "verified"
		return
	}
	event.IntegrityStatus = "tampered"
}

func (p *PasswordManager) listAudit(ctx context.Context, workspace string) ([]auditEvent, error) {
	if p.memory {
		p.mu.RLock()
		defer p.mu.RUnlock()
		result := make([]auditEvent, 0, len(p.events))
		for index := len(p.events) - 1; index >= 0; index-- {
			if p.events[index].WorkspaceID == workspace {
				event := p.events[index]
				verifyAuditIntegrity(&event)
				result = append(result, event)
			}
		}
		return result, nil
	}
	rows, err := p.db.QueryContext(ctx, `SELECT id,workspace_id,actor_id,action,COALESCE(item_id,''),COALESCE(request_id,''),COALESCE(correlation_id,''),COALESCE(event_key,''),outcome,COALESCE(decision,''),COALESCE(destination,''),COALESCE(detail,''),COALESCE(integrity_hash,''),COALESCE(integrity_status,'unknown'),created_at FROM pm_audit_events WHERE workspace_id=$1 ORDER BY created_at DESC LIMIT 100`, workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]auditEvent, 0)
	for rows.Next() {
		var event auditEvent
		var created string
		if err := rows.Scan(&event.ID, &event.WorkspaceID, &event.ActorID, &event.Action, &event.ItemID, &event.RequestID, &event.CorrelationID, &event.EventKey, &event.Outcome, &event.Decision, &event.Destination, &event.Detail, &event.IntegrityHash, &event.IntegrityStatus, &created); err != nil {
			return nil, err
		}
		event.CreatedAt = parseTime(created)
		verifyAuditIntegrity(&event)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (p *PasswordManager) metadata(ctx context.Context, workspace, vaultID, id string) (VaultItem, error) {
	if p.memory {
		p.mu.RLock()
		defer p.mu.RUnlock()
		item, ok := p.items[id]
		if !ok || item.Workspace != workspace || item.Meta.VaultID != vaultID {
			return VaultItem{}, sql.ErrNoRows
		}
		return item.Meta, nil
	}
	var item VaultItem
	var tags, created, updated string
	err := p.db.QueryRowContext(ctx, `SELECT id,vault_id,item_type,name,username,uri,folder,tags,favorite,trashed,version,created_at,updated_at FROM pm_items WHERE id=$1 AND workspace_id=$2 AND vault_id=$3`, id, workspace, vaultID).Scan(&item.ID, &item.VaultID, &item.Type, &item.Name, &item.Username, &item.URI, &item.Folder, &tags, &item.Favorite, &item.Trashed, &item.Revision, &created, &updated)
	if err != nil {
		return VaultItem{}, err
	}
	_ = json.Unmarshal([]byte(tags), &item.Tags)
	item.CreatedAt, item.UpdatedAt = parseTime(created), parseTime(updated)
	return item, nil
}

func (p *PasswordManager) listItems(ctx context.Context, workspace, vaultID, query string, includeTrashed bool, page, limit int) ([]VaultItem, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	if p.memory {
		p.mu.RLock()
		items := make([]VaultItem, 0)
		for _, item := range p.items {
			if item.Workspace != workspace || item.Meta.VaultID != vaultID || (!includeTrashed && item.Meta.Trashed) {
				continue
			}
			if query != "" && !strings.Contains(strings.ToLower(item.Meta.Name+" "+item.Meta.Username+" "+item.Meta.URI), strings.ToLower(query)) {
				continue
			}
			items = append(items, item.Meta)
		}
		p.mu.RUnlock()
		sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
		start := (page - 1) * limit
		if start >= len(items) {
			return []VaultItem{}, nil
		}
		end := start + limit
		if end > len(items) {
			end = len(items)
		}
		return items[start:end], nil
	}
	// Search is deliberately metadata-only. The encrypted payload is never
	// loaded for an ordinary list operation.
	like := "%" + query + "%"
	rows, err := p.db.QueryContext(ctx, `SELECT id,vault_id,item_type,name,username,uri,folder,tags,favorite,trashed,version,created_at,updated_at FROM pm_items WHERE workspace_id=$1 AND vault_id=$2 AND ($3 OR trashed=FALSE) AND ($4='' OR LOWER(name) LIKE LOWER($5) OR LOWER(COALESCE(username,'')) LIKE LOWER($5) OR LOWER(COALESCE(uri,'')) LIKE LOWER($5)) ORDER BY updated_at DESC LIMIT $6 OFFSET $7`, workspace, vaultID, includeTrashed, query, like, limit, (page-1)*limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]VaultItem, 0)
	for rows.Next() {
		var item VaultItem
		var tags, created, updated string
		if err := rows.Scan(&item.ID, &item.VaultID, &item.Type, &item.Name, &item.Username, &item.URI, &item.Folder, &tags, &item.Favorite, &item.Trashed, &item.Revision, &created, &updated); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(tags), &item.Tags)
		item.CreatedAt, item.UpdatedAt = parseTime(created), parseTime(updated)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *PasswordManager) updateItem(ctx context.Context, workspace, vaultID, actor, id string, expected int, input createItemInput) (VaultItem, error) {
	if input.Fields == nil {
		input.Fields = map[string]string{}
	}
	input.Fields["username"], input.Fields["uri"] = input.Username, input.URI
	if err := validateItemInput(input.Type, input.Name, input.Fields); err != nil {
		return VaultItem{}, err
	}
	v, err := p.ensureVault(ctx, workspace, vaultID)
	if err != nil {
		return VaultItem{}, err
	}
	if v.Status == "locked" {
		return VaultItem{}, errVaultLocked
	}
	if p.memory {
		p.mu.Lock()
		old, ok := p.items[id]
		if !ok || old.Workspace != workspace || old.Meta.VaultID != vaultID {
			p.mu.Unlock()
			return VaultItem{}, sql.ErrNoRows
		}
		if old.Meta.Revision != expected {
			p.mu.Unlock()
			return VaultItem{}, errItemConflict
		}
		sealed, err := p.encrypt(vaultID, id, input.Fields)
		if err != nil {
			p.mu.Unlock()
			return VaultItem{}, err
		}
		item := old.Meta
		item.Type, item.Name, item.Username, item.URI, item.Folder, item.Tags, item.Favorite = input.Type, input.Name, input.Username, input.URI, input.Folder, normalizeTags(input.Tags), input.Favorite
		item.Revision++
		item.UpdatedAt = time.Now().UTC()
		p.history[id] = append(p.history[id], itemHistoryRecord{ID: uuid.NewString(), ItemID: id, Version: old.Meta.Revision, ChangedBy: actor, Ciphertext: old.Ciphertext, ChangedAt: time.Now().UTC()})
		if len(p.history[id]) > 20 {
			p.history[id] = p.history[id][len(p.history[id])-20:]
		}
		old.Meta = item
		old.Ciphertext = sealed
		p.items[id] = old
		p.mu.Unlock()
		p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "item.update", ItemID: id, Outcome: "success"})
		return item, nil
	}
	sealed, err := p.encrypt(vaultID, id, input.Fields)
	if err != nil {
		return VaultItem{}, err
	}
	tags, _ := json.Marshal(normalizeTags(input.Tags))
	next := expected + 1
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return VaultItem{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var currentVersion int
	if err := tx.QueryRowContext(ctx, `SELECT version FROM pm_items WHERE id=$1 AND workspace_id=$2 AND vault_id=$3`, id, workspace, vaultID).Scan(&currentVersion); err != nil {
		return VaultItem{}, err
	}
	if currentVersion != expected {
		return VaultItem{}, errItemConflict
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO pm_item_history(id,item_id,version,encrypted_payload,changed_by) SELECT $1,id,version,encrypted_payload,$2 FROM pm_items WHERE id=$3`, uuid.NewString(), actor, id); err != nil {
		return VaultItem{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE pm_items SET item_type=$1,name=$2,username=$3,uri=$4,folder=$5,tags=$6,favorite=$7,version=$8,encrypted_payload=$9,updated_at=CURRENT_TIMESTAMP WHERE id=$10 AND workspace_id=$11 AND version=$12`, input.Type, input.Name, input.Username, input.URI, input.Folder, string(tags), input.Favorite, next, sealed, id, workspace, expected)
	if err != nil {
		return VaultItem{}, err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return VaultItem{}, errItemConflict
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM pm_item_history WHERE item_id=$1 AND id NOT IN (SELECT id FROM pm_item_history WHERE item_id=$1 ORDER BY version DESC LIMIT 20)`, id); err != nil {
		return VaultItem{}, err
	}
	if err = tx.Commit(); err != nil {
		return VaultItem{}, err
	}
	p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "item.update", ItemID: id, Outcome: "success"})
	return p.metadata(ctx, workspace, vaultID, id)
}

func (p *PasswordManager) listHistory(ctx context.Context, workspace, vaultID, itemID string) ([]map[string]any, error) {
	if _, err := p.metadata(ctx, workspace, vaultID, itemID); err != nil {
		return nil, err
	}
	if p.memory {
		p.mu.RLock()
		records := append([]itemHistoryRecord(nil), p.history[itemID]...)
		p.mu.RUnlock()
		result := make([]map[string]any, 0, len(records))
		for i := len(records) - 1; i >= 0; i-- {
			record := records[i]
			result = append(result, map[string]any{"id": record.ID, "item_id": record.ItemID, "version": record.Version, "changed_by": record.ChangedBy, "changed_at": record.ChangedAt})
		}
		return result, nil
	}
	rows, err := p.db.QueryContext(ctx, `SELECT id,item_id,version,changed_at,changed_by FROM pm_item_history WHERE item_id=$1 ORDER BY version DESC LIMIT 20`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]map[string]any, 0)
	for rows.Next() {
		var id, changedAt, changedBy string
		var version int
		if err := rows.Scan(&id, &itemID, &version, &changedAt, &changedBy); err != nil {
			return nil, err
		}
		result = append(result, map[string]any{"id": id, "item_id": itemID, "version": version, "changed_by": changedBy, "changed_at": parseTime(changedAt)})
	}
	return result, rows.Err()
}

func (p *PasswordManager) setTrashed(ctx context.Context, workspace, vaultID, id string, trashed bool) error {
	if p.memory {
		p.mu.Lock()
		defer p.mu.Unlock()
		item, ok := p.items[id]
		if !ok || item.Workspace != workspace || item.Meta.VaultID != vaultID {
			return sql.ErrNoRows
		}
		item.Meta.Trashed = trashed
		item.Meta.Revision++
		item.Meta.UpdatedAt = time.Now().UTC()
		p.items[id] = item
		return nil
	}
	_, err := p.db.ExecContext(ctx, `UPDATE pm_items SET trashed=$1,version=version+1,updated_at=CURRENT_TIMESTAMP WHERE id=$2 AND workspace_id=$3 AND vault_id=$4`, trashed, id, workspace, vaultID)
	return err
}

func (p *PasswordManager) reveal(ctx context.Context, workspace, vaultID, id, actor, assurance, requestDigest string, fields []string) (map[string]string, error) {
	v, err := p.ensureVault(ctx, workspace, vaultID)
	if err != nil {
		return nil, err
	}
	if v.Status == "locked" {
		return nil, errVaultLocked
	}
	if err := p.consumeAssurance(ctx, workspace, actor, "reveal:"+id, requestDigest, assurance); err != nil {
		return nil, err
	}
	var record vaultRecord
	if p.memory {
		p.mu.RLock()
		record = p.items[id]
		p.mu.RUnlock()
		if record.Workspace != workspace || record.Meta.VaultID != vaultID {
			return nil, sql.ErrNoRows
		}
	} else {
		var tags, created, updated string
		var item VaultItem
		err = p.db.QueryRowContext(ctx, `SELECT id,vault_id,item_type,name,username,uri,folder,tags,favorite,trashed,version,encrypted_payload,created_at,updated_at FROM pm_items WHERE id=$1 AND workspace_id=$2 AND vault_id=$3`, id, workspace, vaultID).Scan(&item.ID, &item.VaultID, &item.Type, &item.Name, &item.Username, &item.URI, &item.Folder, &tags, &item.Favorite, &item.Trashed, &item.Revision, &record.Ciphertext, &created, &updated)
		if err != nil {
			return nil, err
		}
		record.Meta = item
		record.Workspace = workspace
	}
	if record.Meta.Trashed {
		return nil, errors.New("trashed items cannot be revealed until restored")
	}
	result, err := p.decrypt(vaultID, id, record.Ciphertext)
	if err != nil {
		return nil, err
	}
	p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "item.reveal", ItemID: id, Outcome: "success", Detail: "selected fields only"})
	if len(fields) == 0 {
		return result, nil
	}
	selected := make(map[string]string, len(fields))
	for _, field := range fields {
		if value, ok := result[field]; ok {
			selected[field] = value
		}
	}
	return selected, nil
}

func (p *PasswordManager) lock(ctx context.Context, workspace, vaultID string, locked bool) error {
	if _, err := p.ensureVault(ctx, workspace, vaultID); err != nil {
		return err
	}
	if p.memory {
		p.mu.Lock()
		v, ok := p.vaults[vaultID]
		if !ok || v.Workspace != workspace {
			p.mu.Unlock()
			return sql.ErrNoRows
		}
		if locked {
			v.Status = "locked"
		} else {
			v.Status = "unlocked"
		}
		p.vaults[vaultID] = v
		p.mu.Unlock()
		if locked {
			p.terminateCredentialUse(ctx, workspace, "", vaultID, "vault_locked")
		}
		return nil
	}
	status := "locked"
	if !locked {
		status = "unlocked"
	}
	_, err := p.db.ExecContext(ctx, `UPDATE pm_vaults SET status=$1,updated_at=CURRENT_TIMESTAMP WHERE id=$2 AND workspace_id=$3`, status, vaultID, workspace)
	if err == nil && locked {
		p.terminateCredentialUse(ctx, workspace, "", vaultID, "vault_locked")
	}
	return err
}

func (p *PasswordManager) issueAssurance(ctx context.Context, workspace, actor, operation, requestDigest string) (string, time.Time, error) {
	if operation == "" {
		return "", time.Time{}, errors.New("assurance operation is required")
	}
	raw := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(digest[:])
	expires := time.Now().UTC().Add(2 * time.Minute)
	record := assuranceRecord{Workspace: workspace, Actor: actor, Operation: operation, Request: requestDigest, Expires: expires}
	if p.memory {
		p.mu.Lock()
		p.assurance[key] = record
		p.mu.Unlock()
		return token, expires, nil
	}
	_, err := p.db.ExecContext(ctx, `INSERT INTO pm_assurance_tokens(digest,workspace_id,actor_id,operation,request_digest,expires_at) VALUES($1,$2,$3,$4,$5,$6)`, key, workspace, actor, operation, requestDigest, expires)
	return token, expires, err
}

func (p *PasswordManager) consumeAssurance(ctx context.Context, workspace, actor, operation, expectedRequest, token string) error {
	if token == "" {
		return errAssuranceRequired
	}
	digest := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(digest[:])
	if p.memory {
		p.mu.Lock()
		defer p.mu.Unlock()
		r, ok := p.assurance[key]
		if !ok || r.Consumed || time.Now().After(r.Expires) || r.Workspace != workspace || r.Actor != actor || r.Operation != operation || (r.Request != "" && r.Request != expectedRequest) {
			return errAssuranceRequired
		}
		r.Consumed = true
		p.assurance[key] = r
		return nil
	}
	var storedWorkspace, storedActor, storedOperation, storedRequest string
	var expires string
	err := p.db.QueryRowContext(ctx, `SELECT workspace_id,actor_id,operation,request_digest,expires_at FROM pm_assurance_tokens WHERE digest=$1 AND consumed_at IS NULL`, key).Scan(&storedWorkspace, &storedActor, &storedOperation, &storedRequest, &expires)
	if err != nil || storedWorkspace != workspace || storedActor != actor || storedOperation != operation || (storedRequest != "" && storedRequest != expectedRequest) || time.Now().After(parseTime(expires)) {
		return errAssuranceRequired
	}
	_, err = p.db.ExecContext(ctx, `UPDATE pm_assurance_tokens SET consumed_at=CURRENT_TIMESTAMP WHERE digest=$1 AND consumed_at IS NULL`, key)
	if err != nil {
		return errAssuranceRequired
	}
	return nil
}

func (p *PasswordManager) createGrant(ctx context.Context, workspace, actor string, input createGrantInput) (grantRecord, error) {
	if input.VaultID == "" || input.PrincipalType == "" || input.PrincipalID == "" {
		return grantRecord{}, errors.New("vault, principal type, and principal id are required")
	}
	if input.SelectorMode == "" {
		input.SelectorMode = "current_snapshot"
	}
	if input.SelectorMode != "current_snapshot" && input.SelectorMode != "dynamic" {
		return grantRecord{}, errors.New("selector_mode must be current_snapshot or dynamic")
	}
	if input.SelectorMode == "current_snapshot" && len(input.Members) == 0 {
		input.Members = []string{input.PrincipalID}
	}
	if len(input.Operations) == 0 {
		return grantRecord{}, errors.New("at least one grant operation is required")
	}
	for _, operation := range input.Operations {
		if !supportedGrantOperations[operation] {
			return grantRecord{}, fmt.Errorf("unsupported grant operation %q", operation)
		}
	}
	if input.ExpiresAt.IsZero() {
		input.ExpiresAt = time.Now().UTC().Add(24 * time.Hour)
	}
	if input.ExpiresAt.Before(time.Now().UTC()) || input.ExpiresAt.After(time.Now().UTC().Add(365*24*time.Hour)) {
		return grantRecord{}, errors.New("grant expiry must be in the future and within one year")
	}
	grant := grantRecord{ID: uuid.NewString(), WorkspaceID: workspace, VaultID: input.VaultID, ItemID: input.ItemID, ParentGrantID: strings.TrimSpace(input.ParentGrantID), PrincipalType: input.PrincipalType, PrincipalID: input.PrincipalID, SelectorMode: input.SelectorMode, Members: normalizeTags(input.Members), Operations: normalizeTags(input.Operations), Target: input.Target, ExpiresAt: input.ExpiresAt.UTC(), Status: "active", CreatedBy: actor, CreatedAt: time.Now().UTC()}
	if grant.ParentGrantID != "" {
		parent, parentErr := p.getGrant(ctx, workspace, grant.ParentGrantID)
		if parentErr != nil {
			return grantRecord{}, errAccessDenied
		}
		if !grantIsSubset(grant, parent) {
			return grantRecord{}, errors.New("child grant must be a subset of its parent grant")
		}
	}
	if p.memory {
		p.mu.Lock()
		p.grants[grant.ID] = grant
		p.mu.Unlock()
		return grant, nil
	}
	members, _ := json.Marshal(grant.Members)
	operations, _ := json.Marshal(grant.Operations)
	_, err := p.db.ExecContext(ctx, `INSERT INTO pm_grants(id,workspace_id,vault_id,item_id,parent_grant_id,principal_type,principal_id,selector_mode,members,operations,target,expires_at,status,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'active',$13)`, grant.ID, grant.WorkspaceID, grant.VaultID, grant.ItemID, grant.ParentGrantID, grant.PrincipalType, grant.PrincipalID, grant.SelectorMode, string(members), string(operations), grant.Target, grant.ExpiresAt, grant.CreatedBy)
	return grant, err
}

func (p *PasswordManager) getGrant(ctx context.Context, workspace, id string) (grantRecord, error) {
	if p.memory {
		p.mu.RLock()
		grant, ok := p.grants[id]
		p.mu.RUnlock()
		if !ok || grant.WorkspaceID != workspace {
			return grantRecord{}, sql.ErrNoRows
		}
		return grant, nil
	}
	var grant grantRecord
	var members, operations string
	err := p.db.QueryRowContext(ctx, `SELECT id,workspace_id,vault_id,COALESCE(item_id,''),COALESCE(parent_grant_id,''),principal_type,principal_id,selector_mode,members,operations,COALESCE(target,''),expires_at,status,created_by,created_at FROM pm_grants WHERE id=$1 AND workspace_id=$2`, id, workspace).Scan(&grant.ID, &grant.WorkspaceID, &grant.VaultID, &grant.ItemID, &grant.ParentGrantID, &grant.PrincipalType, &grant.PrincipalID, &grant.SelectorMode, &members, &operations, &grant.Target, &grant.ExpiresAt, &grant.Status, &grant.CreatedBy, &grant.CreatedAt)
	if err != nil {
		return grantRecord{}, err
	}
	_ = json.Unmarshal([]byte(members), &grant.Members)
	_ = json.Unmarshal([]byte(operations), &grant.Operations)
	return grant, nil
}

func (p *PasswordManager) listGrants(ctx context.Context, workspace string) ([]grantRecord, error) {
	if p.memory {
		p.mu.RLock()
		defer p.mu.RUnlock()
		result := make([]grantRecord, 0)
		for _, grant := range p.grants {
			if grant.WorkspaceID == workspace {
				result = append(result, grant)
			}
		}
		sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
		return result, nil
	}
	rows, err := p.db.QueryContext(ctx, `SELECT id,workspace_id,vault_id,COALESCE(item_id,''),COALESCE(parent_grant_id,''),principal_type,principal_id,selector_mode,members,operations,COALESCE(target,''),expires_at,status,created_by,created_at FROM pm_grants WHERE workspace_id=$1 ORDER BY created_at DESC LIMIT 200`, workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]grantRecord, 0)
	for rows.Next() {
		var grant grantRecord
		var members, operations, expires, created string
		if err := rows.Scan(&grant.ID, &grant.WorkspaceID, &grant.VaultID, &grant.ItemID, &grant.ParentGrantID, &grant.PrincipalType, &grant.PrincipalID, &grant.SelectorMode, &members, &operations, &grant.Target, &expires, &grant.Status, &grant.CreatedBy, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(members), &grant.Members)
		_ = json.Unmarshal([]byte(operations), &grant.Operations)
		grant.ExpiresAt, grant.CreatedAt = parseTime(expires), parseTime(created)
		result = append(result, grant)
	}
	return result, rows.Err()
}

func (p *PasswordManager) listAccessRequests(ctx context.Context, workspace string) ([]accessRequest, error) {
	if p.memory {
		p.mu.Lock()
		defer p.mu.Unlock()
		result := make([]accessRequest, 0)
		now := time.Now().UTC()
		for id, request := range p.requests {
			if request.Status == "pending" && now.After(request.ExpiresAt) {
				request.Status = "expired"
				p.requests[id] = request
			}
			if request.WorkspaceID == workspace {
				result = append(result, request)
			}
		}
		sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
		return result, nil
	}
	_, _ = p.db.ExecContext(ctx, `UPDATE pm_access_requests SET status='expired' WHERE workspace_id=$1 AND status='pending' AND expires_at <= CURRENT_TIMESTAMP`, workspace)
	rows, err := p.db.QueryContext(ctx, `SELECT id,workspace_id,grant_id,item_id,operation,request_digest,status,requested_by,COALESCE(decided_by,''),COALESCE(decision_reason,''),expires_at,created_at,decided_at FROM pm_access_requests WHERE workspace_id=$1 ORDER BY created_at DESC LIMIT 200`, workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]accessRequest, 0)
	for rows.Next() {
		var request accessRequest
		var expires, created string
		var decided sql.NullString
		if err := rows.Scan(&request.ID, &request.WorkspaceID, &request.GrantID, &request.ItemID, &request.Operation, &request.RequestDigest, &request.Status, &request.RequestedBy, &request.DecidedBy, &request.DecisionReason, &expires, &created, &decided); err != nil {
			return nil, err
		}
		request.ExpiresAt, request.CreatedAt = parseTime(expires), parseTime(created)
		if decided.Valid && decided.String != "" {
			value := parseTime(decided.String)
			request.DecidedAt = &value
		}
		result = append(result, request)
	}
	return result, rows.Err()
}

func grantAllows(grant grantRecord, actor, itemID, operation string, now time.Time) bool {
	if grant.Status != "active" || now.After(grant.ExpiresAt) || (grant.ItemID != "" && grant.ItemID != itemID) {
		return false
	}
	foundOperation := false
	for _, candidate := range grant.Operations {
		if candidate == operation {
			foundOperation = true
			break
		}
	}
	if !foundOperation {
		return false
	}
	if grant.SelectorMode == "dynamic" {
		if actor == grant.PrincipalID || contains(grant.Members, actor) {
			return true
		}
		return grant.PrincipalID == "*" || (grant.PrincipalType == "workspace-agent" && grant.PrincipalID == "workspace-agent:*")
	}
	return contains(grant.Members, actor)
}

func (p *PasswordManager) grantAllows(ctx context.Context, grant grantRecord, actor, itemID, operation string, now time.Time) bool {
	if !grantAllows(grant, actor, itemID, operation, now) {
		return false
	}
	if grant.SelectorMode != "dynamic" || (grant.PrincipalID != "*" && !(grant.PrincipalType == "workspace-agent" && grant.PrincipalID == "workspace-agent:*")) {
		return true
	}
	if p.memory {
		p.mu.RLock()
		defer p.mu.RUnlock()
		hasMembers := false
		for _, member := range p.members {
			if member.WorkspaceID == grant.WorkspaceID {
				hasMembers = true
				if member.PrincipalID == actor && member.Status == "active" {
					return true
				}
			}
		}
		return !hasMembers && strings.TrimSpace(actor) != ""
	}
	var total, active int
	if err := p.db.QueryRowContext(ctx, `SELECT COUNT(*),COUNT(*) FILTER (WHERE principal_id=$2 AND status='active') FROM pm_workspace_members WHERE workspace_id=$1`, grant.WorkspaceID, actor).Scan(&total, &active); err != nil {
		return false
	}
	return (total == 0 && strings.TrimSpace(actor) != "") || active > 0
}

func grantIsSubset(child, parent grantRecord) bool {
	if child.WorkspaceID != parent.WorkspaceID || child.VaultID != parent.VaultID || parent.Status != "active" {
		return false
	}
	if parent.ItemID != "" && child.ItemID != parent.ItemID {
		return false
	}
	if !containsAll(parent.Operations, child.Operations) || child.ExpiresAt.After(parent.ExpiresAt) {
		return false
	}
	if parent.Target != "" && child.Target != parent.Target {
		return false
	}
	if parent.SelectorMode == "current_snapshot" {
		if child.SelectorMode != "current_snapshot" || !containsAll(parent.Members, child.Members) {
			return false
		}
	} else if child.SelectorMode == "dynamic" && parent.PrincipalID != "*" && child.PrincipalID != parent.PrincipalID {
		return false
	}
	if parent.PrincipalType != child.PrincipalType && parent.PrincipalID != "*" {
		return false
	}
	if parent.PrincipalID != "*" && child.PrincipalID != parent.PrincipalID && !contains(parent.Members, child.PrincipalID) {
		return false
	}
	return true
}

func containsAll(haystack, values []string) bool {
	for _, value := range values {
		if !contains(haystack, value) {
			return false
		}
	}
	return true
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func (p *PasswordManager) createAccessRequest(ctx context.Context, workspace, actor string, input createAccessRequestInput) (accessRequest, error) {
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if len(input.IdempotencyKey) > 200 {
		return accessRequest{}, errors.New("Idempotency-Key must be at most 200 characters")
	}
	grant, err := p.getGrant(ctx, workspace, input.GrantID)
	if err != nil {
		return accessRequest{}, err
	}
	if !p.grantAllows(ctx, grant, actor, input.ItemID, input.Operation, time.Now().UTC()) {
		return accessRequest{}, errAccessDenied
	}
	if !supportedGrantOperations[input.Operation] {
		return accessRequest{}, errors.New("unsupported access operation")
	}
	digestBytes := sha256.Sum256([]byte(strings.Join([]string{workspace, grant.ID, input.ItemID, input.Operation, grant.Target, actor}, "\x00")))
	req := accessRequest{ID: uuid.NewString(), WorkspaceID: workspace, GrantID: grant.ID, ItemID: input.ItemID, Operation: input.Operation, RequestDigest: hex.EncodeToString(digestBytes[:]), Status: "pending", RequestedBy: actor, ExpiresAt: time.Now().UTC().Add(10 * time.Minute), CreatedAt: time.Now().UTC()}
	if input.DurationSec > 0 && input.DurationSec < 600 {
		req.ExpiresAt = time.Now().UTC().Add(time.Duration(input.DurationSec) * time.Second)
	}
	if p.memory {
		p.mu.Lock()
		if input.IdempotencyKey != "" {
			key := "access:" + workspace + ":" + actor + ":" + input.IdempotencyKey
			if knownID := p.idempotent[key]; knownID != "" {
				known := p.requests[knownID]
				p.mu.Unlock()
				if known.RequestDigest != req.RequestDigest {
					return accessRequest{}, errItemConflict
				}
				return known, nil
			}
			p.idempotent[key] = req.ID
		}
		p.requests[req.ID] = req
		p.mu.Unlock()
		p.signalAccessRequestChange()
		return req, nil
	}
	if input.IdempotencyKey == "" {
		_, err = p.db.ExecContext(ctx, `INSERT INTO pm_access_requests(id,workspace_id,grant_id,item_id,operation,request_digest,status,requested_by,expires_at) VALUES($1,$2,$3,$4,$5,$6,'pending',$7,$8)`, req.ID, req.WorkspaceID, req.GrantID, req.ItemID, req.Operation, req.RequestDigest, req.RequestedBy, req.ExpiresAt)
		if err == nil {
			p.signalAccessRequestChange()
		}
		return req, err
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return accessRequest{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `INSERT INTO pm_access_requests(id,workspace_id,grant_id,item_id,operation,request_digest,status,requested_by,expires_at) VALUES($1,$2,$3,$4,$5,$6,'pending',$7,$8)`, req.ID, req.WorkspaceID, req.GrantID, req.ItemID, req.Operation, req.RequestDigest, req.RequestedBy, req.ExpiresAt); err != nil {
		return accessRequest{}, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO pm_access_request_idempotency(workspace_id,requester_id,idempotency_key,request_id,request_digest) VALUES($1,$2,$3,$4,$5) ON CONFLICT (workspace_id,requester_id,idempotency_key) DO NOTHING`, workspace, actor, input.IdempotencyKey, req.ID, req.RequestDigest)
	if err != nil {
		return accessRequest{}, err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		var knownID, knownDigest string
		if err := tx.QueryRowContext(ctx, `SELECT request_id,request_digest FROM pm_access_request_idempotency WHERE workspace_id=$1 AND requester_id=$2 AND idempotency_key=$3`, workspace, actor, input.IdempotencyKey).Scan(&knownID, &knownDigest); err != nil {
			return accessRequest{}, err
		}
		if knownDigest != req.RequestDigest {
			return accessRequest{}, errItemConflict
		}
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			return accessRequest{}, err
		}
		return p.getAccessRequest(ctx, workspace, knownID)
	}
	if err = tx.Commit(); err != nil {
		return accessRequest{}, err
	}
	p.signalAccessRequestChange()
	return req, nil
}

func (p *PasswordManager) signalAccessRequestChange() {
	if p.requestWake == nil {
		return
	}
	select {
	case p.requestWake <- struct{}{}:
	default:
	}
}

func (p *PasswordManager) getAccessRequest(ctx context.Context, workspace, id string) (accessRequest, error) {
	if p.memory {
		p.mu.Lock()
		defer p.mu.Unlock()
		request, ok := p.requests[id]
		if !ok || request.WorkspaceID != workspace {
			return accessRequest{}, sql.ErrNoRows
		}
		if request.Status == "pending" && time.Now().UTC().After(request.ExpiresAt) {
			request.Status = "expired"
			p.requests[id] = request
		}
		return request, nil
	}
	var request accessRequest
	var expires, created string
	var decidedAt sql.NullString
	err := p.db.QueryRowContext(ctx, `SELECT id,workspace_id,grant_id,item_id,operation,request_digest,status,requested_by,COALESCE(decided_by,''),COALESCE(decision_reason,''),expires_at,created_at,decided_at FROM pm_access_requests WHERE id=$1 AND workspace_id=$2`, id, workspace).Scan(&request.ID, &request.WorkspaceID, &request.GrantID, &request.ItemID, &request.Operation, &request.RequestDigest, &request.Status, &request.RequestedBy, &request.DecidedBy, &request.DecisionReason, &expires, &created, &decidedAt)
	if err != nil {
		return accessRequest{}, err
	}
	request.ExpiresAt, request.CreatedAt = parseTime(expires), parseTime(created)
	if decidedAt.Valid && decidedAt.String != "" {
		value := parseTime(decidedAt.String)
		request.DecidedAt = &value
	}
	if request.Status == "pending" && time.Now().UTC().After(request.ExpiresAt) {
		_, _ = p.db.ExecContext(ctx, `UPDATE pm_access_requests SET status='expired' WHERE id=$1 AND workspace_id=$2 AND status='pending'`, id, workspace)
		request.Status = "expired"
	}
	return request, nil
}

func (p *PasswordManager) waitForAccessRequest(ctx context.Context, workspace, id string, timeout time.Duration) (accessRequest, bool, error) {
	if timeout <= 0 || timeout > 10*time.Minute {
		timeout = 30 * time.Second
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		request, err := p.getAccessRequest(ctx, workspace, id)
		if err != nil {
			return accessRequest{}, false, err
		}
		if request.Status != "pending" {
			return request, false, nil
		}
		select {
		case <-ctx.Done():
			return request, true, ctx.Err()
		case <-deadline.C:
			return request, true, nil
		case <-p.requestWake:
		}
	}
}

func (p *PasswordManager) decideAccessRequest(ctx context.Context, workspace, actor, id, expectedDigest, decision, reason string) (accessRequest, error) {
	if decision != "approved" && decision != "denied" {
		return accessRequest{}, errors.New("invalid access decision")
	}
	if p.memory {
		p.mu.Lock()
		req, ok := p.requests[id]
		if !ok || req.WorkspaceID != workspace {
			p.mu.Unlock()
			return accessRequest{}, sql.ErrNoRows
		}
		if req.RequestedBy == actor {
			p.mu.Unlock()
			return accessRequest{}, errAccessDenied
		}
		if req.Status != "pending" || time.Now().After(req.ExpiresAt) {
			if req.Status == "pending" && time.Now().After(req.ExpiresAt) {
				req.Status = "expired"
				p.requests[id] = req
			}
			p.mu.Unlock()
			if req.Status == "expired" {
				return accessRequest{}, errAccessDenied
			}
			return accessRequest{}, errDecisionConflict
		}
		if expectedDigest != "" && expectedDigest != req.RequestDigest {
			p.mu.Unlock()
			return accessRequest{}, errItemConflict
		}
		now := time.Now().UTC()
		req.Status, req.DecidedBy, req.DecisionReason, req.DecidedAt = decision, actor, reason, &now
		p.requests[id] = req
		p.mu.Unlock()
		p.signalAccessRequestChange()
		return req, nil
	}
	req, err := p.getAccessRequest(ctx, workspace, id)
	if err != nil {
		return accessRequest{}, err
	}
	if req.RequestedBy == actor {
		return accessRequest{}, errAccessDenied
	}
	if req.Status != "pending" {
		if req.Status == "expired" {
			return accessRequest{}, errAccessDenied
		}
		return accessRequest{}, errDecisionConflict
	}
	if time.Now().UTC().After(req.ExpiresAt) {
		_, _ = p.db.ExecContext(ctx, `UPDATE pm_access_requests SET status='expired' WHERE id=$1 AND workspace_id=$2 AND status='pending'`, id, workspace)
		return accessRequest{}, errAccessDenied
	}
	if expectedDigest != "" && expectedDigest != req.RequestDigest {
		return accessRequest{}, errItemConflict
	}
	result, err := p.db.ExecContext(ctx, `UPDATE pm_access_requests SET status=$1,decided_by=$2,decision_reason=$3,decided_at=CURRENT_TIMESTAMP WHERE id=$4 AND workspace_id=$5 AND status='pending' AND expires_at > CURRENT_TIMESTAMP`, decision, actor, reason, id, workspace)
	if err != nil {
		return accessRequest{}, err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		current, getErr := p.getAccessRequest(ctx, workspace, id)
		if getErr == nil && current.Status != "pending" && current.Status != "expired" {
			return accessRequest{}, errDecisionConflict
		}
		return accessRequest{}, errAccessDenied
	}
	req.Status, req.DecidedBy, req.DecisionReason = decision, actor, reason
	now := time.Now().UTC()
	req.DecidedAt = &now
	p.signalAccessRequestChange()
	return req, nil
}

type passwordManagerHandlers struct {
	manager        *PasswordManager
	backupEvidence func(context.Context) recoveryEvidenceSnapshot
}

func newPasswordManagerHandlers(db *database.RoutedDB) *passwordManagerHandlers {
	return &passwordManagerHandlers{manager: newPasswordManager(db)}
}

func (h *passwordManagerHandlers) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/enrollment/status", h.enrollmentStatus).Methods(http.MethodGet)
	r.HandleFunc("/enrollment/complete", h.completeEnrollment).Methods(http.MethodPost)
	r.HandleFunc("/vaults/{vault}/status", h.status).Methods(http.MethodGet)
	r.HandleFunc("/vaults/{vault}/unlock", h.unlock).Methods(http.MethodPost)
	r.HandleFunc("/vaults/{vault}/lock", h.lock).Methods(http.MethodPost)
	r.HandleFunc("/vaults/{vault}/items", h.list).Methods(http.MethodGet)
	r.HandleFunc("/vaults/{vault}/items", h.create).Methods(http.MethodPost)
	r.HandleFunc("/vaults/{vault}/items/{id}", h.get).Methods(http.MethodGet)
	r.HandleFunc("/vaults/{vault}/items/{id}", h.update).Methods(http.MethodPut)
	r.HandleFunc("/vaults/{vault}/items/{id}", h.trash).Methods(http.MethodDelete)
	r.HandleFunc("/vaults/{vault}/items/{id}/restore", h.restore).Methods(http.MethodPost)
	r.HandleFunc("/vaults/{vault}/items/{id}/reveal", h.reveal).Methods(http.MethodPost)
	r.HandleFunc("/vaults/{vault}/items/{id}/totp", h.totpCode).Methods(http.MethodPost)
	r.HandleFunc("/vaults/{vault}/items/{id}/history", h.history).Methods(http.MethodGet)
	r.HandleFunc("/vaults/{vault}/export", h.export).Methods(http.MethodPost)
	r.HandleFunc("/exports/{id}/redeem", h.redeemExport).Methods(http.MethodPost)
	r.HandleFunc("/vaults/{vault}/imports/preview", h.previewImport).Methods(http.MethodPost)
	r.HandleFunc("/vaults/{vault}/imports", h.commitImport).Methods(http.MethodPost)
	r.HandleFunc("/recovery/status", h.recoveryStatus).Methods(http.MethodGet)
	r.HandleFunc("/recovery/activate", h.activateRecovery).Methods(http.MethodPost)
	r.HandleFunc("/assurance", h.assurance).Methods(http.MethodPost)
	r.HandleFunc("/password/generate", h.generate).Methods(http.MethodPost)
	r.HandleFunc("/grants", h.createGrant).Methods(http.MethodPost)
	r.HandleFunc("/grants", h.listGrants).Methods(http.MethodGet)
	r.HandleFunc("/grants/{id}", h.getGrant).Methods(http.MethodGet)
	r.HandleFunc("/grants/{id}/effective-access", h.effectiveGrantAccess).Methods(http.MethodGet)
	r.HandleFunc("/grants/{id}/revoke", h.revokeGrant).Methods(http.MethodPost)
	r.HandleFunc("/access-requests", h.createAccessRequest).Methods(http.MethodPost)
	r.HandleFunc("/access-requests", h.listAccessRequests).Methods(http.MethodGet)
	r.HandleFunc("/access-requests/{id}", h.getAccessRequest).Methods(http.MethodGet)
	r.HandleFunc("/access-requests/{id}/wait", h.waitAccessRequest).Methods(http.MethodPost)
	r.HandleFunc("/access-requests/{id}/approve", h.approveAccessRequest).Methods(http.MethodPost)
	r.HandleFunc("/access-requests/{id}/deny", h.denyAccessRequest).Methods(http.MethodPost)
	r.HandleFunc("/audit", h.audit).Methods(http.MethodGet)
	r.HandleFunc("/audit/export", h.auditExport).Methods(http.MethodGet)
	r.HandleFunc("/members", h.listMembers).Methods(http.MethodGet)
	r.HandleFunc("/members", h.addMember).Methods(http.MethodPost)
	r.HandleFunc("/members/{principal}", h.removeMember).Methods(http.MethodDelete)
	r.HandleFunc("/machine-principals", h.enrollMachinePrincipal).Methods(http.MethodPost)
	r.HandleFunc("/machine-principals", h.listMachinePrincipals).Methods(http.MethodGet)
	r.HandleFunc("/machine-principals/{id}", h.revokeMachinePrincipal).Methods(http.MethodDelete)
	r.HandleFunc("/sources", h.createSource).Methods(http.MethodPost)
	r.HandleFunc("/sources", h.listSources).Methods(http.MethodGet)
	r.HandleFunc("/sources/{id}/health", h.sourceHealth).Methods(http.MethodGet)
	r.HandleFunc("/vaults/{vault}/items/{id}/source-binding", h.bindSource).Methods(http.MethodPost)
	h.brokerRegisterRoutes(r)
	h.nativeHostRegisterRoutes(r)
	h.credentialUseRegisterRoutes(r)
}

func requestIdentity(r *http.Request) (string, string) {
	if auth, ok := r.Context().Value(ownerAuthContextKey{}).(ownerAuthContext); ok {
		return auth.WorkspaceID, auth.PrincipalID
	}
	workspace := strings.TrimSpace(r.Header.Get("X-Workspace-ID"))
	if workspace == "" {
		workspace = "local"
	}
	actor := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
	if actor == "" {
		actor = "local-owner"
	}
	return workspace, actor
}

func (h *passwordManagerHandlers) authorize(r *http.Request) error {
	if h.manager.memory {
		if auth, ok := r.Context().Value(ownerAuthContextKey{}).(ownerAuthContext); ok && (auth.Role == "agent" || auth.Role == "machine") {
			return nil
		}
		workspace, actor := requestIdentity(r)
		setOwnerAuthContext(r, ownerAuthContext{WorkspaceID: workspace, PrincipalID: actor, Role: "owner"})
		return nil
	}
	if auth, ok := r.Context().Value(ownerAuthContextKey{}).(ownerAuthContext); ok && auth.PrincipalID != "" && auth.Role == "" {
		if !workspaceAllowed(auth.WorkspaceID) || h.manager.db == nil {
			return errAccessDenied
		}
		var role, status string
		if err := h.manager.db.QueryRowContext(r.Context(), `SELECT role,status FROM pm_workspace_members WHERE workspace_id=$1 AND principal_id=$2`, auth.WorkspaceID, auth.PrincipalID).Scan(&role, &status); err != nil {
			return errAccessDenied
		}
		if status != "active" || !supportedWorkspaceRole(role) {
			return errAccessDenied
		}
		setOwnerAuthContext(r, ownerAuthContext{WorkspaceID: auth.WorkspaceID, PrincipalID: auth.PrincipalID, Role: role, AuthenticatedAt: auth.AuthenticatedAt})
		return nil
	}
	if auth, ok := r.Context().Value(ownerAuthContextKey{}).(ownerAuthContext); ok && auth.PrincipalID != "" && (auth.Role == "agent" || auth.Role == "machine") {
		return nil
	}
	workspace, _ := requestIdentityFromHeaders(r)
	if !workspaceAllowed(workspace) {
		return errAccessDenied
	}
	configured, _ := configuredSecretsManagerOwnerToken()
	raw := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer "))
	if raw == "" {
		return errAccessDenied
	}
	if configured != "" {
		if subtle.ConstantTimeCompare([]byte(raw), []byte(configured)) != 1 {
			return errAccessDenied
		}
		setOwnerAuthContext(r, ownerAuthContext{WorkspaceID: workspace, PrincipalID: "local-owner", Role: "owner"})
		return nil
	}
	if h.manager.db == nil {
		return errOwnerAuthMissing
	}
	digest := ownerTokenDigest(raw)
	var principalID, role, status string
	if err := h.manager.db.QueryRowContext(r.Context(), `SELECT principal_id,role,status FROM pm_owner_tokens WHERE token_digest=$1 AND workspace_id=$2`, digest, workspace).Scan(&principalID, &role, &status); err != nil {
		return errAccessDenied
	}
	if status != "active" || !supportedWorkspaceRole(role) {
		return errAccessDenied
	}
	setOwnerAuthContext(r, ownerAuthContext{WorkspaceID: workspace, PrincipalID: principalID, Role: role})
	return nil
}

func supportedWorkspaceRole(role string) bool {
	switch role {
	case "owner", "admin", "member", "viewer", "security-operator":
		return true
	default:
		return false
	}
}

type workspaceCapability string

const (
	capabilityMetadataRead workspaceCapability = "metadata.read"
	capabilityItemWrite    workspaceCapability = "item.write"
	capabilitySecretReveal workspaceCapability = "secret.reveal"
	capabilityGrantManage  workspaceCapability = "grant.manage"
	capabilityAuditRead    workspaceCapability = "audit.read"
	capabilityMemberManage workspaceCapability = "member.manage"
	capabilityVaultManage  workspaceCapability = "vault.manage"
)

func requireCapability(r *http.Request, capability workspaceCapability) error {
	auth, ok := r.Context().Value(ownerAuthContextKey{}).(ownerAuthContext)
	if !ok || auth.PrincipalID == "" {
		return errAccessDenied
	}
	if auth.Role == "agent" {
		if capability == capabilitySecretReveal || capability == capabilityMetadataRead {
			return nil
		}
		return errAccessDenied
	}
	if auth.Role == "machine" {
		if capability == capabilitySecretReveal || capability == capabilityMetadataRead {
			return nil
		}
		return errAccessDenied
	}
	allowed := map[string]map[workspaceCapability]bool{
		"owner": {
			capabilityMetadataRead: true, capabilityItemWrite: true, capabilitySecretReveal: true,
			capabilityGrantManage: true, capabilityAuditRead: true, capabilityMemberManage: true, capabilityVaultManage: true,
		},
		"admin": {
			capabilityMetadataRead: true, capabilityItemWrite: true, capabilitySecretReveal: true,
			capabilityGrantManage: true, capabilityAuditRead: true, capabilityMemberManage: true, capabilityVaultManage: true,
		},
		"member": {
			capabilityMetadataRead: true, capabilityItemWrite: true, capabilitySecretReveal: true, capabilityVaultManage: true,
		},
		"viewer": {
			capabilityMetadataRead: true,
		},
		"security-operator": {
			capabilityMetadataRead: true, capabilityAuditRead: true,
		},
	}
	if allowed[auth.Role][capability] {
		return nil
	}
	return errAccessDenied
}

func requestIdentityFromHeaders(r *http.Request) (string, string) {
	workspace := strings.TrimSpace(r.Header.Get("X-Workspace-ID"))
	if workspace == "" {
		workspace = "local"
	}
	actor := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
	if actor == "" {
		actor = "local-owner"
	}
	return workspace, actor
}

func setOwnerAuthContext(r *http.Request, auth ownerAuthContext) {
	*r = *r.WithContext(context.WithValue(r.Context(), ownerAuthContextKey{}, auth))
}

func (p *PasswordManager) enrollMachinePrincipal(ctx context.Context, workspace, actor string, input machinePrincipalInput) (machinePrincipalRecord, string, error) {
	input.MachineID = strings.TrimSpace(input.MachineID)
	input.Label = strings.TrimSpace(input.Label)
	if input.MachineID == "" || len(input.MachineID) > 128 || input.Label == "" || len(input.Label) > 128 {
		return machinePrincipalRecord{}, "", errors.New("machine_id and label are required and must be at most 128 characters")
	}
	raw := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return machinePrincipalRecord{}, "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	record := machinePrincipalRecord{ID: "machine:" + uuid.NewString(), WorkspaceID: workspace, MachineID: input.MachineID, Label: input.Label, TokenDigest: ownerTokenDigest(token), Status: "active", CreatedBy: actor, CreatedAt: time.Now().UTC()}
	if p.memory {
		p.mu.Lock()
		defer p.mu.Unlock()
		for _, existing := range p.machines {
			if existing.WorkspaceID == workspace && existing.MachineID == input.MachineID && existing.Status == "active" {
				return machinePrincipalRecord{}, "", errMachineExists
			}
		}
		p.machines[record.ID] = record
		return record, token, nil
	}
	_, err := p.db.ExecContext(ctx, `INSERT INTO pm_machine_principals(id,workspace_id,machine_id,label,token_digest,status,created_by) VALUES($1,$2,$3,$4,$5,'active',$6)`, record.ID, record.WorkspaceID, record.MachineID, record.Label, record.TokenDigest, record.CreatedBy)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return machinePrincipalRecord{}, "", errMachineExists
		}
		return machinePrincipalRecord{}, "", err
	}
	return record, token, nil
}

// AuthenticateMachinePrincipal verifies a one-time-disclosed machine token.
// It returns a machine actor context; it never creates an Agent Manager
// identity and never accepts a caller-supplied principal ID.
func (p *PasswordManager) AuthenticateMachinePrincipal(ctx context.Context, workspace, token string) (ownerAuthContext, error) {
	workspace = strings.TrimSpace(workspace)
	token = strings.TrimSpace(token)
	if workspace == "" || token == "" {
		return ownerAuthContext{}, errAccessDenied
	}
	digest := ownerTokenDigest(token)
	if p.memory {
		p.mu.RLock()
		defer p.mu.RUnlock()
		for _, record := range p.machines {
			if record.WorkspaceID == workspace && record.TokenDigest == digest && record.Status == "active" {
				return ownerAuthContext{WorkspaceID: workspace, PrincipalID: record.ID, Role: "machine"}, nil
			}
		}
		return ownerAuthContext{}, errAccessDenied
	}
	var principalID, status string
	if err := p.db.QueryRowContext(ctx, `SELECT id,status FROM pm_machine_principals WHERE token_digest=$1 AND workspace_id=$2`, digest, workspace).Scan(&principalID, &status); err != nil || status != "active" {
		return ownerAuthContext{}, errAccessDenied
	}
	return ownerAuthContext{WorkspaceID: workspace, PrincipalID: principalID, Role: "machine"}, nil
}

func (h *passwordManagerHandlers) enrollMachinePrincipal(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireApprover(r); err != nil {
		writeVaultError(w, err)
		return
	}
	var input machinePrincipalInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	record, token, err := h.manager.enrollMachinePrincipal(r.Context(), workspace, actor, input)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	h.manager.recordAudit(r.Context(), auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "machine_principal.enroll", Outcome: "success", Detail: "token disclosed once"})
	writeJSON(w, http.StatusCreated, map[string]any{"id": record.ID, "workspace_id": workspace, "machine_id": record.MachineID, "label": record.Label, "status": record.Status, "enrollment_token": token, "token_once": true})
}

func (h *passwordManagerHandlers) listMachinePrincipals(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMemberManage); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	principals := make([]map[string]any, 0)
	if h.manager.memory {
		h.manager.mu.RLock()
		for _, record := range h.manager.machines {
			if record.WorkspaceID == workspace {
				principals = append(principals, map[string]any{"id": record.ID, "workspace_id": workspace, "machine_id": record.MachineID, "label": record.Label, "status": record.Status, "created_at": record.CreatedAt})
			}
		}
		h.manager.mu.RUnlock()
	} else {
		rows, err := h.manager.db.QueryContext(r.Context(), `SELECT id,workspace_id,machine_id,label,status,created_at FROM pm_machine_principals WHERE workspace_id=$1 ORDER BY created_at`, workspace)
		if err != nil {
			writeVaultError(w, err)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id, workspaceID, machineID, label, status string
			var createdAt time.Time
			if err := rows.Scan(&id, &workspaceID, &machineID, &label, &status, &createdAt); err != nil {
				writeVaultError(w, err)
				return
			}
			principals = append(principals, map[string]any{"id": id, "workspace_id": workspaceID, "machine_id": machineID, "label": label, "status": status, "created_at": createdAt})
		}
		if err := rows.Err(); err != nil {
			writeVaultError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"machine_principals": principals})
}

func (h *passwordManagerHandlers) revokeMachinePrincipal(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireApprover(r); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	id := mux.Vars(r)["id"]
	if h.manager.memory {
		h.manager.mu.Lock()
		record, ok := h.manager.machines[id]
		if !ok || record.WorkspaceID != workspace {
			h.manager.mu.Unlock()
			writeVaultError(w, sql.ErrNoRows)
			return
		}
		record.Status = "revoked"
		record.RevokedAt = time.Now().UTC()
		h.manager.machines[id] = record
		h.manager.mu.Unlock()
	} else {
		result, err := h.manager.db.ExecContext(r.Context(), `UPDATE pm_machine_principals SET status='revoked',revoked_at=CURRENT_TIMESTAMP WHERE id=$1 AND workspace_id=$2 AND status='active'`, id, workspace)
		if err != nil {
			writeVaultError(w, err)
			return
		}
		if count, _ := result.RowsAffected(); count == 0 {
			writeVaultError(w, sql.ErrNoRows)
			return
		}
	}
	h.manager.recordAudit(r.Context(), auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "machine_principal.revoke", Outcome: "success", Detail: "machine token revoked"})
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": "revoked"})
}

func ownerTokenDigest(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func workspaceAllowed(workspace string) bool {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return false
	}
	configured := strings.TrimSpace(getEnv("SECRETS_MANAGER_WORKSPACE_ID"))
	if configured == "" {
		configured = "local"
	}
	if workspace == configured {
		return true
	}
	for _, candidate := range strings.Split(getEnv("SECRETS_MANAGER_ALLOWED_WORKSPACES"), ",") {
		if strings.TrimSpace(candidate) == workspace {
			return true
		}
	}
	return false
}

type enrollmentInput struct {
	BootstrapToken string `json:"bootstrap_token"`
}

func (h *passwordManagerHandlers) enrollmentStatus(w http.ResponseWriter, r *http.Request) {
	workspace, _ := requestIdentityFromHeaders(r)
	if !workspaceAllowed(workspace) {
		writeVaultError(w, errAccessDenied)
		return
	}
	status := map[string]any{
		"workspace_id":         workspace,
		"bootstrap_configured": strings.TrimSpace(getEnv("SECRETS_MANAGER_OWNER_BOOTSTRAP_TOKEN")) != "",
		"role_model":           []string{"owner", "admin", "member", "viewer"},
		"enrolled":             false,
	}
	if h.manager.memory {
		h.manager.mu.RLock()
		_, status["enrolled"] = h.manager.enrollments[workspace]
		h.manager.mu.RUnlock()
		writeJSON(w, http.StatusOK, status)
		return
	}
	var enrolled bool
	err := h.manager.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM pm_workspace_enrollments WHERE workspace_id=$1 AND status='enrolled')`, workspace).Scan(&enrolled)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	status["enrolled"] = enrolled
	writeJSON(w, http.StatusOK, status)
}

func (h *passwordManagerHandlers) completeEnrollment(w http.ResponseWriter, r *http.Request) {
	var input enrollmentInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	configured := strings.TrimSpace(getEnv("SECRETS_MANAGER_OWNER_BOOTSTRAP_TOKEN"))
	provided := strings.TrimSpace(input.BootstrapToken)
	if configured == "" || provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(configured)) != 1 {
		writeVaultError(w, errOwnerAuthMissing)
		return
	}
	workspace, _ := requestIdentityFromHeaders(r)
	if !workspaceAllowed(workspace) {
		writeVaultError(w, errAccessDenied)
		return
	}
	ownerToken, _, err := randomCapability()
	if err != nil {
		writeVaultError(w, err)
		return
	}
	digest := ownerTokenDigest(ownerToken)
	principalID := "owner"
	if h.manager.memory {
		h.manager.mu.Lock()
		defer h.manager.mu.Unlock()
		if _, exists := h.manager.enrollments[workspace]; exists {
			writeVaultError(w, errWorkspaceEnrolled)
			return
		}
		h.manager.enrollments[workspace] = workspaceEnrollment{WorkspaceID: workspace, Status: "enrolled", EnrolledBy: principalID, EnrolledAt: time.Now().UTC()}
		h.manager.members[workspace+":"+principalID] = workspaceMember{ID: uuid.NewString(), WorkspaceID: workspace, PrincipalID: principalID, Role: "owner", Status: "active", CreatedAt: time.Now().UTC()}
		h.manager.ownerTokens[digest] = ownerTokenRecord{WorkspaceID: workspace, PrincipalID: principalID, Role: "owner", Status: "active"}
		writeJSON(w, http.StatusCreated, map[string]any{"workspace_id": workspace, "principal_id": principalID, "role": "owner", "owner_token": ownerToken, "token_once": true})
		return
	}
	tx, err := h.manager.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	result, err := tx.ExecContext(r.Context(), `INSERT INTO pm_workspace_enrollments(workspace_id,status,enrolled_by) VALUES($1,'enrolled',$2) ON CONFLICT (workspace_id) DO NOTHING`, workspace, principalID)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	if count, _ := result.RowsAffected(); count != 1 {
		writeVaultError(w, errWorkspaceEnrolled)
		return
	}
	if _, err = tx.ExecContext(r.Context(), `INSERT INTO pm_workspace_members(id,workspace_id,principal_id,role,status) VALUES($1,$2,$3,'owner','active')`, uuid.NewString(), workspace, principalID); err != nil {
		writeVaultError(w, err)
		return
	}
	if _, err = tx.ExecContext(r.Context(), `INSERT INTO pm_owner_tokens(token_digest,workspace_id,principal_id,role,status) VALUES($1,$2,$3,'owner','active')`, digest, workspace, principalID); err != nil {
		writeVaultError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		writeVaultError(w, err)
		return
	}
	committed = true
	writeJSON(w, http.StatusCreated, map[string]any{"workspace_id": workspace, "principal_id": principalID, "role": "owner", "owner_token": ownerToken, "token_once": true})
}

func (h *passwordManagerHandlers) status(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	id := mux.Vars(r)["vault"]
	v, err := h.manager.ensureVault(r.Context(), workspace, id)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vault_id": id, "workspace_id": workspace, "status": v.Status, "key_available": len(h.manager.key) == 32, "supports_recovery": true})
}

// recoveryStatus reports local custody and the evidence actually available to
// this scenario. Backup and replacement-host restore are owned by Data Backup
// Manager; they must not be presented as complete until that owner supplies a
// verified receipt.
func (h *passwordManagerHandlers) recoveryStatus(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	epoch, epochErr := h.manager.currentRecoveryEpoch(r.Context(), workspace)
	if epochErr != nil {
		writeVaultError(w, epochErr)
		return
	}
	if h.backupEvidence != nil {
		evidence := h.backupEvidence(r.Context())
		writeJSON(w, http.StatusOK, map[string]any{
			"key_available":      len(h.manager.key) == 32,
			"backup_status":      evidence.BackupStatus,
			"restore_status":     evidence.RestoreStatus,
			"recovery_epoch":     epoch,
			"activation_status":  "active",
			"stale_capabilities": "rejected",
			"evidence_owner":     "data-backup-manager",
			"recovery_ready":     evidence.RecoveryReady,
			"evidence":           evidence.Evidence,
			"remediation":        evidence.Remediation,
			"evidence_updated":   evidence.UpdatedAt,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"key_available":      len(h.manager.key) == 32,
		"backup_status":      "not_configured",
		"restore_status":     "not_run",
		"recovery_epoch":     epoch,
		"activation_status":  "active",
		"stale_capabilities": "rejected",
		"evidence_owner":     "data-backup-manager",
		"recovery_ready":     false,
	})
}

func (h *passwordManagerHandlers) unlock(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityVaultManage); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	id := mux.Vars(r)["vault"]
	if err := h.manager.lock(r.Context(), workspace, id, false); err != nil {
		writeVaultError(w, err)
		return
	}
	h.manager.recordAudit(r.Context(), auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "vault.unlock", Outcome: "success", Detail: "vault state changed"})
	writeJSON(w, http.StatusOK, map[string]any{"vault_id": id, "status": "unlocked", "unlocked_by": actor})
}

func (h *passwordManagerHandlers) lock(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityVaultManage); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	id := mux.Vars(r)["vault"]
	if err := h.manager.lock(r.Context(), workspace, id, true); err != nil {
		writeVaultError(w, err)
		return
	}
	h.manager.recordAudit(r.Context(), auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "vault.lock", Outcome: "success", Detail: "vault state changed"})
	writeJSON(w, http.StatusOK, map[string]any{"vault_id": id, "status": "locked"})
}

func (h *passwordManagerHandlers) list(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	v := mux.Vars(r)["vault"]
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.manager.listItems(r.Context(), workspace, v, r.URL.Query().Get("q"), r.URL.Query().Get("include_trashed") == "true", page, limit)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "page": maxInt(page, 1), "limit": maxInt(limit, 50)})
}

func (h *passwordManagerHandlers) get(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	item, err := h.manager.metadata(r.Context(), workspace, mux.Vars(r)["vault"], mux.Vars(r)["id"])
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *passwordManagerHandlers) history(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityAuditRead); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	vars := mux.Vars(r)
	records, err := h.manager.listHistory(r.Context(), workspace, vars["vault"], vars["id"])
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": records})
}

func decodeJSON(r *http.Request, dst any) error {
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024+1))
	if err != nil {
		return err
	}
	if len(payload) > 1024*1024 {
		return errors.New("request body exceeds 1 MiB")
	}
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func (h *passwordManagerHandlers) create(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityItemWrite); err != nil {
		writeVaultError(w, err)
		return
	}
	var input createItemInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	item, err := h.manager.createItem(r.Context(), workspace, mux.Vars(r)["vault"], actor, r.Header.Get("Idempotency-Key"), input)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *passwordManagerHandlers) update(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityItemWrite); err != nil {
		writeVaultError(w, err)
		return
	}
	var input createItemInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	expected, _ := strconv.Atoi(r.Header.Get("If-Match"))
	if expected < 1 {
		writeVaultError(w, errors.New("If-Match must contain the current item revision"))
		return
	}
	workspace, actor := requestIdentity(r)
	item, err := h.manager.updateItem(r.Context(), workspace, mux.Vars(r)["vault"], actor, mux.Vars(r)["id"], expected, input)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *passwordManagerHandlers) trash(w http.ResponseWriter, r *http.Request) {
	h.setTrash(w, r, true)
}

func (h *passwordManagerHandlers) restore(w http.ResponseWriter, r *http.Request) {
	h.setTrash(w, r, false)
}

func (h *passwordManagerHandlers) setTrash(w http.ResponseWriter, r *http.Request, trashed bool) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityItemWrite); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	err := h.manager.setTrashed(r.Context(), workspace, mux.Vars(r)["vault"], mux.Vars(r)["id"], trashed)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	h.manager.recordAudit(r.Context(), auditEvent{WorkspaceID: workspace, ActorID: actor, Action: map[bool]string{true: "item.trash", false: "item.restore"}[trashed], ItemID: mux.Vars(r)["id"], Outcome: "success"})
	writeJSON(w, http.StatusOK, map[string]any{"id": mux.Vars(r)["id"], "trashed": trashed})
}

func (h *passwordManagerHandlers) assurance(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilitySecretReveal); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireFreshActionAuthentication(r, h.manager.memory); err != nil {
		writeVaultError(w, err)
		return
	}
	var body struct {
		Operation     string `json:"operation"`
		RequestDigest string `json:"request_digest"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	token, expires, err := h.manager.issueAssurance(r.Context(), workspace, actor, body.Operation, body.RequestDigest)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"assurance_token": token, "expires_at": expires, "operation": body.Operation})
}

func requireFreshActionAuthentication(r *http.Request, memoryMode bool) error {
	if memoryMode {
		return nil
	}
	auth, ok := r.Context().Value(ownerAuthContextKey{}).(ownerAuthContext)
	if !ok || auth.Role == "agent" || auth.AuthenticatedAt.IsZero() {
		return errAssuranceRequired
	}
	age := time.Since(auth.AuthenticatedAt)
	if age < -30*time.Second || age > 10*time.Minute {
		return errAssuranceRequired
	}
	return nil
}

func (h *passwordManagerHandlers) reveal(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilitySecretReveal); err != nil {
		writeVaultError(w, err)
		return
	}
	var body struct {
		AssuranceToken string   `json:"assurance_token"`
		RequestDigest  string   `json:"request_digest"`
		Fields         []string `json:"fields"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	vars := mux.Vars(r)
	fields, err := h.manager.reveal(r.Context(), workspace, vars["vault"], vars["id"], actor, body.AssuranceToken, body.RequestDigest, body.Fields)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item_id": vars["id"], "fields": fields, "secret_bearing": true})
}

func (h *passwordManagerHandlers) generate(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	var body struct {
		Length  int  `json:"length"`
		Upper   bool `json:"upper"`
		Lower   bool `json:"lower"`
		Digits  bool `json:"digits"`
		Symbols bool `json:"symbols"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeVaultError(w, err)
		return
	}
	if body.Length == 0 {
		body.Length = 24
	}
	if !body.Upper && !body.Lower && !body.Digits && !body.Symbols {
		body.Upper, body.Lower, body.Digits, body.Symbols = true, true, true, true
	}
	value, err := generatePassword(body.Length, body.Upper, body.Lower, body.Digits, body.Symbols)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"password": value, "length": len([]rune(value))})
}

func (h *passwordManagerHandlers) createGrant(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireApprover(r); err != nil {
		writeVaultError(w, err)
		return
	}
	var input createGrantInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	grant, err := h.manager.createGrant(r.Context(), workspace, actor, input)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, grant)
}

func requireApprover(r *http.Request) error {
	if auth, ok := r.Context().Value(ownerAuthContextKey{}).(ownerAuthContext); ok && (auth.Role == "owner" || auth.Role == "admin") {
		return nil
	}
	return errAccessDenied
}

func (h *passwordManagerHandlers) getGrant(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	grant, err := h.manager.getGrant(r.Context(), workspace, mux.Vars(r)["id"])
	if err != nil {
		writeVaultError(w, err)
		return
	}
	grant.ExpiresAt = grant.ExpiresAt.UTC()
	writeJSON(w, http.StatusOK, grant)
}

func (h *passwordManagerHandlers) effectiveGrantAccess(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	grant, err := h.manager.getGrant(r.Context(), workspace, mux.Vars(r)["id"])
	if err != nil {
		writeVaultError(w, err)
		return
	}
	operation := strings.TrimSpace(r.URL.Query().Get("operation"))
	if operation == "" {
		operation = "use"
	}
	itemID := strings.TrimSpace(r.URL.Query().Get("item_id"))
	if itemID == "" {
		itemID = grant.ItemID
	}
	allowed := h.manager.grantAllows(r.Context(), grant, actor, itemID, operation, time.Now().UTC())
	reason := "selector, operation, target, and lifetime permit the requested use"
	if !allowed {
		reason = "grant is expired, revoked, outside its item or operation scope, or actor membership is unavailable"
	}
	writeJSON(w, http.StatusOK, grantAccessExplanation{
		GrantID: grant.ID, WorkspaceID: grant.WorkspaceID, Actor: actor, ItemID: itemID, Operation: operation,
		Decision: map[bool]string{true: "allow", false: "deny"}[allowed], Reason: reason,
		SelectorMode: grant.SelectorMode, FutureMembers: grant.SelectorMode == "dynamic", RawReadPermitted: contains(grant.Operations, "reveal") || contains(grant.Operations, "export"),
	})
}

func (h *passwordManagerHandlers) listGrants(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	grants, err := h.manager.listGrants(r.Context(), workspace)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"grants": grants})
}

func (h *passwordManagerHandlers) revokeGrant(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireApprover(r); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	id := mux.Vars(r)["id"]
	if h.manager.memory {
		h.manager.mu.Lock()
		grant, ok := h.manager.grants[id]
		if !ok || grant.WorkspaceID != workspace {
			h.manager.mu.Unlock()
			writeVaultError(w, sql.ErrNoRows)
			return
		}
		grant.Status = "revoked"
		h.manager.grants[id] = grant
		for requestID, access := range h.manager.requests {
			if access.GrantID == id && access.Status == "pending" {
				access.Status = "denied"
				h.manager.requests[requestID] = access
			}
		}
		for sessionID, session := range h.manager.brokers {
			if session.GrantID == id && session.Status == "active" {
				session.Status = "revoked"
				h.manager.brokers[sessionID] = session
			}
		}
		h.manager.mu.Unlock()
		h.manager.terminateCredentialUse(r.Context(), workspace, id, "", "grant_revoked")
		writeJSON(w, http.StatusOK, map[string]any{"grant_id": id, "status": "revoked", "remote_purge": "not_applicable"})
		return
	}
	if _, err := h.manager.getGrant(r.Context(), workspace, id); err != nil {
		writeVaultError(w, err)
		return
	}
	if _, err := h.manager.db.ExecContext(r.Context(), `UPDATE pm_grants SET status='revoked' WHERE id=$1 AND workspace_id=$2`, id, workspace); err != nil {
		writeVaultError(w, err)
		return
	}
	_, _ = h.manager.db.ExecContext(r.Context(), `UPDATE pm_broker_sessions SET status='revoked' WHERE grant_id=$1 AND workspace_id=$2 AND status='active'`, id, workspace)
	h.manager.terminateCredentialUse(r.Context(), workspace, id, "", "grant_revoked")
	writeJSON(w, http.StatusOK, map[string]any{"grant_id": id, "status": "revoked", "remote_purge": "pending"})
}

func (h *passwordManagerHandlers) createAccessRequest(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	var input createAccessRequestInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	input.IdempotencyKey = r.Header.Get("Idempotency-Key")
	request, err := h.manager.createAccessRequest(r.Context(), workspace, actor, input)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, request)
}

func (h *passwordManagerHandlers) getAccessRequest(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	request, err := h.manager.getAccessRequest(r.Context(), workspace, mux.Vars(r)["id"])
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, request)
}

func (h *passwordManagerHandlers) waitAccessRequest(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	var body struct {
		TimeoutSeconds int `json:"timeout_seconds"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if err := decodeJSON(r, &body); err != nil {
			writeVaultError(w, err)
			return
		}
	}
	workspace, _ := requestIdentity(r)
	request, timedOut, err := h.manager.waitForAccessRequest(r.Context(), workspace, mux.Vars(r)["id"], time.Duration(body.TimeoutSeconds)*time.Second)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"request": request, "timed_out": timedOut})
}

func (h *passwordManagerHandlers) listAccessRequests(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	requests, err := h.manager.listAccessRequests(r.Context(), workspace)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": requests})
}

func (h *passwordManagerHandlers) decideAccessRequest(w http.ResponseWriter, r *http.Request, decision string) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireApprover(r); err != nil {
		writeVaultError(w, err)
		return
	}
	var body struct {
		RequestDigest  string `json:"request_digest"`
		Reason         string `json:"reason"`
		AssuranceToken string `json:"assurance_token"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if err := decodeJSON(r, &body); err != nil {
			writeVaultError(w, err)
			return
		}
	}
	workspace, actor := requestIdentity(r)
	request, err := h.manager.decideAccessRequest(r.Context(), workspace, actor, mux.Vars(r)["id"], body.RequestDigest, decision, body.Reason)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, request)
}

func (h *passwordManagerHandlers) approveAccessRequest(w http.ResponseWriter, r *http.Request) {
	h.decideAccessRequest(w, r, "approved")
}

func (h *passwordManagerHandlers) denyAccessRequest(w http.ResponseWriter, r *http.Request) {
	h.decideAccessRequest(w, r, "denied")
}

func (h *passwordManagerHandlers) audit(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityAuditRead); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	events, err := h.manager.listAudit(r.Context(), workspace)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events, "integrity": auditIntegritySummary(events)})
}

func auditIntegritySummary(events []auditEvent) string {
	for _, event := range events {
		if event.IntegrityStatus == "tampered" {
			return "tampered"
		}
	}
	for _, event := range events {
		if event.IntegrityStatus == "unknown" {
			return "unknown"
		}
	}
	return "verified"
}

func (h *passwordManagerHandlers) auditExport(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityAuditRead); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	events, err := h.manager.listAudit(r.Context(), workspace)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="audit-export.json"`)
	writeJSON(w, http.StatusOK, map[string]any{
		"version":        1,
		"workspace_id":   workspace,
		"exported_at":    time.Now().UTC(),
		"integrity":      auditIntegritySummary(events),
		"events":         events,
		"payload_policy": "metadata_only",
	})
}

func memberKey(workspace, principal string) string {
	return workspace + "\x00" + principal
}

func (p *PasswordManager) removePrincipalFromGrantSnapshots(ctx context.Context, workspace, principal string) {
	if p.memory {
		p.mu.Lock()
		defer p.mu.Unlock()
		for id, grant := range p.grants {
			if grant.WorkspaceID != workspace || grant.SelectorMode != "current_snapshot" || !contains(grant.Members, principal) {
				continue
			}
			members := make([]string, 0, len(grant.Members))
			for _, member := range grant.Members {
				if member != principal {
					members = append(members, member)
				}
			}
			grant.Members = members
			p.grants[id] = grant
		}
		return
	}
	rows, err := p.db.QueryContext(ctx, `SELECT id,members FROM pm_grants WHERE workspace_id=$1 AND selector_mode='current_snapshot'`, workspace)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, encoded string
		if rows.Scan(&id, &encoded) != nil {
			continue
		}
		var members []string
		if json.Unmarshal([]byte(encoded), &members) != nil || !contains(members, principal) {
			continue
		}
		filtered := make([]string, 0, len(members))
		for _, member := range members {
			if member != principal {
				filtered = append(filtered, member)
			}
		}
		updated, _ := json.Marshal(filtered)
		_, _ = p.db.ExecContext(ctx, `UPDATE pm_grants SET members=$1 WHERE id=$2 AND workspace_id=$3`, string(updated), id, workspace)
	}
}

func (h *passwordManagerHandlers) listMembers(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	if h.manager.memory {
		h.manager.mu.RLock()
		members := make([]workspaceMember, 0)
		for _, member := range h.manager.members {
			if member.WorkspaceID == workspace && member.Status == "active" {
				members = append(members, member)
			}
		}
		h.manager.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"members": members})
		return
	}
	rows, err := h.manager.db.QueryContext(r.Context(), `SELECT id,workspace_id,principal_id,role,status,created_at,COALESCE(revoked_at,'') FROM pm_workspace_members WHERE workspace_id=$1 ORDER BY created_at`, workspace)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	defer rows.Close()
	members := make([]workspaceMember, 0)
	for rows.Next() {
		var member workspaceMember
		var created, revoked string
		if err := rows.Scan(&member.ID, &member.WorkspaceID, &member.PrincipalID, &member.Role, &member.Status, &created, &revoked); err != nil {
			writeVaultError(w, err)
			return
		}
		member.CreatedAt, member.RevokedAt = parseTime(created), parseTime(revoked)
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": members})
}

func (h *passwordManagerHandlers) addMember(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMemberManage); err != nil {
		writeVaultError(w, err)
		return
	}
	var input workspaceMemberInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	input.PrincipalID = strings.TrimSpace(input.PrincipalID)
	input.Role = strings.TrimSpace(input.Role)
	if input.PrincipalID == "" || !supportedWorkspaceRole(input.Role) || input.Role == "owner" {
		writeVaultError(w, errors.New("principal_id and a non-owner workspace role are required"))
		return
	}
	workspace, actor := requestIdentity(r)
	member := workspaceMember{ID: uuid.NewString(), WorkspaceID: workspace, PrincipalID: input.PrincipalID, Role: input.Role, Status: "active", CreatedAt: time.Now().UTC()}
	if h.manager.memory {
		h.manager.mu.Lock()
		key := memberKey(workspace, input.PrincipalID)
		if existing, ok := h.manager.members[key]; ok && existing.Status == "active" {
			h.manager.mu.Unlock()
			writeVaultError(w, errMemberExists)
			return
		}
		h.manager.members[key] = member
		h.manager.mu.Unlock()
		h.manager.recordAudit(r.Context(), auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "workspace.member.add", Outcome: "success", Detail: "member capability granted"})
		writeJSON(w, http.StatusCreated, member)
		return
	}
	_, err := h.manager.db.ExecContext(r.Context(), `INSERT INTO pm_workspace_members(id,workspace_id,principal_id,role,status) VALUES($1,$2,$3,$4,'active')`, member.ID, workspace, member.PrincipalID, member.Role)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "constraint") {
			writeVaultError(w, errMemberExists)
		} else {
			writeVaultError(w, err)
		}
		return
	}
	h.manager.recordAudit(r.Context(), auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "workspace.member.add", Outcome: "success", Detail: "member capability granted"})
	writeJSON(w, http.StatusCreated, member)
}

func (h *passwordManagerHandlers) removeMember(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMemberManage); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	principal := strings.TrimSpace(mux.Vars(r)["principal"])
	if principal == "" {
		writeVaultError(w, errors.New("principal is required"))
		return
	}
	if h.manager.memory {
		h.manager.mu.Lock()
		key := memberKey(workspace, principal)
		member, ok := h.manager.members[key]
		if !ok || member.Status != "active" || member.Role == "owner" {
			h.manager.mu.Unlock()
			writeVaultError(w, errAccessDenied)
			return
		}
		now := time.Now().UTC()
		member.Status, member.RevokedAt = "revoked", now
		h.manager.members[key] = member
		h.manager.mu.Unlock()
		h.manager.removePrincipalFromGrantSnapshots(r.Context(), workspace, principal)
		writeJSON(w, http.StatusOK, map[string]any{"principal_id": principal, "status": "revoked"})
		return
	}
	var role, status string
	if err := h.manager.db.QueryRowContext(r.Context(), `SELECT role,status FROM pm_workspace_members WHERE workspace_id=$1 AND principal_id=$2`, workspace, principal).Scan(&role, &status); err != nil {
		writeVaultError(w, err)
		return
	}
	if status != "active" || role == "owner" {
		writeVaultError(w, errAccessDenied)
		return
	}
	result, err := h.manager.db.ExecContext(r.Context(), `UPDATE pm_workspace_members SET status='revoked',revoked_at=CURRENT_TIMESTAMP WHERE workspace_id=$1 AND principal_id=$2 AND status='active'`, workspace, principal)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	if count, _ := result.RowsAffected(); count != 1 {
		writeVaultError(w, errAccessDenied)
		return
	}
	_, _ = h.manager.db.ExecContext(r.Context(), `UPDATE pm_owner_tokens SET status='revoked',revoked_at=CURRENT_TIMESTAMP WHERE workspace_id=$1 AND principal_id=$2 AND status='active'`, workspace, principal)
	h.manager.removePrincipalFromGrantSnapshots(r.Context(), workspace, principal)
	h.manager.recordAudit(r.Context(), auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "workspace.member.remove", Outcome: "success", Detail: "member capability revoked"})
	writeJSON(w, http.StatusOK, map[string]any{"principal_id": principal, "status": "revoked"})
}

func generatePassword(length int, upper, lower, digits, symbols bool) (string, error) {
	if length < 8 || length > 128 {
		return "", errors.New("password length must be between 8 and 128")
	}
	sets := []string{}
	if upper {
		sets = append(sets, "ABCDEFGHJKLMNPQRSTUVWXYZ")
	}
	if lower {
		sets = append(sets, "abcdefghijkmnopqrstuvwxyz")
	}
	if digits {
		sets = append(sets, "23456789")
	}
	if symbols {
		sets = append(sets, "!@#$%^&*()-_=+")
	}
	if len(sets) == 0 {
		return "", errors.New("at least one character class is required")
	}
	all := strings.Join(sets, "")
	out := make([]byte, length)
	for i, set := range sets {
		n, err := randomIndex(len(set))
		if err != nil {
			return "", err
		}
		out[i] = set[n]
	}
	for i := len(sets); i < length; i++ {
		n, err := randomIndex(len(all))
		if err != nil {
			return "", err
		}
		out[i] = all[n]
	}
	for i := len(out) - 1; i > 0; i-- {
		n, err := randomIndex(i + 1)
		if err != nil {
			return "", err
		}
		out[i], out[n] = out[n], out[i]
	}
	return string(out), nil
}

func randomIndex(max int) (int, error) {
	if max <= 0 {
		return 0, errors.New("invalid random bound")
	}
	var b [1]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return 0, err
	}
	return int(b[0]) % max, nil
}

func maxInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func writeVaultError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	switch {
	case errors.Is(err, errVaultLocked):
		status = http.StatusLocked
		code = "vault_locked"
	case errors.Is(err, errVaultKeyMissing):
		status = http.StatusServiceUnavailable
		code = "recovery_required"
	case errors.Is(err, errItemConflict):
		status = http.StatusConflict
		code = "revision_conflict"
	case errors.Is(err, errDecisionConflict):
		status = http.StatusConflict
		code = "decision_conflict"
	case errors.Is(err, errRecoveryEpochConflict):
		status = http.StatusConflict
		code = "recovery_epoch_conflict"
	case errors.Is(err, errAssuranceRequired):
		status = http.StatusPreconditionRequired
		code = "assurance_required"
	case errors.Is(err, errAccessDenied):
		status = http.StatusForbidden
		code = "access_denied"
	case errors.Is(err, errOwnerAuthMissing):
		status = http.StatusServiceUnavailable
		code = "owner_auth_unconfigured"
	case errors.Is(err, errWorkspaceEnrolled):
		status = http.StatusConflict
		code = "workspace_already_enrolled"
	case errors.Is(err, errMemberExists):
		status = http.StatusConflict
		code = "workspace_member_exists"
	case errors.Is(err, sql.ErrNoRows):
		status = http.StatusNotFound
		code = "not_found"
	default:
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "unsupported") || strings.Contains(err.Error(), "must be") || strings.Contains(err.Error(), "invalid") {
			status = http.StatusBadRequest
			code = "invalid_request"
		}
	}
	writeJSON(w, status, map[string]any{"error": code, "message": err.Error()})
}

// compile-time use of the domain-owned schema keeps the owner visible to the
// scenario composition code and prevents accidental removal from boot.
var _ = vault.Schema
