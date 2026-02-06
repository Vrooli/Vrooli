## Tools focus: Scenario to Cloud

Use `scenario-to-cloud` to:
- Deploy a {{TARGET}} to a VPS, or
- Find and manage existing deployments

---

### **1. When to Use This Tool**

| Goal | Command |
|------|---------|
| Deploy scenario to VPS (one-shot) | `scenario-to-cloud redeploy cloud-manifest.json --preflight` |
| List existing deployments | `scenario-to-cloud deployment list` |
| Check deployment health | `scenario-to-cloud inspect live <deployment-id>` |
| View deployment logs | `scenario-to-cloud inspect logs <deployment-id>` |
| Detect config drift | `scenario-to-cloud inspect drift <deployment-id>` |
| Verify DNS + TLS | `scenario-to-cloud edge dns-check <deployment-id>` |
| Test SSH connectivity | `scenario-to-cloud ssh test <host>` |

**Scope boundaries:**
- **In scope:** VPS deployment lifecycle, manifest validation, bundle building, SSH/DNS/TLS management, remote inspection, process control, secrets
- **Out of scope:** Building desktop apps (scenario-to-desktop skill), LPBS remote profiles / desktop upload (landing-page-desktop-upload skill), VPS provisioning (cloud provider console), domain registration

---

### **2. Command Reference**

The CLI uses **subcommand groups**. Run `<group> help` for detailed options:

| Group | Purpose | Key Subcommands |
|-------|---------|-----------------|
| `status` | API health check | (flat) |
| `manifest` | Manifest validation | `validate` |
| `bundle` | Bundle operations | `build`, `list`, `stats`, `delete`, `cleanup`, `vps-list`, `vps-delete` |
| `deployment` | Deployment lifecycle | `create`, `list`, `get`, `delete`, `execute`, `start`, `stop`, `history`, `plan` |
| `redeploy` | One-shot create + execute | (flat) |
| `preflight` | VPS preflight checks | `run`, `fix-ports`, `fix-firewall`, `fix-processes`, `disk-usage`, `disk-cleanup` |
| `vps` | Low-level VPS operations | `setup {plan\|apply}`, `deploy {plan\|apply}` |
| `inspect` | Remote state inspection | `plan`, `status`, `live`, `drift`, `logs`, `files` |
| `process` | Process management | `kill`, `restart`, `control`, `vps-action` |
| `edge` | Edge & TLS management | `dns-check`, `dns-records`, `caddy`, `tls`, `tls-renew` |
| `ssh` | SSH key management | `keys`, `generate`, `delete`, `test`, `copy-key` |
| `secrets` | Secrets management | `get` |
| `scenario` | Scenario discovery | `list`, `ports`, `deps` |
| `task` | AI task management | `create`, `list`, `get`, `stop`, `agent-status` |

**Global options** (all commands):
- `--api-base <url>`: Override API URL
- `--json`: Output raw JSON (available on most subcommands)

---

### **3. Primary Workflow: Deploy a Scenario**

#### Step 1: Write a cloud manifest

```json
{
  "scenario": { "id": "{{SCENARIO_NAME}}" },
  "target": {
    "vps": {
      "host": "{{VPS_HOST}}",
      "user": "root",
      "port": 22
    }
  },
  "edge": {
    "domain": "{{DOMAIN}}",
    "dns_policy": "required",
    "caddy": {
      "enabled": true,
      "email": "{{ADMIN_EMAIL}}"
    }
  }
}
```

Save as `cloud-manifest.json`.

#### Step 2: Validate the manifest

```bash
scenario-to-cloud manifest validate cloud-manifest.json
```

#### Step 3: Ensure SSH access

```bash
scenario-to-cloud ssh test {{VPS_HOST}} --user root
```

If no key exists:
```bash
scenario-to-cloud ssh generate s2c-deploy
scenario-to-cloud ssh copy-key {{VPS_HOST}} --key s2c-deploy --user root
```

#### Step 4: Deploy (one-shot)

```bash
scenario-to-cloud redeploy cloud-manifest.json --preflight
```

This creates/updates the deployment record, runs VPS preflight checks, and executes the full pipeline (bundle -> setup -> deploy -> health check).

#### Step 5: Verify

```bash
scenario-to-cloud deployment list --scenario {{SCENARIO_NAME}}
scenario-to-cloud inspect live <deployment-id>
scenario-to-cloud edge dns-check <deployment-id>
```

**Deployment status flow:** `pending` -> `setup_running` -> `setup_complete` -> `deploying` -> `deployed`

---

### **4. Inspect & Monitor Deployments**

#### Find your deployment

```bash
# List all
scenario-to-cloud deployment list

# Filter by scenario
scenario-to-cloud deployment list --scenario landing-page-business-suite

# Filter by status
scenario-to-cloud deployment list --status deployed

# Full details
scenario-to-cloud deployment get <deployment-id>
```

#### Live state (processes, resources, health)

```bash
scenario-to-cloud inspect live <deployment-id>
```

Shows: running/healthy status, uptime, public/internal IP, CPU/memory/disk, process list, resource list.

#### Configuration drift detection

```bash
scenario-to-cloud inspect drift <deployment-id>
```

Reports differences between expected and actual VPS state with severity levels.

#### Aggregated logs

```bash
scenario-to-cloud inspect logs <deployment-id> --tail 200
scenario-to-cloud inspect logs <deployment-id> --level error --since 1h
scenario-to-cloud inspect logs <deployment-id> --source postgres --search "connection refused"
```

#### Remote file inspection

```bash
scenario-to-cloud inspect files <deployment-id> /var/log
scenario-to-cloud inspect files <deployment-id> /etc/caddy/Caddyfile --content
```

#### Deployment history

```bash
scenario-to-cloud deployment history <deployment-id>
```

---

### **5. Edge & TLS Management**

#### DNS validation

```bash
scenario-to-cloud edge dns-check <deployment-id>
scenario-to-cloud edge dns-records <deployment-id>
```

#### Caddy reverse proxy control

```bash
scenario-to-cloud edge caddy <deployment-id> status
scenario-to-cloud edge caddy <deployment-id> validate
scenario-to-cloud edge caddy <deployment-id> reload
```

#### TLS certificates

```bash
# View certificate info (issuer, expiry, SANs, auto-renew status)
scenario-to-cloud edge tls <deployment-id>

# Renew certificates
scenario-to-cloud edge tls-renew <deployment-id>
scenario-to-cloud edge tls-renew <deployment-id> --domain app.example.com --force
```

#### DNS + TLS notes

- `dns_policy` in the manifest controls validation: `required` (fail if wrong), `warn` (log warning), `skip` (no check).
- Cloudflare proxied records need DNS-01 challenge -- provide `CLOUDFLARE_API_TOKEN` via deployment secrets so Caddy can use DNS-01.
- During initial TLS issuance, set DNS to "DNS only" (grey cloud in Cloudflare), then re-enable proxying after certificate is issued.

---

### **6. Process Management**

```bash
# Restart a scenario or resource
scenario-to-cloud process restart <deployment-id> scenario {{SCENARIO_NAME}}
scenario-to-cloud process restart <deployment-id> resource redis

# Bulk control (start/stop/restart)
scenario-to-cloud process control <deployment-id> restart
scenario-to-cloud process control <deployment-id> stop

# Kill by PID
scenario-to-cloud process kill <deployment-id> <pid>
scenario-to-cloud process kill <deployment-id> <pid> --signal SIGKILL

# VPS-level actions (affects the ENTIRE VPS, not just the deployment)
scenario-to-cloud process vps-action <deployment-id> reboot
scenario-to-cloud process vps-action <deployment-id> shutdown
```

**Warning:** `vps-action` commands affect the entire VPS. Only use when explicitly requested.

---

### **7. Manifest Reference**

Full manifest schema with all optional fields:

```json
{
  "scenario": { "id": "my-scenario" },
  "target": {
    "vps": {
      "host": "vps.example.com",
      "user": "root",
      "port": 22,
      "key_path": "~/.ssh/id_rsa"
    }
  },
  "edge": {
    "domain": "app.example.com",
    "dns_policy": "required",
    "caddy": { "enabled": true, "email": "admin@example.com" }
  },
  "ports": { "ui": 3000, "api": 8080, "ws": 8081 },
  "dependencies": { "resources": ["postgres", "redis"], "scenarios": [] },
  "options": { "include_packages": true, "autoheal": true, "force_rebuild": false }
}
```

| Field | Required | Default | Notes |
|-------|----------|---------|-------|
| `scenario.id` | Yes | -- | Must match a local scenario directory name |
| `target.vps.host` | Yes | -- | VPS hostname or IP |
| `target.vps.user` | No | `root` | SSH user |
| `target.vps.port` | No | `22` | SSH port |
| `target.vps.key_path` | No | auto-detected | Path to SSH private key |
| `edge.domain` | Yes | -- | Domain with A record pointing to VPS IP |
| `edge.dns_policy` | No | `required` | `required`, `warn`, or `skip` |
| `edge.caddy.enabled` | No | `true` | Enable Caddy reverse proxy + auto-HTTPS |
| `edge.caddy.email` | No | -- | Email for Let's Encrypt ACME |
| `ports` | No | scenario defaults | Override port mappings |
| `dependencies.resources` | No | from service.json | Resource IDs to start on VPS |
| `options.autoheal` | No | `true` | Auto-restart crashed processes |
| `options.force_rebuild` | No | `false` | Rebuild bundle even if cached |

**VPS prerequisites:**
- OS: Ubuntu 22.04+ or Debian 11+
- RAM: 2 GB minimum (4 GB+ recommended)
- Storage: 20 GB minimum
- SSH key auth configured (password auth not recommended)
- Domain A record pointing to VPS IP
- Ports 22 (SSH), 80 (HTTP), 443 (HTTPS) open

---

### **8. SSH & Secrets Management**

#### SSH keys

```bash
scenario-to-cloud ssh keys                                      # List all keys
scenario-to-cloud ssh generate my-key                           # Generate ed25519 key
scenario-to-cloud ssh generate my-key --type rsa --bits 4096    # Generate RSA key
scenario-to-cloud ssh delete my-key                             # Delete key
scenario-to-cloud ssh test vps.example.com --user root          # Test connection
scenario-to-cloud ssh copy-key vps.example.com --key my-key --user root  # Install key on VPS
```

#### Secrets

```bash
# View required/optional secrets for a scenario (values masked)
scenario-to-cloud secrets get {{SCENARIO_NAME}}

# Show actual values
scenario-to-cloud secrets get {{SCENARIO_NAME}} --reveal
```

---

### **9. Troubleshooting**

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| `ssh test` fails | Key not on VPS | `ssh copy-key <host> --key <name> --user root` |
| `manifest validate` errors | Missing required fields | Check `scenario.id`, `target.vps.host`, `edge.domain` |
| Stuck in `setup_running` | VPS unreachable or slow | Check `ssh test`, verify ports 22/80/443 open |
| DNS check shows `FAIL` | A record not propagated | Wait for propagation or verify at registrar |
| TLS renewal fails | Cloudflare proxy enabled | Set DNS-only, renew, re-enable proxy |
| `inspect live` shows unhealthy | Process crashed | `inspect logs <id> --level error`, then `process restart` |
| Drift detected | Manual VPS changes | Review drift items, `redeploy` to restore expected state |
| `deployment execute` timeout | Large bundle or slow VPS | Check `inspect live`, try `--force-bundle` |
| Preflight fails | VPS requirements not met | Review preflight output; use `preflight fix-ports`/`fix-firewall` |

Add `--json` to any command for structured error details.

---

### **10. Guardrails**

**Do:**
- Validate manifests before deploying (`manifest validate`)
- Run preflight on first deploy to a new VPS (`redeploy --preflight`)
- Test SSH connectivity before deploying (`ssh test`)
- Use `--json` for scripted/programmatic workflows
- Check `deployment list` before creating duplicate deployments

**Do NOT:**
- Deploy without a valid manifest
- Use `process vps-action reboot/shutdown` without explicit user request
- Skip DNS validation for production domains (`dns_policy: "skip"` is for testing only)
- Use deprecated aliases (`manifest-validate`, `vps-setup-plan`, etc.)
- Hardcode passwords in commands

---

### **11. Output Expectations**

**After successful deployment:**
- `deployment list` shows status `deployed`
- `inspect live <id>` shows `Running: true`, `Healthy: true`
- `edge dns-check <id>` shows `HEALTHY`
- `edge tls <id>` shows valid certificate with `Auto-Renew: true`
- Scenario is accessible at `https://{{DOMAIN}}`

**May create/update:**
- Deployment records (via `deployment create` / `redeploy`)
- SSH keys (via `ssh generate`)
- VPS state (packages, services, Caddy config, TLS certificates)
- Bundle artifacts (via `bundle build`)

**Must not:**
- Install local dependencies without permission
- Modify scenario source code
- Provision or destroy VPS instances (use cloud provider console)
- Build desktop apps (use scenario-to-desktop skill)
