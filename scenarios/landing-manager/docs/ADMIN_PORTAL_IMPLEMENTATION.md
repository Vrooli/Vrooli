---
title: "Admin Portal Implementation"
description: "Implementation guide for the admin portal"
category: "technical"
order: 4
audience: ["developers"]
---

# Admin Portal Implementation Guide

> **Status**: Foundation Created (P10)
> **Priority**: P0 - Blocks 7 critical requirements
> **Last Updated**: 2025-11-21

## Overview

The admin portal is a P0 feature that enables landing page owners to:
1. **Authenticate** securely (email/password with bcrypt)
2. **View analytics** (conversion rates, A/B test results, visitor metrics)
3. **Customize content** (trigger agent-based improvements, manage variants)
4. **Navigate efficiently** (≤ 3 clicks to any customization card)

## Current Status

### ✅ Completed (P10)
- Created `ui/src/pages/AdminLogin.tsx` - Login page stub with selectors
- Created `ui/src/pages/AdminHome.tsx` - Mode switcher (Analytics/Customization)
- Added admin portal selectors to `ui/src/consts/selectors.ts`:
  - `admin.login.{email, password, submit, error}`
  - `admin.breadcrumb`
  - `admin.mode.{analytics, customization}`
- Documented requirements and implementation TODOs in component comments
- Agent handoff is expected to go through **app-issue-tracker** (no on-box model dependency).

### ❌ Blocked Requirements (7 P0)

| Requirement | PRD Ref | Status | Blocker |
|-------------|---------|--------|---------|
| AGENT-TRIGGER | OT-P0-005 | Not Implemented | No verified agent handoff via app-issue-tracker |
| AGENT-INPUT | OT-P0-006 | Not Implemented | Depends on AGENT-TRIGGER |
| ADMIN-HIDDEN | OT-P0-007 | Not Implemented | No routing system |
| ADMIN-AUTH | OT-P0-008 | Partial (UI only) | No API endpoint, no session management |
| ADMIN-MODES | OT-P0-009 | Partial (UI only) | No routing, no analytics/customization pages |
| ADMIN-NAV | OT-P0-010 | Not Implemented | No customization cards to navigate to |
| ADMIN-BREADCRUMB | OT-P0-011 | Partial (stub) | No routing context |

## Implementation Roadmap

### Phase 1: Routing & Authentication (Est. 4-6 hours)

#### 1.1 Install React Router
```bash
cd ui
pnpm add react-router-dom@^6.26.0
pnpm add -D @types/react-router-dom
```

#### 1.2 Update App.tsx with Routes
```tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AdminLogin } from './pages/AdminLogin';
import { AdminHome } from './pages/AdminHome';
import { ProtectedRoute } from './components/ProtectedRoute';

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Public route - health check (existing) */}
        <Route path="/" element={<PublicHome />} />

        {/* Admin routes - hidden from public (OT-P0-007) */}
        <Route path="/admin/login" element={<AdminLogin />} />
        <Route path="/admin" element={
          <ProtectedRoute>
            <AdminHome />
          </ProtectedRoute>
        } />

        {/* Future: Analytics & Customization pages */}
        <Route path="/admin/analytics" element={<ProtectedRoute><Analytics /></ProtectedRoute>} />
        <Route path="/admin/customization" element={<ProtectedRoute><Customization /></ProtectedRoute>} />

        {/* 404 redirect */}
        <Route path="*" element={<Navigate to="/" />} />
      </Routes>
    </BrowserRouter>
  );
}
```

#### 1.3 Create Authentication Context
```tsx
// ui/src/contexts/AuthContext.tsx
import { createContext, useContext, useState, useEffect } from 'react';

interface AuthContextValue {
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  user: { email: string } | null;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState<{ email: string } | null>(null);

  // Check for existing session on mount
  useEffect(() => {
    const checkSession = async () => {
      // TODO: Call API to verify session cookie
      // For now, check localStorage (NOT secure, for development only)
      const session = localStorage.getItem('admin_session');
      if (session) {
        try {
          const userData = JSON.parse(session);
          setIsAuthenticated(true);
          setUser(userData);
        } catch (e) {
          console.error('Invalid session data');
        }
      }
    };
    checkSession();
  }, []);

  const login = async (email: string, password: string) => {
    // TODO: Call POST /api/v1/admin/login with { email, password }
    // API should return httpOnly cookie with session token
    const response = await fetch('/api/v1/admin/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
      credentials: 'include', // Include cookies
    });

    if (!response.ok) {
      throw new Error('Login failed');
    }

    const userData = await response.json();
    setIsAuthenticated(true);
    setUser(userData);
    localStorage.setItem('admin_session', JSON.stringify(userData)); // Temporary, remove when using httpOnly cookies
  };

  const logout = () => {
    setIsAuthenticated(false);
    setUser(null);
    localStorage.removeItem('admin_session');
    // TODO: Call POST /api/v1/admin/logout to clear session cookie
  };

  return (
    <AuthContext.Provider value={{ isAuthenticated, login, logout, user }}>
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
};
```

#### 1.4 Create ProtectedRoute Component
```tsx
// ui/src/components/ProtectedRoute.tsx
import { Navigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth();

  if (!isAuthenticated) {
    return <Navigate to="/admin/login" replace />;
  }

  return <>{children}</>;
}
```

#### 1.5 Update AdminLogin to Use Auth Context
```tsx
import { useAuth } from '../contexts/AuthContext';
import { useNavigate } from 'react-router-dom';

export function AdminLogin() {
  const { login } = useAuth();
  const navigate = useNavigate();
  // ... existing state

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    try {
      await login(email, password);
      navigate("/admin");
    } catch (err) {
      setError("Invalid email or password");
    }
  };
  // ... rest of component
}
```

### Phase 2: Backend Authentication API (Est. 3-4 hours)

#### 2.1 Add Go Dependencies
```bash
cd api
go get golang.org/x/crypto/bcrypt
go get github.com/gorilla/sessions
```

#### 2.2 Create Database Schema
```sql
-- api/migrations/001_admin_users.sql
CREATE TABLE IF NOT EXISTS admin_users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    last_login TIMESTAMP
);

CREATE INDEX idx_admin_users_email ON admin_users(email);

-- Insert default admin (password: changeme123)
-- bcrypt hash generated with: bcrypt.GenerateFromPassword([]byte("changeme123"), 10)
INSERT INTO admin_users (email, password_hash) VALUES
('admin@localhost', '$2a$10$...');  -- Replace with actual hash
```

#### 2.3 Implement Authentication Handlers
```go
// api/auth.go
package main

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "time"

    "github.com/gorilla/sessions"
    "golang.org/x/crypto/bcrypt"
)

var sessionStore = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))

type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type LoginResponse struct {
    Email string `json:"email"`
}

func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Query user from database
    var passwordHash string
    err := s.db.QueryRow(
        "SELECT password_hash FROM admin_users WHERE email = $1",
        req.Email,
    ).Scan(&passwordHash)

    if err == sql.ErrNoRows {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    } else if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    // Verify password
    if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }

    // Update last login
    _, _ = s.db.Exec("UPDATE admin_users SET last_login = NOW() WHERE email = $1", req.Email)

    // Create session
    session, _ := sessionStore.Get(r, "admin_session")
    session.Values["email"] = req.Email
    session.Options.HttpOnly = true
    session.Options.Secure = false // Set to true in production with HTTPS
    session.Options.MaxAge = 86400 * 7 // 7 days
    session.Save(r, w)

    // Return user data
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(LoginResponse{Email: req.Email})
}

func (s *Server) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
    session, _ := sessionStore.Get(r, "admin_session")
    session.Options.MaxAge = -1
    session.Save(r, w)

    w.WriteHeader(http.StatusNoContent)
}

// Middleware to protect admin routes
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        session, _ := sessionStore.Get(r, "admin_session")
        email, ok := session.Values["email"].(string)
        if !ok || email == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}
```

#### 2.4 Register Auth Routes
```go
// In setupRoutes()
s.router.HandleFunc("/api/v1/admin/login", s.handleAdminLogin).Methods("POST")
s.router.HandleFunc("/api/v1/admin/logout", s.handleAdminLogout).Methods("POST")
```

### Phase 3: Analytics Dashboard (Est. 6-8 hours)

Requirements: OT-P0-019 through OT-P0-024

#### 3.1 Create Analytics Page Component
```tsx
// ui/src/pages/Analytics.tsx
import { useQuery } from '@tanstack/react-query';
import { fetchAnalytics } from '../lib/api';

export function Analytics() {
  const { data, isLoading } = useQuery({
    queryKey: ['analytics'],
    queryFn: fetchAnalytics,
  });

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 p-6">
      <nav className="mb-6 text-sm text-slate-400" data-testid="admin-breadcrumb">
        Admin / Analytics
      </nav>

      <h1 className="text-3xl font-semibold mb-8">Analytics Dashboard</h1>

      {/* OT-P0-023: Summary metrics */}
      <div className="grid md:grid-cols-3 gap-6 mb-8">
        <MetricCard title="Total Visitors" value={data?.totalVisitors ?? 0} />
        <MetricCard title="Conversion Rate" value={`${data?.conversionRate ?? 0}%`} />
        <MetricCard title="Top CTA CTR" value={`${data?.topCtaCtr ?? 0}%`} />
      </div>

      {/* OT-P0-020: Variant filtering */}
      <VariantFilter />

      {/* OT-P0-024: Variant detail views */}
      <VariantTable variants={data?.variants ?? []} />
    </div>
  );
}
```

#### 3.2 Implement Metrics Collection
- Event tracking on frontend (page_view, scroll_depth, click, form_submit, conversion)
- POST /api/v1/metrics endpoint to store events
- Database schema for metrics (events table with variant_id, event_type, timestamp, metadata)

#### 3.3 Implement Analytics API
- GET /api/v1/analytics?variant_id=...&start_date=...&end_date=...
- Aggregate queries for conversion rates, CTR, visitor counts
- Group by variant for comparison

### Phase 4: Customization Interface (Est. 8-10 hours)

Requirements: OT-P0-005, OT-P0-006, OT-P0-012, OT-P0-013

#### 4.1 Create Customization Page
```tsx
// ui/src/pages/Customization.tsx
export function Customization() {
  return (
    <div className="min-h-screen bg-slate-950 text-slate-50">
      <nav className="p-6 text-sm text-slate-400" data-testid="admin-breadcrumb">
        Admin / Customization
      </nav>

      {/* Split layout: Form + Preview (OT-P0-012) */}
      <div className="grid lg:grid-cols-2 gap-6 p-6">
        {/* Left: Form */}
        <div>
          <CustomizationForm />
          <AgentTriggerButton /> {/* OT-P0-005 */}
        </div>

        {/* Right: Live Preview (OT-P0-013) */}
        <div className="sticky top-6">
          <LivePreview />
        </div>
      </div>
    </div>
  );
}
```

#### 4.2 Implement Agent Integration
- POST /api/v1/admin/trigger-agent endpoint
- Accepts structured brief (text + assets + goals) - OT-P0-006
- Files customization requests into app-issue-tracker and triggers investigation
- Returns job_id for polling status

#### 4.3 Live Preview System
- Debounced form updates (300ms) - OT-P0-013
- Iframe preview or component preview
- Real-time variant rendering

### Phase 5: BAS Playbooks (Est. 2-3 hours)

Once UI is implemented, create automated test workflows:

#### 5.1 Admin Portal Playbook
```json
// bas/cases/admin-portal/ui/admin-portal.json
{
  "name": "Admin Portal Core Functionality",
  "description": "Validates admin login, authentication, mode switcher, and navigation",
  "requirements": [
    "AGENT-TRIGGER",
    "AGENT-INPUT",
    "ADMIN-HIDDEN",
    "ADMIN-AUTH",
    "ADMIN-MODES",
    "ADMIN-NAV",
    "ADMIN-BREADCRUMB"
  ],
  "steps": [
    {
      "action": "navigate",
      "url": "http://localhost:${UI_PORT}/admin/login"
    },
    {
      "action": "fill",
      "selector": "[data-testid='admin-login-email']",
      "value": "admin@localhost"
    },
    {
      "action": "fill",
      "selector": "[data-testid='admin-login-password']",
      "value": "test-password"
    },
    {
      "action": "click",
      "selector": "[data-testid='admin-login-submit']"
    },
    {
      "action": "waitForNavigation",
      "url": "/admin"
    },
    {
      "action": "assertVisible",
      "selector": "[data-testid='admin-breadcrumb']"
    },
    {
      "action": "assertText",
      "selector": "[data-testid='admin-breadcrumb']",
      "value": "Admin"
    },
    {
      "action": "assertVisible",
      "selector": "[data-testid='admin-mode-analytics']"
    },
    {
      "action": "assertVisible",
      "selector": "[data-testid='admin-mode-customization']"
    }
  ]
}
```

#### 5.2 Rebuild Registry
```bash
node scripts/scenarios/testing/playbooks/build-registry.mjs --scenario landing-manager
```

## Testing Strategy

### Manual Testing Checklist
- [ ] Can access /admin/login (not linked from public pages)
- [ ] Cannot access /admin without authentication
- [ ] Login with valid credentials redirects to admin home
- [ ] Login with invalid credentials shows error
- [ ] Admin home shows exactly 2 mode cards
- [ ] Breadcrumb shows "Admin" on home page
- [ ] Clicking Analytics mode navigates to analytics dashboard
- [ ] Clicking Customization mode navigates to customization interface
- [ ] Can navigate to any customization card in ≤ 3 clicks

### Automated Testing
- Structure phase: BAS playbook validation (9 requirements)
- Integration phase: API endpoint tests (login, logout, session)
- Business phase: Requirement coverage (7 P0 requirements)

## Security Considerations

### Authentication
- ✅ Use bcrypt for password hashing (cost factor 10+)
- ✅ Store session tokens in httpOnly cookies (prevent XSS)
- ✅ Add CSRF protection for state-changing requests
- ✅ Implement rate limiting on login endpoint (prevent brute force)
- ✅ Log failed login attempts for monitoring

### Authorization
- ✅ Verify session on every admin route
- ✅ Never expose admin routes in sitemap or public navigation (OT-P0-007)
- ✅ Add Content-Security-Policy headers
- ✅ Validate all user inputs on backend

### Session Management
- ✅ 7-day session expiry with sliding window
- ✅ Secure flag on cookies in production (HTTPS only)
- ✅ Generate session secret from environment variable
- ✅ Implement logout on all devices functionality

## Dependencies

### Frontend
- react-router-dom@^6.26.0 (routing)
- @tanstack/react-query (already installed, for API calls)
- shadcn/ui components (already installed, for form inputs)

### Backend
- golang.org/x/crypto/bcrypt (password hashing)
- github.com/gorilla/sessions (session management)
- Existing: gorilla/mux (routing), lib/pq (PostgreSQL)

### Database
- PostgreSQL (already dependency)
- New tables: admin_users, admin_sessions (optional), metrics

## Estimated Total Effort

| Phase | Effort | Priority |
|-------|--------|----------|
| Phase 1: Routing & Auth Frontend | 4-6 hours | P0 |
| Phase 2: Backend Auth API | 3-4 hours | P0 |
| Phase 3: Analytics Dashboard | 6-8 hours | P0 |
| Phase 4: Customization Interface | 8-10 hours | P0 |
| Phase 5: BAS Playbooks | 2-3 hours | P0 |
| **Total** | **23-31 hours** | **P0** |

## Next Steps for Future Agents

1. **Start with Phase 1** (Routing & Auth Frontend) - This unblocks UI development
2. **Implement Phase 2** (Backend Auth API) - This enables actual authentication
3. **Build Phase 3** (Analytics) OR **Phase 4** (Customization) - Choose based on priority
4. **Create BAS Playbooks** (Phase 5) - Once UI is complete
5. **Run full test suite** - Verify all 7 P0 requirements pass

## References

- Requirements: `/scenarios/landing-manager/requirements/02-admin-portal/module.json`
- PRD: `/scenarios/landing-manager/PRD.md` (OT-P0-005 through OT-P0-011)
- Selector Registry: `/scenarios/landing-manager/ui/src/consts/selectors.ts`
- Test Structure: `/docs/testing/guides/ui-automation-with-bas.md`

---

**Author**: Improver Agent P10
**Date**: 2025-11-21
**Status**: Foundation created, ready for Phase 1 implementation
