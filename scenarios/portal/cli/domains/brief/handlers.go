package brief

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/agentbrief-go/runtimecap"
	"github.com/vrooli/cli-core/cliapp"
	briefv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/brief"
	briefconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/brief/brief_v1connect"
)

type handlers struct {
	client briefconnect.BriefServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: briefconnect.NewBriefServiceClient(httpClient, baseURL)}
}

func (h *handlers) build(ctx cliapp.RunContext) error {
	prompt, err := readPrompt(ctx)
	if err != nil {
		return err
	}
	req := &briefv1.BuildBriefRequest{Prompt: prompt, Consumer: parseConsumer(ctx.Flag("consumer")), BudgetMs: parseInt32(ctx.Flag("budget-ms"))}
	resp, err := h.client.Build(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("build context brief", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetBrief() == nil {
		return fmt.Errorf("server returned no brief")
	}
	brief := resp.Msg.GetBrief()
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	if strings.EqualFold(ctx.Flag("render"), "agent") {
		if strings.EqualFold(brief.GetVerdict().String(), "BRIEF_VERDICT_DELIVER") {
			_, err = io.WriteString(ctx.Stdout(), brief.GetRendered())
			return err
		}
		fmt.Fprintf(ctx.Stderr(), "%s: %s\n", brief.GetVerdict().String(), brief.GetReason())
		return nil
	}
	fmt.Fprintf(ctx.Stdout(), "Brief %s: %s (%s)\n%s\n", brief.GetId(), brief.GetVerdict().String(), brief.GetReason(), brief.GetRendered())
	return nil
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	resp, err := h.client.Get(context.Background(), connect.NewRequest(&briefv1.GetBriefRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("get context brief", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no brief")
	}
	return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.List(context.Background(), connect.NewRequest(&briefv1.ListBriefsRequest{Consumer: parseConsumer(ctx.Flag("consumer")), ChatId: ctx.Flag("chat-id"), SessionRef: ctx.Flag("session-ref"), Limit: parseInt32(ctx.Flag("limit"))}))
	if err != nil {
		return cliapp.WrapAPIError("list context briefs", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no briefs")
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	for _, brief := range resp.Msg.GetBriefs() {
		fmt.Fprintf(ctx.Stdout(), "%s %s %s %s\n", brief.GetId(), brief.GetConsumer().String(), brief.GetVerdict().String(), brief.GetReason())
	}
	return nil
}

func (h *handlers) recordUse(ctx cliapp.RunContext) error {
	index, err := strconv.ParseInt(ctx.Positional("item-index"), 10, 32)
	if err != nil {
		return fmt.Errorf("item-index must be an integer: %w", err)
	}
	resp, err := h.client.RecordUse(context.Background(), connect.NewRequest(&briefv1.RecordBriefUseRequest{BriefId: ctx.Positional("brief-id"), ItemIndex: int32(index), Kind: parseUseKind(ctx.Positional("kind"))}))
	if err != nil {
		return cliapp.WrapAPIError("record context brief use", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no use result")
	}
	return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
}

func (h *handlers) stats(ctx cliapp.RunContext) error {
	request := &briefv1.BriefStatsRequest{WindowDays: parseInt32(ctx.Flag("window")), Consumer: parseConsumer(ctx.Flag("consumer"))}
	if strings.TrimSpace(ctx.Flag("consumer")) == "" {
		request.Consumer = briefv1.BriefConsumer_BRIEF_CONSUMER_UNSPECIFIED
	}
	resp, err := h.client.Stats(context.Background(), connect.NewRequest(request))
	if err != nil {
		return cliapp.WrapAPIError("read context brief stats", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no brief stats")
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	fmt.Fprintln(ctx.Stdout(), "Usage rate = used items / delivered items; withheld rate = withheld briefs / built briefs.")
	for _, row := range resp.Msg.GetRows() {
		fmt.Fprintf(ctx.Stdout(), "%s built=%d delivered=%d withheld_rate=%.3f items=%d used=%d usage_rate=%.3f\n", row.GetConsumer().String(), row.GetBriefsBuilt(), row.GetBriefsDelivered(), row.GetWithheldRate(), row.GetItemsDelivered(), row.GetItemsUsed(), row.GetUsageRate())
		for verdict, count := range row.GetWithheldByVerdict() {
			fmt.Fprintf(ctx.Stdout(), "  %s=%d\n", verdict, count)
		}
	}
	return nil
}

func readPrompt(ctx cliapp.RunContext) (string, error) {
	if value := strings.TrimSpace(ctx.Flag("prompt")); value != "" {
		return value, nil
	}
	path := strings.TrimSpace(ctx.Flag("prompt-file"))
	if path == "" {
		return "", fmt.Errorf("one of --prompt or --prompt-file is required")
	}
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}
func parseConsumer(value string) briefv1.BriefConsumer {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "portal-llm", "llm":
		return briefv1.BriefConsumer_BRIEF_CONSUMER_PORTAL_LLM
	case "portal-agent", "agent":
		return briefv1.BriefConsumer_BRIEF_CONSUMER_PORTAL_AGENT
	default:
		return briefv1.BriefConsumer_BRIEF_CONSUMER_EXTERNAL_HARNESS
	}
}
func parseUseKind(value string) briefv1.BriefUseKind {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "opened":
		return briefv1.BriefUseKind_BRIEF_USE_KIND_OPENED
	case "copied":
		return briefv1.BriefUseKind_BRIEF_USE_KIND_COPIED
	case "referenced":
		return briefv1.BriefUseKind_BRIEF_USE_KIND_REFERENCED
	case "rejected":
		return briefv1.BriefUseKind_BRIEF_USE_KIND_REJECTED
	default:
		return briefv1.BriefUseKind_BRIEF_USE_KIND_UNSPECIFIED
	}
}
func parseInt32(value string) int32 {
	if value == "" {
		return 0
	}
	n, _ := strconv.ParseInt(value, 10, 32)
	return int32(n)
}

type hookEvent struct {
	Prompt string `json:"prompt"`
}

func (h *handlers) hook(ctx cliapp.RunContext) error {
	// Hook stdout is a host protocol. Every malformed or unavailable path is a
	// successful no-op so the host never renders a hook error to the operator.
	var event hookEvent
	if err := json.NewDecoder(os.Stdin).Decode(&event); err != nil || strings.TrimSpace(event.Prompt) == "" {
		return nil
	}
	requestCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	response, err := h.client.Build(requestCtx, connect.NewRequest(&briefv1.BuildBriefRequest{
		Prompt:   event.Prompt,
		Consumer: briefv1.BriefConsumer_BRIEF_CONSUMER_EXTERNAL_HARNESS,
		BudgetMs: 3800,
	}))
	if err != nil || response == nil || response.Msg == nil || response.Msg.GetBrief() == nil {
		return nil
	}
	brief := response.Msg.GetBrief()
	if brief.GetVerdict() != briefv1.BriefVerdict_BRIEF_VERDICT_DELIVER || strings.TrimSpace(brief.GetRendered()) == "" {
		return nil
	}
	payload := map[string]any{"hookSpecificOutput": map[string]string{
		"hookEventName":     "UserPromptSubmit",
		"additionalContext": brief.GetRendered(),
	}}
	return json.NewEncoder(ctx.Stdout()).Encode(payload)
}

func (h *handlers) hooks(ctx cliapp.RunContext) error {
	action := strings.ToLower(strings.TrimSpace(ctx.Positional("action")))
	runtime := strings.TrimSpace(ctx.Flag("runtime"))
	if action == "status" {
		runtimes := []string{"claude-code", "codex", "grok", "antigravity", "opencode"}
		if runtime != "" {
			runtimes = []string{runtime}
		}
		for _, name := range runtimes {
			capability, err := runtimeCapability(name)
			if err != nil {
				return err
			}
			fmt.Fprintf(ctx.Stdout(), "%s UserPromptSubmit supported=%t can_inject_context=%t verified_by=%s reason=%s\n", name, capability.Supported, capability.CanInjectContext, capability.VerifiedBy, capability.Reason)
		}
		return nil
	}
	if runtime == "" {
		return fmt.Errorf("--runtime is required for hooks %s", action)
	}
	capability, err := runtimeCapability(runtime)
	if err != nil {
		return err
	}
	switch action {
	case "install":
		if !capability.Supported || !capability.CanInjectContext || capability.VerifiedBy != "canary" {
			fmt.Fprintf(ctx.Stdout(), "refused: runtime %s UserPromptSubmit is not canary-verified (supported=%t can_inject_context=%t verified_by=%q reason=%s)\n", runtime, capability.Supported, capability.CanInjectContext, capability.VerifiedBy, capability.Reason)
			return nil
		}
		return reconcileHook(ctx, runtime)
	case "remove":
		return removeHook(ctx, runtime)
	default:
		return fmt.Errorf("hooks action must be install, remove, or status")
	}
}

func runtimeCapability(runtime string) (runtimecap.Capability, error) {
	root, err := repoRoot()
	if err != nil {
		return runtimecap.Capability{}, err
	}
	return runtimecap.Lookup(root, runtime, "UserPromptSubmit")
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, ".vrooli")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not locate Vrooli repository root")
		}
		dir = parent
	}
}

func reconcileHook(ctx cliapp.RunContext, runtime string) error {
	args := []string{"hooks", "reconcile", "--event", "UserPromptSubmit", "--id", "portal-context-brief", "--hook-json", fmt.Sprintf(`{"type":"command","command":"portal brief hook --runtime %s"}`, runtime), "--scope", hookScope(runtime)}
	command := exec.Command("resource-"+runtime, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("install context hook for %s: %w: %s", runtime, err, strings.TrimSpace(string(output)))
	}
	fmt.Fprint(ctx.Stdout(), string(output))
	return nil
}

func removeHook(ctx cliapp.RunContext, runtime string) error {
	command := exec.Command("resource-"+runtime, "hooks", "remove", "--event", "UserPromptSubmit", "--id", "portal-context-brief", "--scope", hookScope(runtime))
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove context hook for %s: %w: %s", runtime, err, strings.TrimSpace(string(output)))
	}
	fmt.Fprint(ctx.Stdout(), string(output))
	return nil
}

func hookScope(runtime string) string {
	if runtime == "grok" {
		return "user"
	}
	return "global"
}
