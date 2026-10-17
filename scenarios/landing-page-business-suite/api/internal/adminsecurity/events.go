package adminsecurity

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

type Store interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type queryStore interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Repository struct{ DB Store }

type Event struct {
	EventType string         `json:"event"`
	CreatedAt string         `json:"created_at"`
	IPHint    string         `json:"ip_hint,omitempty"`
	UserAgent string         `json:"user_agent,omitempty"`
	Detail    map[string]any `json:"detail,omitempty"`
}

func HashSessionID(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}

func (r *Repository) Record(ctx context.Context, event, email, sessionID, ip, userAgent string, detail map[string]any) error {
	if r == nil || r.DB == nil {
		return fmt.Errorf("admin security event store unavailable")
	}
	if detail == nil {
		detail = map[string]any{}
	}
	payload, err := json.Marshal(detail)
	if err != nil {
		return fmt.Errorf("marshal admin security event: %w", err)
	}
	var hash any
	if strings.TrimSpace(sessionID) != "" {
		hash = HashSessionID(sessionID)
	}
	_, err = r.DB.ExecContext(ctx, `INSERT INTO admin_security_events (admin_email, session_id_hash, event_type, ip_address, user_agent, detail) VALUES ($1, $2, $3, $4, $5, $6::jsonb)`, email, hash, event, ip, userAgent, string(payload))
	return err
}

func DevicePrefix(ip string) string {
	host := net.ParseIP(strings.TrimSpace(ip))
	if host == nil {
		return ""
	}
	if v4 := host.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d.0/24", v4[0], v4[1], v4[2])
	}
	mask := host.Mask(net.CIDRMask(48, 128))
	return fmt.Sprintf("%x:%x:%x:%x::/48", mask[0:2], mask[2:4], mask[4:6], mask[6:8])
}

func (r *Repository) IsNewDevice(ctx context.Context, email, userAgent, ip string) (bool, error) {
	if r == nil || r.DB == nil {
		return false, fmt.Errorf("admin security event store unavailable")
	}
	reader, ok := r.DB.(queryStore)
	if !ok {
		return false, fmt.Errorf("admin security event read store unavailable")
	}
	rows, err := reader.QueryContext(ctx, `SELECT ip_address FROM admin_security_events WHERE admin_email = $1 AND event_type IN ('login_success', 'new_device_login') AND created_at > NOW() - INTERVAL '30 days' AND user_agent = $2`, email, userAgent)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	current := DevicePrefix(ip)
	for rows.Next() {
		var previous sql.NullString
		if err := rows.Scan(&previous); err != nil {
			return false, err
		}
		if current != "" && current == DevicePrefix(previous.String) {
			return false, nil
		}
	}
	return true, rows.Err()
}

func (r *Repository) List(ctx context.Context, email string, limit int) ([]Event, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	reader, ok := r.DB.(queryStore)
	if !ok {
		return nil, fmt.Errorf("admin security event read store unavailable")
	}
	rows, err := reader.QueryContext(ctx, `SELECT event_type, created_at, ip_address, user_agent, detail FROM admin_security_events WHERE admin_email = $1 ORDER BY created_at DESC LIMIT $2`, email, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Event, 0, limit)
	for rows.Next() {
		var event Event
		var created time.Time
		var ip, ua sql.NullString
		var detail []byte
		if err := rows.Scan(&event.EventType, &created, &ip, &ua, &detail); err != nil {
			return nil, err
		}
		event.CreatedAt = created.UTC().Format(time.RFC3339)
		event.IPHint = ip.String
		event.UserAgent = ua.String
		if len(detail) > 0 {
			_ = json.Unmarshal(detail, &event.Detail)
		}
		result = append(result, event)
	}
	return result, rows.Err()
}
