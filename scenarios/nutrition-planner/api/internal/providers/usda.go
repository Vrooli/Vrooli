package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"nutrition-planner/internal/httpc"
)

// USDAFoodDataCentral is the source-aware FoodData Central adapter. It is
// intentionally small: callers receive provider identity and raw nutrient
// values, while local catalog revisions remain the durable source of truth.
type USDAFoodDataCentral struct {
	APIKey  string
	BaseURL string
	Client  httpc.Doer
	Now     func() time.Time
}

type usdaSearchResponse struct {
	Foods []struct {
		FDCID       int64  `json:"fdcId"`
		Description string `json:"description"`
		DataType    string `json:"dataType"`
		BrandOwner  string `json:"brandOwner"`
	} `json:"foods"`
}

type usdaFood struct {
	FDCID           int64  `json:"fdcId"`
	Description     string `json:"description"`
	DataType        string `json:"dataType"`
	BrandOwner      string `json:"brandOwner"`
	PublicationDate string `json:"publicationDate"`
	Nutrients       []struct {
		Number string  `json:"nutrientNumber"`
		Name   string  `json:"nutrientName"`
		Amount float64 `json:"value"`
		Unit   string  `json:"unitName"`
	} `json:"foodNutrients"`
}

func (u USDAFoodDataCentral) endpoint() string {
	base := strings.TrimRight(strings.TrimSpace(u.BaseURL), "/")
	if base == "" {
		base = "https://api.nal.usda.gov/fdc/v1"
	}
	return base
}

func (u USDAFoodDataCentral) now() time.Time {
	if u.Now != nil {
		return u.Now().UTC()
	}
	return time.Now().UTC()
}

func (u USDAFoodDataCentral) request(ctx context.Context, path string, query url.Values, target any) error {
	if strings.TrimSpace(u.APIKey) == "" {
		return ErrNotConfigured
	}
	client := u.Client
	if client == nil {
		client = http.DefaultClient
	}
	if query == nil {
		query = url.Values{}
	}
	query.Set("api_key", u.APIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.endpoint()+path+"?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("usda request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func (u USDAFoodDataCentral) Search(ctx context.Context, query NutritionSearchQuery) ([]NutritionSearchResult, error) {
	var response usdaSearchResponse
	params := url.Values{"query": {query.Query}, "pageSize": {"50"}}
	if query.DataType != "" {
		params.Set("dataType", query.DataType)
	}
	if err := u.request(ctx, "/foods/search", params, &response); err != nil {
		return nil, err
	}
	observed := u.now().Format("2006-01-02")
	out := make([]NutritionSearchResult, 0, len(response.Foods))
	for _, food := range response.Foods {
		out = append(out, NutritionSearchResult{
			SourceID: strconv.FormatInt(food.FDCID, 10), SourceVersion: "fdc-v1", ObservedAt: observed,
			ConceptID: strconv.FormatInt(food.FDCID, 10), Name: food.Description, ProductName: food.BrandOwner,
			DataType: food.DataType, Preparation: query.Preparation,
		})
	}
	return out, nil
}

func (u USDAFoodDataCentral) Detail(ctx context.Context, query NutritionQuery) (NutritionResult, error) {
	if strings.TrimSpace(query.FoodID) == "" {
		return NutritionResult{}, fmt.Errorf("food id is required")
	}
	var food usdaFood
	if err := u.request(ctx, "/food/"+url.PathEscape(query.FoodID), nil, &food); err != nil {
		return NutritionResult{}, err
	}
	values := make(map[string]string, len(food.Nutrients))
	for _, nutrient := range food.Nutrients {
		key := strings.TrimSpace(nutrient.Number)
		if key == "" {
			key = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(nutrient.Name), " ", "_"))
		}
		values[key] = strconv.FormatFloat(nutrient.Amount, 'f', -1, 64) + " " + strings.TrimSpace(nutrient.Unit)
	}
	return NutritionResult{SourceID: strconv.FormatInt(food.FDCID, 10), SourceVersion: "fdc-v1", ObservedAt: u.now().Format("2006-01-02"), Values: values, Evidence: []string{"usda:fdc:" + strconv.FormatInt(food.FDCID, 10)}}, nil
}

var _ NutritionCatalog = USDAFoodDataCentral{}
