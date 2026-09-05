package coupons

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
)

func TestRegisterProtectsEveryCouponProcedure(t *testing.T) {
	router := mux.NewRouter()
	Register(router, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("authorization middleware must run before the coupon handler")
	}), func(http.HandlerFunc) http.HandlerFunc {
		return func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(http.StatusUnauthorized) }
	})

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
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, procedure, nil))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s status = %d, want %d", procedure, response.Code, http.StatusUnauthorized)
		}
	}
}
