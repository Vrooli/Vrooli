# Problems — Calendar

Persistent register of known Calendar defects, contract drift, and deferred
work that future maintainers must not rediscover from runtime failures.

## Work ladder

- Rung: W0
- Evidence: the active user-owned Plan Manager execution
  `credential-blast-radius-make-minted-secrets-un-losable` requires Calendar to
  prove its generated JWT secret through the scenario lifecycle. No separate
  active Calendar goal establishes a different contract.
- Blocker: the embedded category schema declared a global UUID category model,
  while the API and existing database implement per-user string categories.
  Schema application failed before the API could start.
- Measured: 2026-08-27

## Entries

### 2026-08-27 — Category schema contradicted the API model

**Symptom:** Calendar startup stopped while applying its embedded schema because
`event_categories.is_system` did not exist. Test Genie could not start the
scenario and therefore ran zero validation phases.

**Root cause:** The embedded schema declared and seeded global system categories
with UUID identifiers. `categorization.go` implements built-in defaults in code
and stores only user-owned categories with string identifiers and a required
`user_id`. Existing databases used the API model, so the provider was not
idempotent and a clean database would have received a table incompatible with
the API.

**Workaround:** None. Do not delete the Calendar database or add global-category
columns to conceal the contract conflict.

**Real fix:** Keep the declarative table aligned with the per-user API model.
Ensure the `updated_at` column on existing tables before installing the update
trigger. Validate both lifecycle startup and the comprehensive Calendar suite.

**Owner:** Calendar maintainers.

**Refs:** `api/internal/calendar/schema.sql`, `api/categorization.go`, Test Genie
run `20260827-181746-9f0eb8a3`.

