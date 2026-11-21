package capabilities

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SQLExecutor is the engine-independent database surface the repository needs.
// The package owns its schema and queries; the engine stays behind this seams.
type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// Repository is the capability catalog's persistence boundary.
type Repository interface {
	Upsert(context.Context, Capability) (Capability, error)
	Seed(context.Context, []Capability) (SeedResult, error)
	Get(context.Context, string, string) (Capability, error)
	List(context.Context, string) ([]Capability, error)
	RecordQualification(context.Context, CapabilityQualification) (CapabilityQualification, error)
	LatestQualification(context.Context, string) (CapabilityQualification, error)
	Link(context.Context, CapabilityLink) (CapabilityLink, error)
	ListLinks(context.Context) ([]CapabilityLink, error)
	LinksForCapability(context.Context, string) ([]CapabilityLink, error)
}

type repository struct{ db SQLExecutor }

// NewRepository builds a capability repository over the given executor.
func NewRepository(db SQLExecutor) Repository { return &repository{db: db} }

type executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

const capabilityColumns = `id, name, medium, aliases, channels, audience_applicability, delivery_applicability, producing_operation, prerequisites, priority, priority_reason, priority_scope, definition_status, implementation_status, operational_readiness, output_quality, distribution_connectivity, owner, source_refs, next_action, created_at, updated_at`

const qualificationColumns = `id, capability_id, latest_artifact_id, latest_run_id, environment, validated_at, observed_at, freshness_basis, candidate_identity, max_age_seconds, limitation, next_action, created_at`

const linkColumns = `id, capability_id, relation, target_id, created_at`

func encodeStrings(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func decodeStrings(raw string) []string {
	if raw == "" || raw == "[]" {
		return nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

// dedupeStrings drops empty entries and repeats, preserving first-seen order.
// Capability aliases are semantically a set: the catalog prepends each slug to
// its entry aliases, so an entry that also lists its own slug must not surface a
// duplicated alias through the API, the CLI report or the alias index.
func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func nowString(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseTime(raw string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

// validateCapability enforces every closed readiness/enumeration set at the
// domain boundary. Both Upsert and Seed call it so a seeded record can never
// persist an invalid dimension.
func validateCapability(capability Capability) error {
	if capability.Name == "" {
		return ErrInvalidName
	}
	if !IsValidMedium(capability.Medium) {
		return fmt.Errorf("%w: %q", ErrInvalidMedium, capability.Medium)
	}
	if !IsValidDefinitionStatus(capability.DefinitionStatus) {
		return fmt.Errorf("%w: definition_status %q", ErrInvalidStatus, capability.DefinitionStatus)
	}
	if !IsValidImplementationStatus(capability.ImplementationStatus) {
		return fmt.Errorf("%w: implementation_status %q", ErrInvalidStatus, capability.ImplementationStatus)
	}
	if !IsValidOperationalReadiness(capability.OperationalReadiness) {
		return fmt.Errorf("%w: operational_readiness %q", ErrInvalidStatus, capability.OperationalReadiness)
	}
	if !IsValidOutputQuality(capability.OutputQuality) {
		return fmt.Errorf("%w: output_quality %q", ErrInvalidStatus, capability.OutputQuality)
	}
	if !IsValidDistributionConnectivity(capability.DistributionConnectivity) {
		return fmt.Errorf("%w: distribution_connectivity %q", ErrInvalidStatus, capability.DistributionConnectivity)
	}
	return nil
}

func (r *repository) Upsert(ctx context.Context, capability Capability) (Capability, error) {
	if err := validateCapability(capability); err != nil {
		return Capability{}, err
	}
	if capability.ID == "" {
		capability.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Capability{}, err
	}
	defer tx.Rollback()

	createdAt := now
	var existing string
	if err := tx.QueryRowContext(ctx, `SELECT created_at FROM capabilities WHERE id = ?`, capability.ID).Scan(&existing); err == nil {
		if parsed := parseTime(existing); !parsed.IsZero() {
			createdAt = parsed
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Capability{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO capabilities (`+capabilityColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, medium=excluded.medium, aliases=excluded.aliases, channels=excluded.channels,
		audience_applicability=excluded.audience_applicability, delivery_applicability=excluded.delivery_applicability,
		producing_operation=excluded.producing_operation, prerequisites=excluded.prerequisites, priority=excluded.priority,
		priority_reason=excluded.priority_reason, priority_scope=excluded.priority_scope, definition_status=excluded.definition_status,
		implementation_status=excluded.implementation_status, operational_readiness=excluded.operational_readiness,
		output_quality=excluded.output_quality, distribution_connectivity=excluded.distribution_connectivity, owner=excluded.owner,
		source_refs=excluded.source_refs, next_action=excluded.next_action, updated_at=excluded.updated_at`,
		capability.ID, capability.Name, capability.Medium, encodeStrings(capability.Aliases), encodeStrings(capability.Channels),
		capability.AudienceApplicability, capability.DeliveryApplicability, capability.ProducingOperation,
		encodeStrings(capability.Prerequisites), capability.Priority, capability.PriorityReason, capability.PriorityScope,
		capability.DefinitionStatus, capability.ImplementationStatus, capability.OperationalReadiness, capability.OutputQuality,
		capability.DistributionConnectivity, capability.Owner, encodeStrings(capability.SourceRefs), capability.NextAction,
		nowString(createdAt), nowString(now))
	if err != nil {
		return Capability{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM capability_aliases WHERE capability_id = ?`, capability.ID); err != nil {
		return Capability{}, err
	}
	for _, alias := range capability.Aliases {
		if alias == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO capability_aliases (alias, capability_id) VALUES (?, ?) ON CONFLICT(alias) DO UPDATE SET capability_id=excluded.capability_id`, alias, capability.ID); err != nil {
			return Capability{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Capability{}, err
	}
	capability.CreatedAt = createdAt
	capability.UpdatedAt = now
	return capability, nil
}

// Seed inserts catalog capabilities only when their id is absent, in one
// transaction. Existing rows (including operator edits) are left untouched, so
// seeding is idempotent and never clobbers observed readiness.
func (r *repository) Seed(ctx context.Context, capabilities []Capability) (SeedResult, error) {
	var result SeedResult
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	for _, capability := range capabilities {
		if err := validateCapability(capability); err != nil {
			return result, err
		}
		if capability.ID == "" {
			return result, ErrReferenceRequired
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO capabilities (`+capabilityColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO NOTHING`,
			capability.ID, capability.Name, capability.Medium, encodeStrings(capability.Aliases), encodeStrings(capability.Channels),
			capability.AudienceApplicability, capability.DeliveryApplicability, capability.ProducingOperation,
			encodeStrings(capability.Prerequisites), capability.Priority, capability.PriorityReason, capability.PriorityScope,
			capability.DefinitionStatus, capability.ImplementationStatus, capability.OperationalReadiness, capability.OutputQuality,
			capability.DistributionConnectivity, capability.Owner, encodeStrings(capability.SourceRefs), capability.NextAction,
			nowString(now), nowString(now))
		if err != nil {
			return result, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return result, err
		}
		if affected == 0 {
			result.Existing++
			continue
		}
		result.Inserted++
		for _, alias := range capability.Aliases {
			if alias == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO capability_aliases (alias, capability_id) VALUES (?, ?) ON CONFLICT(alias) DO NOTHING`, alias, capability.ID); err != nil {
				return result, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return SeedResult{}, err
	}
	return result, nil
}

func (r *repository) Get(ctx context.Context, id, alias string) (Capability, error) {
	if id == "" && alias == "" {
		return Capability{}, ErrReferenceRequired
	}
	if id == "" {
		if err := r.db.QueryRowContext(ctx, `SELECT capability_id FROM capability_aliases WHERE alias = ?`, alias).Scan(&id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return Capability{}, ErrNotFound
			}
			return Capability{}, err
		}
	}
	row := r.db.QueryRowContext(ctx, `SELECT `+capabilityColumns+` FROM capabilities WHERE id = ?`, id)
	capability, err := scanCapability(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Capability{}, ErrNotFound
	}
	if err != nil {
		return Capability{}, err
	}
	qualification, err := r.latestQualification(ctx, capability.ID)
	if err != nil {
		return Capability{}, err
	}
	capability.LatestQualification = &qualification
	return capability, nil
}

func (r *repository) List(ctx context.Context, medium string) ([]Capability, error) {
	query := `SELECT ` + capabilityColumns + ` FROM capabilities`
	args := []any{}
	if medium != "" {
		if !IsValidMedium(medium) {
			return nil, fmt.Errorf("%w: %q", ErrInvalidMedium, medium)
		}
		query += ` WHERE medium = ?`
		args = append(args, medium)
	}
	query += ` ORDER BY name, id`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Capability
	for rows.Next() {
		capability, err := scanCapability(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, capability)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		qualification, err := r.latestQualification(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].LatestQualification = &qualification
	}
	return out, nil
}

func (r *repository) RecordQualification(ctx context.Context, qualification CapabilityQualification) (CapabilityQualification, error) {
	if qualification.CapabilityID == "" {
		return CapabilityQualification{}, ErrReferenceRequired
	}
	if !IsValidFreshnessBasis(qualification.FreshnessBasis) {
		return CapabilityQualification{}, fmt.Errorf("%w: %q", ErrInvalidFreshness, qualification.FreshnessBasis)
	}
	if qualification.ID == "" {
		qualification.ID = uuid.NewString()
	}
	qualification.CreatedAt = time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `INSERT INTO capability_qualifications (`+qualificationColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		qualification.ID, qualification.CapabilityID, qualification.LatestArtifactID, qualification.LatestRunID,
		qualification.Environment, qualification.ValidatedAt, qualification.ObservedAt, qualification.FreshnessBasis,
		qualification.CandidateIdentity, qualification.MaxAgeSeconds, qualification.Limitation, qualification.NextAction,
		nowString(qualification.CreatedAt))
	if err != nil {
		return CapabilityQualification{}, err
	}
	return qualification, nil
}

func (r *repository) LatestQualification(ctx context.Context, capabilityID string) (CapabilityQualification, error) {
	return r.latestQualification(ctx, capabilityID)
}

func (r *repository) latestQualification(ctx context.Context, capabilityID string) (CapabilityQualification, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+qualificationColumns+` FROM capability_qualifications WHERE capability_id = ? ORDER BY created_at DESC, id DESC LIMIT 1`, capabilityID)
	qualification, err := scanQualification(row)
	if errors.Is(err, sql.ErrNoRows) {
		return UnknownQualification(capabilityID), nil
	}
	if err != nil {
		return CapabilityQualification{}, err
	}
	return qualification, nil
}

func (r *repository) Link(ctx context.Context, link CapabilityLink) (CapabilityLink, error) {
	if link.CapabilityID == "" {
		return CapabilityLink{}, ErrReferenceRequired
	}
	if !IsValidRelation(link.Relation) {
		return CapabilityLink{}, fmt.Errorf("%w: %q", ErrInvalidRelation, link.Relation)
	}
	if link.TargetID == "" {
		return CapabilityLink{}, ErrInvalidTarget
	}
	if link.ID == "" {
		link.ID = uuid.NewString()
	}
	link.CreatedAt = time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `INSERT INTO capability_links (`+linkColumns+`) VALUES (?, ?, ?, ?, ?)`,
		link.ID, link.CapabilityID, link.Relation, link.TargetID, nowString(link.CreatedAt))
	if err != nil {
		return CapabilityLink{}, err
	}
	return link, nil
}

func (r *repository) ListLinks(ctx context.Context) ([]CapabilityLink, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+linkColumns+` FROM capability_links ORDER BY capability_id, relation, target_id, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CapabilityLink
	for rows.Next() {
		link, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, link)
	}
	return out, rows.Err()
}

func (r *repository) LinksForCapability(ctx context.Context, capabilityID string) ([]CapabilityLink, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+linkColumns+` FROM capability_links WHERE capability_id = ? ORDER BY relation, target_id, id`, capabilityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CapabilityLink
	for rows.Next() {
		link, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, link)
	}
	return out, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func scanCapability(row rowScanner) (Capability, error) {
	var (
		capability                       Capability
		aliases, channels, prereqs, refs string
		priority                         int64
		createdAt, updatedAt             string
	)
	if err := row.Scan(&capability.ID, &capability.Name, &capability.Medium, &aliases, &channels,
		&capability.AudienceApplicability, &capability.DeliveryApplicability, &capability.ProducingOperation,
		&prereqs, &priority, &capability.PriorityReason, &capability.PriorityScope,
		&capability.DefinitionStatus, &capability.ImplementationStatus, &capability.OperationalReadiness,
		&capability.OutputQuality, &capability.DistributionConnectivity, &capability.Owner, &refs,
		&capability.NextAction, &createdAt, &updatedAt); err != nil {
		return Capability{}, err
	}
	capability.Aliases = dedupeStrings(decodeStrings(aliases))
	capability.Channels = decodeStrings(channels)
	capability.Prerequisites = decodeStrings(prereqs)
	capability.SourceRefs = decodeStrings(refs)
	capability.Priority = int32(priority)
	capability.CreatedAt = parseTime(createdAt)
	capability.UpdatedAt = parseTime(updatedAt)
	return capability, nil
}

func scanQualification(row rowScanner) (CapabilityQualification, error) {
	var (
		qualification CapabilityQualification
		createdAt     string
	)
	if err := row.Scan(&qualification.ID, &qualification.CapabilityID, &qualification.LatestArtifactID,
		&qualification.LatestRunID, &qualification.Environment, &qualification.ValidatedAt, &qualification.ObservedAt,
		&qualification.FreshnessBasis, &qualification.CandidateIdentity, &qualification.MaxAgeSeconds,
		&qualification.Limitation, &qualification.NextAction, &createdAt); err != nil {
		return CapabilityQualification{}, err
	}
	qualification.CreatedAt = parseTime(createdAt)
	return qualification, nil
}

func scanLink(row rowScanner) (CapabilityLink, error) {
	var (
		link      CapabilityLink
		createdAt string
	)
	if err := row.Scan(&link.ID, &link.CapabilityID, &link.Relation, &link.TargetID, &createdAt); err != nil {
		return CapabilityLink{}, err
	}
	link.CreatedAt = parseTime(createdAt)
	return link, nil
}
