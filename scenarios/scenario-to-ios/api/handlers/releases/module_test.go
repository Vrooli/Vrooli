package releases

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
)

func TestModuleMountsVersionedSummaryAndValidationRoutes(t *testing.T) {
	store, err := validationmatrix.NewFileStore(t.TempDir())
	require.NoError(t, err)
	matrixHandler := validationmatrix.NewHandler(validationmatrix.NewService(store, validationmatrix.Executors{}))
	m := Module([]*validationmatrix.Handler{matrixHandler}, Surface{
		Probe: func(context.Context) (deliveryramp.Inventory, error) {
			return deliveryramp.Inventory{Targets: []deliveryramp.Target{{ID: "ios:simulator:linux", Available: false}}}, nil
		},
		ChapterCount: 12,
	})
	router := mux.NewRouter()
	m.Mount(router)

	for _, path := range []string{"/api/v1/ios/matrix", "/api/v1/validation/profiles", "/api/v1/validation/matrices"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code, path)
	}

	legacy := httptest.NewRequest(http.MethodGet, "/ios/matrix", nil)
	legacyResponse := httptest.NewRecorder()
	router.ServeHTTP(legacyResponse, legacy)
	require.Equal(t, http.StatusNotFound, legacyResponse.Code)
}
