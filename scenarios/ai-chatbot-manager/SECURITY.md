# Security Considerations

## CORS Configuration

### Embeddable Widget Design

The AI Chatbot Manager is designed to provide **embeddable chatbot widgets** that can be integrated into any website. This design requires CORS wildcard (`*`) configuration by default.

**Why CORS Wildcard is Necessary:**
- Widgets must load and function on any customer website
- Customers embed the widget via `<script>` tags on their domains
- WebSocket connections from arbitrary domains need to work
- This is a standard pattern for embeddable widgets (similar to Intercom, Drift, etc.)

### Security Measures in Place

Even with CORS wildcard, the system remains secure through:

1. **Rate Limiting**: All API endpoints have rate limiting to prevent abuse
2. **WebSocket Authentication**: Each chatbot has unique IDs that must be known to connect
3. **Input Validation**: All user inputs are validated and sanitized
4. **SQL Injection Prevention**: Using parameterized queries throughout
5. **No Sensitive Data Exposure**: CORS endpoints only serve public chatbot data

### Production Deployment Options

For production deployments with known customer domains, you can restrict CORS:

```bash
# Restrict to specific domains
export CORS_ALLOWED_ORIGINS="https://example.com,https://app.example.com"
```

**Note:** Using restricted CORS means the widget will **only work** on the specified domains.

### Audit Findings

**Scenario-Auditor Reports:**
- 2 CORS wildcard findings (middleware.go:61, widget.go:25)
- Severity: High (by default scanner settings)
- Status: **Accepted Risk** - Required for widget functionality
- Mitigation: Rate limiting, authentication, input validation in place

### Recommendations

**Development/Testing:**
- Use wildcard CORS (`CORS_ALLOWED_ORIGINS=*`) for flexibility

**Production - Public Widget:**
- Keep wildcard if widget needs to work on any domain
- Ensure rate limiting is properly configured
- Monitor for abuse patterns

**Production - Known Customers:**
- Set specific allowed origins via `CORS_ALLOWED_ORIGINS`
- Update configuration as new customers are added
- This provides maximum security at cost of flexibility

## Additional Security Measures

### Database Security
- PostgreSQL connections use credentials (not exposed in CORS)
- Connection pooling with limits to prevent resource exhaustion

### API Security
- Health checks don't expose sensitive information
- Chatbot configurations are only accessible via API with proper IDs
- No authentication tokens or secrets exposed in widget code

### Monitoring
- Rate limiting logs help identify potential abuse
- Health checks monitor system status
- All errors logged for security review

## Reporting Security Issues

If you discover a security vulnerability, please contact the Vrooli security team immediately.
