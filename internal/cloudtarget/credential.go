package cloudtarget

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/credentialauthority"
)

// Credential lifecycle reason codes. The cloud side maps these onto its
// typed error model; do not rename without updating the certification matrix.
const (
	CodeCredentialStoreLocked     = "credential_store_locked"
	CodeCredentialPayloadInvalid  = "credential_payload_invalid"
	CodeCredentialVersionStale    = "credential_version_stale"
	CodeCredentialVersionRevoked  = "credential_version_revoked"
	CodeCredentialNotIngested     = "credential_not_ingested"
	CodeCredentialVersionMismatch = "credential_version_mismatch"
	CodeCredentialAuthorityFailed = "credential_authority_failed"
)

// IngestSchemaVersion is the sealed-payload schema `credential ingest`
// accepts on standard input. It matches scenario-to-cloud's IngestPayload.
const IngestSchemaVersion = 1

const credentialsDirName = "credentials"

// RevocationLimitation is stated on every revocation receipt.
const RevocationLimitation = "revocation deletes the stored value and refuses re-ingest of this version; it cannot prove that a process which already read the value forgot it, nor that no copy left this host"

// CredentialAuthority is the target-local credential authority surface the
// verbs need. *credentialauthority.Authority satisfies it.
type CredentialAuthority interface {
	Availability() error
	Put(identity credentialauthority.Identity, field, value string) error
	Delete(identity credentialauthority.Identity, field string) error
	Status(identity credentialauthority.Identity, field string) credentialauthority.Status
}

// CredentialDeps are the seams the credential verbs depend on. Zero values
// select the host authority, the process environment and standard input.
type CredentialDeps struct {
	Authority func() (CredentialAuthority, error)
	Env       func(string) string
	Stdin     io.Reader
}

func (d CredentialDeps) authority() (CredentialAuthority, error) {
	if d.Authority != nil {
		return d.Authority()
	}
	authority, err := credentialauthority.DefaultAuthority()
	if err != nil {
		return nil, err
	}
	return authority, nil
}

func (d CredentialDeps) env(name string) string {
	if d.Env != nil {
		return d.Env(name)
	}
	return os.Getenv(name)
}

func (d CredentialDeps) stdin() io.Reader {
	if d.Stdin != nil {
		return d.Stdin
	}
	return os.Stdin
}

// CredentialRecord is the target-local ledger entry for one binding: which
// version the authority currently holds and which versions were revoked.
// It never holds a value.
type CredentialRecord struct {
	SchemaVersion   int     `json:"schema_version"`
	DeploymentID    string  `json:"deployment_id"`
	BindingID       string  `json:"binding_id"`
	LogicalID       string  `json:"logical_id"`
	Field           string  `json:"field"`
	Version         int64   `json:"version"`
	ContentRef      string  `json:"content_ref"`
	GrantRef        string  `json:"grant_ref,omitempty"`
	IngestedAt      string  `json:"ingested_at"`
	Revoked         bool    `json:"revoked"`
	RevokedVersions []int64 `json:"revoked_versions"`
	UpdatedAt       string  `json:"updated_at"`
}

// IngestPayload is the standard-input document. The value is the only field
// that never appears anywhere else.
type IngestPayload struct {
	SchemaVersion int    `json:"schema_version"`
	DeploymentID  string `json:"deployment_id"`
	BindingID     string `json:"binding_id"`
	LogicalID     string `json:"logical_id"`
	Field         string `json:"field"`
	Version       int64  `json:"version"`
	ContentRef    string `json:"content_ref"`
	Value         string `json:"value,omitempty"`
	GrantRef      string `json:"grant_ref,omitempty"`
}

// CredentialIngestRequest identifies one version to ingest. Metadata may
// come from flags (bridge dispatch) or the stdin payload (ssh); the value
// comes from the payload or from the environment variable FromEnv names.
type CredentialIngestRequest struct {
	Effect     EffectRequest
	BindingID  string
	LogicalID  string
	Field      string
	Version    int64
	ContentRef string
	GrantRef   string
	FromEnv    string
}

// CredentialAckRequest asks the target to prove one consumer can use a
// version.
type CredentialAckRequest struct {
	Effect    EffectRequest
	BindingID string
	Version   int64
	Consumer  string
}

// CredentialRevokeRequest purges one version.
type CredentialRevokeRequest struct {
	Effect    EffectRequest
	BindingID string
	Version   int64
}

func (s *Store) credentialPath(deploymentID, bindingID string) (string, error) {
	dir, err := s.DeploymentDir(deploymentID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(bindingID) == "" || strings.ContainsAny(bindingID, "\n\r\x00") || len(bindingID) > 512 {
		return "", refuse(CodeInvalidArgument, "binding id %q is not a valid credential binding id", bindingID)
	}
	return filepath.Join(dir, credentialsDirName, bindingSlug(bindingID)+".json"), nil
}

// bindingSlug turns a cloud binding id (`<deployment>/<logical_id>:<field>`,
// which carries `/` and `:`) into one flat file name that still reads as
// the binding and cannot collide across ids that fold to the same text.
func bindingSlug(bindingID string) string {
	sum := sha256.Sum256([]byte(bindingID))
	var b strings.Builder
	for _, r := range bindingID {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '.', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	slug := b.String()
	if len(slug) > 96 {
		slug = slug[:96]
	}
	return slug + "-" + hex.EncodeToString(sum[:6])
}

// ReadCredentialRecord returns the ledger entry for a binding, or false.
func (s *Store) ReadCredentialRecord(deploymentID, bindingID string) (CredentialRecord, bool, error) {
	path, err := s.credentialPath(deploymentID, bindingID)
	if err != nil {
		return CredentialRecord{}, false, err
	}
	var record CredentialRecord
	switch err := readJSON(path, &record); {
	case err == nil:
		return record, true, nil
	case errors.Is(err, os.ErrNotExist):
		return CredentialRecord{}, false, nil
	default:
		return CredentialRecord{}, false, fail(CodeStoreIO, "read credential record: %v", err)
	}
}

func (s *Store) writeCredentialRecord(record CredentialRecord) error {
	path, err := s.credentialPath(record.DeploymentID, record.BindingID)
	if err != nil {
		return err
	}
	record.SchemaVersion = SchemaVersion
	record.UpdatedAt = s.now().Format(time.RFC3339Nano)
	if err := writeJSONAtomic(path, record); err != nil {
		return fail(CodeStoreIO, "write credential record: %v", err)
	}
	return nil
}

// authorityReady classifies the store condition before any write: a locked
// or unreachable store is credential_store_locked (refused, exit 2) and an
// absent store is credential_authority_failed.
func authorityReady(deps CredentialDeps) (CredentialAuthority, error) {
	authority, err := deps.authority()
	if err != nil {
		return nil, fail(CodeCredentialAuthorityFailed, "open credential authority: %v", err)
	}
	if err := authority.Availability(); err != nil {
		if errors.Is(err, credentialauthority.ErrProviderUnavailable) {
			return nil, refuse(CodeCredentialStoreLocked, "the credential store is locked or unreachable; nothing was written").withDetails(map[string]any{"detail": credentialauthority.ProviderDetail(err), "next_action": "vrooli credentials doctor"})
		}
		return nil, fail(CodeCredentialAuthorityFailed, "credential store unavailable: %v", err)
	}
	return authority, nil
}

// CredentialIngest writes one version into the target authority and records
// it in the ledger. The receipt input digest covers the metadata only, so a
// replay with a different value under the same (operation, step) is caught
// by version/content_ref, never by hashing the value.
func (s *Store) CredentialIngest(ctx context.Context, req CredentialIngestRequest, deps CredentialDeps) (EffectResult, error) {
	value, meta, err := resolveIngest(req, deps)
	if err != nil {
		return EffectResult{}, err
	}
	effect := req.Effect
	effect.Verb = "credential ingest"
	effect.Input = map[string]any{"binding_id": meta.BindingID, "logical_id": meta.LogicalID, "field": meta.Field, "version": meta.Version, "content_ref": meta.ContentRef, "grant_ref": meta.GrantRef}
	return s.RunEffect(ctx, effect, func(context.Context) (map[string]any, Outcome, error) {
		details := map[string]any{"binding_id": meta.BindingID, "logical_id": meta.LogicalID, "field": meta.Field, "version": meta.Version, "content_ref": meta.ContentRef, "value_channel": meta.channel}
		if meta.GrantRef != "" {
			details["grant_ref"] = meta.GrantRef
		}
		authority, err := authorityReady(deps)
		if err != nil {
			return details, OutcomeFailed, err
		}
		record, found, err := s.ReadCredentialRecord(effect.DeploymentID, meta.BindingID)
		if err != nil {
			return details, OutcomeFailed, err
		}
		if found {
			for _, revoked := range record.RevokedVersions {
				if revoked == meta.Version {
					return details, OutcomeFailed, refuse(CodeCredentialVersionRevoked, "version %d of binding %s was revoked on this target; it cannot be ingested again", meta.Version, meta.BindingID)
				}
			}
			if meta.Version < record.Version {
				return details, OutcomeFailed, refuse(CodeCredentialVersionStale, "version %d is below the ingested version %d", meta.Version, record.Version).withDetails(map[string]any{"current_version": record.Version})
			}
			if meta.Version == record.Version && record.ContentRef == meta.ContentRef && !record.Revoked {
				details["unchanged"] = true
				return details, OutcomeUnchanged, nil
			}
		}
		identity, err := credentialauthority.ParseIdentity(meta.LogicalID)
		if err != nil {
			return details, OutcomeFailed, refuse(CodeCredentialPayloadInvalid, "%v", err)
		}
		if err := authority.Put(identity, meta.Field, value); err != nil {
			if errors.Is(err, credentialauthority.ErrProviderUnavailable) {
				return details, OutcomeFailed, refuse(CodeCredentialStoreLocked, "the credential store refused the write: %s", credentialauthority.ProviderDetail(err))
			}
			return details, OutcomeFailed, fail(CodeCredentialAuthorityFailed, "write credential: %v", err)
		}
		next := CredentialRecord{DeploymentID: effect.DeploymentID, BindingID: meta.BindingID, LogicalID: meta.LogicalID, Field: meta.Field, Version: meta.Version, ContentRef: meta.ContentRef, GrantRef: meta.GrantRef, IngestedAt: s.now().Format(time.RFC3339Nano), RevokedVersions: record.RevokedVersions}
		if next.RevokedVersions == nil {
			next.RevokedVersions = []int64{}
		}
		if err := s.writeCredentialRecord(next); err != nil {
			return details, OutcomeFailed, err
		}
		details["previous_version"] = record.Version
		return details, OutcomeSucceeded, nil
	})
}

type ingestMeta struct {
	BindingID, LogicalID, Field, ContentRef, GrantRef string
	Version                                           int64
	channel                                           string
}

// resolveIngest merges flag metadata with the payload and locates the value.
// Flags and payload must agree on binding, version and deployment.
func resolveIngest(req CredentialIngestRequest, deps CredentialDeps) (string, ingestMeta, error) {
	meta := ingestMeta{BindingID: strings.TrimSpace(req.BindingID), LogicalID: strings.TrimSpace(req.LogicalID), Field: strings.TrimSpace(req.Field), ContentRef: strings.TrimSpace(req.ContentRef), GrantRef: strings.TrimSpace(req.GrantRef), Version: req.Version}
	if env := strings.TrimSpace(req.FromEnv); env != "" {
		value := deps.env(env)
		if value == "" {
			return "", meta, refuse(CodeCredentialPayloadInvalid, "environment variable %s carries no value; the Bridge injection did not arrive", env)
		}
		if meta.LogicalID == "" || meta.Field == "" || meta.ContentRef == "" || meta.Version <= 0 {
			return "", meta, refuse(CodeCredentialPayloadInvalid, "--logical-id, --field, --version and --content-ref are required with --from-env")
		}
		meta.channel = "env"
		return value, meta, nil
	}
	data, err := io.ReadAll(io.LimitReader(deps.stdin(), 1<<20))
	if err != nil {
		return "", meta, refuse(CodeCredentialPayloadInvalid, "read payload: %v", err)
	}
	var payload IngestPayload
	if len(strings.TrimSpace(string(data))) == 0 {
		return "", meta, refuse(CodeCredentialPayloadInvalid, "no sealed payload on standard input")
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", meta, refuse(CodeCredentialPayloadInvalid, "payload is not JSON")
	}
	if payload.SchemaVersion != IngestSchemaVersion {
		return "", meta, refuse(CodeCredentialPayloadInvalid, "unsupported payload schema_version %d", payload.SchemaVersion)
	}
	if payload.Value == "" {
		return "", meta, refuse(CodeCredentialPayloadInvalid, "payload carries no value")
	}
	if payload.DeploymentID != req.Effect.DeploymentID || (meta.BindingID != "" && payload.BindingID != meta.BindingID) || (meta.Version > 0 && payload.Version != meta.Version) {
		return "", meta, refuse(CodeCredentialPayloadInvalid, "payload identity does not match the requested deployment, binding and version")
	}
	if meta.BindingID == "" {
		meta.BindingID = payload.BindingID
	}
	if meta.Version == 0 {
		meta.Version = payload.Version
	}
	meta.LogicalID, meta.Field, meta.ContentRef = payload.LogicalID, payload.Field, payload.ContentRef
	if payload.GrantRef != "" {
		meta.GrantRef = payload.GrantRef
	}
	if meta.BindingID == "" || meta.LogicalID == "" || meta.Field == "" || meta.ContentRef == "" || meta.Version <= 0 {
		return "", meta, refuse(CodeCredentialPayloadInvalid, "payload lacks binding_id, logical_id, field, version or content_ref")
	}
	meta.channel = "stdin"
	return payload.Value, meta, nil
}

// CredentialAcknowledge proves that the requested version is the one the
// authority holds for the binding and that the store can still answer for
// it. The basis is target-local: it does not observe the consumer process.
func (s *Store) CredentialAcknowledge(ctx context.Context, req CredentialAckRequest, deps CredentialDeps) (EffectResult, error) {
	consumer := strings.TrimSpace(req.Consumer)
	if consumer == "" {
		return EffectResult{}, refuse(CodeInvalidArgument, "--consumer is required")
	}
	effect := req.Effect
	effect.Verb = "credential acknowledge"
	effect.Input = map[string]any{"binding_id": req.BindingID, "version": req.Version, "consumer": consumer}
	return s.RunEffect(ctx, effect, func(context.Context) (map[string]any, Outcome, error) {
		details := map[string]any{"binding_id": req.BindingID, "version": req.Version, "consumer": consumer, "basis": "ledger version and authority status; the consumer process itself is not observed"}
		record, found, err := s.ReadCredentialRecord(effect.DeploymentID, req.BindingID)
		if err != nil {
			return details, OutcomeFailed, err
		}
		if !found {
			return details, OutcomeFailed, refuse(CodeCredentialNotIngested, "binding %s was never ingested on this target", req.BindingID)
		}
		if record.Revoked || record.Version != req.Version {
			return details, OutcomeFailed, refuse(CodeCredentialVersionMismatch, "target holds version %d (revoked=%t), not %d", record.Version, record.Revoked, req.Version).withDetails(map[string]any{"current_version": record.Version, "revoked": record.Revoked})
		}
		authority, err := authorityReady(deps)
		if err != nil {
			return details, OutcomeFailed, err
		}
		identity, err := credentialauthority.ParseIdentity(record.LogicalID)
		if err != nil {
			return details, OutcomeFailed, fail(CodeCredentialAuthorityFailed, "%v", err)
		}
		status := authority.Status(identity, record.Field)
		if status.ProviderState != credentialauthority.ProviderAvailable || !status.Configured {
			return details, OutcomeFailed, refuse(CodeCredentialNotIngested, "the authority no longer holds %s:%s (provider %s)", record.LogicalID, record.Field, status.ProviderState)
		}
		details["content_ref"] = record.ContentRef
		details["verified"] = true
		return details, OutcomeSucceeded, nil
	})
}

// CredentialRevoke purges a version. The active version is deleted from the
// authority; a predecessor that the authority already replaced is only
// marked revoked so it can never be re-ingested. The receipt states what
// revocation proves and what it cannot.
func (s *Store) CredentialRevoke(ctx context.Context, req CredentialRevokeRequest, deps CredentialDeps) (EffectResult, error) {
	effect := req.Effect
	effect.Verb = "credential revoke"
	effect.Input = map[string]any{"binding_id": req.BindingID, "version": req.Version}
	return s.RunEffect(ctx, effect, func(context.Context) (map[string]any, Outcome, error) {
		details := map[string]any{
			"binding_id": req.BindingID, "version": req.Version,
			"proven":      []string{"store_no_longer_holds_version", "version_refused_on_reingest"},
			"unproven":    []string{"process_memory_cleared", "no_prior_copies_left_host"},
			"limitations": []string{RevocationLimitation},
		}
		record, found, err := s.ReadCredentialRecord(effect.DeploymentID, req.BindingID)
		if err != nil {
			return details, OutcomeFailed, err
		}
		if !found {
			record = CredentialRecord{DeploymentID: effect.DeploymentID, BindingID: req.BindingID, RevokedVersions: []int64{}}
			details["proven"] = []string{"version_refused_on_reingest"}
			details["note"] = "binding was never ingested on this target; the version is recorded as revoked so it can never be ingested"
		}
		for _, v := range record.RevokedVersions {
			if v == req.Version {
				details["unchanged"] = true
				return details, OutcomeUnchanged, nil
			}
		}
		purged := false
		if found && record.Version == req.Version && !record.Revoked {
			authority, err := authorityReady(deps)
			if err != nil {
				return details, OutcomeFailed, err
			}
			identity, err := credentialauthority.ParseIdentity(record.LogicalID)
			if err != nil {
				return details, OutcomeFailed, fail(CodeCredentialAuthorityFailed, "%v", err)
			}
			if err := authority.Delete(identity, record.Field); err != nil && !errors.Is(err, credentialauthority.ErrUnconfigured) {
				if errors.Is(err, credentialauthority.ErrProviderUnavailable) {
					return details, OutcomeFailed, refuse(CodeCredentialStoreLocked, "the credential store refused the purge: %s", credentialauthority.ProviderDetail(err))
				}
				return details, OutcomeFailed, fail(CodeCredentialAuthorityFailed, "purge credential: %v", err)
			}
			record.Revoked = true
			purged = true
		}
		record.RevokedVersions = append(record.RevokedVersions, req.Version)
		sort.Slice(record.RevokedVersions, func(i, j int) bool { return record.RevokedVersions[i] < record.RevokedVersions[j] })
		if err := s.writeCredentialRecord(record); err != nil {
			return details, OutcomeFailed, err
		}
		details["store_purged"] = purged
		details["active_version_after"] = fmt.Sprint(activeAfter(record))
		return details, OutcomeSucceeded, nil
	})
}

func activeAfter(record CredentialRecord) any {
	if record.Revoked || record.Version == 0 {
		return "none"
	}
	return record.Version
}
