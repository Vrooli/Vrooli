package conversations

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
)

const GroupName = "conversations"

// service holds the Hub clients the conversations commands share. They differ
// only in their client-side timeout.
type service struct {
	// client serves ordinary requests with the scenario's default timeout.
	client connectv1.ConversationsServiceClient
	// waitClient serves Wait, a server-owned long-poll that blocks until an
	// answer or the requested deadline, so a short default timeout would abort
	// it long before either. Its context is bounded by that deadline.
	waitClient connectv1.ConversationsServiceClient
	// uploadClient serves attachment blobs of up to 64 MiB.
	uploadClient connectv1.ConversationsServiceClient
	now          func() time.Time
	// senderSource resolves the label that names the sender of an ask with no
	// --from (see sender.go); nil means there is none. senderOnce makes the
	// resolution happen at most once per process.
	senderSource senderSource
	senderOnce   sync.Once
	sender       string
	// stdin is read by --body-file - ; nil means the process's standard input.
	stdin io.Reader
	// sleep waits between attempts to reach the Hub; nil means time.Sleep.
	sleep func(time.Duration)
	// identityBrokerUnavailable is replaceable in tests. It is consulted only
	// after an unauthenticated wait response from an inherited run identity.
	identityBrokerUnavailable func() bool
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	var defaultTimeout time.Duration
	if core != nil && core.HTTPClient != nil {
		defaultTimeout = core.HTTPClient.Timeout()
	}
	configuredToken := ""
	if core != nil {
		configuredToken = strings.TrimSpace(core.Config.Token)
	}
	h, base := cliapp.NewConnectHTTPClient(core)
	waitHTTP, waitBase := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	uploadHTTP, uploadBase := cliapp.NewConnectHTTPClientWithTimeout(core, uploadTimeout)
	// NOTIFICATION_HUB_SEND_AS swaps each of these for one that presents a
	// service principal's token (see sendas.go); unset, they are used as they are.
	h = hubHTTPClient(core, h, base, defaultTimeout)
	waitHTTP = hubHTTPClient(core, waitHTTP, waitBase, 0)
	uploadHTTP = hubHTTPClient(core, uploadHTTP, uploadBase, uploadTimeout)
	return register(&service{
		client:                    connectv1.NewConversationsServiceClient(h, base),
		waitClient:                connectv1.NewConversationsServiceClient(waitHTTP, waitBase),
		uploadClient:              connectv1.NewConversationsServiceClient(uploadHTTP, uploadBase),
		now:                       time.Now,
		senderSource:              senderOfThisRun,
		identityBrokerUnavailable: func() bool { return agentIdentityBrokerUnavailable(configuredToken) },
	}, manifest)
}

// agentIdentityBrokerUnavailable only marks the retryable case where an
// inherited run identity could not be refreshed because its local authority is
// offline. A live authority or an explicit owner/service token makes a Hub
// refusal authoritative and non-retryable.
func agentIdentityBrokerUnavailable(configuredToken string) bool {
	if strings.TrimSpace(os.Getenv(cliutil.EnvIdentityToken)) == "" ||
		os.Getenv("NOTIFICATION_HUB_API_TOKEN") != "" || os.Getenv("VROOLI_API_TOKEN") != "" ||
		strings.TrimSpace(configuredToken) != "" {
		return false
	}
	conn, err := net.DialTimeout("unix", cliutil.AgentCredentialSocket(), 100*time.Millisecond)
	if err != nil {
		return true
	}
	_ = conn.Close()
	return false
}

func register(s *service, manifest []byte) (cliapp.SubcommandGroup, error) {
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ConversationsService.Ask":            s.ask,
		"ConversationsService.Answer":         s.answer,
		"ConversationsService.RequestContext": s.requestContext,
		"ConversationsService.Wait":           s.wait,
		"ConversationsService.Dismiss": func(ctx cliapp.RunContext) error {
			resp, callErr := s.client.Dismiss(context.Background(), connect.NewRequest(&v1.DismissRequest{
				AskId: ctx.Flag("ask-id"),
				Note:  ctx.Flag("note"),
			}))
			if callErr != nil {
				return cliapp.WrapAPIError("dismiss notification question", callErr, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Dismissed ask %s without answering.", resp.Msg.GetAskId())}})
		},
		"ConversationsService.Withdraw": func(ctx cliapp.RunContext) error {
			askID, err := askIDFrom(ctx)
			if err != nil {
				return err
			}
			subjectKey := ctx.Flag("subject-key")
			if (askID == "") == (subjectKey == "") {
				return fmt.Errorf("give exactly one of the ask id or --subject-key")
			}
			resp, callErr := s.client.Withdraw(context.Background(), connect.NewRequest(&v1.WithdrawRequest{
				AskId:      askID,
				SubjectKey: subjectKey,
				Reason:     ctx.Flag("reason"),
			}))
			if callErr != nil {
				return cliapp.WrapAPIError("withdraw notification question", callErr, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
				Result:      []string{fmt.Sprintf("Withdrew ask %s.", resp.Msg.GetAskId())},
				Changes:     []string{"The ask is closed with your reason, and its record is kept. A run waiting on it ends with status 5 and sees the reason."},
				NextCommand: []string{getCommand(resp.Msg.GetAskId())},
			})
		},
		"ConversationsService.GetAsk": func(ctx cliapp.RunContext) error {
			askID, err := askIDFrom(ctx)
			if err != nil {
				return err
			}
			subjectKey := ctx.Flag("subject-key")
			if (askID == "") == (subjectKey == "") {
				return fmt.Errorf("give exactly one of the ask id or --subject-key")
			}
			resp, callErr := s.client.GetAsk(context.Background(), connect.NewRequest(&v1.GetAskRequest{AskId: askID, SubjectKey: subjectKey}))
			if callErr != nil {
				return readError("read ask", callErr)
			}
			return cliapp.RenderProtoList(ctx, resp.Msg, askDetailReport(resp.Msg.GetAsk()))
		},
		"ConversationsService.ListAsks": s.list,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("conversations: load manifest: %w", err)
	}
	return group, nil
}

// maxPageLimit is the largest page the Hub returns.
const maxPageLimit = 200

// parsePageLimit reads --limit: empty means the Hub's default page, otherwise a
// whole number from 1 to maxPageLimit. A value outside that range is an error
// that names the range, not a page size the Hub silently changes.
func parsePageLimit(value string) (int32, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	limit, err := strconv.ParseInt(value, 10, 32)
	if err != nil || limit < 1 || limit > maxPageLimit {
		return 0, fmt.Errorf("--limit must be a whole number from 1 to %d, got %q", maxPageLimit, value)
	}
	return int32(limit), nil
}
