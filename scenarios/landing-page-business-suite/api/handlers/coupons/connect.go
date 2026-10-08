// Package coupons owns the typed HTTP boundary for coupon administration.
package coupons

import (
	"net/http"

	"github.com/gorilla/mux"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
)

// Register mounts every generated CouponAdminService procedure behind the
// scenario's administrator authorization middleware. The handler implementation
// is supplied by the commerce composition layer so this package stays transport-only.
func Register(router *mux.Router, handler http.Handler, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	for _, procedure := range []string{
		lpbsconnect.CouponAdminServiceListCouponsProcedure,
		lpbsconnect.CouponAdminServiceCreateCouponProcedure,
		lpbsconnect.CouponAdminServiceGetCouponProcedure,
		lpbsconnect.CouponAdminServiceUpdateCouponProcedure,
		lpbsconnect.CouponAdminServiceDeleteCouponProcedure,
		lpbsconnect.CouponAdminServiceListCouponUsageProcedure,
		lpbsconnect.CouponAdminServiceGetCouponMappingsProcedure,
		lpbsconnect.CouponAdminServiceSetCouponForPlanProcedure,
		lpbsconnect.CouponAdminServiceRemoveCouponFromPlanProcedure,
		lpbsconnect.CouponAdminServiceGetCouponImportPreviewProcedure,
	} {
		router.Handle(procedure, requireAdmin(handler.ServeHTTP)).Methods(http.MethodPost)
	}
}
