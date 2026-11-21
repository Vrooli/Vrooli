package hub

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode"
)

// templateConversationTitle gives the inbox a useful identifier even when the
// page headline is an outcome sentence rather than a name for the report.
func templateConversationTitle(name string, data map[string]any, question string) string {
	meta, _ := data["meta"].(map[string]any)
	field := func(key string) string { v, _ := meta[key].(string); return strings.TrimSpace(v) }
	title := field("title")
	switch name {
	case "daily-briefing", "opportunity-roadmap":
		label, format := "Daily briefing", "Mon 2 Jan"
		if name == "opportunity-roadmap" {
			label, format = "Opportunity roadmap", "2 Jan"
		}
		// Steward's data date is authoritative; sentAt is a fallback for other producers.
		date := ""
		if i := strings.LastIndex(title, " for "); i >= 0 {
			date = title[i+5:]
		}
		if len(date) < 10 {
			date = field("sentAt")
		}
		if len(date) >= 10 {
			if day, err := time.Parse("2006-01-02", date[:10]); err == nil {
				return label + " · " + day.Format(format)
			}
		}
		if title != "" {
			return label + " · " + title
		}
		return label
	case "research-report":
		if topic := field("topic"); topic != "" {
			if !strings.Contains(topic, " ") && strings.Contains(topic, "-") {
				topic = strings.ReplaceAll(topic, "-", " ")
				runes := []rune(topic)
				if len(runes) > 0 {
					runes[0] = unicode.ToUpper(runes[0])
					topic = string(runes)
				}
			}
			return "Research report · " + topic
		}
		if title != "" {
			if strings.HasPrefix(strings.ToLower(title), "research report") {
				return title
			}
			return "Research report · " + title
		}
		return "Research report"
	case "usage-mode":
		if title != "" {
			if strings.HasPrefix(strings.ToLower(title), "usage mode:") {
				return title
			}
			return "Usage mode: " + title
		}
	case "structural-change":
		if title != "" {
			if strings.HasPrefix(strings.ToLower(title), "structural change:") {
				return title
			}
			return "Structural change: " + title
		}
	default:
		if title != "" {
			return title
		}
	}
	return strings.TrimSpace(question)
}

// backfillTemplateTitles only changes open generic titles. The predicate is
// rechecked on update so an intervening revision or answer is never overwritten.
func (s *Service) backfillTemplateTitles(ctx context.Context) {
	rows, err := s.db.QueryContext(ctx, `SELECT a.id, a.template_name, a.template_data_json, a.question FROM asks a JOIN notifications n ON n.id=a.notification_id WHERE a.state IN ('pending','escalated') AND a.template_name <> '' AND (n.title IN ('Status update','Decision required','Decision needed') OR n.title LIKE 'Decision needed: %')`)
	if err != nil {
		s.log.Printf("backfill template titles: %v", err)
		return
	}
	type item struct{ id, name, raw, question string }
	var items []item
	for rows.Next() {
		var v item
		if err := rows.Scan(&v.id, &v.name, &v.raw, &v.question); err != nil {
			s.log.Printf("backfill template title scan: %v", err)
			continue
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		s.log.Printf("backfill template title rows: %v", err)
	}
	rows.Close()
	for _, v := range items {
		var data map[string]any
		if json.Unmarshal([]byte(v.raw), &data) != nil {
			continue
		}
		title := templateConversationTitle(v.name, data, v.question)
		if title == "" {
			continue
		}
		_, err := s.db.ExecContext(ctx, `UPDATE notifications SET title=? WHERE id=(SELECT notification_id FROM asks WHERE id=? AND state IN ('pending','escalated') AND template_name=? AND template_data_json=? AND question=?) AND (title IN ('Status update','Decision required','Decision needed') OR title LIKE 'Decision needed: %')`, title, v.id, v.name, v.raw, v.question)
		if err != nil {
			s.log.Printf("backfill template title: %v", err)
		}
	}
}
