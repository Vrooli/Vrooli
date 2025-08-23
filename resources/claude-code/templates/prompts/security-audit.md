# Security Audit Template

Comprehensive security analysis focusing on vulnerabilities and attack vectors.

## Variables
This template supports the following variables:
- `{target}` - Target files/directories to audit (e.g., "src/auth/", "**/*.js")
- `{scope}` - Audit scope (e.g., "web application", "API endpoints", "authentication system")
- `{threat_model}` - Specific threats to focus on (e.g., "OWASP Top 10", "data breaches")

## Template Content

You are a cybersecurity expert performing a comprehensive security audit.

**Audit target:** {target}
**Scope:** {scope}
**Threat model:** {threat_model}

Perform a thorough security analysis of the specified code, examining all potential vulnerabilities and attack vectors.

## 🛡️ OWASP Top 10 Analysis

### 1. Injection Vulnerabilities
- **SQL Injection:** Check database queries for parameter binding
- **NoSQL Injection:** Review NoSQL query construction
- **Command Injection:** Examine system command execution
- **LDAP Injection:** Analyze LDAP query construction
- **Code Injection:** Look for dynamic code execution

### 2. Broken Authentication
- **Password handling:** Check hashing, storage, and validation
- **Session management:** Review session creation, storage, and expiration
- **Multi-factor authentication:** Verify MFA implementation
- **Account lockout:** Check brute force protection
- **Password reset:** Analyze reset mechanisms

### 3. Sensitive Data Exposure
- **Data at rest:** Check encryption of stored data
- **Data in transit:** Verify HTTPS/TLS usage
- **Logging:** Ensure sensitive data isn't logged
- **Error messages:** Check for information disclosure
- **API responses:** Verify no sensitive data leakage

### 4. XML External Entities (XXE)
- **XML parsing:** Check for secure XML parser configuration
- **File uploads:** Review XML file handling
- **External entity processing:** Verify XXE prevention

### 5. Broken Access Control
- **Authorization checks:** Verify proper access controls
- **Direct object references:** Check for IDOR vulnerabilities
- **Privilege escalation:** Look for elevation opportunities
- **CORS configuration:** Review cross-origin policies

### 6. Security Misconfiguration
- **Default credentials:** Check for hardcoded or default passwords
- **Debug information:** Verify debug mode is disabled in production
- **HTTP headers:** Review security headers implementation
- **Server configuration:** Check for secure defaults

### 7. Cross-Site Scripting (XSS)
- **Input validation:** Check user input sanitization
- **Output encoding:** Verify proper data encoding
- **Content Security Policy:** Review CSP implementation
- **DOM manipulation:** Check client-side code safety

### 8. Insecure Deserialization
- **Object deserialization:** Check for unsafe deserialization
- **Input validation:** Verify serialized data validation
- **Type safety:** Review deserialization type checking

### 9. Using Components with Known Vulnerabilities
- **Dependency analysis:** Check for outdated dependencies
- **Vulnerability scanning:** Identify known CVEs
- **Update policy:** Review dependency update practices

### 10. Insufficient Logging & Monitoring
- **Security events:** Check logging of security-relevant events
- **Log integrity:** Verify log protection mechanisms
- **Monitoring:** Review security monitoring implementation
- **Incident response:** Check alerting and response procedures

## 🔐 Additional Security Areas

### Cryptography
- **Algorithm choices:** Verify use of strong crypto algorithms
- **Key management:** Check key generation, storage, and rotation
- **Random number generation:** Verify use of cryptographically secure RNG
- **Digital signatures:** Review signature implementation

### Input Validation
- **Boundary checking:** Verify input length and format validation
- **Type validation:** Check data type enforcement
- **Whitelist validation:** Prefer whitelisting over blacklisting
- **File upload validation:** Check file type and content validation

### Error Handling
- **Information disclosure:** Ensure errors don't reveal system details
- **Error logging:** Verify comprehensive error logging
- **Graceful degradation:** Check system behavior under error conditions

### Business Logic Security
- **Workflow validation:** Check business process security
- **Race conditions:** Look for timing-based vulnerabilities
- **State management:** Verify secure state transitions
- **Rate limiting:** Check for abuse prevention

## 📊 Risk Assessment

For each vulnerability found, provide:

### Risk Rating
- **Critical:** Immediate exploitation possible, severe impact
- **High:** Likely exploitation, significant impact
- **Medium:** Possible exploitation, moderate impact
- **Low:** Difficult exploitation, minimal impact

### Vulnerability Details
- **Location:** Specific file and line number
- **Description:** Clear explanation of the vulnerability
- **Attack vector:** How an attacker could exploit this
- **Impact:** What damage could result
- **Likelihood:** Probability of exploitation
- **Remediation:** Specific steps to fix the issue
- **Prevention:** How to prevent similar issues

## 🛠️ Remediation Priorities

### Immediate Actions (Critical/High Risk)
List vulnerabilities requiring immediate attention

### Short-term Fixes (Medium Risk)
Issues to address in the next development cycle

### Long-term Improvements (Low Risk)
Security enhancements for ongoing improvement

## 📋 Security Checklist

Provide a summary checklist:
- [ ] Authentication mechanisms secure
- [ ] Authorization properly implemented
- [ ] Input validation comprehensive
- [ ] Output encoding consistent
- [ ] Sensitive data protected
- [ ] Error handling secure
- [ ] Logging adequate
- [ ] Dependencies up to date
- [ ] Security headers configured
- [ ] Cryptography properly implemented

## 🔍 Recommendations

### Code-Level Fixes
Specific code changes needed to address vulnerabilities

### Architectural Improvements
System-level security enhancements

### Process Improvements
Development and deployment process security enhancements

### Monitoring and Detection
Security monitoring and incident response improvements

Focus on providing actionable, specific recommendations that developers can implement immediately to improve the security posture of the application.