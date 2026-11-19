package brand_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	brandH "react-component-library/handlers/brand"
	internalbrand "react-component-library/internal/brand"
)

type stubResolver struct {
	brand internalbrand.Brand
	err   error
}

func (s stubResolver) Resolve(context.Context) (internalbrand.Brand, error) {
	return s.brand, s.err
}

func call(t *testing.T, resolver internalbrand.Resolver) *httptest.ResponseRecorder {
	t.Helper()
	m := brandH.Module(resolver)
	router := mux.NewRouter()
	m.Mount(router)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/brand", nil))
	return rec
}

func TestModuleShape(t *testing.T) {
	require.Len(t, brandH.Endpoints, 1)
	require.Equal(t, "/api/brand", brandH.Endpoints[0].Path)
	require.NotNil(t, brandH.Endpoints[0].RESTException)
}

func TestAssignedBrandResponse(t *testing.T) {
	accent, mark, wordmark := "#C0FFEE", "data:image/svg+xml;base64,PHN2Zy8+", "Aperture Labs"
	rec := call(t, stubResolver{brand: internalbrand.Brand{
		Assigned: true,
		Name:     "Aperture",
		Mark:     &mark,
		Accent:   &accent,
		Wordmark: &wordmark,
	}})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	require.JSONEq(t, `{"assigned":true,"name":"Aperture","mark":"data:image/svg+xml;base64,PHN2Zy8+","accent":"#C0FFEE","wordmark":"Aperture Labs"}`, rec.Body.String())
}

func TestUnassignedBrandResponse(t *testing.T) {
	rec := call(t, stubResolver{})
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"assigned":false,"name":"","mark":null,"accent":null,"wordmark":null}`, rec.Body.String())
}

func TestResolverErrorFallsBackToUnassigned(t *testing.T) {
	rec := call(t, stubResolver{err: errors.New("unexpected")})
	require.Equal(t, http.StatusOK, rec.Code)
	var payload internalbrand.Brand
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.False(t, payload.Assigned)
}
