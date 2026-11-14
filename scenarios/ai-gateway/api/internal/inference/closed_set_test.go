package inference

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

const october25ScreenShortPolicy = "Return only JSON with keys action and net. Rule: if inventory after reservations is below 12, action is reorder; otherwise hold. Current inventory 52, reservations 43. net is inventory minus reservations. Do not explain."

const inventoryReservationSchema = `{"type":"object","properties":{"action":{"type":"string","enum":["reorder","hold"]},"net":{"type":"integer"}},"required":["action","net"],"additionalProperties":false}`

func TestOct25ClosedSetPolicyRunsInCodeWithoutProvider(t *testing.T) {
	response := inventoryReservationCodeService(t).Run(context.Background(), ProviderRequest{PrivacyProvenance: "tests.explicit_internal_input",
		Role: "policy.inventory-reservation", Source: october25ScreenShortPolicy,
		SchemaJSON: inventoryReservationSchema,
	})
	require.Nil(t, response.GetError())
	require.True(t, response.GetValidated())
	require.JSONEq(t, `{"action":"reorder","net":9}`, response.GetValueJson())
	require.Equal(t, "vrooli-code", response.GetProvider())
	require.Equal(t, int32(0), response.GetProviderCallsMade())
}

func TestClosedSetPolicyRejectsUnrecognizedRuleShape(t *testing.T) {
	response := inventoryReservationCodeService(t).Run(context.Background(), ProviderRequest{PrivacyProvenance: "tests.explicit_internal_input",
		Role: "policy.inventory-reservation", Source: "choose reorder or hold based on the prompt",
		SchemaJSON: inventoryReservationSchema,
	})
	require.NotNil(t, response.GetError())
	require.Contains(t, response.GetError().GetMessage(), "does not match")
	require.Equal(t, int32(0), response.GetProviderCallsMade())
}

func TestDeclaredCodeRoleEnforcesBoundsAndCallerSchemaThroughService(t *testing.T) {
	service := inventoryReservationCodeService(t)
	base := ProviderRequest{PrivacyProvenance: "tests.explicit_internal_input", Role: "policy.inventory-reservation", Source: october25ScreenShortPolicy, SchemaJSON: inventoryReservationSchema}

	tooLong := base
	tooLong.Source = strings.Repeat("x", 4097)
	response := service.Run(context.Background(), tooLong)
	require.NotNil(t, response.GetError())
	require.Contains(t, response.GetError().GetMessage(), "source exceeds")
	require.Equal(t, int32(0), response.GetProviderCallsMade())

	wrongSchema := base
	wrongSchema.SchemaJSON = `{"type":"string"}`
	response = service.Run(context.Background(), wrongSchema)
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED, response.GetError().GetCode())
	require.Equal(t, "schema_json", response.GetError().GetConstruct())
	require.Equal(t, int32(0), response.GetProviderCallsMade())
}

func inventoryReservationCodeService(tb testing.TB) *Service {
	tb.Helper()
	catalog, err := LoadCatalog(filepath.Join("..", "..", "..", "config", "inference-role-catalog.json"))
	require.NoError(tb, err)
	repository, err := NewResourceRepository(catalog, nil)
	require.NoError(tb, err)
	return NewService(repository)
}

func TestDeclaredCodeRoleEnforcesCatalogBoundsAndSchemas(t *testing.T) {
	catalogPath := filepath.Join("..", "..", "..", "config", "inference-role-catalog.json")
	catalog, err := LoadCatalog(catalogPath)
	require.NoError(t, err)
	contract := catalog.CodeRoles["policy.inventory-reservation"]
	callerSchema, err := (SchemaGate{}).Parse(inventoryReservationSchema)
	require.NoError(t, err)

	value, err := evaluateDeclaredCodeRole(contract, callerSchema, october25ScreenShortPolicy)
	require.NoError(t, err)
	require.JSONEq(t, `{"action":"reorder","net":9}`, value)

	_, err = evaluateDeclaredCodeRole(contract, callerSchema, strings.Repeat("x", contract.MaxSourceBytes+1))
	require.ErrorContains(t, err, "source exceeds")

	wrongCallerSchema, err := (SchemaGate{}).Parse(`{"type":"string"}`)
	require.NoError(t, err)
	_, err = evaluateDeclaredCodeRole(contract, wrongCallerSchema, october25ScreenShortPolicy)
	require.ErrorContains(t, err, "caller schema")

	tooSmallOutput := contract
	tooSmallOutput.MaxOutputBytes = 1
	_, err = evaluateDeclaredCodeRole(tooSmallOutput, callerSchema, october25ScreenShortPolicy)
	require.ErrorContains(t, err, "output exceeds")

	unknownHandler := contract
	unknownHandler.HandlerID = "unknown-handler/v1"
	_, err = evaluateDeclaredCodeRole(unknownHandler, callerSchema, october25ScreenShortPolicy)
	require.ErrorContains(t, err, "not registered")
}

func BenchmarkOct25ClosedSetPolicyCodePath(b *testing.B) {
	service := inventoryReservationCodeService(b)
	request := ProviderRequest{PrivacyProvenance: "tests.explicit_internal_input",
		Role: "policy.inventory-reservation", Source: october25ScreenShortPolicy,
		SchemaJSON: inventoryReservationSchema,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response := service.Run(context.Background(), request)
		if response.GetError() != nil || !response.GetValidated() {
			b.Fatal("closed-set code path did not validate the task")
		}
	}
}
