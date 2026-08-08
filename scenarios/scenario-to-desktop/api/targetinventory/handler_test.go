package targetinventory

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestHandlerExposesTargetInventory(t *testing.T) {
	router := mux.NewRouter()
	NewHandler(LocalProbe{LookPath: func(string) (string, error) { return "/bin/tool", nil }}).RegisterRoutes(router)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/validation/targets", nil))
	if recorder.Code != http.StatusOK || recorder.Body.Len() == 0 {
		t.Fatalf("inventory response status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	var inventory Inventory
	if err := json.Unmarshal(recorder.Body.Bytes(), &inventory); err != nil {
		t.Fatalf("decode inventory: %v", err)
	}
	if len(inventory.Targets) != 1 || inventory.Targets[0].Descriptor == nil {
		t.Fatalf("unexpected inventory: %+v", inventory)
	}
}
