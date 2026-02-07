package ssh

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type matrixEndpoint struct {
	RequestRequiredFields  []string `json:"request_required_fields"`
	ResponseRequiredFields []string `json:"response_required_fields"`
	ResponseStatuses       []string `json:"response_statuses"`
}

type contractMatrix struct {
	Endpoints             map[string]matrixEndpoint `json:"endpoints"`
	LegacyDisallowedField []string                  `json:"legacy_disallowed_fields"`
}

func loadContractMatrix(t *testing.T) contractMatrix {
	t.Helper()
	path := filepath.Join("..", "..", "contracts", "ssh-contract-matrix.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read matrix: %v", err)
	}
	var m contractMatrix
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("decode matrix: %v", err)
	}
	return m
}

func TestContractMatrixCLIRequestsIncludeRequiredFields(t *testing.T) {
	t.Parallel()

	matrix := loadContractMatrix(t)

	assertJSONHasFields(t, "DELETE /ssh/keys request", DeleteRequest{
		KeyPath: "/home/me/.ssh/id_ed25519",
	}, matrix.Endpoints["/api/v1/ssh/keys:delete"].RequestRequiredFields)

	assertJSONHasFields(t, "POST /ssh/test request", TestRequest{
		Host:    "example.com",
		Port:    22,
		User:    "root",
		KeyPath: "/home/me/.ssh/id_ed25519",
	}, matrix.Endpoints["/api/v1/ssh/test"].RequestRequiredFields)

	assertJSONHasFields(t, "POST /ssh/copy-key request", CopyKeyRequest{
		Host:     "example.com",
		Port:     22,
		User:     "root",
		KeyPath:  "/home/me/.ssh/id_ed25519",
		Password: "secret",
	}, matrix.Endpoints["/api/v1/ssh/copy-key"].RequestRequiredFields)
}

func TestContractMatrixCLITagsExcludeLegacyFields(t *testing.T) {
	t.Parallel()

	matrix := loadContractMatrix(t)
	disallowed := toSet(matrix.LegacyDisallowedField)

	types := []reflect.Type{
		reflect.TypeFor[DeleteRequest](),
		reflect.TypeFor[TestRequest](),
		reflect.TypeFor[CopyKeyRequest](),
		reflect.TypeFor[Outcome](),
		reflect.TypeFor[GenerateResponse](),
		reflect.TypeFor[TestResponse](),
		reflect.TypeFor[CopyKeyResponse](),
	}

	for _, typ := range types {
		fields := jsonFieldSet(typ)
		for name := range disallowed {
			if fields[name] {
				t.Fatalf("%s must not contain legacy JSON field %q", typ.Name(), name)
			}
		}
	}
}

func TestContractMatrixCLIResponsesAcceptRequiredFields(t *testing.T) {
	t.Parallel()

	matrix := loadContractMatrix(t)

	testJSON := []byte(`{"ok":true,"status":"success","timestamp":"2026-01-01T00:00:00Z"}`)
	var testResp TestResponse
	if err := json.Unmarshal(testJSON, &testResp); err != nil {
		t.Fatalf("decode TestResponse: %v", err)
	}
	assertResponseFieldsPresent(t, "TestResponse", matrix.Endpoints["/api/v1/ssh/test"].ResponseRequiredFields, testJSON)

	copyJSON := []byte(`{"ok":true,"status":"already_exists","timestamp":"2026-01-01T00:00:00Z","key_copied":false,"already_exists":true}`)
	var copyResp CopyKeyResponse
	if err := json.Unmarshal(copyJSON, &copyResp); err != nil {
		t.Fatalf("decode CopyKeyResponse: %v", err)
	}
	assertResponseFieldsPresent(t, "CopyKeyResponse", matrix.Endpoints["/api/v1/ssh/copy-key"].ResponseRequiredFields, copyJSON)
}

func assertJSONHasFields(t *testing.T, label string, payload any, required []string) {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("%s marshal: %v", label, err)
	}
	assertResponseFieldsPresent(t, label, required, encoded)
}

func assertResponseFieldsPresent(t *testing.T, label string, required []string, raw []byte) {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("%s decode: %v", label, err)
	}
	for _, field := range required {
		if _, ok := m[field]; !ok {
			t.Fatalf("%s missing field %q", label, field)
		}
	}
}

func jsonFieldSet(typ reflect.Type) map[string]bool {
	fields := map[string]bool{}
	var walk func(reflect.Type)
	walk = func(t reflect.Type) {
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return
		}
		for i := range t.NumField() {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			if f.Anonymous {
				walk(f.Type)
				continue
			}
			tag := f.Tag.Get("json")
			if tag == "" || tag == "-" {
				continue
			}
			name := tag
			for i := 0; i < len(tag); i++ {
				if tag[i] == ',' {
					name = tag[:i]
					break
				}
			}
			if name != "" {
				fields[name] = true
			}
		}
	}
	walk(typ)
	return fields
}

func toSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, v := range values {
		out[v] = true
	}
	return out
}
