package main

import (
	"net/http"
	"strings"

	"connectrpc.com/connect"
)

// couponProviderError is API composition's Stripe-specific error policy. The
// coupon transport receives it as an injected mapper and never imports main.
func couponProviderError(err error) error {
	if status, _, _, ok := classifyStripeError(err); ok {
		switch status {
		case http.StatusNotFound:
			return connect.NewError(connect.CodeNotFound, err)
		case http.StatusBadGateway, http.StatusServiceUnavailable:
			return connect.NewError(connect.CodeUnavailable, err)
		case http.StatusUnauthorized:
			return connect.NewError(connect.CodeUnauthenticated, err)
		case http.StatusForbidden:
			return connect.NewError(connect.CodePermissionDenied, err)
		}
	}
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInvalidArgument, err)
}
