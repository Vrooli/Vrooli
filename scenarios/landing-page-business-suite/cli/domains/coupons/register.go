package coupons

import (
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Admin Commerce - Coupons",
		Commands: deps.EndpointCommands([]support.EndpointDef{
			{Name: "admin-coupons-list", Method: "GET", Path: "/admin/coupons", Description: "List coupons"},
			{Name: "admin-coupons-create", Method: "POST", Path: "/admin/coupons", Description: "Create coupon"},
			{Name: "admin-coupons-usage", Method: "GET", Path: "/admin/coupons/usage", Description: "Coupon usage"},
			{Name: "admin-coupons-get", Method: "GET", Path: "/admin/coupons/{coupon_id}", Description: "Get coupon"},
			{Name: "admin-coupons-update", Method: "PATCH", Path: "/admin/coupons/{coupon_id}", Description: "Update coupon"},
			{Name: "admin-coupons-delete", Method: "DELETE", Path: "/admin/coupons/{coupon_id}", Description: "Delete coupon"},
			{Name: "admin-coupon-mappings", Method: "GET", Path: "/admin/coupon-mappings", Description: "List coupon mappings"},
			{Name: "admin-plan-coupon-set", Method: "PUT", Path: "/admin/plans/{price_id}/coupon", Description: "Set coupon for plan"},
			{Name: "admin-plan-coupon-remove", Method: "DELETE", Path: "/admin/plans/{price_id}/coupon", Description: "Remove coupon from plan"},
			{Name: "admin-stripe-coupons-preview", Method: "GET", Path: "/admin/stripe/coupons-preview", Description: "Stripe coupons preview"},
		}),
	}
}
