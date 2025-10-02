# Quick Integration Guide

Add authentication to any scenario in 5 minutes.

## For Go API (Minimal Example)

```go
// 1. Add middleware function
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

        req, _ := http.NewRequest("GET", "http://localhost:15785/api/v1/auth/validate", nil)
        req.Header.Set("Authorization", "Bearer "+token)

        resp, err := http.DefaultClient.Do(req)
        if err != nil || resp.StatusCode != 200 {
            http.Error(w, "Unauthorized", 401)
            return
        }

        next(w, r)
    }
}

// 2. Wrap your handlers
http.HandleFunc("/api/data", requireAuth(dataHandler))
```

## For JavaScript UI

```javascript
// 1. Check auth on page load
const token = localStorage.getItem('auth_token');
if (!token) {
    window.location.href = 'http://localhost:15785/ui';
}

// 2. Add token to API calls
fetch('/api/data', {
    headers: { 'Authorization': `Bearer ${token}` }
});
```

## For CLI Integration

```bash
# Validate token
scenario-authenticator token validate "$TOKEN"

# Create user
scenario-authenticator user create user@example.com password123
```

## Test Your Integration

```bash
# Get a token
TOKEN=$(curl -X POST http://localhost:15785/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "test123"}' \
  | jq -r '.token')

# Test protected endpoint
curl http://localhost:YOUR_PORT/api/protected \
  -H "Authorization: Bearer $TOKEN"
```

That's it! For a complete working example, see [integration-example.md](./integration-example.md).
