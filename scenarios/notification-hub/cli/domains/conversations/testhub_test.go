package conversations

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
)

// hubStub is a fake Hub for the conversations commands. Tests set the fields
// they care about before running a command; every request is recorded.
type hubStub struct {
	connectv1.UnimplementedConversationsServiceHandler

	mu sync.Mutex
	// held are the blob hashes the fake Hub already stores.
	held map[string]bool

	statCalls [][]string
	puts      []*v1.PutAttachmentBlobRequest
	asks      []*v1.AskRequest
	waits     []*v1.WaitRequest
	gets      []*v1.GetAskRequest
	answers   []*v1.AnswerRequest
	contexts  []*v1.RequestContextRequest
	lists     []*v1.ListAsksRequest

	// listResp answers ListAsks; the zero value is an empty page.
	listResp *v1.ListAsksResponse
	askResp  *v1.AskResponse
	askErr   error
	waitResp *v1.WaitResponse
	// waitSeq answers successive Wait calls in order and then repeats its last
	// entry; a nil entry in waitErrSeq lets the call through, an error fails it.
	waitSeq    []*v1.WaitResponse
	waitErrSeq []error
	checkResp  *v1.CheckAskResponse
	checks     []*v1.CheckAskRequest
	getAsk     *v1.Ask
	putErr     error
	// putSeq fails the first len(putSeq) Put calls with these errors in order
	// (a nil entry succeeds); later calls succeed. statErr fails every Stat.
	putSeq  []error
	statErr error
	// readErr fails every GetAsk, ListAsks and Wait.
	readErr  error
	putCalls int

	inFlight, maxInFlight int
}

func (h *hubStub) StatAttachmentBlobs(_ context.Context, req *connect.Request[v1.StatAttachmentBlobsRequest]) (*connect.Response[v1.StatAttachmentBlobsResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.statErr != nil {
		return nil, h.statErr
	}
	h.statCalls = append(h.statCalls, req.Msg.GetSha256())
	var missing []string
	for _, hash := range req.Msg.GetSha256() {
		if !h.held[hash] {
			missing = append(missing, hash)
		}
	}
	return connect.NewResponse(&v1.StatAttachmentBlobsResponse{Missing: missing}), nil
}

func (h *hubStub) PutAttachmentBlob(_ context.Context, req *connect.Request[v1.PutAttachmentBlobRequest]) (*connect.Response[v1.PutAttachmentBlobResponse], error) {
	h.mu.Lock()
	h.inFlight++
	h.maxInFlight = max(h.maxInFlight, h.inFlight)
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		h.inFlight--
		h.mu.Unlock()
	}()
	if h.putErr != nil {
		return nil, h.putErr
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.putCalls++
	if h.putCalls <= len(h.putSeq) && h.putSeq[h.putCalls-1] != nil {
		return nil, h.putSeq[h.putCalls-1]
	}
	h.puts = append(h.puts, req.Msg)
	if h.held == nil {
		h.held = map[string]bool{}
	}
	h.held[req.Msg.GetSha256()] = true
	return connect.NewResponse(&v1.PutAttachmentBlobResponse{Sha256: req.Msg.GetSha256(), SizeBytes: int64(len(req.Msg.GetContent()))}), nil
}

func (h *hubStub) Ask(_ context.Context, req *connect.Request[v1.AskRequest]) (*connect.Response[v1.AskResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.asks = append(h.asks, req.Msg)
	if h.askErr != nil {
		return nil, h.askErr
	}
	if h.askResp != nil {
		return connect.NewResponse(h.askResp), nil
	}
	return connect.NewResponse(&v1.AskResponse{AskId: "ask-1"}), nil
}

func (h *hubStub) Wait(_ context.Context, req *connect.Request[v1.WaitRequest]) (*connect.Response[v1.WaitResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.readErr != nil {
		return nil, h.readErr
	}
	call := len(h.waits)
	h.waits = append(h.waits, req.Msg)
	if call < len(h.waitErrSeq) && h.waitErrSeq[call] != nil {
		return nil, h.waitErrSeq[call]
	}
	if len(h.waitSeq) > 0 {
		return connect.NewResponse(h.waitSeq[min(call, len(h.waitSeq)-1)]), nil
	}
	return connect.NewResponse(h.waitResp), nil
}

func (h *hubStub) CheckAsk(_ context.Context, req *connect.Request[v1.CheckAskRequest]) (*connect.Response[v1.CheckAskResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks = append(h.checks, req.Msg)
	if h.checkResp != nil {
		return connect.NewResponse(h.checkResp), nil
	}
	return connect.NewResponse(&v1.CheckAskResponse{PushTitle: "Decision required", PushBody: req.Msg.GetAsk().GetQuestion()}), nil
}

func (h *hubStub) ListAsks(_ context.Context, req *connect.Request[v1.ListAsksRequest]) (*connect.Response[v1.ListAsksResponse], error) {
	h.mu.Lock()
	h.lists = append(h.lists, req.Msg)
	h.mu.Unlock()
	if h.readErr != nil {
		return nil, h.readErr
	}
	if h.listResp != nil {
		return connect.NewResponse(h.listResp), nil
	}
	return connect.NewResponse(&v1.ListAsksResponse{}), nil
}

func (h *hubStub) GetAsk(_ context.Context, req *connect.Request[v1.GetAskRequest]) (*connect.Response[v1.GetAskResponse], error) {
	h.mu.Lock()
	h.gets = append(h.gets, req.Msg)
	h.mu.Unlock()
	if h.readErr != nil {
		return nil, h.readErr
	}
	if h.getAsk == nil {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&v1.GetAskResponse{Ask: h.getAsk}), nil
}

func (h *hubStub) Answer(_ context.Context, req *connect.Request[v1.AnswerRequest]) (*connect.Response[v1.AnswerResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.answers = append(h.answers, req.Msg)
	return connect.NewResponse(&v1.AnswerResponse{AskId: req.Msg.GetAskId()}), nil
}

func (h *hubStub) RequestContext(_ context.Context, req *connect.Request[v1.RequestContextRequest]) (*connect.Response[v1.RequestContextResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.contexts = append(h.contexts, req.Msg)
	return connect.NewResponse(&v1.RequestContextResponse{AskId: req.Msg.GetAskId()}), nil
}

// newTestService serves the stub over HTTP and returns a service wired to it.
func newTestService(t *testing.T, stub *hubStub) *service {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(connectv1.NewConversationsServiceHandler(stub))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := connectv1.NewConversationsServiceClient(server.Client(), server.URL)
	return &service{client: client, waitClient: client, uploadClient: client, now: time.Now}
}

// runCommand runs `conversations <name> args...` through the real manifest and
// argument parser, with the guide check stubbed out so nothing touches the network.
func runCommand(t *testing.T, svc *service, name string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	restore := checkGuide
	checkGuide = func(context.Context) guideReceipt { return guideReceipt{Status: guideRead} }
	t.Cleanup(func() { checkGuide = restore })

	manifest, readErr := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, readErr)
	group, regErr := register(svc, manifest)
	require.NoError(t, regErr)
	for _, cmd := range group.Subcommands {
		if cmd.Name != name {
			continue
		}
		var out, errOut bytes.Buffer
		ctx, parseErr := cliapp.NewTestRunContextFromArgs(cmd.Args, args, nil, &out, &errOut)
		require.NoError(t, parseErr)
		err = cmd.RunCtx(ctx)
		return out.String(), errOut.String(), err
	}
	t.Fatalf("no conversations command %q", name)
	return "", "", fmt.Errorf("unreachable")
}

// askArgs are the flags every ask needs.
func askArgs(extra ...string) []string {
	base := []string{
		"--question", "Ship it?", "--option", "yes", "--option", "no",
		"--expires", "24h", "--sensitivity-label", "private", "--idempotency-key", "k1",
	}
	return append(base, extra...)
}

func writeFile(t *testing.T, dir, rel, content string) string {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func connectError(message string) error {
	return connect.NewError(connect.CodeInternal, fmt.Errorf("%s", message))
}

func itoa(n int) string { return strconv.Itoa(n) }
