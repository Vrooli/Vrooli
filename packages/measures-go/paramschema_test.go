package measures

import (
	"strings"
	"testing"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

// descriptorImagePath is the committed proto descriptor image relative to this
// package directory (go test runs with the package dir as the working dir).
const descriptorImagePath = "../proto/gen/descriptor/image.binpb"

func loadReader(t *testing.T) *SchemaReader {
	t.Helper()
	r, err := NewSchemaReaderFromFile(descriptorImagePath)
	if err != nil {
		t.Fatalf("NewSchemaReaderFromFile: %v", err)
	}
	return r
}

func byName(params []ParamSchema, name string) (ParamSchema, bool) {
	for _, p := range params {
		if p.Name == name {
			return p, true
		}
	}
	return ParamSchema{}, false
}

// TestDescriptorRoundTrip re-asserts (in CI) that `buf build` preserves the
// buf.validate constraints, enum membership, optionality, and leading comments
// the measures layer depends on. If a future proto/buf change silently drops
// any of these from the descriptor image, this test fails.
func TestDescriptorRoundTrip(t *testing.T) {
	r := loadReader(t)

	// Resolve via the fully-qualified service name.
	params, err := r.RequestParams("agent_manager.v1.AgentManagerService", "ListProfiles")
	if err != nil {
		t.Fatalf("RequestParams(ListProfiles): %v", err)
	}

	limit, ok := byName(params, "limit")
	if !ok {
		t.Fatalf("limit param not found; got %+v", params)
	}
	if limit.Type != "int32" {
		t.Errorf("limit.Type = %q, want int32", limit.Type)
	}
	if !limit.Optional {
		t.Errorf("limit.Optional = false, want true (proto3 optional)")
	}
	if limit.Min == nil || *limit.Min != 1 {
		t.Errorf("limit.Min = %v, want 1", limit.Min)
	}
	if limit.Max == nil || *limit.Max != 100 {
		t.Errorf("limit.Max = %v, want 100", limit.Max)
	}
	if !strings.Contains(limit.Description, "Pagination limit") {
		t.Errorf("limit.Description = %q, want it to contain leading comment %q", limit.Description, "Pagination limit")
	}

	offset, ok := byName(params, "offset")
	if !ok {
		t.Fatalf("offset param not found")
	}
	if offset.Min == nil || *offset.Min != 0 {
		t.Errorf("offset.Min = %v, want 0", offset.Min)
	}

	runnerType, ok := byName(params, "runner_type")
	if !ok {
		t.Fatalf("runner_type param not found")
	}
	if runnerType.Type != "enum" {
		t.Errorf("runner_type.Type = %q, want enum", runnerType.Type)
	}
	if len(runnerType.EnumValues) == 0 {
		t.Errorf("runner_type.EnumValues empty, want declared values minus not_in[0]")
	}
	// not_in: [0] must exclude the zero (UNSPECIFIED) value.
	for _, v := range runnerType.EnumValues {
		if strings.HasSuffix(v, "UNSPECIFIED") {
			t.Errorf("runner_type.EnumValues = %v, must exclude the not_in[0] UNSPECIFIED value", runnerType.EnumValues)
		}
	}
}

// TestUUIDFormat confirms the string.uuid constraint surfaces as a format hint.
func TestUUIDFormat(t *testing.T) {
	r := loadReader(t)
	params, err := r.RequestParams("agent_manager.v1.AgentManagerService", "GetProfile")
	if err != nil {
		t.Fatalf("RequestParams(GetProfile): %v", err)
	}
	id, ok := byName(params, "profile_id")
	if !ok {
		t.Fatalf("profile_id param not found; got %+v", params)
	}
	if id.Type != "string" {
		t.Errorf("profile_id.Type = %q, want string", id.Type)
	}
	if id.Format != "uuid" {
		t.Errorf("profile_id.Format = %q, want uuid", id.Format)
	}
}

// TestBareServiceLookup confirms a manifest-style bare service name (no proto
// package) resolves the same method.
func TestBareServiceLookup(t *testing.T) {
	r := loadReader(t)
	params, err := r.RequestParams("AgentManagerService", "ListProfiles")
	if err != nil {
		t.Fatalf("RequestParams(bare AgentManagerService): %v", err)
	}
	if _, ok := byName(params, "limit"); !ok {
		t.Fatalf("bare lookup did not return the expected params: %+v", params)
	}
}

// TestUnknownMethod returns an error rather than panicking.
func TestUnknownMethod(t *testing.T) {
	r := loadReader(t)
	if _, err := r.RequestParams("NoSuchService", "Nope"); err == nil {
		t.Fatalf("expected error for unknown service/method, got nil")
	}
}

// TestTimeWindowImportable proves the shared canonical TimeWindow type compiles
// and is importable, and that the reader recognizes it as the canonical
// "time_window" param type by message name.
func TestTimeWindowImportable(t *testing.T) {
	tw := &measuresv1.TimeWindow{
		Window: &measuresv1.TimeWindow_Token{
			Token: measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_WEEK,
		},
	}
	if tw.GetToken() != measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_WEEK {
		t.Fatalf("TimeWindow token round-trip failed")
	}
	if string(tw.ProtoReflect().Descriptor().FullName()) != TimeWindowMessageName {
		t.Fatalf("TimeWindowMessageName = %q, want %q", TimeWindowMessageName, tw.ProtoReflect().Descriptor().FullName())
	}
}
