---
title: "Troubleshooting"
description: "Solutions for common landing page issues"
category: "operational"
order: 10
audience: ["users", "developers"]
---

# Troubleshooting

Solutions for common issues with your landing page.

---

## Quick Diagnostics

Run these commands to diagnose most issues:

```bash
# Check scenario status
vrooli scenario status <your-scenario>

# Check PostgreSQL
resource-postgres status

# View logs
vrooli scenario logs <your-scenario> --tail 50

# Check health endpoint
curl http://localhost:<port>/health
```

---

## Startup Issues

### "Port already in use"

**Symptoms**: Scenario fails to start with port binding error

**Solutions**:

1. **Find what's using the port**:
   ```bash
   lsof -i :<port>
   # or
   ss -tlnp | grep <port>
   ```

2. **Stop the conflicting process**:
   ```bash
   kill <pid>
   ```

3. **Let Vrooli reallocate**:
   ```bash
   make stop && make start
   ```

### "Database connection failed"

**Solutions**:

1. **Start PostgreSQL**:
   ```bash
   resource-postgres start
   ```

2. **Check connection parameters**:
   ```bash
   resource-postgres status
   ```

3. **Verify database exists**:
   ```bash
   psql -h localhost -U postgres -c "\l" | grep <scenario-slug>
   ```

4. **Run schema setup**:
   ```bash
   cd <scenario-path>
   make setup
   ```

### "Go build failed"

**Solutions**:

1. **Check Go is installed**:
   ```bash
   go version
   ```

2. **Check for syntax errors** in the logs

3. **Run manually to see errors**:
   ```bash
   cd <scenario-path>/api
   go build -o <slug>-api .
   ```

### UI won't start

**Solutions**:

1. **Check node_modules**:
   ```bash
   cd <scenario-path>/ui
   pnpm install
   ```

2. **Check Vite config**:
   ```bash
   pnpm run dev
   ```

3. **Check for port conflicts** on UI port

---

## Admin Portal Issues

### "Session expired" / Can't log in

**Solutions**:

1. **Clear cookies** for the domain
2. **Check API is running** at `/api/v1/health`
3. **Verify credentials**:
   - Default: `admin@localhost` / `changeme123` (change via `/admin/profile`)

### Changes not saving

**Solutions**:

1. **Check browser console** for API errors
2. **Verify API health**:
   ```bash
   curl http://localhost:<port>/api/v1/health
   ```
3. **Check database connection**

### Live preview not updating

**Solutions**:

1. **Check for JavaScript errors** in browser console
2. **Ensure debounce isn't stuck** - wait 500ms after typing
3. **Refresh the page**

### Analytics showing no data

**Solutions**:

1. **Check time range filter** - may be set to empty period
2. **Verify events are being tracked**:
   ```bash
   curl http://localhost:<port>/api/v1/metrics/summary
   ```
3. **Check database has events**:
   ```sql
   SELECT COUNT(*) FROM events;
   ```

---

## A/B Testing Issues

### Same variant always shown

**Possible causes**:

1. **URL override active** - check for `?variant=` in URL
2. **localStorage stuck** - clear localStorage
3. **Only one active variant** - check variant status

**Solutions**:
```javascript
// Clear variant from localStorage (run in browser console)
localStorage.removeItem('variant_id');
```

### Variant weights not working

**Cause**: Weights are normalized - they don't need to equal 100

**Verify normalization**:
```
If weights are [50, 30, 20]:
Total = 100
Control: 50/100 = 50%
Variant A: 30/100 = 30%
Variant B: 20/100 = 20%
```

### Can't create new variant

**Solutions**:

1. **Check slug is unique** - no duplicate slugs
2. **Check Control variant exists** - new variants copy from Control
3. **Check API logs** for validation errors

---

## Payment Issues

### Stripe checkout not working

**Solutions**:

1. **Verify Stripe keys are set**:
   - Admin -> Stripe Settings
   - Check both publishable and restricted keys

2. **Check Stripe Dashboard** for errors

3. **Verify webhook endpoint** is accessible:
   ```bash
   curl -X POST http://localhost:<port>/api/v1/webhooks/stripe
   # Should return 400 (no signature), not 404
   ```

### Webhooks not received

**Solutions**:

1. **Check webhook secret** matches Stripe Dashboard

2. **Verify endpoint URL** in Stripe:
   ```
   https://your-domain.com/api/v1/webhooks/stripe
   ```

3. **Check Stripe Dashboard -> Webhooks** for delivery failures

4. **For local testing**, use Stripe CLI:
   ```bash
   stripe listen --forward-to localhost:<port>/api/v1/webhooks/stripe
   ```

### Subscription status not updating

**Solutions**:

1. **Check webhook events** are being received
2. **Verify database** has subscription records:
   ```sql
   SELECT * FROM subscriptions WHERE user_email = 'user@example.com';
   ```
3. **Check cache** - status is cached for up to 60 seconds

---

## Performance Issues

### Slow page load

**Solutions**:

1. **Check Lighthouse score**:
   ```bash
   npx lighthouse http://localhost:<port> --view
   ```

2. **Optimize images** - use WebP, compress below 200KB

3. **Check for unnecessary re-renders** in React DevTools

4. **Enable production build**:
   ```bash
   cd ui && pnpm run build && pnpm run preview
   ```

### API slow responses

**Solutions**:

1. **Check database indexes**:
   ```sql
   EXPLAIN ANALYZE SELECT * FROM sections WHERE variant_id = 1;
   ```

2. **Add missing indexes**:
   ```sql
   CREATE INDEX idx_sections_variant_id ON sections(variant_id);
   ```

3. **Check for N+1 queries** in API logs

---

## Deployment Issues

### "502 Bad Gateway"

The API isn't responding:
```bash
# Check if API is running
ps aux | grep landing-api

# Check logs
journalctl -u landing -n 100
```

### "Database connection refused"

```bash
# Verify DATABASE_URL
echo $DATABASE_URL

# Test connection
psql $DATABASE_URL -c "SELECT 1"

# Check PostgreSQL is running
systemctl status postgresql
```

### SSL certificate errors

```bash
# Check certificate validity
openssl s_client -connect landing.yourdomain.com:443 -servername landing.yourdomain.com

# Renew Let's Encrypt
sudo certbot renew --dry-run
```

### Webhook failures in production

```bash
# Check webhook endpoint is accessible
curl -X POST https://landing.yourdomain.com/api/v1/webhooks/stripe

# Should return 400 (no signature), not 404 or 5xx
```

---

## Getting More Help

### Collecting Debug Information

```bash
# System info
vrooli info

# Scenario status
vrooli scenario status <name>

# Recent logs
vrooli scenario logs <name> --tail 200

# Port allocations
./scripts/resources/port-registry.sh list
```

### Reporting Issues

When reporting issues, include:

1. **Exact error message**
2. **Steps to reproduce**
3. **Output of diagnostic commands** above
4. **Relevant log snippets**

---

## See Also

- [FAQ](FAQ.md) - Frequently asked questions
- [Concepts](CONCEPTS.md) - Architecture understanding
- [Deployment Guide](DEPLOYMENT.md) - Deployment troubleshooting
