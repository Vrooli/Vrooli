package chat

import (
	"agent-inbox/cli/internal/support"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type chatRecord struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Preview               string   `json:"preview"`
	Model                 string   `json:"model"`
	IsRead                bool     `json:"is_read"`
	IsArchived            bool     `json:"is_archived"`
	IsStarred             bool     `json:"is_starred"`
	LabelIDs              []string `json:"label_ids"`
	ToolsEnabled          bool     `json:"tools_enabled"`
	WebSearchEnabled      bool     `json:"web_search_enabled"`
	ChatMode              string   `json:"chat_mode"`
	AgentRunID            string   `json:"agent_run_id"`
	ActiveTemplateID      string   `json:"active_template_id"`
	ActiveTemplateToolIDs []string `json:"active_template_tool_ids"`
	CreatedAt             string   `json:"created_at"`
	UpdatedAt             string   `json:"updated_at"`
}

type toolCallRecord struct {
	ID           string `json:"id"`
	ToolName     string `json:"tool_name"`
	Status       string `json:"status"`
	ScenarioName string `json:"scenario_name"`
	StartedAt    string `json:"started_at"`
}

type messageRecord struct {
	ID              string `json:"id"`
	Role            string `json:"role"`
	Content         string `json:"content"`
	Model           string `json:"model"`
	ToolCallID      string `json:"tool_call_id"`
	ParentMessageID string `json:"parent_message_id"`
	CreatedAt       string `json:"created_at"`
}

type chatDetailResponse struct {
	Chat            chatRecord       `json:"chat"`
	Messages        []messageRecord  `json:"messages"`
	ToolCallRecords []toolCallRecord `json:"tool_call_records"`
}

type searchResult struct {
	Chat      chatRecord `json:"chat"`
	MessageID string     `json:"message_id"`
	Snippet   string     `json:"snippet"`
	Rank      float64    `json:"rank"`
	MatchType string     `json:"match_type"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "chat",
		Description: "Inbox chat operations",
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, NeedsAPI: true, Description: "List chats [--archived] [--starred]", Run: func(args []string) error { return RunList(core, args) }},
			{Name: "get", Aliases: []string{"open"}, NeedsAPI: true, Description: "Get one chat with messages", Run: func(args []string) error { return RunGet(core, args) }},
			{Name: "create", Aliases: []string{"new"}, NeedsAPI: true, Description: "Create a chat", Run: func(args []string) error { return RunCreate(core, args) }},
			{Name: "update", NeedsAPI: true, Description: "Update chat metadata", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", NeedsAPI: true, Description: "Delete a chat", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "message", Aliases: []string{"send"}, NeedsAPI: true, Description: "Append a message to a chat", Run: func(args []string) error { return runMessage(core, args) }},
			{Name: "search", NeedsAPI: true, Description: "Search chats and messages", Run: func(args []string) error { return runSearch(core, args) }},
			{Name: "archive", NeedsAPI: true, Description: "Archive or unarchive a chat", Run: func(args []string) error { return runArchive(core, args) }},
			{Name: "read", NeedsAPI: true, Description: "Mark a chat read or unread", Run: func(args []string) error { return runRead(core, args) }},
			{Name: "star", NeedsAPI: true, Description: "Star or unstar a chat", Run: func(args []string) error { return runStar(core, args) }},
			{Name: "mark-all-read", NeedsAPI: true, Description: "Mark all chats as read", Run: func(args []string) error { return runMarkAllRead(core, args) }},
			{Name: "delete-archived", NeedsAPI: true, Description: "Delete all archived chats", Run: func(args []string) error { return runDeleteArchived(core, args) }},
			{Name: "auto-name", NeedsAPI: true, Description: "Generate a name for a chat", Run: func(args []string) error { return runAutoName(core, args) }},
			{Name: "export", NeedsAPI: true, Description: "Export a chat to stdout or a file", Run: func(args []string) error { return runExport(core, args) }},
		},
	}
}

func RunList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chat list")
	archived := fs.Bool("archived", false, "Only archived chats")
	starred := fs.Bool("starred", false, "Only starred chats")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *archived {
		query.Set("archived", "true")
	}
	if *starred {
		query.Set("starred", "true")
	}

	body, err := core.Get("/chats", query)
	if err != nil {
		return err
	}

	var chats []chatRecord
	if err := support.Decode(body, &chats); err != nil {
		return err
	}

	results := make([]string, 0, len(chats))
	for _, chat := range chats {
		state := make([]string, 0, 4)
		if chat.IsArchived {
			state = append(state, "archived")
		}
		if chat.IsStarred {
			state = append(state, "starred")
		}
		if !chat.IsRead {
			state = append(state, "unread")
		}
		if chat.ChatMode != "" {
			state = append(state, "mode="+chat.ChatMode)
		}
		line := fmt.Sprintf("%s | %s | %s", chat.ID, support.Truncate(chat.Name, 48), support.FormatTime(chat.UpdatedAt))
		if len(state) > 0 {
			line += " | " + strings.Join(state, ", ")
		}
		if preview := support.Truncate(chat.Preview, 72); preview != "" {
			line += "\n  " + preview
		}
		results = append(results, line)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Chats: %d", len(chats)), fmt.Sprintf("Filters: archived=%t starred=%t", *archived, *starred)},
		ResultsHeading: "Chats",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " chat get <chat-id>", support.CLIName + " chat search --query <text>"},
	}
	return support.PrintList(*jsonOutput, report)
}

func RunGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chat get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: chat get <chat-id> [--json]")
	}
	id := fs.Arg(0)

	body, err := core.Get("/chats/"+id, nil)
	if err != nil {
		return err
	}

	var resp chatDetailResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{
		"Name: " + resp.Chat.Name,
		"Model: " + resp.Chat.Model,
		"Mode: " + resp.Chat.ChatMode,
		fmt.Sprintf("Messages: %d", len(resp.Messages)),
		fmt.Sprintf("Tool calls: %d", len(resp.ToolCallRecords)),
	}
	if len(resp.Chat.LabelIDs) > 0 {
		results = append(results, "Labels: "+strings.Join(resp.Chat.LabelIDs, ", "))
	}
	if resp.Chat.ActiveTemplateID != "" {
		results = append(results, "Active template: "+resp.Chat.ActiveTemplateID)
	}
	for _, msg := range resp.Messages {
		line := fmt.Sprintf("%s | %s | %s", support.FormatTime(msg.CreatedAt), msg.Role, support.Truncate(msg.Content, 96))
		results = append(results, line)
	}
	if len(resp.ToolCallRecords) > 0 {
		results = append(results, "Recent tool calls:")
		for _, call := range resp.ToolCallRecords {
			results = append(results, fmt.Sprintf("  %s | %s | %s", call.Status, call.ToolName, support.FormatTime(call.StartedAt)))
		}
	}

	report := cliapp.ListReport{
		Summary:        []string{"Chat: " + resp.Chat.ID, fmt.Sprintf("Updated: %s", support.FormatTime(resp.Chat.UpdatedAt))},
		ResultsHeading: "Conversation",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " chat message " + resp.Chat.ID + " --content \"...\"", support.CLIName + " agent status " + resp.Chat.ID},
	}
	return support.PrintList(*jsonOutput, report)
}

func RunCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chat create")
	name := fs.String("name", "", "Chat name")
	model := fs.String("model", "", "Model ID")
	mode := fs.String("mode", "llm", "Chat mode: llm or agent")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	input := map[string]interface{}{
		"chat_mode": *mode,
	}
	if strings.TrimSpace(*name) != "" {
		input["name"] = strings.TrimSpace(*name)
	}
	if strings.TrimSpace(*model) != "" {
		input["model"] = strings.TrimSpace(*model)
	}

	body, err := core.Request("POST", "/chats", nil, input)
	if err != nil {
		return err
	}

	var chat chatRecord
	if err := support.Decode(body, &chat); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Chat created", "Chat ID: " + chat.ID},
		Changes: []string{
			"Name: " + chat.Name,
			"Mode: " + chat.ChatMode,
			"Model: " + chat.Model,
		},
		NextCommand: []string{support.CLIName + " chat get " + chat.ID, support.CLIName + " chat message " + chat.ID + " --content \"...\""},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chat update")
	name := fs.String("name", "", "New chat name")
	model := fs.String("model", "", "New model")
	toolsEnabled := fs.String("tools-enabled", "", "Set tools enabled to true or false")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: chat update <chat-id> [--name NAME] [--model MODEL] [--tools-enabled true|false] [--json]")
	}
	id := fs.Arg(0)

	input := map[string]interface{}{}
	if strings.TrimSpace(*name) != "" {
		input["name"] = strings.TrimSpace(*name)
	}
	if strings.TrimSpace(*model) != "" {
		input["model"] = strings.TrimSpace(*model)
	}
	if value, err := support.ParseOptionalBool(*toolsEnabled); err != nil {
		return err
	} else if value != nil {
		input["tools_enabled"] = *value
	}
	if len(input) == 0 {
		return fmt.Errorf("at least one field must be provided")
	}

	body, err := core.Request("PATCH", "/chats/"+id, nil, input)
	if err != nil {
		return err
	}

	var chat chatRecord
	if err := support.Decode(body, &chat); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Chat updated", "Chat ID: " + chat.ID},
		Changes: []string{
			"Name: " + chat.Name,
			"Model: " + chat.Model,
			"Tools: " + support.BoolLabel(chat.ToolsEnabled),
		},
		NextCommand: []string{support.CLIName + " chat get " + chat.ID},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chat delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: chat delete <chat-id> [--json]")
	}
	id := fs.Arg(0)

	if _, err := core.Request("DELETE", "/chats/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Chat deleted", "Chat ID: " + id},
		Changes:     []string{"Removed the chat and its message history"},
		NextCommand: []string{support.CLIName + " chat list"},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runMessage(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chat message")
	content := fs.String("content", "", "Message content")
	role := fs.String("role", "user", "Message role")
	model := fs.String("model", "", "Model identifier for assistant messages")
	parentID := fs.String("parent", "", "Parent message ID")
	webSearch := fs.String("web-search", "", "Override web search with true or false")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: chat message <chat-id> --content TEXT [--role user|assistant|system|tool] [--json]")
	}
	if strings.TrimSpace(*content) == "" {
		return fmt.Errorf("--content is required")
	}
	id := fs.Arg(0)

	input := map[string]interface{}{
		"role":    strings.TrimSpace(*role),
		"content": *content,
	}
	if strings.TrimSpace(*model) != "" {
		input["model"] = strings.TrimSpace(*model)
	}
	if strings.TrimSpace(*parentID) != "" {
		input["parent_message_id"] = strings.TrimSpace(*parentID)
	}
	if value, err := support.ParseOptionalBool(*webSearch); err != nil {
		return err
	} else if value != nil {
		input["web_search"] = *value
	}

	body, err := core.Request("POST", "/chats/"+id+"/messages", nil, input)
	if err != nil {
		return err
	}

	var msg messageRecord
	if err := support.Decode(body, &msg); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Message added", "Message ID: " + msg.ID},
		Changes: []string{
			"Chat ID: " + id,
			"Role: " + msg.Role,
			"Content: " + support.Truncate(msg.Content, 96),
		},
		NextCommand: []string{support.CLIName + " chat get " + id},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chat search")
	queryText := fs.String("query", "", "Search query")
	limit := fs.Int("limit", 20, "Maximum results")
	perChat := fs.Int("per-chat", 1, "Maximum results per chat")
	caseSensitive := fs.Bool("case-sensitive", false, "Case-sensitive search")
	wholeWord := fs.Bool("whole-word", false, "Match whole words only")
	regex := fs.Bool("regex", false, "Treat query as regex")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*queryText) == "" {
		return fmt.Errorf("--query is required")
	}

	query := url.Values{}
	query.Set("q", *queryText)
	query.Set("limit", strconv.Itoa(*limit))
	query.Set("per_chat", strconv.Itoa(*perChat))
	if *caseSensitive {
		query.Set("case_sensitive", "true")
	}
	if *wholeWord {
		query.Set("whole_word", "true")
	}
	if *regex {
		query.Set("regex", "true")
	}

	body, err := core.Get("/search", query)
	if err != nil {
		return err
	}

	var resultsResp []searchResult
	if err := support.Decode(body, &resultsResp); err != nil {
		return err
	}

	results := make([]string, 0, len(resultsResp))
	for _, item := range resultsResp {
		line := fmt.Sprintf("%s | %s | %s", item.Chat.ID, item.MatchType, support.Truncate(item.Chat.Name, 40))
		if strings.TrimSpace(item.Snippet) != "" {
			line += "\n  " + support.Truncate(item.Snippet, 120)
		}
		results = append(results, line)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Matches: %d", len(resultsResp)), "Query: " + *queryText},
		ResultsHeading: "Search Results",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " chat get <chat-id>", support.CLIName + " chat search --query \"" + *queryText + "\" --regex"},
	}
	return support.PrintList(*jsonOutput, report)
}

func runArchive(core *cliapp.ScenarioApp, args []string) error {
	return runToggle(core, "archive", "/chats/%s/archive", "is_archived", args)
}

func runRead(core *cliapp.ScenarioApp, args []string) error {
	return runToggle(core, "read", "/chats/%s/read", "is_read", args)
}

func runStar(core *cliapp.ScenarioApp, args []string) error {
	return runToggle(core, "star", "/chats/%s/star", "is_starred", args)
}

func runToggle(core *cliapp.ScenarioApp, commandName, pathTemplate, field string, args []string) error {
	fs := support.NewFlagSet("chat " + commandName)
	setValue := fs.String("set", "", "Explicit true or false value")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: chat %s <chat-id> [--set true|false] [--json]", commandName)
	}
	id := fs.Arg(0)
	input := map[string]interface{}{}
	if value, err := support.ParseOptionalBool(*setValue); err != nil {
		return err
	} else if value != nil {
		input["value"] = *value
	}

	body, err := core.Request("POST", fmt.Sprintf(pathTemplate, id), nil, input)
	if err != nil {
		return err
	}

	var result map[string]bool
	if err := support.Decode(body, &result); err != nil {
		return err
	}
	value := result[field]

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Chat %s updated", commandName), fmt.Sprintf("%s: %t", field, value)},
		Changes:     []string{"Chat ID: " + id},
		NextCommand: []string{support.CLIName + " chat get " + id},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runMarkAllRead(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chat mark-all-read")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Request("POST", "/chats/mark-all-read", nil, map[string]interface{}{})
	if err != nil {
		return err
	}

	var resp struct {
		Updated int `json:"updated"`
	}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"All chats marked as read"},
		Changes:     []string{fmt.Sprintf("Updated chats: %d", resp.Updated)},
		NextCommand: []string{support.CLIName + " chat list"},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runDeleteArchived(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chat delete-archived")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Request("DELETE", "/chats/archived", nil, nil)
	if err != nil {
		return err
	}

	var resp struct {
		Deleted int `json:"deleted"`
	}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Archived chats deleted"},
		Changes:     []string{fmt.Sprintf("Deleted chats: %d", resp.Deleted)},
		NextCommand: []string{support.CLIName + " chat list"},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runAutoName(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chat auto-name")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: chat auto-name <chat-id> [--json]")
	}
	id := fs.Arg(0)

	body, err := core.Request("POST", "/chats/"+id+"/auto-name", nil, map[string]interface{}{})
	if err != nil {
		return err
	}

	var chat chatRecord
	if err := support.Decode(body, &chat); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Chat renamed", "Chat ID: " + chat.ID},
		Changes:     []string{"New name: " + chat.Name},
		NextCommand: []string{support.CLIName + " chat get " + chat.ID},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runExport(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chat export")
	format := fs.String("format", "markdown", "Export format: markdown, json, txt")
	output := fs.String("output", "", "Write export to this file instead of stdout")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: chat export <chat-id> [--format markdown|json|txt] [--output path]")
	}
	id := fs.Arg(0)

	query := url.Values{}
	query.Set("format", *format)
	body, err := core.Get("/chats/"+id+"/export", query)
	if err != nil {
		return err
	}

	if strings.TrimSpace(*output) == "" {
		_, err = os.Stdout.Write(body)
		return err
	}

	target := support.AbsPath(*output)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(target, body, 0o644); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Chat exported", "File: " + target},
		Changes:     []string{"Format: " + *format, fmt.Sprintf("Bytes written: %d", len(body))},
		NextCommand: []string{support.CLIName + " chat get " + id},
	}
	return support.PrintMutation(false, report)
}
