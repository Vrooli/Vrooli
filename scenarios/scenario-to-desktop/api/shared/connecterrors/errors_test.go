package connecterrors

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
	domainerrors "scenario-to-desktop-api/shared/errors"
)

func TestWithEnvelopePreservesConnectCodeAndDomainRemediation(t *testing.T) {
	domainErr := domainerrors.ErrPipelineNotFound("pipe-42")
	err := WithEnvelope(connect.NewError(connect.CodeNotFound, domainErr))
	connectErr := new(connect.Error)
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeNotFound {
		t.Fatalf("Connect error = %#v, want not found", err)
	}
	envelope := errorEnvelope(t, connectErr)
	if envelope.GetCode() != string(domainerrors.CodePipelineNotFound) || envelope.GetCategory() == "" || envelope.GetRecovery() == "" || envelope.GetRecoveryHint() == "" {
		t.Fatalf("envelope is missing remediation fields: %#v", envelope)
	}
	if envelope.GetDetails()["pipeline_id"] != `"pipe-42"` {
		t.Fatalf("pipeline detail = %q", envelope.GetDetails()["pipeline_id"])
	}
	if got := WithEnvelope(connectErr); got != connectErr || len(connectErr.Details()) != 1 {
		t.Fatal("adding an envelope twice must preserve one canonical detail")
	}
}

func TestWithEnvelopeMakesUnexpectedFailuresActionable(t *testing.T) {
	connectErr := WithEnvelope(errors.New("database driver failed")).(*connect.Error)
	if connectErr.Code() != connect.CodeInternal {
		t.Fatalf("code = %v, want internal", connectErr.Code())
	}
	envelope := errorEnvelope(t, connectErr)
	if envelope.GetCode() != string(domainerrors.CodeInternal) || envelope.GetCategory() == "" || envelope.GetRecovery() == "" || envelope.GetRecoveryHint() == "" {
		t.Fatalf("generic envelope is not actionable: %#v", envelope)
	}
}

func errorEnvelope(t *testing.T, err *connect.Error) *sharedv1.ErrorEnvelope {
	t.Helper()
	for _, detail := range err.Details() {
		value, valueErr := detail.Value()
		if valueErr != nil {
			t.Fatalf("decode error detail: %v", valueErr)
		}
		if envelope, ok := value.(*sharedv1.ErrorEnvelope); ok {
			return envelope
		}
	}
	t.Fatal("Connect error did not carry ErrorEnvelope")
	return nil
}
