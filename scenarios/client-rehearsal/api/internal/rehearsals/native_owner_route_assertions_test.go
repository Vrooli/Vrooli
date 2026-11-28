package rehearsals

import (
	"net/http"
	"strings"
	"testing"
)

func exactHTTPMockRouteHitCount(receipt HTTPMockRouteHitsReceipt, expected HTTPMockRoute) uint64 {
	if receipt.Status != "available" {
		return 0
	}
	for _, hit := range receipt.Routes {
		if strings.EqualFold(hit.Host, expected.Host) && strings.EqualFold(hit.Method, expected.Method) && hit.Path == expected.Path && hit.Query == expected.Query {
			return hit.Count
		}
	}
	return 0
}

func seededNativeFaultRouteSatisfied(receipt HTTPMockRouteHitsReceipt, expected HTTPMockRoute) bool {
	return exactHTTPMockRouteHitCount(receipt, expected) == 1
}

func TestSeededNativeFaultRequiresExactRouteHitForScheduleAndWebhook(t *testing.T) {
	faultRoutes := []HTTPMockRoute{
		{Host: "generativelanguage-googleapis-co.synthetic.invalid", Method: http.MethodGet, Path: "/v1beta/models/gemini-2.0-flash:generateContent"},
		{Host: "mail.synthetic.invalid", Method: http.MethodGet, Path: "/v1/messages"},
	}
	for _, route := range faultRoutes {
		t.Run(route.Host+route.Path, func(t *testing.T) {
			if seededNativeFaultRouteSatisfied(HTTPMockRouteHitsReceipt{Status: "available"}, route) {
				t.Fatal("zero-hit owner failure must not count as a seeded service fault")
			}
			if seededNativeFaultRouteSatisfied(HTTPMockRouteHitsReceipt{Status: "unavailable", Routes: []HTTPMockRouteHit{{Host: route.Host, Method: route.Method, Path: route.Path, Count: 1}}}, route) {
				t.Fatal("unavailable route receipt must not count as a seeded service fault")
			}
			wrongRoute := HTTPMockRouteHitsReceipt{Status: "available", Routes: []HTTPMockRouteHit{{Host: route.Host, Method: http.MethodPost, Path: route.Path, Count: 1}}}
			if seededNativeFaultRouteSatisfied(wrongRoute, route) {
				t.Fatal("method-mismatched route must not count as a seeded service fault")
			}
			exactRoute := HTTPMockRouteHitsReceipt{Status: "available", Routes: []HTTPMockRouteHit{{Host: route.Host, Method: route.Method, Path: route.Path, Count: 1}}}
			if !seededNativeFaultRouteSatisfied(exactRoute, route) {
				t.Fatal("one exact fault route hit should satisfy the seeded service-fault receipt")
			}
			exactRoute.Routes[0].Query = "different=value"
			if seededNativeFaultRouteSatisfied(exactRoute, route) {
				t.Fatal("query-mismatched route must not count as the seeded service fault")
			}
		})
	}
}
