package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"portal/internal/clock"

	"github.com/google/uuid"
)

type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	if clk == nil {
		clk = clock.System{}
	}
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const (
	timeFormat   = time.RFC3339Nano
	chatColumns  = `id, title, preview, COALESCE(group_id, ''), sort_order, model, web_search_enabled, mode, agent_harness, COALESCE(active_leaf_message_id, ''), system_prompt, created_at, updated_at`
	groupColumns = `id, name, color, collapsed, sort_order, created_at, updated_at`
	msgColumns   = `id, chat_id, COALESCE(parent_message_id, ''), sibling_index, role, content, model, token_count, response_id, finish_reason, web_search, created_at, updated_at`
)

func (r *sqliteRepository) ListChats(ctx context.Context, input SearchInput) ([]Chat, []ChatGroup, error) {
	groups, err := r.ListGroups(ctx)
	if err != nil {
		return nil, nil, err
	}

	query := "SELECT " + chatColumns + " FROM chats"
	var clauses []string
	var args []any
	if input.GroupID != "" {
		clauses = append(clauses, "group_id = ?")
		args = append(args, input.GroupID)
	}
	if strings.TrimSpace(input.Query) != "" {
		clauses = append(clauses, "rowid IN (SELECT rowid FROM chats_fts WHERE chats_fts MATCH ?)")
		args = append(args, input.Query)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY sort_order ASC, updated_at DESC, title ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("list chats: %w", err)
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		c, err := scanChat(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan chat: %w", err)
		}
		chats = append(chats, c)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate chats: %w", err)
	}
	return chats, groups, nil
}

func (r *sqliteRepository) CreateChat(ctx context.Context, input CreateChatInput) (Chat, error) {
	now := r.clock.Now().UTC()
	c := Chat{
		ID:               uuid.NewString(),
		Title:            defaultString(strings.TrimSpace(input.Title), "New chat"),
		GroupID:          strings.TrimSpace(input.GroupID),
		Model:            defaultString(strings.TrimSpace(input.Model), DefaultModel),
		WebSearchEnabled: input.WebSearchEnabled,
		Mode:             NormalizeMode(input.Mode),
		AgentHarness:     NormalizeAgentHarness(input.AgentHarness),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := validateChat(c); err != nil {
		return Chat{}, err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO chats (id, title, preview, group_id, sort_order, model, web_search_enabled, mode, agent_harness, active_leaf_message_id, system_prompt, created_at, updated_at)
VALUES (?, ?, '', nullif(?, ''), 0, ?, ?, ?, ?, NULL, '', ?, ?)`,
		c.ID, c.Title, c.GroupID, c.Model, boolToInt(c.WebSearchEnabled), string(c.Mode), string(c.AgentHarness),
		c.CreatedAt.Format(timeFormat), c.UpdatedAt.Format(timeFormat),
	)
	if err != nil {
		return Chat{}, fmt.Errorf("create chat %q: %w", c.ID, err)
	}
	return c, nil
}

func (r *sqliteRepository) GetChat(ctx context.Context, id string) (Chat, error) {
	c, err := scanChat(r.db.QueryRowContext(ctx, "SELECT "+chatColumns+" FROM chats WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return Chat{}, ErrNotFound{Resource: "chat", ID: id}
	}
	if err != nil {
		return Chat{}, fmt.Errorf("get chat %q: %w", id, err)
	}
	return c, nil
}

func (r *sqliteRepository) UpdateChat(ctx context.Context, input UpdateChatInput) (Chat, error) {
	current, err := r.GetChat(ctx, input.ID)
	if err != nil {
		return Chat{}, err
	}
	if input.Title != nil {
		current.Title = defaultString(strings.TrimSpace(*input.Title), "New chat")
	}
	if input.GroupID != nil {
		current.GroupID = strings.TrimSpace(*input.GroupID)
	}
	if input.Model != nil {
		current.Model = defaultString(strings.TrimSpace(*input.Model), DefaultModel)
	}
	if input.WebSearchEnabled != nil {
		current.WebSearchEnabled = *input.WebSearchEnabled
	}
	if input.ClearActiveLeafMessageID {
		current.ActiveLeafMessageID = ""
	} else if input.ActiveLeafMessageID != nil {
		current.ActiveLeafMessageID = strings.TrimSpace(*input.ActiveLeafMessageID)
	}
	current.UpdatedAt = r.clock.Now().UTC()
	if err := validateChat(current); err != nil {
		return Chat{}, err
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE chats
SET title = ?, group_id = nullif(?, ''), model = ?, web_search_enabled = ?, active_leaf_message_id = nullif(?, ''), updated_at = ?
WHERE id = ?`,
		current.Title, current.GroupID, current.Model, boolToInt(current.WebSearchEnabled), current.ActiveLeafMessageID,
		current.UpdatedAt.Format(timeFormat), current.ID,
	)
	if err != nil {
		return Chat{}, fmt.Errorf("update chat %q: %w", input.ID, err)
	}
	return current, rowsAffectedOrNotFound(res, "chat", input.ID)
}

func (r *sqliteRepository) DeleteChat(ctx context.Context, id string) (bool, error) {
	res, err := r.db.ExecContext(ctx, "DELETE FROM chats WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("delete chat %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete chat %q rows affected: %w", id, err)
	}
	return n > 0, nil
}

func (r *sqliteRepository) ListGroups(ctx context.Context) ([]ChatGroup, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+groupColumns+" FROM chat_groups ORDER BY sort_order ASC, name ASC")
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()
	var groups []ChatGroup
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate groups: %w", err)
	}
	return groups, nil
}

func (r *sqliteRepository) CreateGroup(ctx context.Context, input CreateGroupInput) (ChatGroup, error) {
	now := r.clock.Now().UTC()
	g := ChatGroup{
		ID:        uuid.NewString(),
		Name:      defaultString(strings.TrimSpace(input.Name), "Group"),
		Color:     defaultString(strings.TrimSpace(input.Color), "#2563eb"),
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO chat_groups (id, name, color, collapsed, sort_order, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		g.ID, g.Name, g.Color, boolToInt(g.Collapsed), g.SortOrder, g.CreatedAt.Format(timeFormat), g.UpdatedAt.Format(timeFormat),
	)
	if err != nil {
		return ChatGroup{}, fmt.Errorf("create group %q: %w", g.ID, err)
	}
	return g, nil
}

func (r *sqliteRepository) UpdateGroup(ctx context.Context, input UpdateGroupInput) (ChatGroup, error) {
	current, err := r.getGroup(ctx, input.ID)
	if err != nil {
		return ChatGroup{}, err
	}
	if input.Name != nil {
		current.Name = defaultString(strings.TrimSpace(*input.Name), "Group")
	}
	if input.Color != nil {
		current.Color = defaultString(strings.TrimSpace(*input.Color), "#2563eb")
	}
	if input.Collapsed != nil {
		current.Collapsed = *input.Collapsed
	}
	if input.SortOrder != nil {
		current.SortOrder = *input.SortOrder
	}
	current.UpdatedAt = r.clock.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
UPDATE chat_groups
SET name = ?, color = ?, collapsed = ?, sort_order = ?, updated_at = ?
WHERE id = ?`,
		current.Name, current.Color, boolToInt(current.Collapsed), current.SortOrder, current.UpdatedAt.Format(timeFormat), current.ID,
	)
	if err != nil {
		return ChatGroup{}, fmt.Errorf("update group %q: %w", input.ID, err)
	}
	return current, rowsAffectedOrNotFound(res, "group", input.ID)
}

func (r *sqliteRepository) DeleteGroup(ctx context.Context, id string) (bool, error) {
	res, err := r.db.ExecContext(ctx, "DELETE FROM chat_groups WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("delete group %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete group %q rows affected: %w", id, err)
	}
	return n > 0, nil
}

func (r *sqliteRepository) ListMessages(ctx context.Context, chatID string) ([]Message, string, error) {
	if _, err := r.GetChat(ctx, chatID); err != nil {
		return nil, "", err
	}
	rows, err := r.db.QueryContext(ctx, "SELECT "+msgColumns+" FROM messages WHERE chat_id = ? ORDER BY created_at ASC, sibling_index ASC", chatID)
	if err != nil {
		return nil, "", fmt.Errorf("list messages for chat %q: %w", chatID, err)
	}
	defer rows.Close()
	var messages []Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, "", fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate messages: %w", err)
	}
	if err := r.hydrateSearchAttachments(ctx, messages); err != nil {
		return nil, "", err
	}
	leaf, err := r.activeLeaf(ctx, chatID)
	if err != nil {
		return nil, "", err
	}
	return messages, leaf, nil
}

func (r *sqliteRepository) SendMessage(ctx context.Context, input SendMessageInput) (Message, error) {
	return r.AppendMessage(ctx, input)
}

func (r *sqliteRepository) AppendMessage(ctx context.Context, input SendMessageInput) (Message, error) {
	if strings.TrimSpace(input.ChatID) == "" || strings.TrimSpace(input.Content) == "" {
		return Message{}, ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Message{}, fmt.Errorf("begin send message: %w", err)
	}
	defer rollback(tx)

	if err := ensureChatExists(ctx, tx, input.ChatID); err != nil {
		return Message{}, err
	}
	if input.ParentMessageID != "" {
		if err := ensureMessageInChat(ctx, tx, input.ParentMessageID, input.ChatID); err != nil {
			return Message{}, err
		}
	}
	msg, err := r.insertMessage(ctx, tx, input.ChatID, input.ParentMessageID, input.Role, input.Content, input.Model, input.WebSearch)
	if err != nil {
		return Message{}, err
	}
	if err := updateChatAfterMessage(ctx, tx, input.ChatID, msg.ID, msg.Content, msg.UpdatedAt); err != nil {
		return Message{}, err
	}
	if err := tx.Commit(); err != nil {
		return Message{}, fmt.Errorf("commit send message: %w", err)
	}
	return msg, nil
}

func (r *sqliteRepository) CreateUsageRecord(ctx context.Context, input CreateUsageInput) (UsageRecord, error) {
	if strings.TrimSpace(input.ChatID) == "" || strings.TrimSpace(input.MessageID) == "" || strings.TrimSpace(input.Model) == "" {
		return UsageRecord{}, ErrInvalidInput
	}
	now := r.clock.Now().UTC()
	record := UsageRecord{
		ID:               uuid.NewString(),
		ChatID:           strings.TrimSpace(input.ChatID),
		MessageID:        strings.TrimSpace(input.MessageID),
		Provider:         defaultString(strings.TrimSpace(input.Provider), "openrouter"),
		Model:            strings.TrimSpace(input.Model),
		PromptTokens:     input.PromptTokens,
		CompletionTokens: input.CompletionTokens,
		TotalTokens:      input.PromptTokens + input.CompletionTokens,
		CostUSD:          input.CostUSD,
		CreatedAt:        now,
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO usage_records (id, chat_id, message_id, provider, model, prompt_tokens, completion_tokens, total_tokens, cost_usd, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID, record.ChatID, record.MessageID, record.Provider, record.Model, record.PromptTokens,
		record.CompletionTokens, record.TotalTokens, record.CostUSD, record.CreatedAt.Format(timeFormat),
	)
	if err != nil {
		return UsageRecord{}, fmt.Errorf("create usage record for message %q: %w", input.MessageID, err)
	}
	return record, nil
}

func (r *sqliteRepository) CreateSearchAttachment(ctx context.Context, input CreateSearchAttachmentInput) (SearchAttachment, error) {
	if strings.TrimSpace(input.ChatID) == "" || strings.TrimSpace(input.MessageID) == "" || strings.TrimSpace(input.Query) == "" {
		return SearchAttachment{}, ErrInvalidInput
	}
	hitsJSON, err := json.Marshal(input.Hits)
	if err != nil {
		return SearchAttachment{}, fmt.Errorf("encode search attachment hits: %w", err)
	}
	now := r.clock.Now().UTC()
	attachment := SearchAttachment{
		ID:        uuid.NewString(),
		ChatID:    strings.TrimSpace(input.ChatID),
		MessageID: strings.TrimSpace(input.MessageID),
		Query:     strings.TrimSpace(input.Query),
		Hits:      input.Hits,
		Degraded:  input.Degraded,
		Reason:    strings.TrimSpace(input.Reason),
		LatencyMS: input.LatencyMS,
		CreatedAt: now,
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO search_attachments (id, chat_id, message_id, query, hits_json, degraded, reason, latency_ms, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		attachment.ID, attachment.ChatID, attachment.MessageID, attachment.Query, string(hitsJSON), boolToInt(attachment.Degraded),
		attachment.Reason, attachment.LatencyMS, attachment.CreatedAt.Format(timeFormat),
	)
	if err != nil {
		return SearchAttachment{}, fmt.Errorf("create search attachment for message %q: %w", input.MessageID, err)
	}
	return attachment, nil
}

func (r *sqliteRepository) ListSearchAttachments(ctx context.Context, chatID string, limit int) ([]SearchAttachment, error) {
	if strings.TrimSpace(chatID) == "" {
		return nil, ErrInvalidInput
	}
	if limit <= 0 {
		limit = 5
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, chat_id, message_id, query, hits_json, degraded, reason, latency_ms, created_at
FROM search_attachments
WHERE chat_id = ?
ORDER BY created_at DESC
LIMIT ?`, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("list search attachments for chat %q: %w", chatID, err)
	}
	defer rows.Close()
	var attachments []SearchAttachment
	for rows.Next() {
		attachment, err := scanSearchAttachment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan search attachment: %w", err)
		}
		attachments = append(attachments, attachment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search attachments: %w", err)
	}
	return attachments, nil
}

func (r *sqliteRepository) BranchMessage(ctx context.Context, input BranchMessageInput) (Message, error) {
	originalID := strings.TrimSpace(input.MessageID)
	if originalID == "" {
		return Message{}, ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Message{}, fmt.Errorf("begin branch message: %w", err)
	}
	defer rollback(tx)

	original, err := getMessageForUpdate(ctx, tx, originalID)
	if err != nil {
		return Message{}, err
	}
	model := defaultString(strings.TrimSpace(input.Model), original.Model)
	msg, err := r.insertMessage(ctx, tx, original.ChatID, original.ParentMessageID, original.Role, input.Content, model, original.WebSearch)
	if err != nil {
		return Message{}, err
	}
	if err := updateChatAfterMessage(ctx, tx, original.ChatID, msg.ID, msg.Content, msg.UpdatedAt); err != nil {
		return Message{}, err
	}
	if err := tx.Commit(); err != nil {
		return Message{}, fmt.Errorf("commit branch message: %w", err)
	}
	return msg, nil
}

func (r *sqliteRepository) insertMessage(ctx context.Context, tx *sql.Tx, chatID, parentID string, role MessageRole, content, model string, webSearch *bool) (Message, error) {
	if role == "" {
		role = RoleUser
	}
	now := r.clock.Now().UTC()
	msg := Message{
		ID:              uuid.NewString(),
		ChatID:          chatID,
		ParentMessageID: strings.TrimSpace(parentID),
		Role:            role,
		Content:         content,
		Model:           strings.TrimSpace(model),
		WebSearch:       webSearch,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := validateRole(role); err != nil {
		return Message{}, err
	}
	next, err := nextSiblingIndex(ctx, tx, msg.ChatID, msg.ParentMessageID)
	if err != nil {
		return Message{}, err
	}
	msg.SiblingIndex = next
	var web sql.NullInt64
	if msg.WebSearch != nil {
		web = sql.NullInt64{Int64: int64(boolToInt(*msg.WebSearch)), Valid: true}
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO messages (id, chat_id, parent_message_id, sibling_index, role, content, model, token_count, response_id, finish_reason, web_search, created_at, updated_at)
VALUES (?, ?, nullif(?, ''), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.ChatID, msg.ParentMessageID, msg.SiblingIndex, string(msg.Role), msg.Content, msg.Model,
		msg.TokenCount, msg.ResponseID, msg.FinishReason, web, msg.CreatedAt.Format(timeFormat), msg.UpdatedAt.Format(timeFormat),
	)
	if err != nil {
		return Message{}, fmt.Errorf("insert message %q: %w", msg.ID, err)
	}
	return msg, nil
}

func (r *sqliteRepository) getGroup(ctx context.Context, id string) (ChatGroup, error) {
	g, err := scanGroup(r.db.QueryRowContext(ctx, "SELECT "+groupColumns+" FROM chat_groups WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return ChatGroup{}, ErrNotFound{Resource: "group", ID: id}
	}
	if err != nil {
		return ChatGroup{}, fmt.Errorf("get group %q: %w", id, err)
	}
	return g, nil
}

func (r *sqliteRepository) activeLeaf(ctx context.Context, chatID string) (string, error) {
	var leaf sql.NullString
	err := r.db.QueryRowContext(ctx, "SELECT active_leaf_message_id FROM chats WHERE id = ?", chatID).Scan(&leaf)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound{Resource: "chat", ID: chatID}
	}
	if err != nil {
		return "", fmt.Errorf("get active leaf for chat %q: %w", chatID, err)
	}
	return nullStringValue(leaf), nil
}

func nextSiblingIndex(ctx context.Context, tx *sql.Tx, chatID, parentID string) (int32, error) {
	var next sql.NullInt64
	var err error
	if parentID == "" {
		err = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(sibling_index), -1) + 1 FROM messages WHERE chat_id = ? AND parent_message_id IS NULL", chatID).Scan(&next)
	} else {
		err = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(sibling_index), -1) + 1 FROM messages WHERE chat_id = ? AND parent_message_id = ?", chatID, parentID).Scan(&next)
	}
	if err != nil {
		return 0, fmt.Errorf("next sibling index: %w", err)
	}
	return int32(next.Int64), nil
}

func ensureChatExists(ctx context.Context, tx *sql.Tx, chatID string) error {
	var id string
	err := tx.QueryRowContext(ctx, "SELECT id FROM chats WHERE id = ?", chatID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound{Resource: "chat", ID: chatID}
	}
	if err != nil {
		return fmt.Errorf("check chat %q: %w", chatID, err)
	}
	return nil
}

func ensureMessageInChat(ctx context.Context, tx *sql.Tx, messageID, chatID string) error {
	var id string
	err := tx.QueryRowContext(ctx, "SELECT id FROM messages WHERE id = ? AND chat_id = ?", messageID, chatID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound{Resource: "message", ID: messageID}
	}
	if err != nil {
		return fmt.Errorf("check message %q: %w", messageID, err)
	}
	return nil
}

func getMessageForUpdate(ctx context.Context, tx *sql.Tx, id string) (Message, error) {
	msg, err := scanMessage(tx.QueryRowContext(ctx, "SELECT "+msgColumns+" FROM messages WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, ErrNotFound{Resource: "message", ID: id}
	}
	if err != nil {
		return Message{}, fmt.Errorf("get message %q: %w", id, err)
	}
	return msg, nil
}

func updateChatAfterMessage(ctx context.Context, tx *sql.Tx, chatID, leafID, content string, now time.Time) error {
	preview := content
	if len(preview) > 160 {
		preview = preview[:160]
	}
	_, err := tx.ExecContext(ctx, `
UPDATE chats
SET active_leaf_message_id = ?, preview = ?, updated_at = ?
WHERE id = ?`,
		leafID, preview, now.Format(timeFormat), chatID,
	)
	if err != nil {
		return fmt.Errorf("update chat after message %q: %w", chatID, err)
	}
	return nil
}

func (r *sqliteRepository) hydrateSearchAttachments(ctx context.Context, messages []Message) error {
	if len(messages) == 0 {
		return nil
	}
	byID := make(map[string]int, len(messages))
	ids := make([]string, 0, len(messages))
	for i := range messages {
		byID[messages[i].ID] = i
		ids = append(ids, messages[i].ID)
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, chat_id, message_id, query, hits_json, degraded, reason, latency_ms, created_at
FROM search_attachments
WHERE message_id IN (`+placeholders+`)
ORDER BY created_at ASC`, args...)
	if err != nil {
		return fmt.Errorf("list message search attachments: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		attachment, err := scanSearchAttachment(rows)
		if err != nil {
			return fmt.Errorf("scan message search attachment: %w", err)
		}
		if index, ok := byID[attachment.MessageID]; ok {
			messages[index].SearchAttachments = append(messages[index].SearchAttachments, attachment)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate message search attachments: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanChat(row rowScanner) (Chat, error) {
	var c Chat
	var webSearch int
	var mode, harness, created, updated string
	if err := row.Scan(&c.ID, &c.Title, &c.Preview, &c.GroupID, &c.SortOrder, &c.Model, &webSearch, &mode, &harness, &c.ActiveLeafMessageID, &c.SystemPrompt, &created, &updated); err != nil {
		return Chat{}, err
	}
	c.WebSearchEnabled = webSearch != 0
	c.Mode = ChatMode(mode)
	c.AgentHarness = AgentHarness(harness)
	var err error
	c.CreatedAt, err = parseTime(created)
	if err != nil {
		return Chat{}, err
	}
	c.UpdatedAt, err = parseTime(updated)
	if err != nil {
		return Chat{}, err
	}
	return c, nil
}

func scanGroup(row rowScanner) (ChatGroup, error) {
	var g ChatGroup
	var collapsed int
	var created, updated string
	if err := row.Scan(&g.ID, &g.Name, &g.Color, &collapsed, &g.SortOrder, &created, &updated); err != nil {
		return ChatGroup{}, err
	}
	g.Collapsed = collapsed != 0
	var err error
	g.CreatedAt, err = parseTime(created)
	if err != nil {
		return ChatGroup{}, err
	}
	g.UpdatedAt, err = parseTime(updated)
	if err != nil {
		return ChatGroup{}, err
	}
	return g, nil
}

func scanMessage(row rowScanner) (Message, error) {
	var m Message
	var role, created, updated string
	var web sql.NullInt64
	if err := row.Scan(&m.ID, &m.ChatID, &m.ParentMessageID, &m.SiblingIndex, &role, &m.Content, &m.Model, &m.TokenCount, &m.ResponseID, &m.FinishReason, &web, &created, &updated); err != nil {
		return Message{}, err
	}
	m.Role = MessageRole(role)
	if web.Valid {
		value := web.Int64 != 0
		m.WebSearch = &value
	}
	var err error
	m.CreatedAt, err = parseTime(created)
	if err != nil {
		return Message{}, err
	}
	m.UpdatedAt, err = parseTime(updated)
	if err != nil {
		return Message{}, err
	}
	return m, nil
}

func scanSearchAttachment(row rowScanner) (SearchAttachment, error) {
	var attachment SearchAttachment
	var hitsJSON, created string
	var degraded int
	if err := row.Scan(
		&attachment.ID,
		&attachment.ChatID,
		&attachment.MessageID,
		&attachment.Query,
		&hitsJSON,
		&degraded,
		&attachment.Reason,
		&attachment.LatencyMS,
		&created,
	); err != nil {
		return SearchAttachment{}, err
	}
	if strings.TrimSpace(hitsJSON) != "" {
		if err := json.Unmarshal([]byte(hitsJSON), &attachment.Hits); err != nil {
			return SearchAttachment{}, fmt.Errorf("decode hits json: %w", err)
		}
	}
	attachment.Degraded = degraded != 0
	var err error
	attachment.CreatedAt, err = parseTime(created)
	if err != nil {
		return SearchAttachment{}, err
	}
	return attachment, nil
}

func validateChat(c Chat) error {
	if c.Title == "" || c.Model == "" {
		return ErrInvalidInput
	}
	switch c.Mode {
	case ChatModeLLM, ChatModeAgent:
	default:
		return ErrInvalidInput
	}
	switch c.AgentHarness {
	case AgentHarnessClaudeCode, AgentHarnessCodex, AgentHarnessOpencode, AgentHarnessGrok:
	default:
		return ErrInvalidInput
	}
	return nil
}

func validateRole(role MessageRole) error {
	switch role {
	case RoleSystem, RoleUser, RoleAssistant, RoleAgent:
		return nil
	default:
		return ErrInvalidInput
	}
}

func rowsAffectedOrNotFound(res sql.Result, resource, id string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s %q rows affected: %w", resource, id, err)
	}
	if n == 0 {
		return ErrNotFound{Resource: resource, ID: id}
	}
	return nil
}

func parseTime(value string) (time.Time, error) {
	t, err := time.Parse(timeFormat, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", value, err)
	}
	return t, nil
}

func rollback(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullStringValue(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}
