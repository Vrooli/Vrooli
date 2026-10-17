package terminal

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	terminalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal"
)

func TestAnswerPromptRequiresTheSessionAndThePromptHash(t *testing.T) {
	// [REQ:P0-017i] An answer must name the session and the prompt it answers.
	h := NewConnectHandler(Deps{Service: &fakeTerminalService{}})
	for _, msg := range []*terminalv1.AnswerPromptRequest{
		{OptionKey: "1", PromptHash: "h"},
		{SessionId: "s1", OptionKey: "1"},
	} {
		if _, err := h.AnswerPrompt(context.Background(), connect.NewRequest(msg)); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("AnswerPrompt(%+v) error = %v, want invalid argument", msg, err)
		}
	}
	res, err := h.AnswerPrompt(context.Background(), connect.NewRequest(&terminalv1.AnswerPromptRequest{SessionId: "s1", OptionKey: "2", PromptHash: "h"}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetDelivery() != DeliveryKeystrokes || res.Msg.GetAnswer() != "2" {
		t.Fatalf("response = %+v, want the service's delivery and answer", res.Msg)
	}
}
