# Troubleshooting

Common issues and how to resolve them.

## Scenario Won't Start

### Error: "This binary must be run through the Vrooli lifecycle system"

**Cause:** Running the binary directly instead of through `vrooli scenario start`.

**Solution:**
```bash
# Correct way to start
vrooli scenario start vrooli-autoheal

# Wrong - direct execution
./api/vrooli-autoheal-api  # This will fail
```

### Error: "failed to connect to database"

**Cause:** PostgreSQL resource not running or connection string incorrect.

**Solution:**
1. Check if PostgreSQL is running:
   ```bash
   vrooli resource status postgres
   ```

2. Start it if needed:
   ```bash
   vrooli resource start postgres
   ```

3. Verify connection string in environment.

### Error: "port already in use"

**Cause:** Another process is using the allocated port.

**Solution:**
1. Find the conflicting process:
   ```bash
   lsof -i :PORT
   ```

2. Stop the scenario and restart:
   ```bash
   vrooli scenario stop vrooli-autoheal
   vrooli scenario start vrooli-autoheal
   ```

## Dashboard Issues

### Dashboard shows "Loading..." indefinitely

**Causes:**
- API not running
- Network/CORS issues
- API port mismatch

**Solutions:**
1. Check API is running:
   ```bash
   vrooli scenario status vrooli-autoheal
   curl http://localhost:API_PORT/health
   ```

2. Check browser console for errors (F12 → Console)

3. Try hard refresh (Ctrl+Shift+R / Cmd+Shift+R)

### Dashboard shows "Connection Error"

**Cause:** API is not reachable.

**Solution:**
1. Verify API health:
   ```bash
   curl http://localhost:API_PORT/health
   ```

2. Check if running behind a tunnel that needs refresh

3. Check for firewall blocking the port

### All checks show "No data" or "Never"

**Cause:** No tick has been run yet.

**Solution:**
Click "Run Tick" button or wait for auto-refresh (30s).

## Health Check Issues

### Check shows "Critical" but service is running

**Possible causes:**
1. **Interval not elapsed:** Check hasn't run recently
2. **Stale cached data:** Old result being shown
3. **Timeout:** Check took too long

**Solution:**
Click "Run Tick" with force mode to get fresh results.

### Check shows wrong platform detection

**Cause:** Platform detection caching.

**Solution:**
1. Check platform detection:
   ```bash
   curl http://localhost:API_PORT/api/v1/platform | jq
   ```

2. Restart the scenario to re-detect:
   ```bash
   vrooli scenario restart vrooli-autoheal
   ```

### Network check fails but internet works

**Cause:** Usually firewall or DNS configuration.

**Checks:**
```bash
# Test what autoheal tests
nc -zv 8.8.8.8 53

# Check DNS
nslookup google.com
```

### Docker check fails

**Causes:**
1. Docker not installed
2. Docker daemon not running
3. Permission issues

**Solutions:**
```bash
# Check Docker status
docker info

# Start Docker (Linux)
sudo systemctl start docker

# Add user to docker group (avoids sudo)
sudo usermod -aG docker $USER
```

## CLI Issues

### Command "vrooli-autoheal" not found

**Cause:** CLI not in PATH or not installed.

**Solution:**
```bash
# Use full path
./cli/vrooli-autoheal status

# Or set up alias
alias vrooli-autoheal='./cli/vrooli-autoheal'
```

### CLI can't find API port

**Cause:** Scenario not running or port discovery failed.

**Solution:**
1. Check scenario is running:
   ```bash
   vrooli scenario status vrooli-autoheal
   ```

2. Manually specify port:
   ```bash
   AUTOHEAL_API_PORT=12345 vrooli-autoheal status
   ```

## Watchdog Issues

### Watchdog install fails with "permission denied"

**Solution:** Use sudo for system-level installation:
```bash
sudo vrooli-autoheal watchdog install
```

### Watchdog keeps restarting the process

**Cause:** Process is crashing repeatedly.

**Debug:**
```bash
# Linux
journalctl -u vrooli-autoheal -f

# macOS
tail -f /tmp/vrooli-autoheal.err
```

**Common fixes:**
- Ensure PostgreSQL is running
- Check environment variables
- Verify binary path is correct

## Database Issues

### "relation does not exist" error

**Cause:** Database migrations haven't run.

**Solution:**
The schema should be created automatically on first start. If not:
```bash
# Restart the scenario
vrooli scenario restart vrooli-autoheal
```

### Old data not being cleaned up

**Cause:** Cleanup job not running or failing.

**Check:**
```sql
-- Check data age
SELECT MIN(timestamp), MAX(timestamp)
FROM health_results;
```

## Getting Help

### Collect Debug Information

When reporting issues, include:

```bash
# 1. Scenario status
vrooli scenario status vrooli-autoheal

# 2. API health
curl http://localhost:API_PORT/health | jq

# 3. Platform info
curl http://localhost:API_PORT/api/v1/platform | jq

# 4. Recent logs
vrooli scenario logs vrooli-autoheal --lines 100

# 5. System info
uname -a
```

### Log Locations

| Component | Location |
|-----------|----------|
| API logs | `vrooli scenario logs vrooli-autoheal` |
| Watchdog (Linux) | `journalctl -u vrooli-autoheal` |
| Watchdog (macOS) | `/tmp/vrooli-autoheal.log` |
| Watchdog (Windows) | Event Viewer |

### Reset Everything

Nuclear option if nothing else works:

```bash
# Stop everything
vrooli scenario stop vrooli-autoheal

# Clear database (if needed)
vrooli resource restart postgres

# Fresh start
vrooli scenario start vrooli-autoheal
```
