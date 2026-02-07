## Tools focus: Scenario to Cloud

Use `scenario-to-cloud` to:
- Deploy a {{TARGET}} to a VPS, or
- Find and manage existing deployments

---

### **1. When to Use This Tool**

| Goal | Command |
|------|---------|
| Deploy scenario to VPS (one-shot) | `scenario-to-cloud redeploy cloud-manifest.json --preflight --wait` |
| List existing deployments | `scenario-to-cloud deployment list` |
| Check deployment health / triage issues | `scenario-to-cloud deployment health <deployment-id>` |
| View deployment logs | `scenario-to-cloud inspect logs <deployment-id>` |
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
| `manifest` | Manifest schema + lifecycle | `schema`, `init`, `template`, `validate`, `doctor`, `fix` |
| `bundle` | Bundle operations | `build`, `list`, `stats`, `delete`, `cleanup`, `vps-list`, `vps-delete` |
| `deployment` | Deployment lifecycle | `create`, `list`, `get`, `delete`, `execute`, `start`, `stop`, `history`, `health`, `plan` |
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

#### Step 1: Initialize a cloud manifest

```bash
scenario-to-cloud manifest init \
  --scenario {{SCENARIO_NAME}} \
  --host {{VPS_HOST}} \
  --domain {{DOMAIN}} \
  --caddy-email {{ADMIN_EMAIL}} \
  --out cloud-manifest.json
```

Optional checks:
```bash
scenario-to-cloud manifest doctor cloud-manifest.json
scenario-to-cloud manifest fix cloud-manifest.json --write
```

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
scenario-to-cloud redeploy cloud-manifest.json --preflight --wait
```

This creates/updates the deployment record, runs VPS preflight checks, and executes the full pipeline (bundle -> setup -> deploy -> health check).
`--wait` is required so deployment progress is observed to completion, with stage-by-stage timing output.

#### Step 5: Verify

```bash
scenario-to-cloud deployment health <deployment-id>
```

This checks deployment status, processes, DNS, TLS, and system resources in one pass. Follow its recommendations to resolve any issues.

**Deployment status flow:** `pending` -> `setup_running` -> `setup_complete` -> `deploying` -> `deployed`

---

### **4. Monitor & Troubleshoot**

#### Health check (run this first)

```bash
scenario-to-cloud deployment health <deployment-id>
```

Checks deployment status, SSH, processes, DNS, TLS, Caddy, and system resources. Provides specific recommendations with exact commands to run.

**If health is unavailable** (API down, no deployment record):

| Symptom | Fix |
|---------|-----|
| `ssh test` fails | `ssh copy-key <host> --key <name> --user root` |
| `manifest validate` errors | Check `scenario.id`, `target.vps.host`, `edge.domain` |

#### Logs

```bash
scenario-to-cloud inspect logs <deployment-id> --tail 200
scenario-to-cloud inspect logs <deployment-id> --level error --since 1h
scenario-to-cloud inspect logs <deployment-id> --source postgres --search "connection refused"
```

---

### **5. VPS prerequisites:**
- OS: Ubuntu 22.04+ or Debian 11+
- RAM: 2 GB minimum (4 GB+ recommended)
- Storage: 20 GB minimum
- SSH key auth configured (password auth not recommended)
- Domain A record pointing to VPS IP
- Ports 22 (SSH), 80 (HTTP), 443 (HTTPS) open

**Cloudflare note:** Proxied records need DNS-01 challenge — provide `CLOUDFLARE_API_TOKEN` via deployment secrets. During initial TLS issuance, set DNS to "DNS only" (grey cloud), then re-enable proxying after certificate is issued.

---

### **6. SSH & Secrets Management**

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

### **7. Guardrails**

**Do:**
- Validate manifests before deploying (`manifest validate`)
- Run preflight on first deploy to a new VPS (`redeploy --preflight --wait`)
- Always use `--wait` for `deployment execute`, `deployment start`, and `redeploy` so stage completion and durations are visible
- Test SSH connectivity before deploying (`ssh test`)
- Use `--json` for scripted/programmatic workflows
- Check `deployment list` before creating duplicate deployments

**Do NOT:**
- Deploy without a valid manifest
- Use `process vps-action reboot/shutdown` without explicit user request
- Use deprecated aliases (`vps-setup-plan`, etc.)
- Hardcode passwords in commands

---

### **8. Output Expectations**

**After successful deployment:**
- `deployment health <id>` shows `HEALTHY` with all sections passing
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
