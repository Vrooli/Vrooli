# Scenario Authenticator

Universal authentication service that enables any Vrooli scenario to secure its endpoints with user management, JWT tokens, and role-based access control.

## 🎯 Purpose

This scenario provides **foundational authentication capabilities** that transform any Vrooli scenario from a demo into a production-ready, secured application. It's designed to be the permanent intelligence that makes all future scenarios more capable.

## ✨ Key Features

- **JWT Authentication**: Stateless token-based authentication
- **User Management**: Complete CRUD operations for user accounts
- **Session Management**: Redis-backed session storage
- **Role-Based Access Control**: Flexible permission system
- **Password Reset**: Secure email-based password recovery
- **API Security**: Token validation for any scenario's API
- **UI Integration**: Ready-to-use authentication pages
- **CLI Management**: Command-line tools for user administration

## 🚀 Quick Start

### 1. Setup and Run

```bash
# Setup the authentication service (REQUIRED - do not run directly)
vrooli scenario run scenario-authenticator

# Alternative using Makefile
cd scenarios/scenario-authenticator && make run

# ⚠️  IMPORTANT: Always use the lifecycle system, never run ./api/scenario-authenticator-api directly

# Access points will be displayed after startup
# - Authentication API: Dynamic port (check logs or use CLI)
# - Authentication UI: Dynamic port (check logs or use CLI)  
# - CLI: scenario-authenticator --help
```

### 2. Create Your First User

```bash
# Via CLI
scenario-authenticator user create admin@example.com SecurePass123! admin
scenario-authenticator user get <user-id>

# Admin actions require AUTH_TOKEN exported from an admin login
export AUTH_TOKEN=$(curl -s -X POST \
  http://localhost:$(vrooli scenario port scenario-authenticator API_PORT)/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"SecurePass123!"}' | jq -r '.token')

scenario-authenticator user update <user-id> roles admin,moderator
scenario-authenticator user delete <user-id>

# Via API (replace port with actual API port)
curl -X POST http://localhost:$(vrooli scenario port scenario-authenticator API_PORT)/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"SecurePass123!"}'

# Via UI (check logs for actual UI port)
# Visit the UI URL displayed in the startup logs and click "Sign Up"
```

### 3. Validate Tokens

```bash
# Get your token (from registration response or login)
export AUTH_TOKEN="eyJhbGciOiJSUzI1NiIs..."

# Validate via CLI
scenario-authenticator token validate $AUTH_TOKEN

# Validate via API (using dynamic port discovery)
curl -H "Authorization: Bearer $AUTH_TOKEN" \
  http://localhost:$(vrooli scenario port scenario-authenticator API_PORT)/api/v1/auth/validate
```

## 🔧 Integration Guide

### Protecting Your Scenario's API

**Option 1: Direct API Integration (JavaScript/Node.js)**

```javascript
// Add to your API routes
const validateAuth = async (req, res, next) => {
    const token = req.headers.authorization?.replace('Bearer ', '');
    if (!token) {
        return res.status(401).json({ error: 'No token provided' });
    }
    
    try {
        // Get auth API port dynamically
        const authApiPort = process.env.API_PORT;
        const authApiUrl = `http://localhost:${authApiPort}`;

        const response = await fetch(`${authApiUrl}/api/v1/auth/validate`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        
        const validation = await response.json();
        if (validation.valid) {
            req.user = {
                id: validation.user_id,
                email: validation.email,
                roles: validation.roles
            };
            next();
        } else {
            res.status(401).json({ error: 'Invalid token' });
        }
    } catch (error) {
        res.status(500).json({ error: 'Auth service unavailable' });
    }
};

// Protect your routes
router.use('/api/protected', validateAuth);
```

**Option 2: Go API Integration**

```go
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "No token provided", 401)
            return
        }
        
        token = strings.Replace(token, "Bearer ", "", 1)

        // Get auth API port from environment
        authPort := os.Getenv("API_PORT")
        authURL := fmt.Sprintf("http://localhost:%s/api/v1/auth/validate?token=%s", authPort, token)
        
        resp, err := http.Get(authURL)
        if err != nil || resp.StatusCode != 200 {
            http.Error(w, "Invalid token", 401)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

// Use: router.Use(authMiddleware)
```

### Protecting Your Scenario's UI

**Option 1: JavaScript Client-Side**

```javascript
// Add to your UI initialization
async function checkAuth() {
    const token = localStorage.getItem('auth_token');
    if (!token) {
        // Get auth UI port dynamically from environment or configuration
        const authUiPort = window.AUTH_UI_PORT || getAuthUIPortFromConfig();
        window.location.href = `http://localhost:${authUiPort}/login?redirect=` + 
                              encodeURIComponent(window.location.href);
        return false;
    }
    
    try {
        // Get auth API port dynamically from environment or configuration
        const authApiPort = window.API_PORT || getAuthAPIPortFromConfig();
        const authApiUrl = `http://localhost:${authApiPort}`;

        const response = await fetch(`${authApiUrl}/api/v1/auth/validate`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });

        const validation = await response.json();
        if (!validation.valid) {
            localStorage.removeItem('auth_token');
            const authUiPort = window.UI_PORT || getAuthUIPortFromConfig();
            window.location.href = `http://localhost:${authUiPort}/login?redirect=` + 
                                  encodeURIComponent(window.location.href);
            return false;
        }
        
        window.currentUser = validation;
        return true;
    } catch (error) {
        console.error('Auth check failed:', error);
        return false;
    }
}

// Call on page load
document.addEventListener('DOMContentLoaded', checkAuth);
```

**Option 2: Using the VrooliAuth Library**

```javascript
// The auth UI exposes a global VrooliAuth object
const user = VrooliAuth.getUser();
const token = VrooliAuth.getToken();
const isValid = await VrooliAuth.validateToken(token);

// Logout
await VrooliAuth.logout();
```

### CLI Integration Examples

```bash
# Generate integration code
scenario-authenticator protect my-scenario api
scenario-authenticator protect my-scenario ui

# Manage users
scenario-authenticator user list --role admin
scenario-authenticator user create user@example.com Pass123!
scenario-authenticator user update user123 roles "user,moderator"

# Monitor sessions
scenario-authenticator session list --user-id user123
scenario-authenticator session revoke session456

# Check service health
scenario-authenticator status --verbose
```

## 🛡️ Security Features

- **BCrypt Password Hashing**: Industry-standard password protection
- **Password Complexity Requirements**: Enforced minimum length, uppercase, lowercase, and numbers
- **Email Validation**: RFC-compliant email format validation
- **JWT with RSA Signing**: Secure, stateless authentication with configurable expiry
- **Token Blacklisting**: Immediate token revocation via Redis
- **Session Management**: Secure session tracking and cleanup
- **CORS Protection**: Configurable origin whitelisting (defaults to localhost for development)
- **Rate Limiting**: Per-user API rate limiting
- **Audit Logging**: Complete authentication event tracking
- **Password Reset Security**: Time-limited, single-use reset tokens
- **Role-Based Permissions**: Granular access control
- **Two-Factor Authentication (2FA)**: TOTP-based 2FA with backup codes
- **OAuth2 Integration**: Social login with Google and GitHub
- **Security Headers**: Comprehensive HTTP security headers (X-Frame-Options, CSP, X-XSS-Protection, etc.)
- **Request Size Limiting**: DoS protection via configurable request body size limits (default: 1MB)

## 📋 API Reference

### Authentication Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/auth/register` | POST | Create new user account |
| `/api/v1/auth/login` | POST | Authenticate user |
| `/api/v1/auth/validate` | GET | Validate JWT token |
| `/api/v1/auth/refresh` | POST | Refresh expired token |
| `/api/v1/auth/logout` | POST | Invalidate session |
| `/api/v1/auth/reset-password` | POST | Initiate password reset |
| `/api/v1/auth/oauth/providers` | GET | List available OAuth providers |
| `/api/v1/auth/oauth/login?provider=google` | GET | Initiate OAuth2 login |
| `/api/v1/auth/oauth/google/callback` | GET | OAuth2 callback (Google) |
| `/api/v1/auth/oauth/github/callback` | GET | OAuth2 callback (GitHub) |

### User Management Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/users` | GET | List users (paginated) |
| `/api/v1/users/{id}` | GET | Get user details |
| `/api/v1/users/{id}` | PUT | Update user |
| `/api/v1/users/{id}` | DELETE | Delete user (soft delete) |

### Session Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/sessions` | GET | List active sessions |
| `/api/v1/sessions/{id}` | DELETE | Revoke session |

## 🔄 API Endpoints Available

All authentication workflows are implemented as direct API endpoints:

1. **Token Validation**: `/api/v1/auth/validate` - Validate JWT tokens for any scenario
2. **Password Reset**: `/api/v1/auth/reset-password` - Complete password reset flow with email  
3. **Token Refresh**: `/api/v1/auth/refresh` - Automatic token refresh handling
4. **User Registration**: `/api/v1/auth/register` - New user account creation

## 🧪 Testing

```bash
# Run all tests
vrooli scenario test scenario-authenticator

# Test specific components
scenario-authenticator status --json

# Test health endpoints (using dynamic port discovery)
curl -f http://localhost:$(vrooli scenario port scenario-authenticator API_PORT)/health
curl -f http://localhost:$(vrooli scenario port scenario-authenticator UI_PORT)/health
```

## 📊 Monitoring

The authentication service exposes metrics for:

- **Active Sessions**: Real-time session count
- **Login/Registration Rates**: Authentication activity  
- **Token Validation Load**: API usage patterns
- **Failed Authentication Attempts**: Security monitoring
- **Password Reset Requests**: User activity tracking

Access via:
```bash
scenario-authenticator status --verbose
```

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `API_PORT` | Dynamic (15000-19999) | API server port |
| `UI_PORT` | Dynamic (35000-39999) | UI server port |
| `DATABASE_URL` | auto | PostgreSQL connection |
| `REDIS_URL` | auto | Redis connection |
| `CORS_ALLOWED_ORIGINS` | localhost:3000,5173,8080 | Comma-separated list of allowed CORS origins |
| `CSP_POLICY` | (auto) | Custom Content-Security-Policy header (defaults to development-friendly CSP) |
| `JWT_EXPIRY_MINUTES` | 60 | JWT token expiry in minutes (max: 1440 = 24 hours) |
| `MAX_REQUEST_SIZE_MB` | 1 | Maximum request body size in MB (max: 100) |
| `GOOGLE_CLIENT_ID` | (optional) | Google OAuth2 client ID |
| `GOOGLE_CLIENT_SECRET` | (optional) | Google OAuth2 client secret |
| `GITHUB_CLIENT_ID` | (optional) | GitHub OAuth2 client ID |
| `GITHUB_CLIENT_SECRET` | (optional) | GitHub OAuth2 client secret |
| `AUTH_BASE_URL` | http://localhost:8105 | Base URL for OAuth2 callbacks |
| `REFRESH_EXPIRY` | 7d | Refresh token expiry |
| `CONTACT_BOOK_URL` | none | Base URL for the Contact Book UI (used by the Manage Profile action) |

If `CONTACT_BOOK_URL` is unset but the lifecycle exposes `CONTACT_BOOK_API_PORT`, the UI falls back to `http://localhost:${CONTACT_BOOK_API_PORT}`.

### OAuth2 Configuration (Optional)

The scenario includes **full OAuth2 social login support** for Google and GitHub. OAuth2 is disabled by default and can be enabled by setting the required environment variables.

#### Enabling Google OAuth2

1. **Create OAuth2 Credentials** at [Google Cloud Console](https://console.cloud.google.com/):
   - Create a new project or select existing
   - Navigate to "APIs & Services" > "Credentials"
   - Click "Create Credentials" > "OAuth 2.0 Client ID"
   - Application type: "Web application"
   - Authorized redirect URI: `http://localhost:$(vrooli scenario port scenario-authenticator API_PORT)/api/v1/auth/oauth/google/callback`
   - Copy the Client ID and Client Secret

2. **Set Environment Variables**:
   ```bash
   export GOOGLE_CLIENT_ID="your-google-client-id"
   export GOOGLE_CLIENT_SECRET="your-google-client-secret"
   ```

3. **Restart the Service**:
   ```bash
   vrooli scenario restart scenario-authenticator
   ```

4. **Verify OAuth2 is Active**:
   ```bash
   curl http://localhost:$(vrooli scenario port scenario-authenticator API_PORT)/api/v1/auth/oauth/providers | jq .
   # Should show Google provider as enabled
   ```

#### Enabling GitHub OAuth2

1. **Create OAuth App** at [GitHub Developer Settings](https://github.com/settings/developers):
   - Click "New OAuth App"
   - Application name: "Vrooli Authentication"
   - Homepage URL: `http://localhost:$(vrooli scenario port scenario-authenticator API_PORT)`
   - Authorization callback URL: `http://localhost:$(vrooli scenario port scenario-authenticator API_PORT)/api/v1/auth/oauth/github/callback`
   - Copy the Client ID and generate a Client Secret

2. **Set Environment Variables**:
   ```bash
   export GITHUB_CLIENT_ID="your-github-client-id"
   export GITHUB_CLIENT_SECRET="your-github-client-secret"
   ```

3. **Restart the Service** and verify as shown above.

#### OAuth2 Flow

When OAuth2 is enabled:
1. Users see "Sign in with Google/GitHub" buttons in the UI
2. Clicking initiates OAuth2 authorization flow
3. After successful authentication, users are automatically created or linked
4. OAuth users are marked with `email_verified: true`
5. All OAuth events are logged in audit logs

**Security Features**:
- CSRF protection via state tokens (10-minute expiration)
- Automatic user linking by email
- Session creation after OAuth login
- Audit logging for all OAuth events

For detailed OAuth2 implementation details, see `docs/oauth2-implementation-guide.md`.

### Database Configuration

The service automatically creates and manages:
- User accounts and authentication data
- Session storage and token blacklisting  
- Audit logs and security events
- API keys and permissions

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Your Scenario │───▶│ Auth Validator   │───▶│  Auth Service   │
│                 │    │ (n8n workflow)   │    │  (API + Redis)  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
                                                ┌─────────────────┐
                                                │   PostgreSQL    │
                                                │  (User Data)    │
                                                └─────────────────┘
```

## 🚀 Next Steps

After setting up authentication, you can:

1. **Secure Existing Scenarios**: Add auth to your current scenarios
2. **Build Multi-User Apps**: Create user-specific features  
3. **Implement RBAC**: Add role-based permissions
4. **Create SaaS Products**: Build subscription-based scenarios
5. **Add Team Features**: Enable collaborative workflows

## 📚 Related Scenarios

This authentication service enables:
- **SaaS Billing Hub**: User-based subscription management
- **Team Collaboration Tools**: Multi-user project management
- **API Gateway Manager**: Per-user rate limiting and keys
- **Personal Data Vaults**: User-specific secure storage
- **Enterprise Dashboards**: Department-level access control

## 🆘 Support

```bash
# Get help
scenario-authenticator help

# Check service status  
scenario-authenticator status

# View logs
journalctl -u scenario-authenticator -f
```

## 🔐 Security Notice

⚠️ **IMPORTANT - Development Seed Data**
This scenario includes seed data (`initialization/postgres/seed.sql`) with **default test accounts**:
- `admin@vrooli.local` / `Admin123!`
- `test@vrooli.local` / `Test123!`
- `demo@vrooli.local` / `Demo123!`

**For Production Deployments:**
- **DELETE or DISABLE** all seed accounts before production use
- **Change all default passwords** immediately
- **Use HTTPS** for all authentication endpoints
- **Rotate JWT keys** regularly (set up automated rotation)
- **Monitor failed login attempts** for security threats
- **Enable audit logging** and review logs regularly
- **Configure CORS_ALLOWED_ORIGINS** environment variable for production domains

---

**This scenario represents a permanent capability addition to Vrooli - every authentication system built on this foundation enhances the intelligence and security of the entire platform.**
