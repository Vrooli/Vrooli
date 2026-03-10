# Research Notes — Lifestyle Dashboard

## Storage Decision: SQLite vs PostgreSQL

### Decision
SQLite-only storage (accepted suggestion s-mmjrnthj)

### Rationale
- Personal single-user app — performance not a concern
- Portability for future mobile/Electron deployment
- Single file backup/restore
- No external database server management
- Follows storage-steer skill patterns

### Trade-offs
- Single-writer constraint (all writes via API)
- No built-in replication (acceptable for personal use)
- Migration path to PostgreSQL if ever needed for scale

### Implementation Notes
- Use `modernc.org/sqlite` or `mattn/go-sqlite3` Go driver
- JSON queries via SQLite `json_extract()` functions
- Date handling with ISO-8601 strings and `datetime()` functions

## Correlation Engine Research

### Statistical Methods
- Pearson correlation for continuous variables
- Chi-squared for categorical
- P-value threshold: 0.05
- Minimum sample size: 14 (configurable per user)

### Prior Art
- Quantified Self community patterns
- N=1 experiment design from personal science literature
- Seth Roberts self-experimentation methodology

## Domain Integration Patterns

### Event-Driven Architecture
- Loose coupling via event schema
- Domains self-register on startup
- Dashboard discovers capabilities dynamically

### Similar Systems
- Apple Health HealthKit model
- Google Fit aggregation patterns
- Oura Ring cross-metric correlations

## Mobile-First Design Research

### Key Constraints (per clarification q2)
- Minimum touch target: 44x44px
- Primary actions in thumb-reach zone
- Daily brief fits in single screen scroll
- Dark mode default (late-night data checking common)

### Design References
- Apple Health app layout
- Grafana mobile dashboards
- Oura Ring daily readiness view
