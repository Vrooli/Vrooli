// Package safeguardpermissions exposes manifest-defined local host safeguard
// configuration through onboarding's typed permission registry. Host effects
// remain owned by the host control plane; this provider only stores consent
// and configuration in the authoritative OperatorState document.
package safeguardpermissions

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

const (
	ProviderID       = "onboarding.host-safeguards"
	Owner            = "vrooli-onboarding"
	PermissionPrefix = "onboarding/host-safeguard/"
	SubjectKind      = "local-host-safeguard"
	LocalSubject     = "local"
	LocalTarget      = "local"
	SchemaVersion    = "operator-state/host-safeguard/v1"
	Source           = "vrooli-onboarding safeguard manifests + OperatorState"
)

// Safeguard is the current owner-defined manifest projection. SchemaDigest
// binds a permission descriptor to the exact configuration contract.
type Safeguard struct {
	Name         string
	SchemaDigest string
	ConfigSchema map[string]any
	ConfigFields []string
	SecretFields []string
}

type Reader interface {
	Get(context.Context) (operatorstate.Document, error)
}
type Writer interface {
	CompareAndSetSafeguard(context.Context, Mutation) (Receipt, error)
}
type Inventory func(context.Context) ([]Safeguard, error)
type Mutation struct {
	ActorID, Name, ExpectedRevision, RequestID, RequestDigest, DecisionRef string
	ConfigKey                                                              string
	Value                                                                  json.RawMessage
	OptedIn                                                                *bool
	Config                                                                 map[string]json.RawMessage
}
type Receipt struct {
	RequestID, ActorID, Target, Subject, ExpectedRevision, NewRevision, RequestDigest, Outcome string
	At                                                                                         time.Time
}

type Provider struct {
	reader            Reader
	writer            Writer
	inventory         Inventory
	now               func() time.Time
	authorizeMutation func(context.Context) error
}

func New(reader Reader, writer Writer, inventory Inventory) *Provider {
	return &Provider{reader: reader, writer: writer, inventory: inventory, now: func() time.Time { return time.Now().UTC() }, authorizeMutation: authz.RequireMutation}
}

var _ permissions.Provider = (*Provider)(nil)

func (*Provider) ProviderID() string                 { return ProviderID }
func (*Provider) Owner() string                      { return Owner }
func (*Provider) MatchesPermissionID(id string) bool { return strings.HasPrefix(id, PermissionPrefix) }

func (p *Provider) List(ctx context.Context) ([]permissions.Descriptor, error) {
	actor, err := verifiedActor(ctx)
	if err != nil {
		return nil, err
	}
	if p.reader == nil || p.inventory == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("host safeguard owner is unavailable"))
	}
	items, err := p.inventory(ctx)
	if err != nil {
		return nil, err
	}
	doc, err := p.reader.Get(ctx)
	if err != nil {
		return nil, err
	}
	revision := operatorstate.Revision(doc)
	if revision == "" {
		return nil, connect.NewError(connect.CodeDataLoss, errors.New("OperatorState has no revision"))
	}
	result := make([]permissions.Descriptor, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" || item.SchemaDigest == "" || seen[name] {
			return nil, connect.NewError(connect.CodeDataLoss, errors.New("safeguard inventory is incomplete or duplicated"))
		}
		seen[name] = true
		result = append(result, permissions.Descriptor{Version: permissions.ContractVersion, ID: PermissionPrefix + name, Owner: Owner, PrincipalKind: string(actor.Kind), PrincipalID: actor.Subject, Scope: "vrooli-onboarding:write", SubjectKind: SubjectKind, SubjectID: name, Source: Source, SchemaVersion: SchemaVersion + "/" + item.SchemaDigest, CanRead: true, CanWrite: p.writer != nil, Sensitivity: "operator"})
	}
	return result, nil
}

func (p *Provider) descriptor(ctx context.Context, d permissions.Descriptor, actorID, owner string, target permissions.TargetBinding) (permissions.Descriptor, error) {
	actor, err := verifiedActor(ctx)
	if err != nil {
		return permissions.Descriptor{}, err
	}
	if d.Owner != Owner || owner != Owner || actor.Subject != actorID || d.PrincipalID != actorID || d.PrincipalKind != string(actor.Kind) || !strings.HasPrefix(d.ID, PermissionPrefix) || d.SubjectKind != SubjectKind || d.SubjectID == "" || d.ID != PermissionPrefix+d.SubjectID || target != (permissions.TargetBinding{}) || d.TargetBound || d.Target != (permissions.TargetBinding{}) || d.Source != Source || !d.CanRead {
		return permissions.Descriptor{}, connect.NewError(connect.CodePermissionDenied, errors.New("host safeguard actor, owner, permission, or target mismatch"))
	}
	if p.inventory == nil {
		return permissions.Descriptor{}, connect.NewError(connect.CodeUnavailable, errors.New("host safeguard inventory is unavailable"))
	}
	items, err := p.inventory(ctx)
	if err != nil {
		return permissions.Descriptor{}, err
	}
	matched := false
	for _, item := range items {
		if item.Name == d.SubjectID && d.SchemaVersion == SchemaVersion+"/"+item.SchemaDigest {
			matched = true
			break
		}
	}
	if !matched {
		return permissions.Descriptor{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("host safeguard descriptor is stale or unsupported"))
	}
	return d, nil
}

func (p *Provider) AuthorizeRead(ctx context.Context, d permissions.Descriptor, req permissions.ReadRequest) (permissions.Decision, error) {
	if _, err := p.descriptor(ctx, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.Decision{}, err
	}
	return permissions.Decision{Allowed: true, DecisionRef: "host-safeguard:" + req.ActorID + ":local"}, nil
}
func (p *Provider) Read(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.ReadRequest) (permissions.ReadResult, error) {
	if !permit.ValidForRead(d.ID, req) {
		return permissions.ReadResult{}, connect.NewError(connect.CodePermissionDenied, errors.New("host safeguard read permit mismatch"))
	}
	if _, err := p.descriptor(ctx, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.ReadResult{}, err
	}
	doc, err := p.reader.Get(ctx)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	value := map[string]any{"opted_in": false, "config": map[string]any{}}
	if current, ok := doc.HostSafeguards[d.SubjectID]; ok {
		if current.OptedIn != nil {
			value["opted_in"] = *current.OptedIn
		}
		if current.Config != nil {
			config := make(map[string]any, len(current.Config))
			for key, item := range current.Config {
				config[key] = item
			}
			inventory, err := p.inventory(ctx)
			if err != nil {
				return permissions.ReadResult{}, err
			}
			for _, safeguard := range inventory {
				if safeguard.Name == d.SubjectID {
					for _, key := range safeguard.SecretFields {
						delete(config, key)
					}
				}
			}
			value["config"] = config
		}
	}
	b, err := json.Marshal(value)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	sum := sha256.Sum256(b)
	return permissions.ReadResult{PermissionID: d.ID, Owner: d.Owner, Revision: operatorstate.Revision(doc), Value: b, ValueDigest: hex.EncodeToString(sum[:]), ObservedAt: p.now().UTC()}, nil
}

type WriteValue struct {
	ConfigKey string                     `json:"config_key,omitempty"`
	Value     json.RawMessage            `json:"value,omitempty"`
	OptedIn   *bool                      `json:"opted_in,omitempty"`
	Config    map[string]json.RawMessage `json:"config,omitempty"`
}

func (p *Provider) Authorize(ctx context.Context, d permissions.Descriptor, req permissions.WriteRequest) (permissions.Decision, error) {
	if !d.CanWrite || p.writer == nil {
		return permissions.Decision{}, connect.NewError(connect.CodeUnimplemented, errors.New("host safeguard writes require owner CAS and durable receipts"))
	}
	if _, err := p.descriptor(ctx, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.Decision{}, err
	}
	if strings.TrimSpace(req.ExpectedRevision) == "" || strings.TrimSpace(req.IdempotencyKey) == "" {
		return permissions.Decision{}, connect.NewError(connect.CodeInvalidArgument, errors.New("expected revision and request id are required"))
	}
	value, err := decodeWriteValue(req.Value)
	if err != nil {
		return permissions.Decision{}, err
	}
	if err := p.validateConfigFields(ctx, d.SubjectID, value); err != nil {
		return permissions.Decision{}, err
	}
	if value.OptedIn != nil && (value.ConfigKey != "" || len(value.Value) != 0 || len(value.Config) != 0) {
		return permissions.Decision{}, connect.NewError(connect.CodeInvalidArgument, errors.New("host safeguard write requires opted_in or configuration values"))
	}
	if p.authorizeMutation == nil {
		return permissions.Decision{}, connect.NewError(connect.CodePermissionDenied, errors.New("host safeguard mutation authorization is unavailable"))
	}
	if err := p.authorizeMutation(ctx); err != nil {
		return permissions.Decision{}, err
	}
	return permissions.Decision{Allowed: true, DecisionRef: "host-safeguard:" + req.ActorID + ":local:mutation"}, nil
}
func (p *Provider) Write(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.WriteRequest) (permissions.Receipt, error) {
	if !permit.ValidForWrite(d.ID, req) {
		return permissions.Receipt{}, connect.NewError(connect.CodePermissionDenied, errors.New("host safeguard write permit mismatch"))
	}
	if _, err := p.descriptor(ctx, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.Receipt{}, err
	}
	value, err := decodeWriteValue(req.Value)
	if err != nil {
		return permissions.Receipt{}, err
	}
	if err := p.validateConfigFields(ctx, d.SubjectID, value); err != nil {
		return permissions.Receipt{}, err
	}
	payloadSum := sha256.Sum256(req.Value)
	payloadDigest := hex.EncodeToString(payloadSum[:])
	ownerDigest := boundMutationDigest(d.SchemaVersion, req.Value)
	stored, err := p.writer.CompareAndSetSafeguard(ctx, Mutation{ActorID: req.ActorID, Name: d.SubjectID, ExpectedRevision: req.ExpectedRevision, RequestID: req.IdempotencyKey, RequestDigest: ownerDigest, DecisionRef: permit.DecisionRef(), ConfigKey: value.ConfigKey, Value: value.Value, OptedIn: value.OptedIn, Config: value.Config})
	if err != nil {
		return permissions.Receipt{}, err
	}
	if stored.RequestID != req.IdempotencyKey || stored.ActorID != req.ActorID || stored.Target != LocalTarget || stored.Subject != SubjectKind+":"+d.SubjectID || stored.ExpectedRevision != req.ExpectedRevision || stored.NewRevision == "" || stored.RequestDigest != ownerDigest || stored.Outcome == "" || stored.At.IsZero() {
		return permissions.Receipt{}, connect.NewError(connect.CodeDataLoss, errors.New("OperatorState returned an unbound safeguard receipt"))
	}
	return permissions.Receipt{ID: "operator-state-safeguard:" + stored.RequestID, PermissionID: d.ID, Owner: d.Owner, ActorID: stored.ActorID, ExpectedRevision: stored.ExpectedRevision, ResultRevision: stored.NewRevision, IdempotencyKey: stored.RequestID, RequestDigest: payloadDigest, DecisionRef: permit.DecisionRef(), Outcome: stored.Outcome, At: stored.At.UTC()}, nil
}

// boundMutationDigest makes the durable OperatorState receipt cover the exact
// manifest contract used to authorize this partial configuration write. The
// provider rechecks SchemaVersion against the live inventory before this is
// called; binding it here lets replay distinguish writes made under different
// owner schemas even when the submitted JSON bytes are identical.
func boundMutationDigest(schemaVersion string, value []byte) string {
	bound, _ := json.Marshal(struct {
		SchemaVersion string          `json:"schema_version"`
		Value         json.RawMessage `json:"value"`
	}{SchemaVersion: schemaVersion, Value: append(json.RawMessage(nil), value...)})
	sum := sha256.Sum256(bound)
	return hex.EncodeToString(sum[:])
}

func decodeWriteValue(raw []byte) (WriteValue, error) {
	var value WriteValue
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return WriteValue{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if value.OptedIn == nil && len(value.Config) == 0 && (strings.TrimSpace(value.ConfigKey) == "" || len(value.Value) == 0) {
		return WriteValue{}, connect.NewError(connect.CodeInvalidArgument, errors.New("host safeguard write requires opted_in or configuration values"))
	}
	if value.OptedIn != nil && (value.ConfigKey != "" || len(value.Value) != 0 || len(value.Config) != 0) {
		return WriteValue{}, connect.NewError(connect.CodeInvalidArgument, errors.New("host safeguard write requires opted_in or configuration values"))
	}
	if len(value.Config) > 0 && (value.ConfigKey != "" || len(value.Value) != 0) {
		return WriteValue{}, connect.NewError(connect.CodeInvalidArgument, errors.New("host safeguard configuration write must use one field encoding"))
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return WriteValue{}, connect.NewError(connect.CodeInvalidArgument, errors.New("host safeguard write must contain one JSON object"))
	}
	return value, nil
}

// validateConfigFields ensures onboarding only stores manifest-declared,
// non-secret configuration fields. A field absent from the owner's typed
// inventory remains unsupported; secret configuration belongs to the owner
// credential flow and must never be copied into OperatorState as cleartext.
func (p *Provider) validateConfigFields(ctx context.Context, name string, value WriteValue) error {
	if value.OptedIn != nil {
		return nil
	}
	if p.inventory == nil {
		return connect.NewError(connect.CodeUnavailable, errors.New("host safeguard inventory is unavailable"))
	}
	items, err := p.inventory(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Name != name {
			continue
		}
		allowed, secret := map[string]bool{}, map[string]bool{}
		for _, field := range item.ConfigFields {
			allowed[field] = true
		}
		for _, field := range item.SecretFields {
			secret[field] = true
		}
		fields := make(map[string]json.RawMessage, len(value.Config)+1)
		if value.ConfigKey != "" {
			fields[value.ConfigKey] = value.Value
		}
		for key, raw := range value.Config {
			fields[key] = raw
		}
		if len(fields) == 0 {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("host safeguard configuration write has no fields"))
		}
		patch := make(map[string]any, len(fields))
		for key, raw := range fields {
			if !allowed[key] || secret[key] {
				return connect.NewError(connect.CodeInvalidArgument, errors.New("host safeguard configuration field is unsupported"))
			}
			var decoded any
			if err := json.Unmarshal(raw, &decoded); err != nil {
				return connect.NewError(connect.CodeInvalidArgument, errors.New("host safeguard configuration value is invalid"))
			}
			patch[key] = decoded
		}
		if len(item.ConfigSchema) == 0 {
			return connect.NewError(connect.CodeDataLoss, errors.New("host safeguard owner returned no configuration schema"))
		}
		// Validate only submitted fields against the owner schema. Root-level
		// required fields are omitted because this operation is a partial patch;
		// nested required fields remain enforced. Secret values are never read
		// from OperatorState to build a complete object.
		schema := cloneSchema(item.ConfigSchema)
		delete(schema, "required")
		stripSchemaAnnotations(schema)
		schemaJSON, err := json.Marshal(schema)
		if err != nil {
			return connect.NewError(connect.CodeDataLoss, errors.New("host safeguard owner schema is invalid"))
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("host-safeguard.schema.json", bytes.NewReader(schemaJSON)); err != nil {
			return connect.NewError(connect.CodeDataLoss, errors.New("host safeguard owner schema is invalid"))
		}
		compiled, err := compiler.Compile("host-safeguard.schema.json")
		if err != nil {
			return connect.NewError(connect.CodeDataLoss, errors.New("host safeguard owner schema is invalid"))
		}
		if err := compiled.Validate(patch); err != nil {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("host safeguard configuration does not match its owner schema"))
		}
		return nil
	}
	return connect.NewError(connect.CodeFailedPrecondition, errors.New("host safeguard descriptor is stale or unsupported"))
}

func cloneSchema(value map[string]any) map[string]any {
	encoded, _ := json.Marshal(value)
	var clone map[string]any
	_ = json.Unmarshal(encoded, &clone)
	if clone == nil {
		return map[string]any{}
	}
	return clone
}

func stripSchemaAnnotations(value any) {
	switch node := value.(type) {
	case map[string]any:
		delete(node, "description")
		delete(node, "default")
		delete(node, "secret")
		delete(node, "sensitive")
		for _, child := range node {
			stripSchemaAnnotations(child)
		}
	case []any:
		for _, child := range node {
			stripSchemaAnnotations(child)
		}
	}
}
func verifiedActor(ctx context.Context) (identity.Principal, error) {
	a, ok := identity.PrincipalFromContext(ctx)
	if !ok || !a.IsHuman() {
		return identity.Principal{}, connect.NewError(connect.CodeUnauthenticated, errors.New("verified human operator required"))
	}
	return a, nil
}
func PermissionID(name string) string {
	return fmt.Sprintf("%s%s", PermissionPrefix, strings.TrimSpace(name))
}
