package snippets

import (
	"context"
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"connectrpc.com/connect"
	snippetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/snippets"

	snippetdomain "web-console/internal/snippets"
)

type fakeCommandRunner struct {
	name   string
	args   []string
	stdin  string
	output string
	err    error
}

func (f *fakeCommandRunner) Run(_ context.Context, name string, args []string, stdin string) (string, error) {
	f.name = name
	f.args = append([]string(nil), args...)
	f.stdin = stdin
	return f.output, f.err
}

func promotionHandler(t *testing.T, runner CommandRunner) (*connectHandler, string) {
	t.Helper()
	store := snippetdomain.NewMemStore()
	created, err := store.Upsert(context.Background(), snippetdomain.UpsertRequest{Name: "Inspect the seam", Body: "Find {{owner}} first."})
	if err != nil {
		t.Fatal(err)
	}
	return NewConnectHandler(Deps{Service: store, Runner: runner}), created.ID
}

func TestPromoteSnippetInvokesGovernedCLIAndLeavesSnippetUnchanged(t *testing.T) {
	runner := &fakeCommandRunner{output: "Created skill: Inspect the seam [inspect-the-seam] in local/\n"}
	handler, id := promotionHandler(t, runner)

	response, err := handler.PromoteSnippet(context.Background(), connect.NewRequest(&snippetsv1.PromoteSnippetRequest{Id: id}))
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Msg.GetIdentifier(); got != "inspect-the-seam" {
		t.Fatalf("identifier = %q", got)
	}
	wantArgs := []string{"skill", "create", "Inspect the seam", "--folder=local", "--description=Promoted from a Web Console snippet"}
	if runner.name != "prompt-manager" || !reflect.DeepEqual(runner.args, wantArgs) || runner.stdin != "Find {{owner}} first.\n" {
		t.Fatalf("call = %q %#v stdin %q", runner.name, runner.args, runner.stdin)
	}
	items, err := handler.deps.Service.List(context.Background())
	if err != nil || len(items) != 1 || items[0].ID != id || items[0].UseCount != 0 || items[0].Body != "Find {{owner}} first." {
		t.Fatalf("snippet changed after promotion: %#v, %v", items, err)
	}
}

func TestPromoteSnippetSurfacesCommandStderr(t *testing.T) {
	runner := &fakeCommandRunner{output: "skill name already exists\n", err: errors.New("exit status 1")}
	handler, id := promotionHandler(t, runner)

	_, err := handler.PromoteSnippet(context.Background(), connect.NewRequest(&snippetsv1.PromoteSnippetRequest{Id: id}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition || err == nil || !strings.Contains(err.Error(), "skill name already exists") {
		t.Fatalf("error = %v (%v)", err, connect.CodeOf(err))
	}
}

func TestPromoteSnippetMapsMissingBinary(t *testing.T) {
	runner := &fakeCommandRunner{err: &exec.Error{Name: "prompt-manager", Err: exec.ErrNotFound}}
	handler, id := promotionHandler(t, runner)

	_, err := handler.PromoteSnippet(context.Background(), connect.NewRequest(&snippetsv1.PromoteSnippetRequest{Id: id}))
	if connect.CodeOf(err) != connect.CodeUnavailable || err == nil || !strings.Contains(err.Error(), ErrPromptManagerUnavailable.Error()) {
		t.Fatalf("error = %v (%v)", err, connect.CodeOf(err))
	}
}
