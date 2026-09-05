CREATE TABLE IF NOT EXISTS template_records (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  display_name TEXT NOT NULL,
  version TEXT NOT NULL,
  manifest_path TEXT NOT NULL,
  source_path TEXT NOT NULL,
  tags_json TEXT NOT NULL DEFAULT '[]',
  status TEXT NOT NULL,
  current_version TEXT NOT NULL DEFAULT '',
  latest_version TEXT NOT NULL DEFAULT '',
  lag_count INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS validation_runs (
  id TEXT PRIMARY KEY,
  template_id TEXT NOT NULL,
  mode TEXT NOT NULL,
  target TEXT NOT NULL,
  status TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT NOT NULL,
  phase_results_json TEXT NOT NULL DEFAULT '[]',
  findings_json TEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE IF NOT EXISTS validation_run_attributions (
  run_id TEXT PRIMARY KEY,
  trigger TEXT NOT NULL,
  FOREIGN KEY(run_id) REFERENCES validation_runs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS drift_snapshots (
  id TEXT PRIMARY KEY,
  template_id TEXT NOT NULL,
  target TEXT NOT NULL,
  status TEXT NOT NULL,
  drift_count INTEGER NOT NULL DEFAULT 0,
  captured_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS debt_entries (
  key TEXT PRIMARY KEY,
  template_id TEXT NOT NULL,
  source TEXT NOT NULL,
  severity TEXT NOT NULL,
  status TEXT NOT NULL,
  title TEXT NOT NULL,
  detail TEXT NOT NULL,
  first_seen_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS monitor_state (
  id TEXT PRIMARY KEY,
  enabled INTEGER NOT NULL,
  interval_seconds INTEGER NOT NULL,
  in_flight INTEGER NOT NULL,
  last_run_id TEXT NOT NULL DEFAULT '',
  last_status TEXT NOT NULL DEFAULT 'never-run',
  last_started_at TEXT NOT NULL DEFAULT '',
  last_finished_at TEXT NOT NULL DEFAULT '',
  next_run_at TEXT NOT NULL,
  green_streak INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);

INSERT OR IGNORE INTO template_records
  (id, kind, display_name, version, manifest_path, source_path, tags_json, status, current_version, latest_version, lag_count, updated_at)
VALUES
  ('react-vite', 'scenario', 'React Vite Scenario', '1.6.2', 'templates/scenarios/react-vite/template.json', 'templates/scenarios/react-vite', '["scenario","react","vite"]', 'quarantined', '1.6.2', '1.6.2', 0, '2026-07-10T00:00:00Z'),
  ('landing-page-react-vite', 'scenario', 'Landing Page React Vite Scenario', '', 'templates/scenarios/landing-page-react-vite/template.json', 'templates/scenarios/landing-page-react-vite', '["scenario","landing-page"]', 'debt', '', '', 0, '2026-07-09T00:00:00Z'),
  ('vrooli-default', 'design', 'Vrooli Default Design Kit', '', 'templates/design/vrooli-default/metadata.json', 'templates/design/vrooli-default', '["design"]', 'active', '', '', 0, '2026-07-09T00:00:00Z'),
  ('vrooli-command-display', 'design', 'Vrooli Command Display Design Kit', '', 'templates/design/vrooli-command-display/metadata.json', 'templates/design/vrooli-command-display', '["design"]', 'active', '', '', 0, '2026-07-09T00:00:00Z'),
  ('vrooli-conversion-landing', 'design', 'Vrooli Conversion Landing Design Kit', '', 'templates/design/vrooli-conversion-landing/metadata.json', 'templates/design/vrooli-conversion-landing', '["design","landing-page"]', 'active', '', '', 0, '2026-07-09T00:00:00Z'),
  ('cloud-api', 'resource', 'Cloud API Resource Template', '', 'templates/resources/cloud-api/template.json', 'templates/resources/cloud-api', '["resource"]', 'active', '', '', 0, '2026-07-09T00:00:00Z'),
  ('compose-service', 'resource', 'Compose Service Resource Template', '', 'templates/resources/compose-service/template.json', 'templates/resources/compose-service', '["resource"]', 'active', '', '', 0, '2026-07-09T00:00:00Z'),
  ('desktop-app', 'resource', 'Desktop App Resource Template', '', 'templates/resources/desktop-app/template.json', 'templates/resources/desktop-app', '["resource"]', 'active', '', '', 0, '2026-07-09T00:00:00Z'),
  ('docker-service', 'resource', 'Docker Service Resource Template', '', 'templates/resources/docker-service/template.json', 'templates/resources/docker-service', '["resource"]', 'active', '', '', 0, '2026-07-09T00:00:00Z'),
  ('external-cli', 'resource', 'External CLI Resource Template', '', 'templates/resources/external-cli/template.json', 'templates/resources/external-cli', '["resource"]', 'active', '', '', 0, '2026-07-09T00:00:00Z'),
  ('manual-resource', 'resource', 'Manual Resource Template', '', 'templates/resources/manual-resource/template.json', 'templates/resources/manual-resource', '["resource"]', 'active', '', '', 0, '2026-07-09T00:00:00Z'),
  ('native-cli', 'resource', 'Native CLI Resource Template', '', 'templates/resources/native-cli/template.json', 'templates/resources/native-cli', '["resource"]', 'active', '', '', 0, '2026-07-09T00:00:00Z');

INSERT OR IGNORE INTO validation_runs
  (id, template_id, mode, target, status, started_at, finished_at, phase_results_json, findings_json)
VALUES
  ('phase2-bootstrap', 'react-vite', 'shallow', 'templates/scenarios/react-vite', 'recorded', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z', '[{"phase":"schema","status":"seeded","finding_count":0}]', '[]');

INSERT OR IGNORE INTO validation_run_attributions (run_id, trigger)
VALUES ('phase2-bootstrap', 'seed');

INSERT OR IGNORE INTO drift_snapshots
  (id, template_id, target, status, drift_count, captured_at)
VALUES
  ('phase2-bootstrap-drift', 'react-vite', 'fleet', 'pending-live-run', 0, '2026-07-09T00:00:00Z');

INSERT OR IGNORE INTO debt_entries
  (key, template_id, source, severity, status, title, detail, first_seen_at, last_seen_at)
VALUES
  ('react-vite.i18n.theme-choice-unused-key-false-positive', 'react-vite', '2026-07-08 template audit', 'warning', 'open', 'Theme choice key trips no-unused-keys falsely', 'The generated UI reports theme.choice as unused even though the selector flow expects it, creating inherited lint noise.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('react-vite.i18n.orphan-app-eyebrow', 'react-vite', '2026-07-08 template audit', 'warning', 'open', 'Generated locale keeps orphan app.eyebrow key', 'The template ships app.eyebrow after the UI stopped reading it, causing every scenario to inherit stale i18n content.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('react-vite.i18n.orphan-app-description', 'react-vite', '2026-07-08 template audit', 'warning', 'open', 'Generated locale keeps orphan app.description key', 'The template ships app.description after the UI stopped reading it, causing every scenario to inherit stale i18n content.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('react-vite.ui.appshell-literal-aria-label', 'react-vite', '2026-07-08 template audit', 'warning', 'open', 'AppShell uses a literal aria-label', 'The generated AppShell bypasses the typed string registry for its navigation aria-label.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('react-vite.ui.theme-provider-unnecessary-condition', 'react-vite', '2026-07-08 template audit', 'warning', 'open', 'ThemeProvider trips no-unnecessary-condition', 'The generated ThemeProvider contains a condition that eslint flags in every new scenario.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('react-vite.ui.routes-react-refresh-warning', 'react-vite', '2026-07-08 template audit', 'warning', 'open', 'routes.tsx trips react-refresh warning', 'The generated route module exports a shape that triggers the react-refresh lint rule.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('react-vite.ui.router-v7-future-flags-missing', 'react-vite', '2026-07-08 template audit', 'warning', 'open', 'React Router v7 future flags missing', 'The generated router lacks the expected v7 future flags and emits inherited runtime warnings.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('react-vite.a11y.duplicate-primary-navigation-landmark', 'react-vite', '2026-07-08 template audit', 'error', 'open', 'Duplicate Primary navigation landmark', 'The generated shell can expose duplicate Primary navigation landmarks under axe, creating inherited accessibility debt.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('landing-page-react-vite.contract.missing-version', 'landing-page-react-vite', '2026-07-08 template audit', 'error', 'open', 'landing-page-react-vite has no version', 'The landing page template lacks a version, so lifecycle and migration tooling cannot reason about lag.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('landing-page-react-vite.contract.missing-orientation', 'landing-page-react-vite', '2026-07-08 template audit', 'error', 'open', 'landing-page-react-vite has no orientation', 'The landing page template does not declare orientation gates, so orient silently has no useful work order.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('landing-page-react-vite.contract.missing-example-domain', 'landing-page-react-vite', '2026-07-08 template audit', 'warning', 'open', 'landing-page-react-vite has no exampleDomain', 'The landing page template lacks exampleDomain metadata used by detemplate and lifecycle checks.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('landing-page-react-vite.api.gorilla-mux-not-connect', 'landing-page-react-vite', '2026-07-08 template audit', 'error', 'open', 'landing page API uses gorilla/mux instead of Connect', 'The landing page template diverges from the Proto+Connect scenario rule.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('landing-page-react-vite.docs.broken-file-dep-paths', 'landing-page-react-vite', '2026-07-08 template audit', 'warning', 'open', 'landing page docs contain broken file: dependency paths', 'The template documentation references dependency paths that do not resolve under machine-readable reference validation.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('landing-page-react-vite.docs.dead-requirements-guide-pointer', 'landing-page-react-vite', '2026-07-08 template audit', 'warning', 'open', 'landing page docs point at dead requirements guide', 'The template references a requirements guide path that no longer exists.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('landing-page-react-vite.docs.wrong-readme-stack-claim', 'landing-page-react-vite', '2026-07-08 template audit', 'warning', 'open', 'landing page README has wrong stack claim', 'The README describes a stack that does not match the template implementation.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('react-vite.changelog.missing-1.0.1', 'react-vite', '2026-07-08 template audit', 'warning', 'open', 'react-vite CHANGELOG missing 1.0.1 entry', 'The template version exists in lifecycle history but the CHANGELOG does not carry the migration entry required by the read-every-entry-above-your-version protocol.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('react-vite.changelog.missing-1.4.0', 'react-vite', '2026-07-08 template audit', 'warning', 'open', 'react-vite CHANGELOG missing 1.4.0 entry', 'The template version exists in git history but the CHANGELOG does not carry the migration entry required by the read-every-entry-above-your-version protocol.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z'),
  ('react-vite.changelog.entries-out-of-order', 'react-vite', '2026-07-08 template audit', 'warning', 'open', 'react-vite CHANGELOG entries are out of order', 'The CHANGELOG ordering breaks the migration algorithm that asks agents to read every entry above their recorded version.', '2026-07-09T00:00:00Z', '2026-07-09T00:00:00Z');

INSERT OR IGNORE INTO monitor_state
  (id, enabled, interval_seconds, in_flight, last_run_id, last_status, last_started_at, last_finished_at, next_run_at, green_streak, updated_at)
VALUES
  ('default', 1, 86400, 0, '', 'never-run', '', '', '2026-07-10T00:00:00Z', 0, '2026-07-09T00:00:00Z');
