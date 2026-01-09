# Deployment Lifecycle

Understanding the stages and status transitions of a deployment.

## Status Flow

```
pending → setup_running → setup_complete → deploying → deployed
    ↓           ↓              ↓             ↓
  failed      failed        failed       failed
    ↑           ↑              ↑             ↑
    └───────────┴──────────────┴─────────────┘

deployed → stopped (via stop action)
         → deploying (via re-deploy)
```

## Status Definitions

| Status | Description |
|--------|-------------|
| `pending` | Deployment created but not started |
| `setup_running` | VPS setup in progress |
| `setup_complete` | VPS setup finished, ready for deploy |
| `deploying` | Scenario deployment in progress |
| `deployed` | Successfully deployed and running |
| `failed` | Deployment failed at some stage |
| `stopped` | Manually stopped on VPS |

## Deployment Stages

### 1. Create Deployment

A deployment record is created with status `pending`. The manifest is stored for reproducibility.

### 2. Build Bundle

If a bundle doesn't exist (or force rebuild is requested), a mini-Vrooli bundle is created containing:
- Core Vrooli scripts (`manage.sh`, `setup.sh`)
- Scenario files and dependencies
- Resource configurations
- Deployment configuration

### 3. VPS Setup (`setup_running`)

The bundle is transferred to the VPS and setup runs:
1. Extract bundle to `~/Vrooli`
2. Run `./scripts/manage.sh setup`
3. Install required tools (git, curl, etc.)
4. Configure environment

### 4. Deploy Scenario (`deploying`)

With setup complete, the scenario is deployed:
1. Start required resources (postgres, redis, etc.)
2. Build scenario if needed
3. Start scenario services
4. Configure Caddy reverse proxy
5. Obtain SSL certificate

### 5. Health Check

After deployment, health checks verify:
- Services are running
- Ports are accessible
- HTTPS is working

## Actions

### Execute

Runs the full deployment pipeline:
1. Build bundle (if needed)
2. VPS setup
3. Deploy scenario
4. Update status

### Inspect

Checks current state on VPS:
- Scenario process status
- Resource status
- Recent logs

### Stop

Gracefully stops the scenario on VPS:
1. Stop scenario services
2. Optionally stop resources
3. Update status to `stopped`

### Re-deploy

Re-runs deployment on an existing deployment:
1. Uses existing bundle (or rebuilds)
2. Runs VPS setup
3. Deploys scenario
4. Updates `last_deployed_at`

## Error Handling

When a deployment fails:
1. Status is set to `failed`
2. `error_step` indicates where failure occurred
3. `error_message` contains error details
4. Full results stored in `setup_result` or `deploy_result`

### Recovery

To recover from a failed deployment:
1. Review error message and logs
2. Fix the underlying issue
3. Click **Retry** to re-run deployment

## Persistence

Deployment records are stored in PostgreSQL with:
- Full manifest snapshot
- Bundle information
- All stage results
- Error details
- Timestamps for auditing

This enables:
- Reproducible deployments
- History tracking
- Debug information retention
