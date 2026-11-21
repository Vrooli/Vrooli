package conversations_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

// The operator's own agents link local files and relative paths in an ask, so a
// direct ask keeps accepting those (the packet check only advises). What it never
// accepts is a link that runs code when the operator taps it.

func askWithLink(key, link string) *v1.AskRequest {
	return &v1.AskRequest{
		Question: "Continue with " + key + "?", AllowedAnswers: []string{"yes", "no"}, ContextUrl: &link,
		Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339), SensitivityLabel: "private", IdempotencyKey: key,
	}
}

func TestADirectAskRefusesALinkThatRunsCodeAndNamesTheScheme(t *testing.T) {
	f := newReaderFixture(t)
	ctx := context.Background()
	for link, scheme := range map[string]string{
		"javascript:alert(1)":                "javascript",
		"data:text/html;base64,PHNjcmlwdD4=": "data",
		"vbscript:msgbox(1)":                 "vbscript",
		"java\tscript:alert(1)":              "javascript",
	} {
		key := "ask-" + scheme
		_, err := f.client.Ask(ctx, reading("owner", askWithLink(key, link)))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "ask %q", link)
		require.ErrorContains(t, err, `context_url scheme "`+scheme+`" can run code`)

		// The dry run says the same, so an agent finds out before it sends.
		_, err = f.client.CheckAsk(ctx, reading("owner", &v1.CheckAskRequest{Ask: askWithLink(key, link)}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "check %q", link)
		require.ErrorContains(t, err, `context_url scheme "`+scheme+`" can run code`)
	}
}

func TestADirectAskStillAcceptsLocalAndRelativeLinks(t *testing.T) {
	f := newReaderFixture(t)
	for i, link := range []string{
		"", "/asks/ask-1", "https://host.example/review", "steward/delegations/x/RETURN.md",
		"file:///home/op/.vrooli/research/runs/1/RETURN.md",
	} {
		id := f.ask(t, "owner", "keep-"+string(rune('a'+i)), func(m *v1.AskRequest) { m.ContextUrl = &link })
		require.NotEmpty(t, id, "link %q", link)
	}
}
