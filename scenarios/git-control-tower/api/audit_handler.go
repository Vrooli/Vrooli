package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// [REQ:GCT-OT-P0-007] Audit log query endpoint
func (s *Server) handleAuditQuery(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp := NewResponse(w)

	// Parse query parameters
	query := r.URL.Query()
	req := AuditQueryRequest{
		Operation: AuditOperation(query.Get("operation")),
		Branch:    query.Get("branch"),
		Limit:     50, // Default limit
	}

	// Parse optional parameters
	if limitStr := query.Get("limit"); limitStr != "" {
		if _, err := fmt.Sscanf(limitStr, "%d", &req.Limit); err != nil {
			resp.BadRequest("invalid limit parameter")
			return
		}
		if req.Limit > 1000 {
			req.Limit = 1000 // Cap at 1000
		}
	}
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if _, err := fmt.Sscanf(offsetStr, "%d", &req.Offset); err != nil {
			resp.BadRequest("invalid offset parameter")
			return
		}
	}

	result, err := s.audit.Query(ctx, req)
	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	resp.OK(result)
}
