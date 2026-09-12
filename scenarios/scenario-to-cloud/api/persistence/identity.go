package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
)

// identityConflict is the typed error for a second deployment claiming the
// same (scenario, environment, target).
func identityConflict(scenarioID, environment string, target identity.TargetRef) error {
	return apierrors.New(apierrors.CodeDeploymentIdentityConflict, "A deployment already exists for this scenario, environment and target").
		WithDetail("scenario_id", scenarioID).
		WithDetail("environment", environment).
		WithDetail("target_key", target.Key()).
		WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "endpoint", Reference: "/api/v1/deployments/resolve", Label: "Resolve the existing deployment"})
}

// deploymentIdentityColumns are the columns that carry identity beside the
// legacy manifest. They are declared once so PostgreSQL and SQLite stay in
// step; the engines differ only in how "add if missing" is spelled.
var deploymentIdentityColumns = []struct {
	name       string
	postgres   string
	sqlite     string
	definition string
}{
	{name: "environment", postgres: "TEXT NOT NULL DEFAULT 'production'", sqlite: "TEXT NOT NULL DEFAULT 'production'"},
	{name: "target_binding", postgres: "JSONB", sqlite: "TEXT"},
	{name: "target_key", postgres: "TEXT", sqlite: "TEXT"},
	{name: "desired_state", postgres: "TEXT NOT NULL DEFAULT 'running'", sqlite: "TEXT NOT NULL DEFAULT 'running'"},
	{name: "persistent_data", postgres: "JSONB", sqlite: "TEXT"},
	{name: "fence", postgres: "BIGINT NOT NULL DEFAULT 0", sqlite: "INTEGER NOT NULL DEFAULT 0"},
}

// deploymentIdentityIndex is the uniqueness rule for deployment identity.
// target_key is denormalised on every write (see bindingForWrite) because
// SQLite cannot index a JSON expression the way PostgreSQL can, and the
// production authority and the routed test pools must enforce the same rule.
const deploymentIdentityIndex = `CREATE UNIQUE INDEX IF NOT EXISTS uq_deployments_identity ON deployments(scenario_id, environment, target_key)`

// ensureDeploymentIdentity adds the identity columns to an existing
// deployments table, converts rows that predate them, and only then installs
// the unique index. The index is created outside the tolerant migration loop
// on purpose: a duplicate-identity failure must surface, not be swallowed as
// "already exists".
func ensureDeploymentIdentity(ctx context.Context, db DB, dialect string) error {
	if isSQLite(dialect) {
		existing, err := sqliteColumns(ctx, db, "deployments")
		if err != nil {
			return err
		}
		for _, col := range deploymentIdentityColumns {
			if existing[col.name] {
				continue
			}
			if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE deployments ADD COLUMN %s %s", col.name, col.sqlite)); err != nil {
				return fmt.Errorf("add deployments.%s: %w", col.name, err)
			}
		}
	} else {
		for _, col := range deploymentIdentityColumns {
			if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE deployments ADD COLUMN IF NOT EXISTS %s %s", col.name, col.postgres)); err != nil {
				return fmt.Errorf("add deployments.%s: %w", col.name, err)
			}
		}
	}
	if err := convertLegacyDeploymentIdentity(ctx, db); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, deploymentIdentityIndex); err != nil {
		return fmt.Errorf("deployment identity index: %w (two deployments share a scenario, environment and target; resolve the duplicate before starting)", err)
	}
	return nil
}

func isSQLite(dialect string) bool {
	d := strings.ToLower(strings.TrimSpace(dialect))
	return d == "sqlite" || d == "sqlite3"
}

func sqliteColumns(ctx context.Context, db DB, table string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, fmt.Errorf("inspect %s columns: %w", table, err)
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var (
			cid        int
			name       string
			typ        string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultVal, &pk); err != nil {
			return nil, fmt.Errorf("scan %s column: %w", table, err)
		}
		columns[name] = true
	}
	return columns, rows.Err()
}

// legacyDeploymentRow is the predecessor record shape: identity was implied
// by scenario_id plus manifest.target.vps.host.
type legacyDeploymentRow struct {
	id       string
	manifest json.RawMessage
}

// convertLegacyDeploymentIdentity back-fills environment and target_binding
// for rows that predate the identity columns. The conversion is portable Go,
// not engine SQL, so the same code runs against the PostgreSQL authority and a
// routed SQLite pool. Rules:
//   - environment: manifest.environment when set, else "production";
//   - target_binding: transport "ssh", empty machine identity, locator from
//     manifest.target.vps (host, port, user, workdir);
//   - key_path is NOT copied: it stays in the manifest as transport config.
//
// Rows that already carry a binding are never rewritten.
func convertLegacyDeploymentIdentity(ctx context.Context, db DB) error {
	rows, err := db.QueryContext(ctx, `SELECT id, manifest FROM deployments WHERE target_binding IS NULL`)
	if err != nil {
		return fmt.Errorf("list legacy deployments: %w", err)
	}
	var legacy []legacyDeploymentRow
	for rows.Next() {
		var row legacyDeploymentRow
		var manifest []byte
		if err := rows.Scan(&row.id, &manifest); err != nil {
			rows.Close()
			return fmt.Errorf("scan legacy deployment: %w", err)
		}
		row.manifest = manifest
		legacy = append(legacy, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate legacy deployments: %w", err)
	}
	rows.Close()

	for _, row := range legacy {
		var m domain.CloudManifest
		if len(row.manifest) > 0 {
			if err := json.Unmarshal(row.manifest, &m); err != nil {
				return fmt.Errorf("decode manifest for deployment %s during identity conversion: %w", row.id, err)
			}
		}
		target := domain.TargetRefFromManifest(m)
		binding, err := json.Marshal(target)
		if err != nil {
			return fmt.Errorf("encode target binding for deployment %s: %w", row.id, err)
		}
		if _, err := db.ExecContext(ctx,
			`UPDATE deployments SET environment = $2, target_binding = $3, target_key = $4 WHERE id = $1`,
			row.id, identity.NormalizeEnvironment(m.Environment), binding, targetKeyPtr(target),
		); err != nil {
			return fmt.Errorf("convert deployment %s identity: %w", row.id, err)
		}
	}
	return nil
}
