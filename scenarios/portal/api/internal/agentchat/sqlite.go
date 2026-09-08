package agentchat

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"portal/internal/chat"
	"portal/internal/integrations/agentmanager"
)

type SQLExecutor interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

const admissionOwnerClause = "COALESCE((SELECT owner FROM agent_chat_run_owners WHERE admission_id=agent_chat_runs.id),'')=?"

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Reserve(ctx context.Context, chatID, messageID string) (Binding, error) {
	if strings.TrimSpace(chatID) == "" || strings.TrimSpace(messageID) == "" {
		return Binding{}, fmt.Errorf("chat and message identity required")
	}
	b := Binding{ID: uuid.NewString(), ChatID: chatID, MessageID: messageID}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Binding{}, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO agent_chat_runs(id,chat_id,message_id) VALUES(?,?,?) ON CONFLICT(chat_id,message_id) DO NOTHING`, b.ID, chatID, messageID)
	if err != nil {
		return Binding{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return Binding{}, err
	}
	if n != 1 {
		return Binding{}, ErrAlreadyAdmitted
	}
	if owner := chat.RequestOwner(ctx); owner != "" {
		if _, err = tx.ExecContext(ctx, "INSERT INTO agent_chat_run_owners(admission_id,owner) VALUES(?,?)", b.ID, owner); err != nil {
			return Binding{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return Binding{}, err
	}
	return b, nil
}

func (r *sqliteRepository) SetBriefID(ctx context.Context, id, briefID string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE agent_chat_runs SET brief_id=? WHERE id=?`, strings.TrimSpace(briefID), strings.TrimSpace(id))
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *sqliteRepository) Bind(ctx context.Context, id string, session agentmanager.Session) error {
	if strings.TrimSpace(session.RunID) == "" || strings.TrimSpace(session.TaskID) == "" {
		return ErrBindingConflict
	}
	result, err := r.db.ExecContext(ctx, `UPDATE agent_chat_runs SET task_id=?,run_id=? WHERE id=? AND (run_id IS NULL OR (run_id=? AND task_id=?)) AND `+admissionOwnerClause, session.TaskID, session.RunID, id, session.RunID, session.TaskID, chat.RequestOwner(ctx))
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrBindingConflict
	}
	return nil
}

func (r *sqliteRepository) Get(ctx context.Context, chatID, messageID string) (Binding, error) {
	var b Binding
	err := r.db.QueryRowContext(ctx, `SELECT id,chat_id,message_id,task_id,COALESCE(run_id,''),COALESCE(brief_id,'') FROM agent_chat_runs WHERE chat_id=? AND message_id=? AND `+admissionOwnerClause, chatID, messageID, chat.RequestOwner(ctx)).Scan(&b.ID, &b.ChatID, &b.MessageID, &b.TaskID, &b.RunID, &b.BriefID)
	return b, err
}

// List retains unbound admissions: a missing launch reply is not absence of work.
// The cursor is an insertion sequence, independent of randomly generated IDs.
func (r *sqliteRepository) List(ctx context.Context, token string, size int) (AdmissionPage, error) {
	if size == 0 {
		size = 50
	}
	if size < 1 || size > 100 {
		return AdmissionPage{}, ErrInvalidPage
	}
	var after int64
	if token != "" {
		parsed, err := strconv.ParseInt(token, 10, 64)
		if err != nil || parsed < 1 || strconv.FormatInt(parsed, 10) != token {
			return AdmissionPage{}, ErrInvalidPage
		}
		after = parsed
	}
	rows, err := r.db.QueryContext(ctx, `SELECT rowid,id,chat_id,message_id,task_id,COALESCE(run_id,''),COALESCE(brief_id,'') FROM agent_chat_runs WHERE rowid > ? AND `+admissionOwnerClause+` ORDER BY rowid LIMIT ?`, after, chat.RequestOwner(ctx), size+1)
	if err != nil {
		return AdmissionPage{}, err
	}
	defer rows.Close()
	page := AdmissionPage{Bindings: make([]Binding, 0, size)}
	var last int64
	for rows.Next() {
		var sequence int64
		var binding Binding
		if err := rows.Scan(&sequence, &binding.ID, &binding.ChatID, &binding.MessageID, &binding.TaskID, &binding.RunID, &binding.BriefID); err != nil {
			return AdmissionPage{}, err
		}
		if len(page.Bindings) == size {
			page.NextPageToken = strconv.FormatInt(last, 10)
			break
		}
		page.Bindings = append(page.Bindings, binding)
		last = sequence
	}
	if err := rows.Err(); err != nil {
		return AdmissionPage{}, err
	}
	return page, nil
}
