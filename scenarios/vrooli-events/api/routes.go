package main

import "net/http"

func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()

	// Event endpoints
	mux.HandleFunc("POST /api/v1/events", s.handleIngest)
	mux.HandleFunc("GET /api/v1/events", s.handleQuery)
	mux.HandleFunc("DELETE /api/v1/events", s.handleDeleteEventsByType)
	mux.HandleFunc("GET /api/v1/events/subscribe", s.handleSubscribe)

	// Policy endpoints
	mux.HandleFunc("GET /api/v1/policies/snapshot", s.handlePolicySnapshot)
	mux.HandleFunc("POST /api/v1/receipt-capture-policies", s.handleCreateCapturePolicy)
	mux.HandleFunc("GET /api/v1/receipt-capture-policies", s.handleListCapturePolicies)
	mux.HandleFunc("DELETE /api/v1/receipt-capture-policies/{policyID}", s.handleDeleteCapturePolicy)
	mux.HandleFunc("POST /api/v1/receipt-capture-policies/reconcile", s.handleReconcileCapturePolicies)
	mux.HandleFunc("GET /api/v1/policies/subscribe", s.handlePolicySubscribe)

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
	// Keep the canonical root route unqualified so lifecycle and static
	// conformance tools can verify the same health surface callers use.
	mux.HandleFunc("/health", s.handleHealth)
	return mux
}
