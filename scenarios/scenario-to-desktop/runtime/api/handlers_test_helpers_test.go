package api

import "net/http"

// testMux registers the scenario API routes with the supplied test token.
// Tests of authentication itself keep a direct Server so they can wrap custom
// routes and inspect middleware behavior.
func testMux(rt Runtime, token string) *http.ServeMux {
	server := NewServer(rt, token)
	mux := http.NewServeMux()
	server.RegisterHandlers(mux)
	return mux
}

// testHandler registers the owner routes and applies the production auth
// middleware for endpoint tests whose contract includes authentication.
func testHandler(rt Runtime, token string) http.Handler {
	server := NewServer(rt, token)
	mux := http.NewServeMux()
	server.RegisterHandlers(mux)
	return server.AuthMiddleware(mux)
}
