# Security Requirements & Checklists

## Mandatory Security Gates

Every resource and scenario MUST implement these security requirements. No exceptions.

### Phase 1: Design Security (Generators)
All new resources/scenarios must include:

#### Authentication & Authorization
- [ ] **Authentication method defined** - JWT, OAuth, API keys, or none (with justification)
- [ ] **Authorization model specified** - RBAC, permissions, or public (with rationale)  
- [ ] **Session management planned** - Token expiry, refresh, revocation
- [ ] **Default deny policy** - Explicit permissions required for access

#### Input Validation & Sanitization  
- [ ] **Input validation rules defined** - Schema validation, type checking
- [ ] **SQL injection prevention** - Parameterized queries, ORM usage
- [ ] **XSS prevention planned** - Output encoding, CSP headers
- [ ] **Command injection prevention** - Input sanitization, restricted commands

#### Data Protection
- [ ] **Sensitive data identified** - Passwords, tokens, PII, business data
- [ ] **Encryption at rest planned** - Database, file storage encryption
- [ ] **Encryption in transit required** - HTTPS, TLS for all communications  
- [ ] **Secret management designed** - Environment variables, vault integration

#### Error Handling & Logging
- [ ] **Error messages sanitized** - No sensitive data in error responses
- [ ] **Security logging planned** - Authentication failures, access attempts
- [ ] **Rate limiting designed** - Prevent brute force, DDoS protection
- [ ] **Audit trail specified** - User actions, admin operations tracked

### Phase 2: Implementation Security (All Operations)

#### Secure Coding Checklist
- [ ] **No hardcoded secrets** - All secrets from environment/vault
- [ ] **Secure defaults configured** - Fail-safe configurations
- [ ] **Dependencies audited** - Known vulnerabilities checked
- [ ] **Least privilege applied** - Minimum required permissions

#### API Security
- [ ] **HTTPS enforced** - No plain HTTP in production
- [ ] **CORS configured properly** - Restrict origins appropriately  
- [ ] **Request size limits** - Prevent resource exhaustion
- [ ] **Content-Type validation** - Reject unexpected formats

#### Infrastructure Security  
- [ ] **Ports minimized** - Only required ports exposed
- [ ] **Health checks secured** - No sensitive data in health endpoints
- [ ] **File permissions set** - Proper read/write restrictions
- [ ] **Container security** - Non-root user, minimal base image

### Phase 3: Validation Security (All Operations)

#### Security Testing
- [ ] **Authentication tested** - Valid/invalid credentials
- [ ] **Authorization tested** - Access control boundaries
- [ ] **Input validation tested** - Malicious input handling
- [ ] **Error handling tested** - Information disclosure prevention

#### Vulnerability Assessment
- [ ] **Dependency scan clean** - No high/critical vulnerabilities
- [ ] **Static analysis passed** - Code security issues addressed
- [ ] **Configuration review** - Secure settings verified
- [ ] **Secrets audit complete** - No exposed credentials

## Security Implementation Patterns

### Authentication Pattern
```bash
# For resources requiring authentication:
# 1. Environment variable for secret
export JWT_SECRET_KEY="$(generate_secure_random_key)"

# 2. Token validation middleware
validate_token() {
    local token="$1"
    jwt_verify "$token" "$JWT_SECRET_KEY" || {
        echo "HTTP/1.1 401 Unauthorized"
        exit 1
    }
}

# 3. Protected endpoint wrapper
protected_endpoint() {
    local auth_header="${HTTP_AUTHORIZATION:-}"
    local token="${auth_header#Bearer }"
    validate_token "$token" && handle_request
}
```

### Input Validation Pattern
```bash
# Validate all inputs before processing
validate_input() {
    local input="$1"
    local max_length="${2:-1000}"
    
    # Check length
    [ ${#input} -le $max_length ] || return 1
    
    # Check for injection attempts
    echo "$input" | grep -qE '[;&|`$<>]' && return 1
    
    # Check for path traversal
    echo "$input" | grep -qE '\.\./|\.\.\\' && return 1
    
    return 0
}
```

### Secure Error Handling Pattern
```bash
# Never expose sensitive information in errors
secure_error() {
    local internal_error="$1"
    local user_error="${2:-An error occurred}"
    
    # Log detailed error internally
    logger "SECURITY: $internal_error" 
    
    # Return generic error to user
    echo "HTTP/1.1 400 Bad Request"
    echo "Content-Type: application/json"
    echo
    echo "{\"error\": \"$user_error\"}"
}
```

### Rate Limiting Pattern
```bash
# Simple rate limiting implementation
rate_limit() {
    local client_ip="$1"
    local limit="${2:-60}"  # requests per minute
    local window_file="/tmp/rate_limit_${client_ip//[^a-zA-Z0-9]/_}"
    
    # Clean old entries
    find /tmp -name "rate_limit_*" -mmin +1 -delete 2>/dev/null
    
    # Count current requests
    local count=$(ls ${window_file}* 2>/dev/null | wc -l)
    
    if [ $count -ge $limit ]; then
        echo "HTTP/1.1 429 Too Many Requests"
        exit 1
    fi
    
    # Record this request
    touch "${window_file}_$(date +%s)"
}
```

## Security Validation Commands

### Test Authentication
```bash
# Test invalid token rejection
curl -H "Authorization: Bearer invalid_token" http://localhost:$PORT/api/protected
# Expected: 401 Unauthorized

# Test missing token rejection  
curl http://localhost:$PORT/api/protected
# Expected: 401 Unauthorized

# Test valid token acceptance
curl -H "Authorization: Bearer $VALID_TOKEN" http://localhost:$PORT/api/protected  
# Expected: 200 OK with data
```

### Test Input Validation
```bash
# Test SQL injection prevention
curl -X POST http://localhost:$PORT/api/data -d "'; DROP TABLE users; --"
# Expected: 400 Bad Request, no data corruption

# Test XSS prevention
curl -X POST http://localhost:$PORT/api/data -d "<script>alert('xss')</script>"
# Expected: Input rejected or safely encoded

# Test command injection prevention
curl -X POST http://localhost:$PORT/api/exec -d "rm -rf /"
# Expected: 400 Bad Request, command not executed
```

### Test Rate Limiting
```bash
# Test rate limiting enforcement
for i in {1..100}; do
    curl http://localhost:$PORT/api/data &
done
# Expected: Some requests return 429 Too Many Requests
```

### Test HTTPS Enforcement
```bash
# Test HTTP redirect to HTTPS
curl -I http://localhost:$PORT/
# Expected: 301/302 redirect to https:// or connection refused

# Test HTTPS availability  
curl -I https://localhost:$HTTPS_PORT/
# Expected: 200 OK with HTTPS
```

## Common Security Mistakes

### Authentication Mistakes
❌ **Never store passwords in plain text**
✅ **Use bcrypt, scrypt, or argon2 for password hashing**

❌ **Never put secrets in code or logs**  
✅ **Use environment variables or secure vaults**

❌ **Never trust client-side validation**
✅ **Always validate server-side**

### Authorization Mistakes  
❌ **Never rely on client-side access control**
✅ **Enforce permissions server-side**

❌ **Never use predictable session IDs**
✅ **Use cryptographically secure random tokens**

### Input Validation Mistakes
❌ **Never trust any user input**
✅ **Validate everything - type, length, format, content**

❌ **Never use string concatenation for SQL**
✅ **Use parameterized queries or ORM**

## Security Priority Matrix

### P0 (Critical) - Must Fix Before Launch
- Authentication bypass vulnerabilities
- SQL/Command injection possibilities  
- Hardcoded secrets or passwords
- Unencrypted sensitive data transmission

### P1 (High) - Fix Within Week
- Missing input validation  
- Information disclosure in errors
- Missing rate limiting
- Weak session management

### P2 (Medium) - Fix Within Month  
- Missing security headers
- Verbose error messages
- Insufficient logging
- Outdated dependencies

Security is not optional. Every requirement must pass security validation before implementation is considered complete.