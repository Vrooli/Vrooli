package main

import (
	"strings"
	"testing"
)

func TestResolvePlanFromMetadata(t *testing.T) {
	tests := []struct {
		name          string
		metadata      map[string]interface{}
		defaultBundle string
		wantPlanTier  string
		wantBundleKey string
		wantPriceID   string
	}{
		{
			name:          "empty metadata uses default bundle",
			metadata:      map[string]interface{}{},
			defaultBundle: "business_suite",
			wantBundleKey: "business_suite",
		},
		{
			name: "extracts plan_tier from metadata",
			metadata: map[string]interface{}{
				"plan_tier": "pro",
			},
			defaultBundle: "default",
			wantPlanTier:  "pro",
			wantBundleKey: "default",
		},
		{
			name: "extracts bundle_key from metadata",
			metadata: map[string]interface{}{
				"bundle_key": "custom_bundle",
			},
			defaultBundle: "default",
			wantBundleKey: "custom_bundle",
		},
		{
			name: "extracts price_id from metadata",
			metadata: map[string]interface{}{
				"price_id": "price_12345",
			},
			defaultBundle: "default",
			wantPriceID:   "price_12345",
			wantBundleKey: "default",
		},
		{
			name: "extracts all fields from metadata",
			metadata: map[string]interface{}{
				"plan_tier":  "enterprise",
				"bundle_key": "enterprise_bundle",
				"price_id":   "price_enterprise",
			},
			defaultBundle: "default",
			wantPlanTier:  "enterprise",
			wantBundleKey: "enterprise_bundle",
			wantPriceID:   "price_enterprise",
		},
		{
			name: "ignores non-string values",
			metadata: map[string]interface{}{
				"plan_tier":  123,
				"bundle_key": true,
				"price_id":   nil,
			},
			defaultBundle: "default",
			wantBundleKey: "default",
		},
		{
			name: "ignores empty string values",
			metadata: map[string]interface{}{
				"plan_tier":  "",
				"bundle_key": "",
				"price_id":   "",
			},
			defaultBundle: "default",
			wantBundleKey: "default",
		},
		{
			name:          "nil metadata uses default bundle",
			metadata:      nil,
			defaultBundle: "test_bundle",
			wantBundleKey: "test_bundle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolvePlanFromMetadata(tt.metadata, tt.defaultBundle)
			if result.PlanTier != tt.wantPlanTier {
				t.Errorf("PlanTier = %q, want %q", result.PlanTier, tt.wantPlanTier)
			}
			if result.BundleKey != tt.wantBundleKey {
				t.Errorf("BundleKey = %q, want %q", result.BundleKey, tt.wantBundleKey)
			}
			if result.PriceID != tt.wantPriceID {
				t.Errorf("PriceID = %q, want %q", result.PriceID, tt.wantPriceID)
			}
		})
	}
}

func TestInferPlanTierFromPriceID(t *testing.T) {
	tests := []struct {
		name        string
		priceID     string
		currentTier string
		want        string
	}{
		{
			name:        "returns currentTier if already set",
			priceID:     "price_pro_monthly",
			currentTier: "business",
			want:        "business",
		},
		{
			name:        "returns currentTier with whitespace",
			priceID:     "price_pro_monthly",
			currentTier: "  pro  ",
			want:        "  pro  ",
		},
		{
			name:        "returns empty for empty priceID and currentTier",
			priceID:     "",
			currentTier: "",
			want:        "",
		},
		{
			name:        "returns empty for whitespace priceID",
			priceID:     "   ",
			currentTier: "",
			want:        "",
		},
		{
			name:        "infers pro from price ID",
			priceID:     "price_pro_monthly",
			currentTier: "",
			want:        "pro",
		},
		{
			name:        "infers solo from price ID",
			priceID:     "price_solo_annual",
			currentTier: "",
			want:        "solo",
		},
		{
			name:        "infers business from price ID",
			priceID:     "price_business_monthly",
			currentTier: "",
			want:        "business",
		},
		{
			name:        "infers studio from price ID",
			priceID:     "price_studio_monthly",
			currentTier: "",
			want:        "studio",
		},
		{
			name:        "returns empty for unrecognized price ID",
			priceID:     "price_xyz_unknown",
			currentTier: "",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferPlanTierFromPriceID(tt.priceID, tt.currentTier)
			if got != tt.want {
				t.Errorf("InferPlanTierFromPriceID(%q, %q) = %q, want %q", tt.priceID, tt.currentTier, got, tt.want)
			}
		})
	}
}

func TestValidatePlanTier(t *testing.T) {
	tests := []struct {
		name           string
		planTier       string
		priceID        string
		subscriptionID string
		want           string
	}{
		{
			name:           "returns empty for empty planTier",
			planTier:       "",
			priceID:        "price_123",
			subscriptionID: "sub_123",
			want:           "",
		},
		{
			name:           "returns empty for whitespace planTier",
			planTier:       "   ",
			priceID:        "price_123",
			subscriptionID: "sub_123",
			want:           "",
		},
		{
			name:           "returns valid tier for solo",
			planTier:       "solo",
			priceID:        "price_123",
			subscriptionID: "sub_123",
			want:           "solo",
		},
		{
			name:           "returns valid tier for pro",
			planTier:       "pro",
			priceID:        "price_123",
			subscriptionID: "sub_123",
			want:           "pro",
		},
		{
			name:           "returns valid tier for business",
			planTier:       "business",
			priceID:        "price_123",
			subscriptionID: "sub_123",
			want:           "business",
		},
		{
			name:           "returns valid tier for studio",
			planTier:       "studio",
			priceID:        "price_123",
			subscriptionID: "sub_123",
			want:           "studio",
		},
		{
			name:           "returns valid tier for free",
			planTier:       "free",
			priceID:        "price_123",
			subscriptionID: "sub_123",
			want:           "free",
		},
		{
			name:           "returns empty for invalid tier",
			planTier:       "invalid_tier",
			priceID:        "price_123",
			subscriptionID: "sub_123",
			want:           "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePlanTier(tt.planTier, tt.priceID, tt.subscriptionID)
			if got != tt.want {
				t.Errorf("ValidatePlanTier(%q, %q, %q) = %q, want %q", tt.planTier, tt.priceID, tt.subscriptionID, got, tt.want)
			}
		})
	}
}

func TestExtractCouponIDFromDiscount(t *testing.T) {
	tests := []struct {
		name     string
		discount map[string]interface{}
		want     string
	}{
		{
			name:     "returns empty for nil discount",
			discount: nil,
			want:     "",
		},
		{
			name:     "returns empty for empty discount",
			discount: map[string]interface{}{},
			want:     "",
		},
		{
			name: "returns empty when coupon is not a map",
			discount: map[string]interface{}{
				"coupon": "not_a_map",
			},
			want: "",
		},
		{
			name: "returns empty when coupon has no id",
			discount: map[string]interface{}{
				"coupon": map[string]interface{}{
					"name": "Test Coupon",
				},
			},
			want: "",
		},
		{
			name: "returns empty when id is not a string",
			discount: map[string]interface{}{
				"coupon": map[string]interface{}{
					"id": 12345,
				},
			},
			want: "",
		},
		{
			name: "extracts coupon id successfully",
			discount: map[string]interface{}{
				"coupon": map[string]interface{}{
					"id":   "coupon_abc123",
					"name": "Test Coupon",
				},
			},
			want: "coupon_abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCouponIDFromDiscount(tt.discount)
			if got != tt.want {
				t.Errorf("ExtractCouponIDFromDiscount() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractCouponFromDiscountAmount(t *testing.T) {
	tests := []struct {
		name string
		tda  map[string]interface{}
		want string
	}{
		{
			name: "returns empty for nil tda",
			tda:  nil,
			want: "",
		},
		{
			name: "returns empty for empty tda",
			tda:  map[string]interface{}{},
			want: "",
		},
		{
			name: "returns empty when discount is not a map",
			tda: map[string]interface{}{
				"discount": "not_a_map",
			},
			want: "",
		},
		{
			name: "extracts coupon id from nested discount",
			tda: map[string]interface{}{
				"discount": map[string]interface{}{
					"coupon": map[string]interface{}{
						"id": "intro_50_off",
					},
				},
			},
			want: "intro_50_off",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCouponFromDiscountAmount(tt.tda)
			if got != tt.want {
				t.Errorf("ExtractCouponFromDiscountAmount() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractFirstSubscriptionItem(t *testing.T) {
	t.Run("returns empty for nil items data", func(t *testing.T) {
		sub := &stripeSubscription{}
		sub.Items.Data = nil
		got := ExtractFirstSubscriptionItem(sub)
		if got != "" {
			t.Errorf("ExtractFirstSubscriptionItem() = %q, want empty", got)
		}
	})

	t.Run("returns empty for empty items data", func(t *testing.T) {
		sub := &stripeSubscription{}
		sub.Items.Data = []struct {
			Price stripePriceRef `json:"price"`
		}{}
		got := ExtractFirstSubscriptionItem(sub)
		if got != "" {
			t.Errorf("ExtractFirstSubscriptionItem() = %q, want empty", got)
		}
	})

	t.Run("extracts first item price ID", func(t *testing.T) {
		sub := &stripeSubscription{}
		sub.Items.Data = []struct {
			Price stripePriceRef `json:"price"`
		}{
			{Price: stripePriceRef{ID: "price_first"}},
			{Price: stripePriceRef{ID: "price_second"}},
		}
		got := ExtractFirstSubscriptionItem(sub)
		if got != "price_first" {
			t.Errorf("ExtractFirstSubscriptionItem() = %q, want price_first", got)
		}
	})
}

func TestSafeStringFromMap(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{
			name: "returns empty for nil map",
			m:    nil,
			key:  "key",
			want: "",
		},
		{
			name: "returns empty for missing key",
			m:    map[string]interface{}{"other": "value"},
			key:  "key",
			want: "",
		},
		{
			name: "returns empty for non-string value",
			m:    map[string]interface{}{"key": 123},
			key:  "key",
			want: "",
		},
		{
			name: "returns string value",
			m:    map[string]interface{}{"key": "value"},
			key:  "key",
			want: "value",
		},
		{
			name: "returns empty string for empty string value",
			m:    map[string]interface{}{"key": ""},
			key:  "key",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeStringFromMap(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("SafeStringFromMap() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSafeInt64FromMap(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want int64
	}{
		{
			name: "returns 0 for nil map",
			m:    nil,
			key:  "key",
			want: 0,
		},
		{
			name: "returns 0 for missing key",
			m:    map[string]interface{}{"other": 123},
			key:  "key",
			want: 0,
		},
		{
			name: "returns 0 for string value",
			m:    map[string]interface{}{"key": "not a number"},
			key:  "key",
			want: 0,
		},
		{
			name: "returns value from float64 (JSON default)",
			m:    map[string]interface{}{"key": float64(42.0)},
			key:  "key",
			want: 42,
		},
		{
			name: "returns value from int",
			m:    map[string]interface{}{"key": int(100)},
			key:  "key",
			want: 100,
		},
		{
			name: "returns value from int64",
			m:    map[string]interface{}{"key": int64(9999)},
			key:  "key",
			want: 9999,
		},
		{
			name: "truncates float64 to int64",
			m:    map[string]interface{}{"key": float64(42.9)},
			key:  "key",
			want: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeInt64FromMap(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("SafeInt64FromMap() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSafeMapFromMap(t *testing.T) {
	tests := []struct {
		name    string
		m       map[string]interface{}
		key     string
		wantNil bool
	}{
		{
			name:    "returns nil for nil map",
			m:       nil,
			key:     "key",
			wantNil: true,
		},
		{
			name:    "returns nil for missing key",
			m:       map[string]interface{}{"other": map[string]interface{}{}},
			key:     "key",
			wantNil: true,
		},
		{
			name:    "returns nil for non-map value",
			m:       map[string]interface{}{"key": "not a map"},
			key:     "key",
			wantNil: true,
		},
		{
			name:    "returns nested map",
			m:       map[string]interface{}{"key": map[string]interface{}{"nested": "value"}},
			key:     "key",
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeMapFromMap(tt.m, tt.key)
			if tt.wantNil && got != nil {
				t.Errorf("SafeMapFromMap() = %v, want nil", got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("SafeMapFromMap() = nil, want non-nil")
			}
		})
	}
}

func TestSafeArrayFromMap(t *testing.T) {
	tests := []struct {
		name    string
		m       map[string]interface{}
		key     string
		wantNil bool
	}{
		{
			name:    "returns nil for nil map",
			m:       nil,
			key:     "key",
			wantNil: true,
		},
		{
			name:    "returns nil for missing key",
			m:       map[string]interface{}{"other": []interface{}{}},
			key:     "key",
			wantNil: true,
		},
		{
			name:    "returns nil for non-array value",
			m:       map[string]interface{}{"key": "not an array"},
			key:     "key",
			wantNil: true,
		},
		{
			name:    "returns array",
			m:       map[string]interface{}{"key": []interface{}{"a", "b", "c"}},
			key:     "key",
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeArrayFromMap(tt.m, tt.key)
			if tt.wantNil && got != nil {
				t.Errorf("SafeArrayFromMap() = %v, want nil", got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("SafeArrayFromMap() = nil, want non-nil")
			}
		})
	}
}

func TestBuildSubscriptionUpsertSQL(t *testing.T) {
	sql := BuildSubscriptionUpsertSQL()

	// Verify the SQL contains expected parts
	if sql == "" {
		t.Error("BuildSubscriptionUpsertSQL() returned empty string")
	}

	// Check for INSERT statement
	if !strings.Contains(sql, "INSERT INTO subscriptions") {
		t.Error("SQL should contain INSERT INTO subscriptions")
	}

	// Check for ON CONFLICT clause
	if !strings.Contains(sql, "ON CONFLICT (subscription_id)") {
		t.Error("SQL should contain ON CONFLICT clause")
	}

	// Check for key columns
	expectedCols := []string{"subscription_id", "customer_id", "customer_email", "status", "plan_tier", "price_id", "bundle_key"}
	for _, col := range expectedCols {
		if !strings.Contains(sql, col) {
			t.Errorf("SQL should contain column %q", col)
		}
	}
}

func TestCoalesceString(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{
			name:   "returns empty for no values",
			values: []string{},
			want:   "",
		},
		{
			name:   "returns empty for all empty values",
			values: []string{"", "", ""},
			want:   "",
		},
		{
			name:   "returns empty for all whitespace values",
			values: []string{"  ", "\t", "\n"},
			want:   "",
		},
		{
			name:   "returns first non-empty value",
			values: []string{"", "first", "second"},
			want:   "first",
		},
		{
			name:   "returns first value if non-empty",
			values: []string{"first", "second"},
			want:   "first",
		},
		{
			name:   "skips whitespace-only values",
			values: []string{"  ", "valid", "other"},
			want:   "valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CoalesceString(tt.values...)
			if got != tt.want {
				t.Errorf("CoalesceString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChooseUserIdentityFromSubscription(t *testing.T) {
	tests := []struct {
		name     string
		userHint string
		sub      *stripeSubscription
		want     string
	}{
		{
			name:     "returns trimmed userHint if provided",
			userHint: "  user@example.com  ",
			sub:      &stripeSubscription{CustomerEmail: "other@example.com", Customer: "cus_123"},
			want:     "user@example.com",
		},
		{
			name:     "returns customer email if userHint is empty",
			userHint: "",
			sub:      &stripeSubscription{CustomerEmail: "customer@example.com", Customer: "cus_123"},
			want:     "customer@example.com",
		},
		{
			name:     "returns customer email if userHint is whitespace",
			userHint: "   ",
			sub:      &stripeSubscription{CustomerEmail: "customer@example.com", Customer: "cus_123"},
			want:     "customer@example.com",
		},
		{
			name:     "returns customer ID if email is empty",
			userHint: "",
			sub:      &stripeSubscription{CustomerEmail: "", Customer: "cus_fallback"},
			want:     "cus_fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ChooseUserIdentityFromSubscription(tt.userHint, tt.sub)
			if got != tt.want {
				t.Errorf("ChooseUserIdentityFromSubscription() = %q, want %q", got, tt.want)
			}
		})
	}
}
