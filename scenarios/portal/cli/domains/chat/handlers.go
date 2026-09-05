package chat

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	chatv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/chat"
	chatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/chat/chat_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/shared"
)

type handlers struct {
	client chatconnect.ChatServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: chatconnect.NewChatServiceClient(httpClient, baseURL)}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListChats(context.Background(), connect.NewRequest(&chatv1.ListChatsRequest{
		GroupId: ctx.Flag("group-id"),
		Query:   ctx.Flag("query"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list chats", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no chats response")
	}
	results := make([]string, 0, len(resp.Msg.GetChats()))
	for _, c := range resp.Msg.GetChats() {
		results = append(results, formatChat(c))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d chat(s) across %d group(s).", len(resp.Msg.GetChats()), len(resp.Msg.GetGroups()))},
		ResultsHeading: "Chats",
		Results:        results,
		RetrievalHints: []string{
			"`chats show <id>` - show one chat",
			"`messages send <chat-id> <text>` - add a user message",
		},
	})
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	req := &chatv1.CreateChatRequest{
		Title:            ctx.Flag("title"),
		GroupId:          ctx.Flag("group-id"),
		Model:            ctx.Flag("model"),
		WebSearchEnabled: ctx.BoolFlag("web-search"),
		Mode:             parseChatMode(ctx.Flag("mode")),
	}
	resp, err := h.client.CreateChat(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("create chat", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetChat() == nil {
		return fmt.Errorf("server returned no chat")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created chat %s.", resp.Msg.GetChat().GetId())},
		Changes: []string{formatChat(resp.Msg.GetChat())},
		NextCommand: []string{
			fmt.Sprintf("`messages send %s <text>` - add a user message", resp.Msg.GetChat().GetId()),
			fmt.Sprintf("`messages stream %s` - stream a completion", resp.Msg.GetChat().GetId()),
		},
	})
}

func (h *handlers) show(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetChat(context.Background(), connect.NewRequest(&chatv1.GetChatRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("show chat %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetChat() == nil {
		return fmt.Errorf("server returned no chat")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched chat %s.", resp.Msg.GetChat().GetId())},
		ResultsHeading: "Chat",
		Results:        []string{formatChat(resp.Msg.GetChat())},
	})
}

func (h *handlers) update(ctx cliapp.RunContext) error {
	req := &chatv1.UpdateChatRequest{Id: ctx.Positional("id")}
	if ctx.Flag("title") != "" {
		req.Title = ctx.Flag("title")
		req.HasTitle = true
	}
	if ctx.Flag("group-id") != "" {
		req.GroupId = ctx.Flag("group-id")
		req.HasGroupId = true
	}
	if ctx.Flag("model") != "" {
		req.Model = ctx.Flag("model")
		req.HasModel = true
	}
	if ctx.BoolFlag("web-search") {
		req.WebSearchEnabled = true
		req.HasWebSearchEnabled = true
	}
	if ctx.BoolFlag("no-web-search") {
		req.WebSearchEnabled = false
		req.HasWebSearchEnabled = true
	}
	if ctx.Flag("active-leaf") != "" {
		req.ActiveLeafMessageId = ctx.Flag("active-leaf")
		req.HasActiveLeafMessageId = true
	}
	resp, err := h.client.UpdateChat(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("update chat", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetChat() == nil {
		return fmt.Errorf("server returned no chat")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Updated chat %s.", resp.Msg.GetChat().GetId())},
		Changes: []string{formatChat(resp.Msg.GetChat())},
	})
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.DeleteChat(context.Background(), connect.NewRequest(&chatv1.DeleteChatRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("delete chat %q", id), err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Deleted chat %s.", id)},
	})
}

func (h *handlers) groups(ctx cliapp.RunContext) error {
	resp, err := h.client.ListGroups(context.Background(), connect.NewRequest(&chatv1.ListGroupsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list chat groups", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no groups response")
	}
	results := make([]string, 0, len(resp.Msg.GetGroups()))
	for _, g := range resp.Msg.GetGroups() {
		results = append(results, formatGroup(g))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d chat group(s).", len(resp.Msg.GetGroups()))},
		ResultsHeading: "Chat Groups",
		Results:        results,
		RetrievalHints: []string{"`chats group-create --name <name> --color <hex>` - create a group"},
	})
}

func (h *handlers) createGroup(ctx cliapp.RunContext) error {
	resp, err := h.client.CreateGroup(context.Background(), connect.NewRequest(&chatv1.CreateGroupRequest{
		Name:  ctx.Flag("name"),
		Color: ctx.Flag("color"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("create chat group", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetGroup() == nil {
		return fmt.Errorf("server returned no group")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created chat group %s.", resp.Msg.GetGroup().GetId())},
		Changes: []string{formatGroup(resp.Msg.GetGroup())},
	})
}

func (h *handlers) updateGroup(ctx cliapp.RunContext) error {
	req := &chatv1.UpdateGroupRequest{Id: ctx.Positional("id")}
	if ctx.Flag("name") != "" {
		req.Name = ctx.Flag("name")
		req.HasName = true
	}
	if ctx.Flag("color") != "" {
		req.Color = ctx.Flag("color")
		req.HasColor = true
	}
	if ctx.BoolFlag("collapsed") {
		req.Collapsed = true
		req.HasCollapsed = true
	}
	if ctx.BoolFlag("expanded") {
		req.Collapsed = false
		req.HasCollapsed = true
	}
	if value := ctx.Flag("sort-order"); value != "" {
		n, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return fmt.Errorf("--sort-order must be an integer: %w", err)
		}
		req.SortOrder = int32(n)
		req.HasSortOrder = true
	}
	resp, err := h.client.UpdateGroup(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("update chat group", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetGroup() == nil {
		return fmt.Errorf("server returned no group")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Updated chat group %s.", resp.Msg.GetGroup().GetId())},
		Changes: []string{formatGroup(resp.Msg.GetGroup())},
	})
}

func (h *handlers) deleteGroup(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.DeleteGroup(context.Background(), connect.NewRequest(&chatv1.DeleteGroupRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("delete chat group %q", id), err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Deleted chat group %s.", id)},
	})
}

func parseChatMode(value string) sharedv1.ChatMode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "agent":
		return sharedv1.ChatMode_CHAT_MODE_AGENT
	case "llm", "":
		return sharedv1.ChatMode_CHAT_MODE_LLM
	default:
		return sharedv1.ChatMode_CHAT_MODE_UNSPECIFIED
	}
}

func formatChat(c *chatv1.Chat) string {
	if c == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s - %s [group=%s mode=%s model=%s web_search=%t active_leaf=%s]",
		c.GetId(), c.GetTitle(), c.GetGroupId(), c.GetMode().String(), c.GetModel(), c.GetWebSearchEnabled(), c.GetActiveLeafMessageId())
}

func formatGroup(g *chatv1.ChatGroup) string {
	if g == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s - %s [color=%s collapsed=%t sort=%d]", g.GetId(), g.GetName(), g.GetColor(), g.GetCollapsed(), g.GetSortOrder())
}
