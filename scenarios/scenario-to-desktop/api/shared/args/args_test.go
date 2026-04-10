package args

import (
	"errors"
	"testing"
)

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		key      string
		def      string
		expected string
	}{
		{"present string", map[string]interface{}{"k": "val"}, "k", "def", "val"},
		{"missing key", map[string]interface{}{}, "k", "def", "def"},
		{"nil map", nil, "k", "def", "def"},
		{"wrong type int", map[string]interface{}{"k": 42}, "k", "def", "def"},
		{"wrong type bool", map[string]interface{}{"k": true}, "k", "def", "def"},
		{"empty string value", map[string]interface{}{"k": ""}, "k", "def", ""},
		{"empty default", map[string]interface{}{}, "k", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetString(tc.args, tc.key, tc.def)
			if got != tc.expected {
				t.Errorf("GetString() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestGetStringArray(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		key      string
		expected []string
	}{
		{
			"interface slice with strings",
			map[string]interface{}{"k": []interface{}{"a", "b", "c"}},
			"k",
			[]string{"a", "b", "c"},
		},
		{
			"interface slice with mixed types filters non-strings",
			map[string]interface{}{"k": []interface{}{"a", 42, "c"}},
			"k",
			[]string{"a", "c"},
		},
		{
			"typed string slice",
			map[string]interface{}{"k": []string{"x", "y"}},
			"k",
			[]string{"x", "y"},
		},
		{
			"empty interface slice",
			map[string]interface{}{"k": []interface{}{}},
			"k",
			[]string{},
		},
		{
			"missing key",
			map[string]interface{}{},
			"k",
			nil,
		},
		{
			"wrong type",
			map[string]interface{}{"k": "not-a-slice"},
			"k",
			nil,
		},
		{
			"nil map",
			nil,
			"k",
			nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetStringArray(tc.args, tc.key)
			if tc.expected == nil {
				if got != nil {
					t.Errorf("GetStringArray() = %v, want nil", got)
				}
				return
			}
			if len(got) != len(tc.expected) {
				t.Fatalf("GetStringArray() len = %d, want %d", len(got), len(tc.expected))
			}
			for i, v := range got {
				if v != tc.expected[i] {
					t.Errorf("GetStringArray()[%d] = %q, want %q", i, v, tc.expected[i])
				}
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		key      string
		def      bool
		expected bool
	}{
		{"present true", map[string]interface{}{"k": true}, "k", false, true},
		{"present false", map[string]interface{}{"k": false}, "k", true, false},
		{"missing key default true", map[string]interface{}{}, "k", true, true},
		{"missing key default false", map[string]interface{}{}, "k", false, false},
		{"wrong type", map[string]interface{}{"k": "true"}, "k", false, false},
		{"nil map", nil, "k", true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetBool(tc.args, tc.key, tc.def)
			if got != tc.expected {
				t.Errorf("GetBool() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		key      string
		def      int
		expected int
	}{
		{"float64 from JSON", map[string]interface{}{"k": float64(42)}, "k", 0, 42},
		{"float64 with decimals truncates", map[string]interface{}{"k": float64(3.9)}, "k", 0, 3},
		{"native int", map[string]interface{}{"k": int(7)}, "k", 0, 7},
		{"zero float64", map[string]interface{}{"k": float64(0)}, "k", 99, 0},
		{"missing key", map[string]interface{}{}, "k", 5, 5},
		{"wrong type string", map[string]interface{}{"k": "42"}, "k", 5, 5},
		{"nil map", nil, "k", 10, 10},
		{"negative float64", map[string]interface{}{"k": float64(-3)}, "k", 0, -3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetInt(tc.args, tc.key, tc.def)
			if got != tc.expected {
				t.Errorf("GetInt() = %d, want %d", got, tc.expected)
			}
		})
	}
}

func TestGetMap(t *testing.T) {
	inner := map[string]interface{}{"nested": "value"}
	tests := []struct {
		name    string
		args    map[string]interface{}
		key     string
		wantNil bool
	}{
		{"present map", map[string]interface{}{"k": inner}, "k", false},
		{"missing key", map[string]interface{}{}, "k", true},
		{"wrong type", map[string]interface{}{"k": "not-a-map"}, "k", true},
		{"nil map", nil, "k", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetMap(tc.args, tc.key)
			if tc.wantNil {
				if got != nil {
					t.Errorf("GetMap() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Fatal("GetMap() = nil, want non-nil")
				}
				if got["nested"] != "value" {
					t.Errorf("GetMap()[nested] = %v, want 'value'", got["nested"])
				}
			}
		})
	}
}

func TestRequireString(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		key     string
		want    string
		wantErr bool
	}{
		{"present", map[string]interface{}{"k": "val"}, "k", "val", false},
		{"missing", map[string]interface{}{}, "k", "", true},
		{"empty string", map[string]interface{}{"k": ""}, "k", "", true},
		{"wrong type", map[string]interface{}{"k": 42}, "k", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RequireString(tc.args, tc.key)
			if tc.wantErr {
				if err == nil {
					t.Fatal("RequireString() error = nil, want error")
				}
				var reqErr *RequiredArgError
				if !errors.As(err, &reqErr) {
					t.Fatalf("error type = %T, want *RequiredArgError", err)
				}
				if reqErr.Key != tc.key {
					t.Errorf("RequiredArgError.Key = %q, want %q", reqErr.Key, tc.key)
				}
				return
			}
			if err != nil {
				t.Fatalf("RequireString() unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("RequireString() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRequireStringArray(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		key     string
		wantLen int
		wantErr bool
	}{
		{"present", map[string]interface{}{"k": []interface{}{"a", "b"}}, "k", 2, false},
		{"missing", map[string]interface{}{}, "k", 0, true},
		{"empty slice", map[string]interface{}{"k": []interface{}{}}, "k", 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RequireStringArray(tc.args, tc.key)
			if tc.wantErr {
				if err == nil {
					t.Fatal("RequireStringArray() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("RequireStringArray() unexpected error: %v", err)
			}
			if len(got) != tc.wantLen {
				t.Errorf("RequireStringArray() len = %d, want %d", len(got), tc.wantLen)
			}
		})
	}
}

func TestRequireMap(t *testing.T) {
	inner := map[string]interface{}{"a": 1}
	tests := []struct {
		name    string
		args    map[string]interface{}
		key     string
		wantErr bool
	}{
		{"present", map[string]interface{}{"k": inner}, "k", false},
		{"missing", map[string]interface{}{}, "k", true},
		{"wrong type", map[string]interface{}{"k": "str"}, "k", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RequireMap(tc.args, tc.key)
			if tc.wantErr {
				if err == nil {
					t.Fatal("RequireMap() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("RequireMap() unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("RequireMap() = nil, want non-nil")
			}
		})
	}
}

func TestRequiredArgError_Error(t *testing.T) {
	err := &RequiredArgError{Key: "my_field"}
	if err.Error() != "my_field is required" {
		t.Errorf("Error() = %q, want %q", err.Error(), "my_field is required")
	}
}
