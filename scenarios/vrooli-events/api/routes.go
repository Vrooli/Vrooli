package main

import "net/http"

func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()

	// Event endpoints
	mux.HandleFunc("POST /api/v1/events", s.handleIngest)
	mux.HandleFunc("GET /api/v1/events", s.handleQuery)
	mux.HandleFunc("GET /api/v1/events/subscribe", s.handleSubscribe)

	// Policy endpoints
	mux.HandleFunc("POST /api/v1/policies", s.handleCreatePolicy)
	mux.HandleFunc("GET /api/v1/policies", s.handleListPolicies)
	mux.HandleFunc("GET /api/v1/policies/snapshot", s.handlePolicySnapshot)
	mux.HandleFunc("POST /api/v1/receipt-projections", s.handleCreateReceiptProjection)
	mux.HandleFunc("GET /api/v1/receipt-projections", s.handleListReceiptProjections)
	mux.HandleFunc("GET /api/v1/receipt-projections/{id}", s.handleGetReceiptProjection)
	mux.HandleFunc("PUT /api/v1/receipt-projections/{id}", s.handleUpdateReceiptProjection)
	mux.HandleFunc("DELETE /api/v1/receipt-projections/{id}", s.handleDeleteReceiptProjection)
	mux.HandleFunc("GET /api/v1/policies/subscribe", s.handlePolicySubscribe)
	mux.HandleFunc("GET /api/v1/policies/violations", s.handleListViolations)
	mux.HandleFunc("POST /api/v1/policies/evaluate", s.handleEvaluatePolicy)
	mux.HandleFunc("GET /api/v1/policies/{id}", s.handleGetPolicy)
	mux.HandleFunc("PUT /api/v1/policies/{id}", s.handleUpdatePolicy)
	mux.HandleFunc("DELETE /api/v1/policies/{id}", s.handleDeletePolicy)
	mux.HandleFunc("POST /api/v1/policies/{id}/override", s.handleOverrideCircuitBreaker)

	// Subscription endpoints
	mux.HandleFunc("POST /api/v1/subscriptions", s.handleCreateSubscription)
	mux.HandleFunc("GET /api/v1/subscriptions", s.handleListSubscriptions)
	mux.HandleFunc("GET /api/v1/subscriptions/{id}", s.handleGetSubscription)
	mux.HandleFunc("PUT /api/v1/subscriptions/{id}", s.handleUpdateSubscription)
	mux.HandleFunc("DELETE /api/v1/subscriptions/{id}", s.handleDeleteSubscription)
	mux.HandleFunc("GET /api/v1/subscriptions/{id}/health", s.handleGetSubscriptionHealth)
	mux.HandleFunc("POST /api/v1/subscriptions/{id}/test", s.handleTestSubscription)
	mux.HandleFunc("POST /api/v1/subscriptions/{id}/deliver", s.handleDeliverSubscription)

	// Health
	mux.HandleFunc("GET /health", s.handleHealth)
	return mux
}
