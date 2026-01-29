package main

import (
	"database/sql"
	"net/http"
	"strings"
)

// ListCouponsResponse contains the list of coupons and intro coupon mapping.
type ListCouponsResponse struct {
	Coupons       []StripeCoupon    `json:"coupons"`
	IntroCouponMap map[string]string `json:"intro_coupon_map"`
}

// CouponUsageStats contains usage statistics for a coupon from the local database.
type CouponUsageStats struct {
	CouponID   string  `json:"coupon_id"`
	TotalUses  int     `json:"total_uses"`
	LastUsedAt *string `json:"last_used_at,omitempty"`
}

// handleAdminListCoupons lists all coupons from Stripe.
func handleAdminListCoupons(stripe *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if stripe == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "Stripe service unavailable", ApiErrorTypeServerError)
			return
		}

		coupons, err := stripe.ListCoupons(r.Context())
		if err != nil {
			logStructuredError("admin_list_coupons_failed", map[string]interface{}{
				"error": err.Error(),
			})
			if status, errType, message, ok := classifyStripeError(err); ok {
				writeJSONError(w, status, message, errType)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to list coupons", ApiErrorTypeServerError)
			return
		}

		response := ListCouponsResponse{
			Coupons:        coupons,
			IntroCouponMap: stripe.GetIntroCouponMap(),
		}

		writeJSONSuccessData(w, response)
	}
}

// handleAdminCreateCoupon creates a new coupon in Stripe.
func handleAdminCreateCoupon(stripe *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if stripe == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "Stripe service unavailable", ApiErrorTypeServerError)
			return
		}

		var req CreateCouponRequest
		if !decodeJSONBody(w, r, &req) {
			return
		}

		coupon, err := stripe.CreateCoupon(r.Context(), req)
		if err != nil {
			logStructuredError("admin_create_coupon_failed", map[string]interface{}{
				"error":    err.Error(),
				"id":       req.ID,
				"duration": req.Duration,
			})
			if status, errType, message, ok := classifyStripeError(err); ok {
				writeJSONError(w, status, message, errType)
				return
			}
			// Check for validation errors from our own code
			errMsg := err.Error()
			if strings.Contains(errMsg, "required") || strings.Contains(errMsg, "must be") || strings.Contains(errMsg, "cannot specify") {
				writeJSONError(w, http.StatusBadRequest, errMsg, ApiErrorTypeValidation)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to create coupon", ApiErrorTypeServerError)
			return
		}

		logStructured("admin_coupon_created", map[string]interface{}{
			"coupon_id": coupon.ID,
			"duration":  coupon.Duration,
		})

		writeJSONSuccessData(w, coupon)
	}
}

// handleAdminGetCoupon gets a single coupon from Stripe.
func handleAdminGetCoupon(stripe *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if stripe == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "Stripe service unavailable", ApiErrorTypeServerError)
			return
		}

		couponID, ok := getPathParam(r, "coupon_id")
		if !ok || strings.TrimSpace(couponID) == "" {
			writeJSONError(w, http.StatusBadRequest, "Coupon ID is required", ApiErrorTypeValidation)
			return
		}

		coupon, err := stripe.GetCoupon(r.Context(), couponID)
		if err != nil {
			logStructuredError("admin_get_coupon_failed", map[string]interface{}{
				"error":     err.Error(),
				"coupon_id": couponID,
			})
			if status, errType, message, ok := classifyStripeError(err); ok {
				writeJSONError(w, status, message, errType)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to get coupon", ApiErrorTypeServerError)
			return
		}

		writeJSONSuccessData(w, coupon)
	}
}

// handleAdminDeleteCoupon deletes a coupon from Stripe.
func handleAdminDeleteCoupon(stripe *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if stripe == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "Stripe service unavailable", ApiErrorTypeServerError)
			return
		}

		couponID, ok := getPathParam(r, "coupon_id")
		if !ok || strings.TrimSpace(couponID) == "" {
			writeJSONError(w, http.StatusBadRequest, "Coupon ID is required", ApiErrorTypeValidation)
			return
		}

		err := stripe.DeleteCoupon(r.Context(), couponID)
		if err != nil {
			logStructuredError("admin_delete_coupon_failed", map[string]interface{}{
				"error":     err.Error(),
				"coupon_id": couponID,
			})
			if status, errType, message, ok := classifyStripeError(err); ok {
				writeJSONError(w, status, message, errType)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to delete coupon", ApiErrorTypeServerError)
			return
		}

		logStructured("admin_coupon_deleted", map[string]interface{}{
			"coupon_id": couponID,
		})

		writeJSONSuccess(w, "Coupon deleted successfully")
	}
}

// handleAdminCouponUsage returns usage statistics for intro coupons from the local database.
func handleAdminCouponUsage(stripe *StripeService, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "Database unavailable", ApiErrorTypeServerError)
			return
		}

		// Query usage stats from intro_coupon_usage table
		rows, err := db.QueryContext(r.Context(), `
			SELECT coupon_id, COUNT(*) as total_uses, MAX(created_at) as last_used_at
			FROM intro_coupon_usage
			GROUP BY coupon_id
			ORDER BY total_uses DESC
		`)
		if err != nil {
			logStructuredError("admin_coupon_usage_query_failed", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to query coupon usage", ApiErrorTypeServerError)
			return
		}
		defer rows.Close()

		stats := make([]CouponUsageStats, 0)
		for rows.Next() {
			var stat CouponUsageStats
			var lastUsedAt sql.NullTime
			if err := rows.Scan(&stat.CouponID, &stat.TotalUses, &lastUsedAt); err != nil {
				logStructuredError("admin_coupon_usage_scan_failed", map[string]interface{}{
					"error": err.Error(),
				})
				continue
			}
			if lastUsedAt.Valid {
				formatted := lastUsedAt.Time.Format("2006-01-02T15:04:05Z")
				stat.LastUsedAt = &formatted
			}
			stats = append(stats, stat)
		}

		if err := rows.Err(); err != nil {
			logStructuredError("admin_coupon_usage_rows_error", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to read coupon usage", ApiErrorTypeServerError)
			return
		}

		writeJSONSuccessData(w, stats)
	}
}
