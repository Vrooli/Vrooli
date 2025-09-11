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
# Setup the authentication service
vrooli scenario run scenario-authenticator

# Access points:
# - Authentication API: http://localhost:3250
# - Authentication UI: http://localhost:3251
# - CLI: scenario-authenticator --help
```

### 2. Create Your First User

```bash
# Via CLI
scenario-authenticator user create admin@example.com SecurePass123! admin

# Via API
curl -X POST http://localhost:3250/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"SecurePass123!"}'

# Via UI
# Visit http://localhost:3251 and click "Sign Up"
```

### 3. Validate Tokens

```bash
# Get your token (from registration response or login)
export AUTH_TOKEN="eyJhbGciOiJSUzI1NiIs..."

# Validate via CLI
scenario-authenticator token validate $AUTH_TOKEN

# Validate via API
curl -H "Authorization: Bearer $AUTH_TOKEN" \
  http://localhost:3250/api/v1/auth/validate
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
        const response = await fetch('http://localhost:3250/api/v1/auth/validate', {
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
        
        resp, err := http.Get("http://localhost:3250/api/v1/auth/validate?token=" + token)
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
        window.location.href = 'http://localhost:3251/login?redirect=' + 
                              encodeURIComponent(window.location.href);
        return false;
    }
    
    try {
        const response = await fetch('http://localhost:3250/api/v1/auth/validate', {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        
        const validation = await response.json();
        if (!validation.valid) {
            localStorage.removeItem('auth_token');
            window.location.href = 'http://localhost:3251/login?redirect=' + 
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
- **JWT with RSA Signing**: Secure, stateless authentication
- **Token Blacklisting**: Immediate token revocation via Redis
- **Session Management**: Secure session tracking and cleanup  
- **Rate Limiting**: Per-user API rate limiting
- **Audit Logging**: Complete authentication event tracking
- **Password Reset Security**: Time-limited, single-use reset tokens
- **Role-Based Permissions**: Granular access control

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
curl -f http://localhost:3250/health
curl -f http://localhost:3251/health
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
| `AUTH_API_PORT` | 3250 | API server port |
| `AUTH_UI_PORT` | 3251 | UI server port |
| `DATABASE_URL` | auto | PostgreSQL connection |
| `REDIS_URL` | auto | Redis connection |
| `JWT_EXPIRY` | 1h | Token expiration time |
| `REFRESH_EXPIRY` | 7d | Refresh token expiry |

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

- **Change default passwords** immediately in production
- **Use HTTPS** for all authentication endpoints
- **Rotate JWT keys** regularly
- **Monitor failed login attempts**
- **Enable audit logging** in production environments

---

**This scenario represents a permanent capability addition to Vrooli - every authentication system built on this foundation enhances the intelligence and security of the entire platform.**