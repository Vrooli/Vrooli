## Tools focus: Scenario to Cloud

Use `scenario-to-cloud` to deploy a scenario to an existing VPS and operate that deployment.

---

### **1. When to Use This Tool**

| Goal | Command |
|------|---------|
| Resolve existing deployment by selector | `scenario-to-cloud deployment resolve --host <host> [--scenario <id>] [--domain <domain>]` |
| Deploy/redeploy only when required | `scenario-to-cloud redeploy cloud-manifest.json --if-needed --preflight --wait` |
| Validate deployment health by selector or id | `scenario-to-cloud deployment health --host <host> --scenario <id>` |
| Read deployment logs | `scenario-to-cloud inspect logs <deployment-id>` |
| Verify SSH access safely (agent mode) | `scenario-to-cloud ssh bootstrap <host> --user root --non-interactive` |

**Scope boundaries:**
- **In scope:** Manifest validation, deployment lifecycle, preflight validation, SSH/DNS/TLS checks, logs and runtime inspection, process control, deployment secrets.
- **Out of scope:** VPS provisioning, cloud provider account setup, domain registration, desktop packaging.

---

### **2. Core Workflow**

#### Step 1: Check SSH bootstrap readiness (agent-safe)

```bash
scenario-to-cloud ssh bootstrap {{VPS_HOST}} --user root --non-interactive
```

If this fails, interactive password entry is required to authorize key-based access. Stop immediately and include the exact handoff command from the command output in your final response.

#### Step 2: Resolve current deployment state (no manifest required)

```bash
scenario-to-cloud deployment health \
  --host {{VPS_HOST}} \
  --scenario {{SCENARIO_NAME}} \
  --json
```

This command resolves deployment ID first, then returns health/freshness.
If no deployment exists for the selector, it fails with a "no deployment found" error.

#### Step 3: Decision tree (always converge to desired state)

```mermaid
flowchart TD
  A[deployment health by selector] --> B{deployment found?}
  B -->|no| C[create+validate manifest]
  C --> D[redeploy --if-needed --preflight --wait]
  B -->|yes| E{health=healthy AND freshness=current?}
  E -->|yes| F[no-op]
  E -->|no| G{health=stopped AND freshness=current?}
  G -->|yes| H[start deployment]
  G -->|no| I[redeploy --if-needed --preflight --wait]
  D --> J[deployment health id]
  F --> J
  H --> J
  I --> J
```

If deployment is missing (or redeploy is needed and no manifest exists locally), create one in a persistent scenario path:

```bash
scenario-to-cloud manifest init \
  --scenario {{SCENARIO_NAME}} \
  --host {{VPS_HOST}} \
  --domain {{DOMAIN}} \
  --out scenarios/{{SCENARIO_NAME}}/.vrooli/cloud/manifest.{{ENV}}.json

scenario-to-cloud manifest validate \
  scenarios/{{SCENARIO_NAME}}/.vrooli/cloud/manifest.{{ENV}}.json
```

Default convergence command:

```bash
scenario-to-cloud redeploy \
  scenarios/{{SCENARIO_NAME}}/.vrooli/cloud/manifest.{{ENV}}.json \
  --if-needed --preflight --wait
```

#### Step 4: Verify and triage

```bash
scenario-to-cloud deployment health <deployment-id>
scenario-to-cloud inspect logs <deployment-id> --tail 200
```

---

### **3. Troubleshooting Workflow**

1. Run health first:
```bash
scenario-to-cloud deployment health --host {{VPS_HOST}} --scenario {{SCENARIO_NAME}}
```
2. Run the exact recommendation commands from the output.
3. Check logs for errors:
```bash
scenario-to-cloud inspect logs <deployment-id> --level error --since 1h
```

---

### **4. Requirements Source of Truth**

Do not duplicate VPS requirements in this skill.
Use the scenario command that returns canonical policy used by runtime preflight:

```bash
scenario-to-cloud preflight requirements
scenario-to-cloud preflight requirements --json
```

---

### **5. If You Need More Commands**

Use built-in group help:

```bash
scenario-to-cloud manifest help
scenario-to-cloud deployment help
scenario-to-cloud preflight help
scenario-to-cloud ssh help
scenario-to-cloud inspect help
scenario-to-cloud edge help
```

Use these only when the core workflow is insufficient.

---

### **6. Guardrails**

**Do:**
- Use `redeploy --preflight --wait` for first deploys and high-confidence redeploys.
- Use `deployment health --host ... --scenario ...` as the default state-check entrypoint.
- Use `deployment resolve` when you need a deployment ID without running health checks.
- Use `redeploy --if-needed --preflight --wait` as the default automation path.
- Use `deployment health` as the default triage entrypoint.
- Use `ssh bootstrap ... --non-interactive` before deployment in agent-driven workflows.
- Prefer exact commands emitted by tool output over ad-hoc fixes.

**Do NOT:**
- Assume VPS provisioning is handled by this tool.
- Duplicate VPS requirement policy in skill text.
- Use deprecated command aliases.
- Hardcode passwords into scripts.

---

### **7. Output Expectations**

**After a successful deployment:**
- `deployment health <id>` reports `HEALTHY` or actionable next steps.
- Scenario is reachable at `https://{{DOMAIN}}`.

**May create/update:**
- Deployment records and bundle artifacts.
- VPS runtime/configuration state (services, Caddy, TLS certificates).
- Local SSH keys and remote key authorization (via bootstrap/copy-key).

**Must not:**
- Modify scenario source code.
- Provision/destroy VPS instances.
- Install local dependencies without explicit permission.
