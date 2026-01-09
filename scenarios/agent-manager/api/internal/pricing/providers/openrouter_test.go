package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/pricing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOpenRouterResponse struct {
	Data []mockOpenRouterModel `json:"data"`
}

type mockOpenRouterModel struct {
	ID      string                `json:"id"`
	Pricing mockOpenRouterPricing `json:"pricing"`
}

type mockOpenRouterPricing struct {
	Prompt         string `json:"prompt"`
	Completion     string `json:"completion"`
	InputCacheRead string `json:"input_cache_read,omitempty"`
	Request        string `json:"request,omitempty"`
}

func TestOpenRouterProvider_Name(t *testing.T) {
	p := NewOpenRouterProvider()
	assert.Equal(t, "openrouter", p.Name())
}

func TestOpenRouterProvider_RefreshInterval(t *testing.T) {
	p := NewOpenRouterProvider()
	assert.Equal(t, 6*time.Hour, p.RefreshInterval())

	// Test custom TTL
	p2 := NewOpenRouterProvider(WithTTL(2 * time.Hour))
	assert.Equal(t, 2*time.Hour, p2.RefreshInterval())
}

func TestOpenRouterProvider_SupportsModel(t *testing.T) {
	p := NewOpenRouterProvider()

	// Models with provider prefix should be supported
	assert.True(t, p.SupportsModel("anthropic/claude-3-opus"))
	assert.True(t, p.SupportsModel("openai/gpt-4"))
	assert.True(t, p.SupportsModel("google/gemini-pro"))

	// Models without provider prefix should not be supported
	assert.False(t, p.SupportsModel("gpt-4"))
	assert.False(t, p.SupportsModel("claude-3-opus"))
}

func TestOpenRouterProvider_FetchAllPricing_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/models", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		response := mockOpenRouterResponse{
			Data: []mockOpenRouterModel{
				{
					ID: "anthropic/claude-3-opus",
					Pricing: mockOpenRouterPricing{
						Prompt:         "0.000015",
						Completion:     "0.000075",
						InputCacheRead: "0.0000075",
					},
				},
				{
					ID: "openai/gpt-4",
					Pricing: mockOpenRouterPricing{
						Prompt:     "0.00003",
						Completion: "0.00006",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewOpenRouterProvider(WithBaseURL(server.URL))

	pricingList, err := p.FetchAllPricing(context.Background())
	require.NoError(t, err)
	require.Len(t, pricingList, 2)

	// Find Claude model
	var claudePricing *pricing.ModelPricing
	for _, mp := range pricingList {
		if mp.CanonicalModelName == "anthropic/claude-3-opus" {
			claudePricing = mp
			break
		}
	}
	require.NotNil(t, claudePricing)

	assert.Equal(t, "anthropic/claude-3-opus", claudePricing.CanonicalModelName)
	assert.Equal(t, "openrouter", claudePricing.Provider)
	assert.NotNil(t, claudePricing.InputTokenPrice)
	assert.NotNil(t, claudePricing.OutputTokenPrice)
	assert.NotNil(t, claudePricing.CacheReadPrice)
	assert.InDelta(t, 0.000015, *claudePricing.InputTokenPrice, 0.0000001)
	assert.InDelta(t, 0.000075, *claudePricing.OutputTokenPrice, 0.0000001)
	assert.InDelta(t, 0.0000075, *claudePricing.CacheReadPrice, 0.0000001)
	assert.Equal(t, pricing.SourceProviderAPI, claudePricing.InputTokenSource)
	assert.Equal(t, pricing.SourceProviderAPI, claudePricing.OutputTokenSource)
	assert.Equal(t, pricing.SourceProviderAPI, claudePricing.CacheReadSource)
}

func TestOpenRouterProvider_FetchModelPricing_Found(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := mockOpenRouterResponse{
			Data: []mockOpenRouterModel{
				{
					ID: "anthropic/claude-3-sonnet",
					Pricing: mockOpenRouterPricing{
						Prompt:     "0.000003",
						Completion: "0.000015",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewOpenRouterProvider(WithBaseURL(server.URL))

	mp, err := p.FetchModelPricing(context.Background(), "anthropic/claude-3-sonnet")
	require.NoError(t, err)
	require.NotNil(t, mp)

	assert.Equal(t, "anthropic/claude-3-sonnet", mp.CanonicalModelName)
	assert.NotNil(t, mp.InputTokenPrice)
	assert.InDelta(t, 0.000003, *mp.InputTokenPrice, 0.0000001)
}

func TestOpenRouterProvider_FetchModelPricing_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := mockOpenRouterResponse{
			Data: []mockOpenRouterModel{
				{
					ID: "anthropic/claude-3-opus",
					Pricing: mockOpenRouterPricing{
						Prompt:     "0.000015",
						Completion: "0.000075",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewOpenRouterProvider(WithBaseURL(server.URL))

	mp, err := p.FetchModelPricing(context.Background(), "openai/gpt-4")
	require.NoError(t, err)
	assert.Nil(t, mp) // Not found returns nil, nil
}

func TestOpenRouterProvider_FetchAllPricing_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := NewOpenRouterProvider(WithBaseURL(server.URL))

	_, err := p.FetchAllPricing(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestOpenRouterProvider_FetchAllPricing_UsesCache(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		response := mockOpenRouterResponse{
			Data: []mockOpenRouterModel{
				{
					ID: "test/model",
					Pricing: mockOpenRouterPricing{
						Prompt:     "0.000001",
						Completion: "0.000002",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewOpenRouterProvider(WithBaseURL(server.URL), WithTTL(1*time.Hour))

	// First call - should hit API
	_, err := p.FetchAllPricing(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Second call - should use cache
	_, err = p.FetchAllPricing(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, callCount) // Still 1, no new API call
}

func TestOpenRouterProvider_ClearCache_ForcesRefresh(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		response := mockOpenRouterResponse{
			Data: []mockOpenRouterModel{
				{
					ID: "test/model",
					Pricing: mockOpenRouterPricing{
						Prompt:     "0.000001",
						Completion: "0.000002",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewOpenRouterProvider(WithBaseURL(server.URL))

	// First call
	_, err := p.FetchAllPricing(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Clear cache
	p.ClearCache()

	// Second call - should hit API again
	_, err = p.FetchAllPricing(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestOpenRouterProvider_CacheStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := mockOpenRouterResponse{
			Data: []mockOpenRouterModel{
				{ID: "model-1", Pricing: mockOpenRouterPricing{Prompt: "0.001", Completion: "0.002"}},
				{ID: "model-2", Pricing: mockOpenRouterPricing{Prompt: "0.001", Completion: "0.002"}},
				{ID: "model-3", Pricing: mockOpenRouterPricing{Prompt: "0.001", Completion: "0.002"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewOpenRouterProvider(WithBaseURL(server.URL), WithTTL(1*time.Hour))

	// Before any fetch
	status := p.CacheStatus()
	assert.Equal(t, 0, status.ModelCount)
	assert.True(t, status.IsStale) // Empty cache is stale

	// After fetch
	_, err := p.FetchAllPricing(context.Background())
	require.NoError(t, err)

	status = p.CacheStatus()
	assert.Equal(t, 3, status.ModelCount)
	assert.Equal(t, "openrouter", status.Provider)
	assert.False(t, status.IsStale)
	assert.True(t, status.ExpiresAt.After(time.Now()))
}

func TestOpenRouterProvider_FetchAllPricing_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	p := NewOpenRouterProvider(WithBaseURL(server.URL))

	_, err := p.FetchAllPricing(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestOpenRouterProvider_FetchAllPricing_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := mockOpenRouterResponse{Data: []mockOpenRouterModel{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewOpenRouterProvider(WithBaseURL(server.URL))

	pricingList, err := p.FetchAllPricing(context.Background())
	require.NoError(t, err)
	assert.Empty(t, pricingList)
}

func TestOpenRouterProvider_FetchAllPricing_ZeroPriceOmitted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := mockOpenRouterResponse{
			Data: []mockOpenRouterModel{
				{
					ID: "test/model",
					Pricing: mockOpenRouterPricing{
						Prompt:         "0.000001",
						Completion:     "0", // Zero price
						InputCacheRead: "",  // Empty string
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewOpenRouterProvider(WithBaseURL(server.URL))

	pricingList, err := p.FetchAllPricing(context.Background())
	require.NoError(t, err)
	require.Len(t, pricingList, 1)

	mp := pricingList[0]
	assert.NotNil(t, mp.InputTokenPrice) // Has value
	assert.Nil(t, mp.OutputTokenPrice)   // Zero price omitted
	assert.Nil(t, mp.CacheReadPrice)     // Empty string omitted
}

func TestOpenRouterProvider_FetchAllPricing_SkipsEmptyModelID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := mockOpenRouterResponse{
			Data: []mockOpenRouterModel{
				{
					ID: "", // Empty ID
					Pricing: mockOpenRouterPricing{
						Prompt:     "0.000001",
						Completion: "0.000002",
					},
				},
				{
					ID: "valid/model",
					Pricing: mockOpenRouterPricing{
						Prompt:     "0.000001",
						Completion: "0.000002",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewOpenRouterProvider(WithBaseURL(server.URL))

	pricingList, err := p.FetchAllPricing(context.Background())
	require.NoError(t, err)
	require.Len(t, pricingList, 1) // Only valid model included

	assert.Equal(t, "valid/model", pricingList[0].CanonicalModelName)
}

func TestParsePricingValue(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"0.000015", 0.000015},
		{"0.0000075", 0.0000075},
		{"0", 0},
		{"", 0},
		{"   0.000001   ", 0.000001},
		{"invalid", 0},
		{"-0.001", -0.001}, // Negative values parsed (though unusual)
	}

	for _, tc := range tests {
		result := parsePricingValue(tc.input)
		assert.InDelta(t, tc.expected, result, 0.0000001, "input: %q", tc.input)
	}
}

func TestOpenRouterProvider_FetchAllPricing_FallsBackToCacheOnError(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call succeeds
			response := mockOpenRouterResponse{
				Data: []mockOpenRouterModel{
					{ID: "test/model", Pricing: mockOpenRouterPricing{Prompt: "0.001", Completion: "0.002"}},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			// Subsequent calls fail
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	// Use very short TTL so cache expires quickly
	p := NewOpenRouterProvider(WithBaseURL(server.URL), WithTTL(1*time.Millisecond))

	// First call - succeeds and populates cache
	pricingList, err := p.FetchAllPricing(context.Background())
	require.NoError(t, err)
	require.Len(t, pricingList, 1)

	// Wait for cache to expire
	time.Sleep(5 * time.Millisecond)

	// Second call - API fails but returns cached data
	pricingList, err = p.FetchAllPricing(context.Background())
	require.NoError(t, err)
	require.Len(t, pricingList, 1) // Falls back to cache
}
