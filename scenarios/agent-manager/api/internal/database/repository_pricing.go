package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"agent-manager/internal/pricing"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// PricingRepository Implementation
// ============================================================================

type pricingRepository struct {
	db  *DB
	log *logrus.Logger
}

// NewPricingRepository creates a new SQL-backed pricing repository.
func NewPricingRepository(db *DB, log *logrus.Logger) pricing.Repository {
	return &pricingRepository{db: db, log: log}
}

var _ pricing.Repository = (*pricingRepository)(nil)

// --- Row Types ---

type modelPricingRow struct {
	ID                  string       `db:"id"`
	CanonicalModelName  string       `db:"canonical_model_name"`
	Provider            string       `db:"provider"`
	InputTokenPrice     sql.NullFloat64 `db:"input_token_price"`
	OutputTokenPrice    sql.NullFloat64 `db:"output_token_price"`
	CacheReadPrice      sql.NullFloat64 `db:"cache_read_price"`
	CacheCreationPrice  sql.NullFloat64 `db:"cache_creation_price"`
	WebSearchPrice      sql.NullFloat64 `db:"web_search_price"`
	ServerToolUsePrice  sql.NullFloat64 `db:"server_tool_use_price"`
	InputTokenSource    sql.NullString  `db:"input_token_source"`
	OutputTokenSource   sql.NullString  `db:"output_token_source"`
	CacheReadSource     sql.NullString  `db:"cache_read_source"`
	CacheCreationSource sql.NullString  `db:"cache_creation_source"`
	WebSearchSource     sql.NullString  `db:"web_search_source"`
	ServerToolUseSource sql.NullString  `db:"server_tool_use_source"`
	FetchedAt           SQLiteTime   `db:"fetched_at"`
	ExpiresAt           SQLiteTime   `db:"expires_at"`
	PricingVersion      sql.NullString  `db:"pricing_version"`
	CreatedAt           SQLiteTime   `db:"created_at"`
	UpdatedAt           SQLiteTime   `db:"updated_at"`
}

func (row *modelPricingRow) toDomain() *pricing.ModelPricing {
	m := &pricing.ModelPricing{
		ID:                 uuid.MustParse(row.ID),
		CanonicalModelName: row.CanonicalModelName,
		Provider:           row.Provider,
		FetchedAt:          row.FetchedAt.Time(),
		ExpiresAt:          row.ExpiresAt.Time(),
		CreatedAt:          row.CreatedAt.Time(),
		UpdatedAt:          row.UpdatedAt.Time(),
	}

	if row.InputTokenPrice.Valid {
		m.InputTokenPrice = &row.InputTokenPrice.Float64
	}
	if row.OutputTokenPrice.Valid {
		m.OutputTokenPrice = &row.OutputTokenPrice.Float64
	}
	if row.CacheReadPrice.Valid {
		m.CacheReadPrice = &row.CacheReadPrice.Float64
	}
	if row.CacheCreationPrice.Valid {
		m.CacheCreationPrice = &row.CacheCreationPrice.Float64
	}
	if row.WebSearchPrice.Valid {
		m.WebSearchPrice = &row.WebSearchPrice.Float64
	}
	if row.ServerToolUsePrice.Valid {
		m.ServerToolUsePrice = &row.ServerToolUsePrice.Float64
	}

	if row.InputTokenSource.Valid {
		m.InputTokenSource = pricing.PricingSource(row.InputTokenSource.String)
	}
	if row.OutputTokenSource.Valid {
		m.OutputTokenSource = pricing.PricingSource(row.OutputTokenSource.String)
	}
	if row.CacheReadSource.Valid {
		m.CacheReadSource = pricing.PricingSource(row.CacheReadSource.String)
	}
	if row.CacheCreationSource.Valid {
		m.CacheCreationSource = pricing.PricingSource(row.CacheCreationSource.String)
	}
	if row.WebSearchSource.Valid {
		m.WebSearchSource = pricing.PricingSource(row.WebSearchSource.String)
	}
	if row.ServerToolUseSource.Valid {
		m.ServerToolUseSource = pricing.PricingSource(row.ServerToolUseSource.String)
	}
	if row.PricingVersion.Valid {
		m.PricingVersion = row.PricingVersion.String
	}

	return m
}

func modelPricingFromDomain(m *pricing.ModelPricing) *modelPricingRow {
	row := &modelPricingRow{
		ID:                 m.ID.String(),
		CanonicalModelName: m.CanonicalModelName,
		Provider:           m.Provider,
		FetchedAt:          SQLiteTime(m.FetchedAt),
		ExpiresAt:          SQLiteTime(m.ExpiresAt),
		CreatedAt:          SQLiteTime(m.CreatedAt),
		UpdatedAt:          SQLiteTime(m.UpdatedAt),
	}

	if m.InputTokenPrice != nil {
		row.InputTokenPrice = sql.NullFloat64{Float64: *m.InputTokenPrice, Valid: true}
	}
	if m.OutputTokenPrice != nil {
		row.OutputTokenPrice = sql.NullFloat64{Float64: *m.OutputTokenPrice, Valid: true}
	}
	if m.CacheReadPrice != nil {
		row.CacheReadPrice = sql.NullFloat64{Float64: *m.CacheReadPrice, Valid: true}
	}
	if m.CacheCreationPrice != nil {
		row.CacheCreationPrice = sql.NullFloat64{Float64: *m.CacheCreationPrice, Valid: true}
	}
	if m.WebSearchPrice != nil {
		row.WebSearchPrice = sql.NullFloat64{Float64: *m.WebSearchPrice, Valid: true}
	}
	if m.ServerToolUsePrice != nil {
		row.ServerToolUsePrice = sql.NullFloat64{Float64: *m.ServerToolUsePrice, Valid: true}
	}

	if m.InputTokenSource != "" {
		row.InputTokenSource = sql.NullString{String: string(m.InputTokenSource), Valid: true}
	}
	if m.OutputTokenSource != "" {
		row.OutputTokenSource = sql.NullString{String: string(m.OutputTokenSource), Valid: true}
	}
	if m.CacheReadSource != "" {
		row.CacheReadSource = sql.NullString{String: string(m.CacheReadSource), Valid: true}
	}
	if m.CacheCreationSource != "" {
		row.CacheCreationSource = sql.NullString{String: string(m.CacheCreationSource), Valid: true}
	}
	if m.WebSearchSource != "" {
		row.WebSearchSource = sql.NullString{String: string(m.WebSearchSource), Valid: true}
	}
	if m.ServerToolUseSource != "" {
		row.ServerToolUseSource = sql.NullString{String: string(m.ServerToolUseSource), Valid: true}
	}
	if m.PricingVersion != "" {
		row.PricingVersion = sql.NullString{String: m.PricingVersion, Valid: true}
	}

	return row
}

type modelAliasRow struct {
	ID             string     `db:"id"`
	RunnerModel    string     `db:"runner_model"`
	RunnerType     string     `db:"runner_type"`
	CanonicalModel string     `db:"canonical_model"`
	Provider       string     `db:"provider"`
	CreatedAt      SQLiteTime `db:"created_at"`
	UpdatedAt      SQLiteTime `db:"updated_at"`
}

func (row *modelAliasRow) toDomain() *pricing.ModelAlias {
	return &pricing.ModelAlias{
		ID:             uuid.MustParse(row.ID),
		RunnerModel:    row.RunnerModel,
		RunnerType:     row.RunnerType,
		CanonicalModel: row.CanonicalModel,
		Provider:       row.Provider,
		CreatedAt:      row.CreatedAt.Time(),
		UpdatedAt:      row.UpdatedAt.Time(),
	}
}

func modelAliasFromDomain(a *pricing.ModelAlias) *modelAliasRow {
	return &modelAliasRow{
		ID:             a.ID.String(),
		RunnerModel:    a.RunnerModel,
		RunnerType:     a.RunnerType,
		CanonicalModel: a.CanonicalModel,
		Provider:       a.Provider,
		CreatedAt:      SQLiteTime(a.CreatedAt),
		UpdatedAt:      SQLiteTime(a.UpdatedAt),
	}
}

type manualOverrideRow struct {
	ID                 string         `db:"id"`
	CanonicalModelName string         `db:"canonical_model_name"`
	Component          string         `db:"component"`
	PriceUSD           float64        `db:"price_usd"`
	Note               sql.NullString `db:"note"`
	CreatedBy          sql.NullString `db:"created_by"`
	CreatedAt          SQLiteTime     `db:"created_at"`
	ExpiresAt          NullableTime   `db:"expires_at"`
}

func (row *manualOverrideRow) toDomain() *pricing.ManualPriceOverride {
	o := &pricing.ManualPriceOverride{
		ID:                 uuid.MustParse(row.ID),
		CanonicalModelName: row.CanonicalModelName,
		Component:          pricing.PricingComponent(row.Component),
		PriceUSD:           row.PriceUSD,
		CreatedAt:          row.CreatedAt.Time(),
		ExpiresAt:          row.ExpiresAt.ToPtr(),
	}
	if row.Note.Valid {
		o.Note = row.Note.String
	}
	if row.CreatedBy.Valid {
		o.CreatedBy = row.CreatedBy.String
	}
	return o
}

func manualOverrideFromDomain(o *pricing.ManualPriceOverride) *manualOverrideRow {
	row := &manualOverrideRow{
		ID:                 o.ID.String(),
		CanonicalModelName: o.CanonicalModelName,
		Component:          string(o.Component),
		PriceUSD:           o.PriceUSD,
		CreatedAt:          SQLiteTime(o.CreatedAt),
		ExpiresAt:          NewNullableTime(o.ExpiresAt),
	}
	if o.Note != "" {
		row.Note = sql.NullString{String: o.Note, Valid: true}
	}
	if o.CreatedBy != "" {
		row.CreatedBy = sql.NullString{String: o.CreatedBy, Valid: true}
	}
	return row
}

// --- Model Pricing Methods ---

const modelPricingColumns = `id, canonical_model_name, provider,
	input_token_price, output_token_price, cache_read_price, cache_creation_price,
	web_search_price, server_tool_use_price,
	input_token_source, output_token_source, cache_read_source, cache_creation_source,
	web_search_source, server_tool_use_source,
	fetched_at, expires_at, pricing_version, created_at, updated_at`

func (r *pricingRepository) GetPricing(ctx context.Context, canonicalModel, provider string) (*pricing.ModelPricing, error) {
	query := r.db.Rebind(`SELECT ` + modelPricingColumns + ` FROM model_pricing WHERE canonical_model_name = ? AND provider = ?`)
	var row modelPricingRow
	if err := r.db.GetContext(ctx, &row, query, canonicalModel, provider); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBError("get_pricing", "ModelPricing", canonicalModel, err)
	}
	return row.toDomain(), nil
}

func (r *pricingRepository) GetAllPricing(ctx context.Context) ([]*pricing.ModelPricing, error) {
	query := `SELECT ` + modelPricingColumns + ` FROM model_pricing ORDER BY canonical_model_name`
	var rows []modelPricingRow
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, wrapDBError("get_all_pricing", "ModelPricing", "", err)
	}
	result := make([]*pricing.ModelPricing, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *pricingRepository) GetPricingByProvider(ctx context.Context, provider string) ([]*pricing.ModelPricing, error) {
	query := r.db.Rebind(`SELECT ` + modelPricingColumns + ` FROM model_pricing WHERE provider = ? ORDER BY canonical_model_name`)
	var rows []modelPricingRow
	if err := r.db.SelectContext(ctx, &rows, query, provider); err != nil {
		return nil, wrapDBError("get_pricing_by_provider", "ModelPricing", provider, err)
	}
	result := make([]*pricing.ModelPricing, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *pricingRepository) UpsertPricing(ctx context.Context, p *pricing.ModelPricing) error {
	now := time.Now()
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
		p.CreatedAt = now
	}
	p.UpdatedAt = now

	row := modelPricingFromDomain(p)
	query := `INSERT INTO model_pricing (id, canonical_model_name, provider,
		input_token_price, output_token_price, cache_read_price, cache_creation_price,
		web_search_price, server_tool_use_price,
		input_token_source, output_token_source, cache_read_source, cache_creation_source,
		web_search_source, server_tool_use_source,
		fetched_at, expires_at, pricing_version, created_at, updated_at)
		VALUES (:id, :canonical_model_name, :provider,
		:input_token_price, :output_token_price, :cache_read_price, :cache_creation_price,
		:web_search_price, :server_tool_use_price,
		:input_token_source, :output_token_source, :cache_read_source, :cache_creation_source,
		:web_search_source, :server_tool_use_source,
		:fetched_at, :expires_at, :pricing_version, :created_at, :updated_at)
		ON CONFLICT (canonical_model_name, provider) DO UPDATE SET
		input_token_price = EXCLUDED.input_token_price,
		output_token_price = EXCLUDED.output_token_price,
		cache_read_price = EXCLUDED.cache_read_price,
		cache_creation_price = EXCLUDED.cache_creation_price,
		web_search_price = EXCLUDED.web_search_price,
		server_tool_use_price = EXCLUDED.server_tool_use_price,
		input_token_source = EXCLUDED.input_token_source,
		output_token_source = EXCLUDED.output_token_source,
		cache_read_source = EXCLUDED.cache_read_source,
		cache_creation_source = EXCLUDED.cache_creation_source,
		web_search_source = EXCLUDED.web_search_source,
		server_tool_use_source = EXCLUDED.server_tool_use_source,
		fetched_at = EXCLUDED.fetched_at,
		expires_at = EXCLUDED.expires_at,
		pricing_version = EXCLUDED.pricing_version,
		updated_at = EXCLUDED.updated_at`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return wrapDBError("upsert_pricing", "ModelPricing", p.CanonicalModelName, err)
	}
	return nil
}

func (r *pricingRepository) BulkUpsertPricing(ctx context.Context, pricingList []*pricing.ModelPricing) error {
	// Use transaction for bulk operations
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return wrapDBError("begin_tx", "ModelPricing", "", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for _, p := range pricingList {
		now := time.Now()
		if p.ID == uuid.Nil {
			p.ID = uuid.New()
			p.CreatedAt = now
		}
		p.UpdatedAt = now

		row := modelPricingFromDomain(p)
		query := `INSERT INTO model_pricing (id, canonical_model_name, provider,
			input_token_price, output_token_price, cache_read_price, cache_creation_price,
			web_search_price, server_tool_use_price,
			input_token_source, output_token_source, cache_read_source, cache_creation_source,
			web_search_source, server_tool_use_source,
			fetched_at, expires_at, pricing_version, created_at, updated_at)
			VALUES (:id, :canonical_model_name, :provider,
			:input_token_price, :output_token_price, :cache_read_price, :cache_creation_price,
			:web_search_price, :server_tool_use_price,
			:input_token_source, :output_token_source, :cache_read_source, :cache_creation_source,
			:web_search_source, :server_tool_use_source,
			:fetched_at, :expires_at, :pricing_version, :created_at, :updated_at)
			ON CONFLICT (canonical_model_name, provider) DO UPDATE SET
			input_token_price = EXCLUDED.input_token_price,
			output_token_price = EXCLUDED.output_token_price,
			cache_read_price = EXCLUDED.cache_read_price,
			cache_creation_price = EXCLUDED.cache_creation_price,
			web_search_price = EXCLUDED.web_search_price,
			server_tool_use_price = EXCLUDED.server_tool_use_price,
			input_token_source = EXCLUDED.input_token_source,
			output_token_source = EXCLUDED.output_token_source,
			cache_read_source = EXCLUDED.cache_read_source,
			cache_creation_source = EXCLUDED.cache_creation_source,
			web_search_source = EXCLUDED.web_search_source,
			server_tool_use_source = EXCLUDED.server_tool_use_source,
			fetched_at = EXCLUDED.fetched_at,
			expires_at = EXCLUDED.expires_at,
			pricing_version = EXCLUDED.pricing_version,
			updated_at = EXCLUDED.updated_at`

		if _, err = tx.NamedExecContext(ctx, query, row); err != nil {
			return wrapDBError("bulk_upsert_pricing", "ModelPricing", p.CanonicalModelName, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return wrapDBError("commit_tx", "ModelPricing", "", err)
	}
	return nil
}

func (r *pricingRepository) GetExpiredPricing(ctx context.Context, before time.Time) ([]*pricing.ModelPricing, error) {
	query := r.db.Rebind(`SELECT ` + modelPricingColumns + ` FROM model_pricing WHERE expires_at < ?`)
	var rows []modelPricingRow
	if err := r.db.SelectContext(ctx, &rows, query, before); err != nil {
		return nil, wrapDBError("get_expired_pricing", "ModelPricing", "", err)
	}
	result := make([]*pricing.ModelPricing, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *pricingRepository) DeletePricing(ctx context.Context, canonicalModel, provider string) error {
	query := r.db.Rebind(`DELETE FROM model_pricing WHERE canonical_model_name = ? AND provider = ?`)
	_, err := r.db.ExecContext(ctx, query, canonicalModel, provider)
	if err != nil {
		return wrapDBError("delete_pricing", "ModelPricing", canonicalModel, err)
	}
	return nil
}

// --- Model Aliases Methods ---

const modelAliasColumns = `id, runner_model, runner_type, canonical_model, provider, created_at, updated_at`

func (r *pricingRepository) GetAlias(ctx context.Context, runnerModel, runnerType string) (*pricing.ModelAlias, error) {
	query := r.db.Rebind(`SELECT ` + modelAliasColumns + ` FROM model_aliases WHERE runner_model = ? AND runner_type = ?`)
	var row modelAliasRow
	if err := r.db.GetContext(ctx, &row, query, runnerModel, runnerType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBError("get_alias", "ModelAlias", runnerModel, err)
	}
	return row.toDomain(), nil
}

func (r *pricingRepository) GetAllAliases(ctx context.Context) ([]*pricing.ModelAlias, error) {
	query := `SELECT ` + modelAliasColumns + ` FROM model_aliases ORDER BY runner_type, runner_model`
	var rows []modelAliasRow
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, wrapDBError("get_all_aliases", "ModelAlias", "", err)
	}
	result := make([]*pricing.ModelAlias, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *pricingRepository) ListAliases(ctx context.Context, runnerType string) ([]*pricing.ModelAlias, error) {
	query := r.db.Rebind(`SELECT ` + modelAliasColumns + ` FROM model_aliases WHERE runner_type = ? ORDER BY runner_model`)
	var rows []modelAliasRow
	if err := r.db.SelectContext(ctx, &rows, query, runnerType); err != nil {
		return nil, wrapDBError("list_aliases", "ModelAlias", runnerType, err)
	}
	result := make([]*pricing.ModelAlias, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *pricingRepository) UpsertAlias(ctx context.Context, a *pricing.ModelAlias) error {
	now := time.Now()
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
		a.CreatedAt = now
	}
	a.UpdatedAt = now

	row := modelAliasFromDomain(a)
	query := `INSERT INTO model_aliases (id, runner_model, runner_type, canonical_model, provider, created_at, updated_at)
		VALUES (:id, :runner_model, :runner_type, :canonical_model, :provider, :created_at, :updated_at)
		ON CONFLICT (runner_model, runner_type) DO UPDATE SET
		canonical_model = EXCLUDED.canonical_model,
		provider = EXCLUDED.provider,
		updated_at = EXCLUDED.updated_at`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return wrapDBError("upsert_alias", "ModelAlias", a.RunnerModel, err)
	}
	return nil
}

func (r *pricingRepository) DeleteAlias(ctx context.Context, runnerModel, runnerType string) error {
	query := r.db.Rebind(`DELETE FROM model_aliases WHERE runner_model = ? AND runner_type = ?`)
	_, err := r.db.ExecContext(ctx, query, runnerModel, runnerType)
	if err != nil {
		return wrapDBError("delete_alias", "ModelAlias", runnerModel, err)
	}
	return nil
}

// --- Manual Overrides Methods ---

const manualOverrideColumns = `id, canonical_model_name, component, price_usd, note, created_by, created_at, expires_at`

func (r *pricingRepository) GetOverride(ctx context.Context, canonicalModel string, component pricing.PricingComponent) (*pricing.ManualPriceOverride, error) {
	query := r.db.Rebind(`SELECT ` + manualOverrideColumns + ` FROM manual_price_overrides WHERE canonical_model_name = ? AND component = ?`)
	var row manualOverrideRow
	if err := r.db.GetContext(ctx, &row, query, canonicalModel, string(component)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBError("get_override", "ManualPriceOverride", canonicalModel, err)
	}
	return row.toDomain(), nil
}

func (r *pricingRepository) GetOverridesForModel(ctx context.Context, canonicalModel string) ([]*pricing.ManualPriceOverride, error) {
	query := r.db.Rebind(`SELECT ` + manualOverrideColumns + ` FROM manual_price_overrides WHERE canonical_model_name = ? ORDER BY component`)
	var rows []manualOverrideRow
	if err := r.db.SelectContext(ctx, &rows, query, canonicalModel); err != nil {
		return nil, wrapDBError("get_overrides_for_model", "ManualPriceOverride", canonicalModel, err)
	}
	result := make([]*pricing.ManualPriceOverride, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *pricingRepository) GetAllOverrides(ctx context.Context) ([]*pricing.ManualPriceOverride, error) {
	query := `SELECT ` + manualOverrideColumns + ` FROM manual_price_overrides ORDER BY canonical_model_name, component`
	var rows []manualOverrideRow
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, wrapDBError("get_all_overrides", "ManualPriceOverride", "", err)
	}
	result := make([]*pricing.ManualPriceOverride, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *pricingRepository) UpsertOverride(ctx context.Context, o *pricing.ManualPriceOverride) error {
	now := time.Now()
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
		o.CreatedAt = now
	}

	row := manualOverrideFromDomain(o)
	query := `INSERT INTO manual_price_overrides (id, canonical_model_name, component, price_usd, note, created_by, created_at, expires_at)
		VALUES (:id, :canonical_model_name, :component, :price_usd, :note, :created_by, :created_at, :expires_at)
		ON CONFLICT (canonical_model_name, component) DO UPDATE SET
		price_usd = EXCLUDED.price_usd,
		note = EXCLUDED.note,
		created_by = EXCLUDED.created_by,
		expires_at = EXCLUDED.expires_at`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return wrapDBError("upsert_override", "ManualPriceOverride", o.CanonicalModelName, err)
	}
	return nil
}

func (r *pricingRepository) DeleteOverride(ctx context.Context, canonicalModel string, component pricing.PricingComponent) error {
	query := r.db.Rebind(`DELETE FROM manual_price_overrides WHERE canonical_model_name = ? AND component = ?`)
	_, err := r.db.ExecContext(ctx, query, canonicalModel, string(component))
	if err != nil {
		return wrapDBError("delete_override", "ManualPriceOverride", canonicalModel, err)
	}
	return nil
}

func (r *pricingRepository) CleanupExpiredOverrides(ctx context.Context) (int, error) {
	now := time.Now()
	query := r.db.Rebind(`DELETE FROM manual_price_overrides WHERE expires_at IS NOT NULL AND expires_at < ?`)
	result, err := r.db.ExecContext(ctx, query, now)
	if err != nil {
		return 0, wrapDBError("cleanup_expired_overrides", "ManualPriceOverride", "", err)
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// --- Historical Pricing Methods ---

func (r *pricingRepository) GetHistoricalAverages(ctx context.Context, canonicalModel string, since time.Time) (*pricing.HistoricalPricing, error) {
	// Query run_events for metric events and calculate averages
	// This needs to join with runs to get the model, then aggregate cost/token data
	var query string
	args := []interface{}{canonicalModel, since}

	if r.db.Dialect() == DialectPostgres {
		query = `
			SELECT
				$1 as canonical_model,
				AVG(CASE WHEN (e.data->>'inputTokens')::int > 0
					THEN (e.data->>'inputCostUsd')::numeric / (e.data->>'inputTokens')::int
					ELSE NULL END) as input_token_avg_price,
				AVG(CASE WHEN (e.data->>'outputTokens')::int > 0
					THEN (e.data->>'outputCostUsd')::numeric / (e.data->>'outputTokens')::int
					ELSE NULL END) as output_token_avg_price,
				AVG(CASE WHEN (e.data->>'cacheReadTokens')::int > 0
					THEN (e.data->>'cacheReadCostUsd')::numeric / (e.data->>'cacheReadTokens')::int
					ELSE NULL END) as cache_read_avg_price,
				AVG(CASE WHEN (e.data->>'cacheCreationTokens')::int > 0
					THEN (e.data->>'cacheCreationCostUsd')::numeric / (e.data->>'cacheCreationTokens')::int
					ELSE NULL END) as cache_creation_avg_price,
				COUNT(*) as sample_count
			FROM run_events e
			JOIN runs r ON e.run_id = r.id
			WHERE e.event_type = 'metric'
				AND (r.resolved_config->>'model' = $1 OR e.data->>'pricingModel' = $1)
				AND r.created_at >= $2
				AND (e.data->>'costSource' = 'runner_reported' OR e.data->>'costSource' = 'provider_usage_api')`
	} else {
		query = `
			SELECT
				? as canonical_model,
				AVG(CASE WHEN CAST(json_extract(e.data, '$.inputTokens') AS INTEGER) > 0
					THEN CAST(json_extract(e.data, '$.inputCostUsd') AS REAL) / CAST(json_extract(e.data, '$.inputTokens') AS INTEGER)
					ELSE NULL END) as input_token_avg_price,
				AVG(CASE WHEN CAST(json_extract(e.data, '$.outputTokens') AS INTEGER) > 0
					THEN CAST(json_extract(e.data, '$.outputCostUsd') AS REAL) / CAST(json_extract(e.data, '$.outputTokens') AS INTEGER)
					ELSE NULL END) as output_token_avg_price,
				AVG(CASE WHEN CAST(json_extract(e.data, '$.cacheReadTokens') AS INTEGER) > 0
					THEN CAST(json_extract(e.data, '$.cacheReadCostUsd') AS REAL) / CAST(json_extract(e.data, '$.cacheReadTokens') AS INTEGER)
					ELSE NULL END) as cache_read_avg_price,
				AVG(CASE WHEN CAST(json_extract(e.data, '$.cacheCreationTokens') AS INTEGER) > 0
					THEN CAST(json_extract(e.data, '$.cacheCreationCostUsd') AS REAL) / CAST(json_extract(e.data, '$.cacheCreationTokens') AS INTEGER)
					ELSE NULL END) as cache_creation_avg_price,
				COUNT(*) as sample_count
			FROM run_events e
			JOIN runs r ON e.run_id = r.id
			WHERE e.event_type = 'metric'
				AND (json_extract(r.resolved_config, '$.model') = ? OR json_extract(e.data, '$.pricingModel') = ?)
				AND r.created_at >= ?
				AND (json_extract(e.data, '$.costSource') = 'runner_reported' OR json_extract(e.data, '$.costSource') = 'provider_usage_api')`
		// SQLite needs the model param three times and since once more
		args = []interface{}{canonicalModel, canonicalModel, canonicalModel, since}
	}

	query = r.db.Rebind(query)

	var result struct {
		CanonicalModel        string          `db:"canonical_model"`
		InputTokenAvgPrice    sql.NullFloat64 `db:"input_token_avg_price"`
		OutputTokenAvgPrice   sql.NullFloat64 `db:"output_token_avg_price"`
		CacheReadAvgPrice     sql.NullFloat64 `db:"cache_read_avg_price"`
		CacheCreationAvgPrice sql.NullFloat64 `db:"cache_creation_avg_price"`
		SampleCount           int             `db:"sample_count"`
	}

	if err := r.db.GetContext(ctx, &result, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBError("get_historical_averages", "HistoricalPricing", canonicalModel, err)
	}

	if result.SampleCount == 0 {
		return nil, nil
	}

	h := &pricing.HistoricalPricing{
		CanonicalModel: canonicalModel,
		SampleCount:    result.SampleCount,
		Since:          since,
	}

	if result.InputTokenAvgPrice.Valid {
		h.InputTokenAvgPrice = &result.InputTokenAvgPrice.Float64
	}
	if result.OutputTokenAvgPrice.Valid {
		h.OutputTokenAvgPrice = &result.OutputTokenAvgPrice.Float64
	}
	if result.CacheReadAvgPrice.Valid {
		h.CacheReadAvgPrice = &result.CacheReadAvgPrice.Float64
	}
	if result.CacheCreationAvgPrice.Valid {
		h.CacheCreationAvgPrice = &result.CacheCreationAvgPrice.Float64
	}

	return h, nil
}

// --- Settings Methods ---

func (r *pricingRepository) GetSettings(ctx context.Context) (*pricing.PricingSettings, error) {
	query := `SELECT historical_average_days, provider_cache_ttl_seconds FROM pricing_settings WHERE id = 1`

	var row struct {
		HistoricalAverageDays   int `db:"historical_average_days"`
		ProviderCacheTTLSeconds int `db:"provider_cache_ttl_seconds"`
	}

	if err := r.db.GetContext(ctx, &row, query); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return defaults if no settings exist
			return pricing.DefaultPricingSettings(), nil
		}
		return nil, wrapDBError("get_settings", "PricingSettings", "", err)
	}

	return &pricing.PricingSettings{
		HistoricalAverageDays: row.HistoricalAverageDays,
		ProviderCacheTTL:      time.Duration(row.ProviderCacheTTLSeconds) * time.Second,
	}, nil
}

func (r *pricingRepository) UpdateSettings(ctx context.Context, settings *pricing.PricingSettings) error {
	query := `UPDATE pricing_settings SET
		historical_average_days = :historical_average_days,
		provider_cache_ttl_seconds = :provider_cache_ttl_seconds,
		updated_at = :updated_at
		WHERE id = 1`

	row := struct {
		HistoricalAverageDays   int        `db:"historical_average_days"`
		ProviderCacheTTLSeconds int        `db:"provider_cache_ttl_seconds"`
		UpdatedAt               SQLiteTime `db:"updated_at"`
	}{
		HistoricalAverageDays:   settings.HistoricalAverageDays,
		ProviderCacheTTLSeconds: int(settings.ProviderCacheTTL.Seconds()),
		UpdatedAt:               SQLiteTime(time.Now()),
	}

	result, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return wrapDBError("update_settings", "PricingSettings", "", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		// Insert if no row exists
		insertQuery := `INSERT INTO pricing_settings (id, historical_average_days, provider_cache_ttl_seconds, updated_at)
			VALUES (1, :historical_average_days, :provider_cache_ttl_seconds, :updated_at)`
		if _, err = r.db.NamedExecContext(ctx, insertQuery, row); err != nil {
			return wrapDBError("insert_settings", "PricingSettings", "", err)
		}
	}

	return nil
}
