# CLI Reference

The Brand Manager CLI communicates with the API server. All commands require the API to be running unless otherwise noted.

**Implementation**: [CODE: cli/app.go]

## Health

### `brand-manager status`

Check API health and readiness.

**Implementation**: [CODE: cli/app.go#cmdStatus]

**Output**:
```
Status: healthy
Ready: true
Service: brand-manager
Version: 1.0.0
Dependencies:
  sqlite: healthy
```

## Brands

### `brand-manager create`

Create a new brand. [REQ: BM-REQ-CLI-CRUD]

**Implementation**: [CODE: cli/app.go#cmdCreate]

**Flags**:
| Flag | Required | Description |
|------|----------|-------------|
| `--name` | Yes | Brand name |
| `--description` | No | Brand description |

**Example**:
```bash
brand-manager create --name "My Brand" --description "Professional branding"
```

### `brand-manager list`

List all brands in tabular format. [REQ: BM-REQ-CLI-CRUD]

**Implementation**: [CODE: cli/app.go#cmdList]

**Output**:
```
  abc-123  My Brand  (v1)
  def-456  Other Brand  (v3)
```

### `brand-manager get <id>`

Get full brand details as JSON. [REQ: BM-REQ-CLI-CRUD]

**Implementation**: [CODE: cli/app.go#cmdGet]

**Example**:
```bash
brand-manager get abc-123-def-456
```

## Configuration

### `brand-manager configure`

Configure API base URL and authentication token.

**Configurable values**: `api_base`, `token`, `api_token`

## Planned Commands

The following commands are defined in the PRD but not yet implemented:

| Command | Purpose | PRD Target |
|---------|---------|------------|
| `update` | Update an existing brand | OT-P0-010 |
| `generate` | AI-generate brand elements | OT-P0-003, OT-P0-010 |
| `discover` | Scan scenario for existing branding | OT-P0-008, OT-P0-010 |
| `apply` | Apply brand to a scenario | OT-P0-004, OT-P0-010 |
| `status <scenario>` | Check scenario branding status | OT-P0-010 |
