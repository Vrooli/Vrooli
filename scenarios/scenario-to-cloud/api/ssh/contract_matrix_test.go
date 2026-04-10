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
	Version               string                    `json:"version"`
	OutcomeStatuses       []string                  `json:"outcome_statuses"`
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

func TestContractMatrixOutcomeStatusesMatchAPIConstants(t *testing.T) {
	t.Parallel()

	matrix := loadContractMatrix(t)
	want := toSet(matrix.OutcomeStatuses)
	got := toSet([]string{
		StatusSuccess,
		StatusAlreadyExists,
		StatusNotFound,
		StatusAuthFailed,
		StatusTimeout,
		StatusHostUnreachable,
		StatusHostKeyChanged,
		StatusIPv6Unavailable,
		StatusInvalidInput,
		StatusDiskFull,
		StatusDNSFailed,
		StatusKeyError,
		StatusError,
	})

	assertSetEqual(t, "outcome statuses", got, want)
}

func TestContractMatrixEndpointDTOFields(t *testing.T) {
	t.Parallel()

	matrix := loadContractMatrix(t)

	checkReqFields(t, matrix, "/api/v1/ssh/keys/generate", reflect.TypeFor[GenerateKeyRequest]())
	checkRespFields(t, matrix, "/api/v1/ssh/keys/generate", reflect.TypeFor[GenerateKeyResponse]())

	checkReqFields(t, matrix, "/api/v1/ssh/keys/public", reflect.TypeFor[GetPublicKeyRequest]())
	checkRespFields(t, matrix, "/api/v1/ssh/keys/public", reflect.TypeFor[GetPublicKeyResponse]())

	checkReqFields(t, matrix, "/api/v1/ssh/keys:delete", reflect.TypeFor[DeleteKeyRequest]())
	checkRespFields(t, matrix, "/api/v1/ssh/keys:delete", reflect.TypeFor[DeleteKeyResponse]())

	checkReqFields(t, matrix, "/api/v1/ssh/test", reflect.TypeFor[TestConnectionRequest]())
	checkRespFields(t, matrix, "/api/v1/ssh/test", reflect.TypeFor[TestConnectionResponse]())

	checkReqFields(t, matrix, "/api/v1/ssh/copy-key", reflect.TypeFor[CopyKeyRequest]())
	checkRespFields(t, matrix, "/api/v1/ssh/copy-key", reflect.TypeFor[CopyKeyResponse]())

	checkRespFields(t, matrix, "/api/v1/ssh/keys", reflect.TypeFor[ListKeysResponse]())
}

func TestContractMatrixEndpointStatuses(t *testing.T) {
	t.Parallel()

	matrix := loadContractMatrix(t)

	assertSetEqual(
		t,
		"/api/v1/ssh/test statuses",
		toSet([]string{
			StatusSuccess,
			StatusAuthFailed,
			StatusTimeout,
			StatusHostUnreachable,
			StatusHostKeyChanged,
			StatusIPv6Unavailable,
			StatusKeyError,
			StatusDNSFailed,
			StatusDiskFull,
			StatusNotFound,
			StatusError,
		}),
		toSet(matrix.Endpoints["/api/v1/ssh/test"].ResponseStatuses),
	)

	assertSetEqual(
		t,
		"/api/v1/ssh/copy-key statuses",
		toSet([]string{
			StatusSuccess,
			StatusAlreadyExists,
			StatusAuthFailed,
			StatusIPv6Unavailable,
			StatusKeyError,
			StatusError,
		}),
		toSet(matrix.Endpoints["/api/v1/ssh/copy-key"].ResponseStatuses),
	)
}

func checkReqFields(t *testing.T, matrix contractMatrix, endpoint string, typ reflect.Type) {
	t.Helper()
	want := matrix.Endpoints[endpoint].RequestRequiredFields
	got := jsonFieldSet(typ)
	for _, field := range want {
		if !got[field] {
			t.Fatalf("%s request missing JSON field %q", endpoint, field)
		}
	}
}

func checkRespFields(t *testing.T, matrix contractMatrix, endpoint string, typ reflect.Type) {
	t.Helper()
	want := matrix.Endpoints[endpoint].ResponseRequiredFields
	got := jsonFieldSet(typ)
	for _, field := range want {
		if !got[field] {
			t.Fatalf("%s response missing JSON field %q", endpoint, field)
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

func assertSetEqual(t *testing.T, label string, got, want map[string]bool) {
	t.Helper()
	for v := range want {
		if !got[v] {
			t.Fatalf("%s: missing expected value %q", label, v)
		}
	}
	for v := range got {
		if !want[v] {
			t.Fatalf("%s: unexpected value %q", label, v)
		}
	}
}
