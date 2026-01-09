# Bundle Storage Management

This guide covers storage management for scenario-to-cloud deployment bundles. Bundles accumulate on both the local development machine and VPS targets during deployments, and periodic cleanup prevents disk space exhaustion.

## Storage Locations

### Local Development Machine

| Path | Content |
|------|---------|
| `{repo}/scenarios/scenario-to-cloud/coverage/bundles/` | Built deployment bundles |

Bundles are created during the deployment pipeline and cached for potential reuse.

### VPS Target Server

| Path | Content |
|------|---------|
| `{workdir}/.vrooli/cloud/bundles/` | Uploaded deployment bundles |
| `{workdir}/` | Extracted scenario files |

The workdir is typically `/root/Vrooli` or a custom path specified in the deployment manifest.

## Bundle Naming Convention

```
mini-vrooli_{scenario-id}_{sha256}.tar.gz
```

- **scenario-id**: The scenario being deployed (e.g., `landing-page-business-suite`)
- **sha256**: 64-character hash of bundle contents for integrity verification

Bundles with identical content produce identical SHA256 hashes, enabling cache reuse.

## Storage Characteristics

| Scenario Type | Typical Bundle Size |
|---------------|---------------------|
| Simple UI-only | 30-50 MB |
| UI + API | 50-100 MB |
| Full stack with dependencies | 100-200 MB |

Bundle size depends on:
- Number of scenario files
- Included resources
- Dependency scenarios
- Go modules (API binaries)

## Manual Cleanup Commands

### Local Bundles

```bash
# List all bundles with sizes
ls -lh scenarios/scenario-to-cloud/coverage/bundles/

# Show total disk usage
du -sh scenarios/scenario-to-cloud/coverage/bundles/

# Remove bundles older than 7 days
find scenarios/scenario-to-cloud/coverage/bundles/ \
  -name "mini-vrooli_*.tar.gz" \
  -mtime +7 \
  -delete

# Remove bundles for a specific scenario
rm scenarios/scenario-to-cloud/coverage/bundles/mini-vrooli_my-scenario_*.tar.gz

# Remove all bundles (use with caution - forces rebuild on next deploy)
rm scenarios/scenario-to-cloud/coverage/bundles/mini-vrooli_*.tar.gz
```

### VPS Bundles

```bash
# SSH to VPS and list bundles
ssh -i ~/.ssh/my-key user@host "ls -lh /root/Vrooli/.vrooli/cloud/bundles/"

# Show total disk usage on VPS
ssh -i ~/.ssh/my-key user@host "du -sh /root/Vrooli/.vrooli/cloud/bundles/"

# Remove old bundles on VPS (keep last 3 per scenario)
ssh -i ~/.ssh/my-key user@host "cd /root/Vrooli/.vrooli/cloud/bundles && \
  for scenario in \$(ls mini-vrooli_*.tar.gz 2>/dev/null | sed 's/mini-vrooli_\([^_]*\)_.*/\1/' | sort -u); do
    ls -t mini-vrooli_\${scenario}_*.tar.gz 2>/dev/null | tail -n +4 | xargs -r rm -f
  done"

# Remove all bundles on VPS (use with caution)
ssh -i ~/.ssh/my-key user@host "rm -f /root/Vrooli/.vrooli/cloud/bundles/mini-vrooli_*.tar.gz"
```

## Automatic Cleanup (Built-in)

scenario-to-cloud provides automatic cleanup mechanisms:

### On Redeploy

When redeploying a scenario, old bundles for that scenario are automatically cleaned up. By default, the 3 most recent bundles are retained.

### On Deployment Delete

When deleting a deployment via the UI or API, you can optionally clean up associated bundles:

**API:**
```bash
# Delete deployment and clean up bundles
curl -X DELETE "http://localhost:8086/api/v1/deployments/{id}?stop=true&cleanup=true"
```

**UI:**
Check "Also delete associated bundle files (local + VPS)" in the delete confirmation dialog.

### Disk Cleanup Endpoint

The preflight disk cleanup endpoint includes bundle cleanup:

```bash
curl -X POST http://localhost:8086/api/v1/bundles/cleanup \
  -H "Content-Type: application/json" \
  -d '{
    "keep_latest": 3,
    "clean_vps": true,
    "host": "your-vps.example.com",
    "port": 22,
    "user": "root",
    "key_path": "/path/to/ssh/key",
    "workdir": "/root/Vrooli"
  }'
```

## Storage Monitoring

### Check Bundle Statistics

```bash
# Via API
curl http://localhost:8086/api/v1/bundles/stats
```

Response includes:
- Total bundle count
- Total storage used
- Per-scenario breakdown

### UI Indicators

The deployment wizard's Build step displays:
- Current bundle size after build
- Total storage used by all bundles
- Quick cleanup button

## Best Practices

1. **Regular Cleanup**: Run cleanup monthly or when disk usage exceeds thresholds
2. **Retention Policy**: Keep 2-3 versions per scenario for rollback capability
3. **Before Major Deploys**: Check available disk space on VPS (`df -h`)
4. **CI/CD Integration**: Add cleanup step to deployment pipelines
5. **Monitor VPS**: Set up disk usage alerts on production VPS targets

## Troubleshooting

### "No space left on device" during deploy

1. Check VPS disk usage: `ssh user@host "df -h"`
2. Clean old bundles (see commands above)
3. Run VPS disk cleanup: apt, journals, docker prune
4. Re-run deployment

### Bundle not found after cleanup

If a deployment references a deleted bundle, the next execution will rebuild it automatically. This is safe but adds deployment time.

### Identifying orphaned bundles

Bundles are orphaned when their deployment record is deleted without cleanup:

```bash
# List bundles on disk
ls scenarios/scenario-to-cloud/coverage/bundles/

# Compare with active deployments
curl http://localhost:8086/api/v1/deployments | jq '.[].bundle_sha256'
```

Any bundle SHA256 not in the deployments list is orphaned and can be safely deleted.
