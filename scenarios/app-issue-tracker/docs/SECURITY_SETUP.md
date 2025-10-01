# Security Configuration Guide

## Overview

The App Issue Tracker includes comprehensive security features that can be enabled via environment variables. By default, these features are **disabled** for local development convenience.

## Security Features

### 1. CORS Configuration

Control which domains can access your API:

```bash
# Allow all origins (default for development)
ALLOWED_ORIGINS="*"

# Allow specific origins (recommended for production)
ALLOWED_ORIGINS="https://example.com,https://app.example.com"

# Allow localhost during development
ALLOWED_ORIGINS="http://localhost:3000,http://localhost:36221"
```

### 2. API Token Authentication

Enable token-based authentication to protect your API endpoints:

```bash
# Enable authentication
ENABLE_AUTH=true

# Set API tokens (comma-separated for multiple tokens)
API_TOKENS="your-secret-token-1,your-secret-token-2"
```

When authentication is enabled, all API requests must include a valid token:

```bash
# Using curl
curl -H "X-API-Token: your-secret-token-1" http://localhost:19751/api/v1/issues

# Using fetch in JavaScript
fetch('http://localhost:19751/api/v1/issues', {
  headers: {
    'X-API-Token': 'your-secret-token-1'
  }
})
```

### 3. Rate Limiting

Configure rate limiting to prevent abuse:

```bash
# Default: 100 requests per time window
# Set custom rate limit (requests per window)
RATE_LIMIT=50
```

## Production Deployment Checklist

For production deployments, **always** enable these security features:

- [ ] Set `ENABLE_AUTH=true`
- [ ] Configure strong API tokens in `API_TOKENS`
- [ ] Limit `ALLOWED_ORIGINS` to your actual domains
- [ ] Set appropriate `RATE_LIMIT` based on your usage patterns
- [ ] Use HTTPS (not HTTP) in production
- [ ] Never commit tokens to version control
- [ ] Rotate API tokens regularly

## Example Production Configuration

Create a `.env` file (never commit this):

```bash
# Production Security Configuration
ENABLE_AUTH=true
API_TOKENS="prod-token-abc123def456,prod-token-backup-xyz789"
ALLOWED_ORIGINS="https://yourapp.com,https://dashboard.yourapp.com"
RATE_LIMIT=100

# API Configuration
API_PORT=19751
ISSUES_DIR=/var/app/data/issues

# GitHub Integration (optional)
GITHUB_TOKEN=ghp_your_github_token_here
GITHUB_OWNER=your-organization
GITHUB_REPO=your-repository
```

## GitHub Integration Setup

To enable automatic PR creation for fixes:

1. **Create a GitHub Personal Access Token**:
   - Go to GitHub Settings → Developer Settings → Personal Access Tokens → Tokens (classic)
   - Click "Generate new token (classic)"
   - Select scopes: `repo` (full control of private repositories)
   - Copy the token (you won't see it again)

2. **Configure Environment Variables**:

```bash
# Required for PR creation
GITHUB_TOKEN=ghp_your_token_here
GITHUB_OWNER=your-username-or-org
GITHUB_REPO=your-repository-name
```

3. **Test PR Creation**:

```bash
# First, investigate an issue
curl -X POST http://localhost:19751/api/v1/investigate \
  -H "Content-Type: application/json" \
  -d '{"issue_id": "issue-123", "agent_id": "unified-resolver", "auto_resolve": true}'

# Then create a PR with the fixes
curl -X POST http://localhost:19751/api/v1/issues/issue-123/create-pr \
  -H "X-API-Token: your-token"
```

## Input Validation

The system automatically validates and sanitizes:

- **Titles**: Max 200 characters, alphanumeric with common punctuation
- **Descriptions**: Max 5000 characters
- **File Paths**: Protected against directory traversal attacks
- **Issue Types**: Limited to: bug, feature, task, improvement
- **Priorities**: Limited to: critical, high, medium, low
- **Statuses**: Limited to: open, active, completed, failed

No additional configuration required - validation is always active.

## Monitoring Security

Check security status in the logs:

```bash
# View startup logs to verify security configuration
vrooli scenario logs app-issue-tracker --step start-api | grep -E "CORS|Authentication"

# Expected output:
# CORS allowed origins: [https://yourapp.com]
# ⚠️  Authentication is ENABLED - API tokens required
```

## Troubleshooting

### "Forbidden" or "Unauthorized" Errors

- Verify `ENABLE_AUTH` matches your environment
- Check that `X-API-Token` header is included in requests
- Ensure token matches one in `API_TOKENS`

### CORS Errors in Browser

- Add your frontend domain to `ALLOWED_ORIGINS`
- Use full URL with protocol (e.g., `https://app.example.com`, not just `app.example.com`)
- Verify CORS headers in response: `curl -I -X OPTIONS http://localhost:19751/api/v1/issues`

### GitHub PR Creation Fails

- Verify `GITHUB_TOKEN` has `repo` scope
- Check that `GITHUB_OWNER` and `GITHUB_REPO` are correct
- Ensure the token hasn't expired
- Verify the repository exists and token has access

## Security Best Practices

1. **Never hardcode tokens** - Use environment variables
2. **Use HTTPS in production** - Never send tokens over HTTP
3. **Rotate tokens regularly** - Change API tokens every 90 days
4. **Monitor access logs** - Watch for suspicious patterns
5. **Limit CORS strictly** - Only allow domains you control
6. **Use strong tokens** - Generate random tokens at least 32 characters long
7. **Separate dev/prod configs** - Never use production tokens in development

## Generating Strong Tokens

```bash
# Generate a secure random token (Linux/macOS)
openssl rand -base64 32

# Or use Node.js
node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"
```

---

**Note**: For local development, the default open configuration is acceptable. Always enable security features before deploying to production or exposing the API to the internet.
