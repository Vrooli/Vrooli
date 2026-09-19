package mints_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	mintsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/mints"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// [REQ:TKE-P0-014] The public mint contract has no monetary valuation, payout, or off-instance transfer capability.
func TestMintsContractHasNoMonetaryValueSurface(t *testing.T) {
	forbidden := []string{"price", "currency", "monetary", "payout", "fiat", "withdraw", "cash", "external", "off_instance"}
	file := mintsv1.File_token_economy_v1_mints_mints_proto

	var inspectMessage func(protoreflect.MessageDescriptor)
	inspectMessage = func(message protoreflect.MessageDescriptor) {
		for i := 0; i < message.Fields().Len(); i++ {
			field := message.Fields().Get(i)
			assertSafeContractName(t, string(field.Name()), forbidden)
			if field.Message() != nil && field.Message().ParentFile() == file {
				inspectMessage(field.Message())
			}
		}
	}
	for i := 0; i < file.Messages().Len(); i++ {
		inspectMessage(file.Messages().Get(i))
	}
	for i := 0; i < file.Services().Len(); i++ {
		service := file.Services().Get(i)
		for j := 0; j < service.Methods().Len(); j++ {
			assertSafeContractName(t, string(service.Methods().Get(j).Name()), forbidden)
		}
	}
}

func assertSafeContractName(t *testing.T, name string, forbidden []string) {
	t.Helper()
	normalized := strings.ToLower(name)
	for _, term := range forbidden {
		require.NotContains(t, normalized, term, "contract member %q introduces forbidden monetary/off-instance semantics", name)
	}
}
