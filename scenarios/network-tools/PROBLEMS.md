# Network Tools - Known Issues and Solutions

## Issues Resolved (2025-09-28)

### 1. SSL Validation Connection Failures
**Problem**: SSL validation endpoint failed with "missing port in address" error when validating URLs without explicit ports.

**Root Cause**: The TLS dial function requires an address with port, but URLs like "https://google.com" don't include port in the Host field.

**Solution**: Added logic to construct proper address with port:
```go
// Build address with port if missing
address := parsedURL.Host
if !strings.Contains(address, ":") {
    address = fmt.Sprintf("%s:%s", parsedURL.Hostname(), port)
}
```

**Status**: ✅ FIXED

---

### 2. Invalid URL Handling
**Problem**: Invalid URLs caused HTTP handler to hang indefinitely instead of returning proper error.

**Root Cause**: Missing URL validation before attempting HTTP request.

**Solution**: Added comprehensive URL validation:
```go
// Validate URL format
parsedURL, err := url.Parse(req.URL)
if err != nil {
    sendError(w, fmt.Sprintf("Invalid URL format: %v", err), http.StatusBadRequest)
    return
}

// Ensure URL has a scheme
if parsedURL.Scheme == "" {
    sendError(w, "URL must include a scheme (http:// or https://)", http.StatusBadRequest)
    return
}

// Ensure URL has a host
if parsedURL.Host == "" {
    sendError(w, "URL must include a host", http.StatusBadRequest)
    return
}
```

**Status**: ✅ FIXED

---

## Known Limitations

### 1. ICMP Ping Requires Root Privileges
**Limitation**: True ICMP ping requires raw socket access which needs root privileges.

**Workaround**: Using TCP connectivity checks as alternative for non-root users.

**Future Enhancement**: Implement capability-based privilege escalation or use setuid binary for ICMP operations.

---

### 2. Bandwidth Testing Not Implemented
**Limitation**: P0 requirement for bandwidth testing is pending implementation.

**Workaround**: Use external tools or defer to P1 priority.

**Future Enhancement**: Implement using iperf3 library or custom bandwidth measurement logic.

---

### 3. DNS Zone Transfers Not Supported
**Limitation**: AXFR/IXFR zone transfers not implemented due to complexity and security restrictions.

**Workaround**: Use standard DNS queries for individual records.

**Future Enhancement**: Add zone transfer support with proper authentication.

---

### 4. Service Banner Grabbing Limited
**Limitation**: Port scanning doesn't include full banner grabbing for service fingerprinting.

**Workaround**: Basic service detection works based on port numbers and simple probes.

**Future Enhancement**: Implement nmap-style service probes and banner analysis.

---

## Performance Considerations

### Database Connection Pool
**Issue**: Database connections not pooled efficiently.

**Impact**: May affect performance under high load.

**Recommendation**: Configure connection pool settings:
```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

---

## Security Notes

### API Key Storage
**Best Practice**: Never store API keys in code or configuration files.

**Recommendation**: Use environment variables or secret management systems:
```bash
export NETWORK_TOOLS_API_KEY="$(vault read -field=key secret/network-tools)"
```

### Rate Limiting
**Current**: 100 requests/minute per IP

**Customization**: Adjust via environment variables:
```bash
export RATE_LIMIT=200
export RATE_WINDOW=60s
```

---

## Testing Issues

### CLI Installation Path
**Problem**: CLI install script references non-existent template path.

**Workaround**: Manual installation or update install script path.

**Fix**: Update cli/install.sh to use correct installation method.

---

## Deployment Notes

### Port Configuration
**Default Port**: 15000 (configurable via API_PORT or PORT environment variable)

**Dynamic Port Assignment**: Vrooli lifecycle may assign different ports - check logs for actual port.

### Resource Dependencies
**Required**: PostgreSQL must be running before scenario start

**Check**: `vrooli resource status postgres`

**Start if needed**: `vrooli resource start postgres`

---

## Troubleshooting Commands

```bash
# Check scenario status
make status

# View logs
make logs

# Check API health
curl http://localhost:$(vrooli scenario status network-tools --json | jq -r '.port')/api/health

# Run tests
make test

# Clean and restart
make stop && make clean && make run
```

---

## Contact

For unresolved issues or feature requests, please:
1. Check this document for known issues
2. Review API documentation in docs/api.md
3. Check scenario logs: `make logs`
4. Create issue in project repository with reproduction steps