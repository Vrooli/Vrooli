package providers

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"nutrition-planner/internal/testutil/mocks"
)

func TestUSDAAdapterPreservesSourceIdentityAndNutrients(t *testing.T) {
	fake := &mocks.FakeDoer{}
	fake.AddResponse(http.StatusOK, []byte(`{"foods":[{"fdcId":123,"description":"Beans","dataType":"Foundation","brandOwner":"","foodNutrients":[]}]}`))
	fake.AddResponse(http.StatusOK, []byte(`{"fdcId":123,"description":"Beans","dataType":"Foundation","foodNutrients":[{"nutrientNumber":"203","nutrientName":"Protein","value":9.5,"unitName":"G"}]}`))
	adapter := USDAFoodDataCentral{APIKey: "test-key", BaseURL: "https://example.test/fdc/v1", Client: fake, Now: func() time.Time { return time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC) }}
	results, err := adapter.Search(context.Background(), NutritionSearchQuery{Query: "beans", DataType: "Foundation"})
	if err != nil || len(results) != 1 || results[0].SourceID != "123" || results[0].SourceVersion != "fdc-v1" || results[0].ObservedAt != "2026-09-18" {
		t.Fatalf("search=%#v err=%v", results, err)
	}
	detail, err := adapter.Detail(context.Background(), NutritionQuery{FoodID: "123"})
	if err != nil || detail.Values["203"] != "9.5 G" || !strings.Contains(detail.Evidence[0], "123") {
		t.Fatalf("detail=%#v err=%v", detail, err)
	}
	if len(fake.Requests) != 2 || fake.Requests[0].URL.Query().Get("api_key") != "test-key" || fake.Requests[1].URL.Path != "/fdc/v1/food/123" {
		t.Fatalf("requests=%#v", fake.Requests)
	}
}

func TestUSDAAdapterRequiresConfigurationAndClassifiesHTTPFailure(t *testing.T) {
	if _, err := (USDAFoodDataCentral{}).Search(context.Background(), NutritionSearchQuery{Query: "beans"}); err != ErrNotConfigured {
		t.Fatalf("unconfigured error=%v", err)
	}
	fake := &mocks.FakeDoer{}
	fake.AddResponse(http.StatusUnauthorized, []byte("bad key"))
	adapter := USDAFoodDataCentral{APIKey: "bad", BaseURL: "https://example.test", Client: fake}
	if _, err := adapter.Search(context.Background(), NutritionSearchQuery{Query: "beans"}); err == nil || !strings.Contains(err.Error(), "status 401") {
		t.Fatalf("expected classified HTTP failure, got %v", err)
	}
}
