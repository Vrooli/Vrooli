package hub

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Recipient and the following records are projections of the verified
// authenticator subject. They deliberately contain no local credential or
// profile fields.
type Recipient struct {
	ID, Subject, TrustPosture, CreatedAt, UpdatedAt string
}

type ChannelAddress struct {
	ID, DeviceID, Channel, Address string
	ApprovedLabels                 []string
}

type EscalationStep struct {
	Ordinal int
	Channel string
}

type EscalationChain struct {
	RecipientID string
	Steps       []EscalationStep
}

type channelTarget struct {
	Channel, Address, MachineID string
	ApprovedLabels              []string
}

func (s *Service) channelTargets(ctx context.Context, recipient string) ([]channelTarget, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT a.channel, a.address, d.machine_id, a.approved_labels FROM channel_addresses a JOIN devices d ON d.id=a.device_id JOIN recipients r ON r.id=d.recipient_id WHERE r.subject=? ORDER BY a.created_at`, recipient)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []channelTarget
	for rows.Next() {
		var item channelTarget
		var labels string
		if err := rows.Scan(&item.Channel, &item.Address, &item.MachineID, &labels); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(labels), &item.ApprovedLabels)
		out = append(out, item)
	}
	return out, rows.Err()
}

func approvedLabel(labels []string, wanted string) bool {
	for _, label := range labels {
		if strings.EqualFold(strings.TrimSpace(label), wanted) {
			return true
		}
	}
	return false
}

func (s *Service) EnsureRecipient(ctx context.Context, subject string) error {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return fmt.Errorf("recipient subject is required")
	}
	now := s.clock.Now().UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	_, err := s.db.ExecContext(ctx, `INSERT INTO recipients (id, subject, trust_posture, created_at, updated_at) VALUES (?, ?, 'personal', ?, ?) ON CONFLICT(subject) DO UPDATE SET updated_at=excluded.updated_at`, uuid.NewString(), subject, now, now)
	return err
}

func (s *Service) GetRecipient(ctx context.Context, subject string) (Recipient, error) {
	if err := s.EnsureRecipient(ctx, subject); err != nil {
		return Recipient{}, err
	}
	var out Recipient
	err := s.db.QueryRowContext(ctx, `SELECT id, subject, trust_posture, created_at, updated_at FROM recipients WHERE subject=?`, subject).Scan(&out.ID, &out.Subject, &out.TrustPosture, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func (s *Service) UpsertDevice(ctx context.Context, recipient string, device Device) (Device, error) {
	if err := s.EnsureRecipient(ctx, recipient); err != nil {
		return Device{}, err
	}
	device.Name = strings.TrimSpace(device.Name)
	if device.Name == "" {
		return Device{}, fmt.Errorf("device name is required")
	}
	if device.ID == "" {
		device.ID = uuid.NewString()
	}
	now := s.clock.Now().UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	_, err := s.db.ExecContext(ctx, `INSERT INTO devices (id, recipient_id, name, machine_id, created_at) VALUES (?, (SELECT id FROM recipients WHERE subject=?), ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name=excluded.name, machine_id=excluded.machine_id`, device.ID, recipient, device.Name, device.MachineID, now)
	if err != nil {
		return Device{}, err
	}
	return s.deviceByID(ctx, device.ID)
}

func (s *Service) deviceByID(ctx context.Context, id string) (Device, error) {
	var out Device
	err := s.db.QueryRowContext(ctx, `SELECT id, name, machine_id FROM devices WHERE id=?`, id).Scan(&out.ID, &out.Name, &out.MachineID)
	if err != nil {
		return Device{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT channel FROM channel_addresses WHERE device_id=? ORDER BY channel`, id)
	if err != nil {
		return Device{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var channel string
		if err := rows.Scan(&channel); err != nil {
			return Device{}, err
		}
		out.Channels = append(out.Channels, channel)
	}
	return out, rows.Err()
}

func (s *Service) RemoveDevice(ctx context.Context, recipient, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM push_subscriptions WHERE device_id IN (SELECT id FROM devices WHERE id=? AND recipient_id=(SELECT id FROM recipients WHERE subject=?))`, id, recipient)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM devices WHERE id=? AND recipient_id=(SELECT id FROM recipients WHERE subject=?)`, id, recipient)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Service) UpsertChannelAddress(ctx context.Context, recipient string, address ChannelAddress) (ChannelAddress, error) {
	if address.DeviceID == "" || strings.TrimSpace(address.Channel) == "" || strings.TrimSpace(address.Address) == "" {
		return ChannelAddress{}, fmt.Errorf("device_id, channel, and address are required")
	}
	if len(address.ApprovedLabels) == 0 {
		address.ApprovedLabels = []string{"public"}
	}
	labels, err := json.Marshal(uniqueStrings(address.ApprovedLabels))
	if err != nil {
		return ChannelAddress{}, err
	}
	if address.ID == "" {
		address.ID = uuid.NewString()
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO channel_addresses (id, device_id, channel, address, approved_labels, created_at) SELECT ?, d.id, ?, ?, ?, ? FROM devices d JOIN recipients r ON r.id=d.recipient_id WHERE d.id=? AND r.subject=? ON CONFLICT(device_id, channel) DO UPDATE SET address=excluded.address, approved_labels=excluded.approved_labels`, address.ID, address.Channel, address.Address, string(labels), s.clock.Now().UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), address.DeviceID, recipient)
	if err != nil {
		return ChannelAddress{}, err
	}
	return s.channelAddressByID(ctx, recipient, address.ID)
}

func (s *Service) channelAddressByID(ctx context.Context, recipient, id string) (ChannelAddress, error) {
	var out ChannelAddress
	var labels string
	err := s.db.QueryRowContext(ctx, `SELECT a.id, a.device_id, a.channel, a.address, a.approved_labels FROM channel_addresses a JOIN devices d ON d.id=a.device_id JOIN recipients r ON r.id=d.recipient_id WHERE a.id=? AND r.subject=?`, id, recipient).Scan(&out.ID, &out.DeviceID, &out.Channel, &out.Address, &labels)
	if err != nil {
		return ChannelAddress{}, err
	}
	_ = json.Unmarshal([]byte(labels), &out.ApprovedLabels)
	return out, nil
}

func (s *Service) RemoveChannelAddress(ctx context.Context, recipient, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM channel_addresses WHERE id IN (SELECT a.id FROM channel_addresses a JOIN devices d ON d.id=a.device_id JOIN recipients r ON r.id=d.recipient_id WHERE a.id=? AND r.subject=?)`, id, recipient)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Service) ListQuietWindows(ctx context.Context, recipient string) ([]QuietWindow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT q.id, q.weekday, q.start_time, q.end_time, q.timezone, q.critical_override FROM quiet_windows q JOIN recipients r ON r.id=q.recipient_id WHERE r.subject=? ORDER BY q.weekday, q.start_time`, recipient)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QuietWindow
	for rows.Next() {
		var window QuietWindow
		var critical int
		if err := rows.Scan(&window.ID, &window.Weekday, &window.Start, &window.End, &window.Timezone, &critical); err != nil {
			return nil, err
		}
		window.CriticalOverride = critical != 0
		out = append(out, window)
	}
	return out, rows.Err()
}

func (s *Service) DeleteQuietWindow(ctx context.Context, recipient, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM quiet_windows WHERE id IN (SELECT q.id FROM quiet_windows q JOIN recipients r ON r.id=q.recipient_id WHERE q.id=? AND r.subject=?)`, id, recipient)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Service) SetEscalationChain(ctx context.Context, recipient string, channels []string) (EscalationChain, error) {
	if err := s.EnsureRecipient(ctx, recipient); err != nil {
		return EscalationChain{}, err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM escalation_chains WHERE recipient_id=(SELECT id FROM recipients WHERE subject=?)`, recipient)
	if err != nil {
		return EscalationChain{}, err
	}
	for ordinal, channel := range uniqueStrings(channels) {
		channel = strings.TrimSpace(channel)
		if channel == "" {
			continue
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO escalation_chains (id, recipient_id, ordinal, channel) VALUES (?, (SELECT id FROM recipients WHERE subject=?), ?, ?)`, uuid.NewString(), recipient, ordinal, channel); err != nil {
			return EscalationChain{}, err
		}
	}
	return s.GetEscalationChain(ctx, recipient)
}

func (s *Service) GetEscalationChain(ctx context.Context, recipient string) (EscalationChain, error) {
	chain := EscalationChain{}
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM recipients WHERE subject=?`, recipient).Scan(&chain.RecipientID); err != nil {
		if err == sql.ErrNoRows {
			if err := s.EnsureRecipient(ctx, recipient); err != nil {
				return EscalationChain{}, err
			}
			if err := s.db.QueryRowContext(ctx, `SELECT id FROM recipients WHERE subject=?`, recipient).Scan(&chain.RecipientID); err != nil {
				return EscalationChain{}, err
			}
		} else {
			return EscalationChain{}, err
		}
	}
	rows, err := s.db.QueryContext(ctx, `SELECT ordinal, channel FROM escalation_chains WHERE recipient_id=? ORDER BY ordinal`, chain.RecipientID)
	if err != nil {
		return EscalationChain{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var step EscalationStep
		if err := rows.Scan(&step.Ordinal, &step.Channel); err != nil {
			return EscalationChain{}, err
		}
		chain.Steps = append(chain.Steps, step)
	}
	return chain, rows.Err()
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
