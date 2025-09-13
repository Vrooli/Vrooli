# ERPNext Resource - Known Problems and Solutions

## 1. Web Interface Routing Issue

### Problem
When accessing ERPNext at `http://localhost:8020`, users see "localhost does not exist" error. This is because ERPNext uses a multi-tenant architecture where each site expects requests with its configured hostname.

### Root Cause
The ERPNext/Frappe framework is designed for multi-tenant deployments where each site has its own domain. The containerized setup expects the Host header to match the site name (vrooli.local).

### Solutions

#### Option 1: Modify /etc/hosts (Recommended)
```bash
# Add this line to /etc/hosts
sudo echo "127.0.0.1 vrooli.local" >> /etc/hosts

# Then access ERPNext at:
http://vrooli.local:8020
```

#### Option 2: Use curl with Host header
```bash
# For API access
curl -H "Host: vrooli.local" http://localhost:8020/api/method/login \
  -X POST -H "Content-Type: application/json" \
  -d '{"usr":"Administrator","pwd":"admin"}'
```

#### Option 3: Configure serve_default_site
Already attempted - partially works but still requires proper routing for full functionality.

## 2. API Authentication Timeout

### Problem
API authentication requests sometimes timeout or hang indefinitely.

### Root Cause
The Frappe framework's authentication system expects specific headers and session management that may not be properly configured in the containerized environment.

### Current Workaround
- Use the provided API helper functions in `lib/api.sh`
- Always include Host header in requests
- Use timeout wrapper for all API calls

## 3. Integration Test Failures

### Problem
2 integration tests fail:
- API endpoint availability
- Authentication endpoint

### Root Cause
Tests don't include proper Host header required for multi-tenant routing.

### Solution
Tests have been updated to include Host header, but some endpoints still require additional configuration.

## 4. Site Initialization

### Problem
Site initialization sometimes fails or requires manual intervention.

### Root Cause
- Database readiness timing issues
- MariaDB connection parameters need specific host scope settings
- Site creation command requires proper database host configuration

### Solution Implemented
- Added database readiness check with retry logic
- Proper MariaDB host scope configuration in site creation
- Automatic site initialization on first start

## 5. Docker Compose Port Mapping

### Problem
Internal port 8000 mapped to external 8020, but internal routing still expects 8000.

### Root Cause
Mismatch between container's internal port and external mapping can cause routing issues.

### Current State
Working with port mapping, but requires Host header for proper routing.

## Future Improvements

### 1. Nginx Reverse Proxy
Implement a proper nginx reverse proxy that:
- Handles Host header injection automatically
- Routes localhost requests to the correct site
- Manages SSL termination if needed

### 2. Single-Site Mode
Configure ERPNext to run in single-site mode for simpler local development:
- Disable multi-tenant checks
- Serve default site on all requests
- Simplify authentication flow

### 3. Browser Extension
Create a simple browser extension that:
- Automatically adds Host header to requests
- Enables direct localhost access
- Handles cookie management for sessions

## Testing Notes

### What Works
- ✅ Health checks (with Host header)
- ✅ Site creation and initialization
- ✅ Database connectivity (PostgreSQL external, MariaDB internal)
- ✅ Redis integration
- ✅ Content management CLI commands
- ✅ API structure (with proper headers)

### What Needs Work
- ⚠️ Direct browser access without hosts file modification
- ⚠️ Full API authentication flow testing
- ⚠️ Web UI module loading
- ⚠️ Workflow engine implementation
- ⚠️ Report generation

## Recommended Development Approach

1. **For API Development**: Use the provided `lib/api.sh` helper functions
2. **For Web Access**: Modify /etc/hosts as described above
3. **For Testing**: Always include Host header in HTTP requests
4. **For Production**: Implement proper domain routing with nginx

## References

- [Frappe Multi-Tenancy Documentation](https://frappeframework.com/docs/user/en/basics/sites)
- [ERPNext Docker Setup Guide](https://github.com/frappe/frappe_docker)
- [Nginx Proxy Configuration for Frappe](https://discuss.erpnext.com/t/nginx-configuration/)