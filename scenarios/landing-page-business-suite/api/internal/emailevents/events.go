package emailevents

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Event struct {
	Event       string            `json:"event"`
	Timestamp   int64             `json:"timestamp"`
	Email       string            `json:"email"`
	SGEventID   string            `json:"sg_event_id"`
	SGMessageID string            `json:"sg_message_id"`
	Response    string            `json:"response"`
	Reason      string            `json:"reason"`
	Status      string            `json:"status"`
	CustomArgs  map[string]string `json:"custom_args"`
}

type Status struct {
	Provider, Delivery, Reason string
	Terminal                   bool
}

func MapStatus(event Event) Status {
	switch strings.ToLower(strings.TrimSpace(event.Event)) {
	case "delivered":
		return Status{"delivered", "delivered", "", true}
	case "bounce":
		return Status{"bounce", "rejected", bounceReason(event), true}
	case "dropped":
		return Status{"dropped", "rejected", "other", true}
	case "spamreport":
		return Status{"spamreport", "rejected", "spam", true}
	case "deferred":
		return Status{"deferred", "deferred", "other", false}
	case "processed":
		return Status{"processed", "pending", "", false}
	default:
		return Status{Provider: strings.ToLower(strings.TrimSpace(event.Event)), Delivery: "pending"}
	}
}

func bounceReason(event Event) string {
	code := strings.TrimSpace(event.Status)
	if len(code) >= 3 {
		switch code[:3] {
		case "550", "551", "553":
			return "invalid"
		case "552":
			return "mailbox_unavailable"
		case "421", "450", "451", "452":
			return "blocked"
		}
	}
	response := strings.ToLower(event.Response + " " + event.Reason)
	if strings.Contains(response, "spam") || strings.Contains(response, "blocked") {
		return "blocked"
	}
	if strings.Contains(response, "mailbox") || strings.Contains(response, "over quota") {
		return "mailbox_unavailable"
	}
	if strings.Contains(response, "invalid") || strings.Contains(response, "unknown user") {
		return "invalid"
	}
	return "other"
}

type Store interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}
type Repository struct{ DB Store }

func (r *Repository) Record(ctx context.Context, event Event, receivedAt time.Time) (bool, error) {
	if r == nil || r.DB == nil {
		return false, fmt.Errorf("email event store is unavailable")
	}
	status := MapStatus(event)
	args, err := json.Marshal(event.CustomArgs)
	if err != nil {
		return false, err
	}
	requestID := strings.TrimSpace(event.CustomArgs["lpbs_sign_in_request_id"])
	result, err := r.DB.ExecContext(ctx, `INSERT INTO auth_email_events (sg_event_id,sg_message_id,sign_in_request_id,event,event_name,reason_class,occurred_at,event_timestamp,email,provider_status,raw_custom_args,received_at) VALUES ($1,$2,NULLIF($3,'')::uuid,$4,$4,$5,to_timestamp($6),to_timestamp($6),$7,$8,$9,$10) ON CONFLICT (sg_event_id) DO NOTHING`, event.SGEventID, event.SGMessageID, requestID, event.Event, status.Reason, event.Timestamp, strings.ToLower(strings.TrimSpace(event.Email)), status.Provider, args, receivedAt.UTC())
	if err != nil {
		return false, fmt.Errorf("record email event: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return false, nil
	}
	messageID := strings.TrimSpace(event.SGMessageID)
	if requestID != "" {
		_, err = r.updateToken(ctx, "id", requestID, status, event)
	} else if messageID != "" {
		_, err = r.updateToken(ctx, "provider_message_id", messageID, status, event)
	}
	return true, err
}

func (r *Repository) updateToken(ctx context.Context, column, value string, status Status, event Event) (sql.Result, error) {
	if column != "id" && column != "provider_message_id" {
		return nil, fmt.Errorf("invalid token correlation")
	}
	where := fmt.Sprintf("%s=$5", column)
	if column == "provider_message_id" { where = "(provider_message_id=$5 OR split_part(provider_message_id,'.',1)=split_part($5,'.',1))" }
	query := fmt.Sprintf(`UPDATE auth_tokens SET provider_status=$1,provider_status_at=to_timestamp($2),provider_reason_class=$3,delivery_status=$4 WHERE %s AND (provider_status IS NULL OR provider_status NOT IN ('delivered','bounce','dropped','spamreport') OR $1 IN ('delivered','bounce','dropped','spamreport'))`, where)
	return r.DB.ExecContext(ctx, query, status.Provider, event.Timestamp, status.Reason, status.Delivery, value)
}
