package consumerdeclarations

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	consumerdeclarationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/consumer_declarations"
)

func TestValidateReturnsGenericDeclarationResult(t *testing.T) {
	response, err := (&service{}).Validate(context.Background(), connect.NewRequest(&consumerdeclarationsv1.ValidateConsumerDeclarationRequest{DeclarationJson: `{"schemaVersion":"browser-automation-studio.consumer-declaration/v1","profiles":[{"key":"shared-browser","workflowRef":"bas/example.json"}]}`}))
	if err != nil {
		t.Fatal(err)
	}
	if !response.Msg.GetValid() || len(response.Msg.GetProfiles()) != 1 {
		t.Fatalf("response = %#v", response.Msg)
	}
}
