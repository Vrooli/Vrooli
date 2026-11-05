package projects

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

	"connectrpc.com/connect"
)

func TestUnknownInspectionFailureLogsCauseAndReturnsActionableMessage(t *testing.T) {
	var logBytes bytes.Buffer
	h := NewConnectHandler(Deps{Logger: log.New(&logBytes, "", 0)})
	err := h.inspectionConnectError("ensure", "project-1", "media-1", errors.New("native diagnostic detail"))
	connectErr, ok := err.(*connect.Error)
	if !ok || connectErr.Code() != connect.CodeInternal {
		t.Fatalf("inspection error = %T %v, want internal Connect error", err, err)
	}
	if !strings.Contains(connectErr.Message(), "check the Device Studio API log") {
		t.Fatalf("client error message is not actionable: %q", connectErr.Message())
	}
	for _, want := range []string{"project=\"project-1\"", "media=\"media-1\"", "native diagnostic detail"} {
		if !strings.Contains(logBytes.String(), want) {
			t.Fatalf("log %q does not contain %q", logBytes.String(), want)
		}
	}
}
