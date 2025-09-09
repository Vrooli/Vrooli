# Security Requirements

## Essential Security Checklist

### Secure Coding Requirements
- [ ] **No hardcoded secrets** - All secrets from environment variables only
- [ ] **No hardcoded ports** - Use environment variables for all port configurations  
- [ ] **No port fallbacks** - Fail explicitly if required ports are not provided
- [ ] **Dependencies audited** - Check for known vulnerabilities before use
- [ ] **Input validation** - Validate all user input for type, length, and format
- [ ] **Error messages sanitized** - No sensitive data in error responses

### Infrastructure Security  
- [ ] **Minimal port exposure** - Only expose ports that are actually needed
- [ ] **Health checks secured** - No sensitive information in health endpoints
- [ ] **Proper file permissions** - Restrict read/write access appropriately
- [ ] **Container security** - Run as non-root user when possible

### UI Security (Scenarios Only)
- [ ] **Use proxyToApi function** - Never direct HTTP calls from UI to API
- [ ] **HTTPS compatibility** - Ensure UI works with mixed content policies
- [ ] **CORS handled by proxy** - Let the proxy handle cross-origin requests

## Common Security Mistakes

### Critical Mistakes to Avoid
❌ **Never put secrets in code or logs**  
✅ **Use environment variables for all secrets**

❌ **Never hardcode ports or use fallbacks**
✅ **Fail explicitly when ports are missing from environment**

❌ **Never trust any user input**
✅ **Validate everything - type, length, format, content**

❌ **Never use string concatenation for SQL**
✅ **Use parameterized queries or prepared statements**

❌ **Never make direct API calls from scenario UIs**
✅ **Always use proxyToApi function for API communication**

❌ **Never expose sensitive data in error messages**
✅ **Return generic error messages, log details internally**

### Quick Security Priority Guide
**P0 (Must Fix)**: Hardcoded secrets, SQL injection, command injection
**P1 (High Priority)**: Missing input validation, information disclosure  
**P2 (Medium Priority)**: Outdated dependencies, verbose errors

## Security Notes
- Authentication/authorization handled by scenario-authenticator
- API security monitoring handled by api-manager  
- Focus on secure coding practices and proper port/proxy configuration