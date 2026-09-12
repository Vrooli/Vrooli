package integrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/vrooli/api-core/database"
	"notification-hub/internal/integrationconfig"
)

func Schema() string { return integrationconfig.Schema() }

type LiveConfig struct {
	EventsAPIBase         string                   `json:"events_api_base"`
	WebhookURL            string                   `json:"webhook_url"`
	Pattern               string                   `json:"pattern"`
	Templates             map[string]EventTemplate `json:"templates,omitempty"`
	SensitivityBySeverity map[string]string        `json:"sensitivity_by_severity,omitempty"`
}

type EventTemplate struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type LiveConfigStore struct {
	mu     sync.RWMutex
	db     *database.RoutedDB
	config LiveConfig
}

func NewLiveConfigStore(config LiveConfig) *LiveConfigStore { return &LiveConfigStore{config: config} }

// NewPersistentLiveConfigStore loads the runtime integration settings from
// the scenario-owned database. The memory-only constructor remains useful for
// isolated handler tests; production always uses this constructor.
func NewPersistentLiveConfigStore(ctx context.Context, db *database.RoutedDB, initial LiveConfig) (*LiveConfigStore, error) {
	if db == nil {
		return nil, fmt.Errorf("event integration config database is nil")
	}
	store := &LiveConfigStore{db: db}
	var templatesJSON string
	if _, err := db.ExecContext(ctx, `ALTER TABLE event_integration_config ADD COLUMN sensitivity_by_severity_json TEXT NOT NULL DEFAULT '{}'`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil, fmt.Errorf("ensure event integration sensitivity mapping: %w", err)
	}
	var sensitivityJSON string
	err := db.QueryRowContext(ctx, `SELECT events_api_base, webhook_url, pattern, templates_json, sensitivity_by_severity_json FROM event_integration_config WHERE id = 1`).Scan(&initial.EventsAPIBase, &initial.WebhookURL, &initial.Pattern, &templatesJSON, &sensitivityJSON)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("load event integration config: %w", err)
	}
	if err == nil && strings.TrimSpace(templatesJSON) != "" {
		if unmarshalErr := json.Unmarshal([]byte(templatesJSON), &initial.Templates); unmarshalErr != nil {
			return nil, fmt.Errorf("decode event integration templates: %w", unmarshalErr)
		}
	}
	if err == nil && strings.TrimSpace(sensitivityJSON) != "" {
		if unmarshalErr := json.Unmarshal([]byte(sensitivityJSON), &initial.SensitivityBySeverity); unmarshalErr != nil {
			return nil, fmt.Errorf("decode event sensitivity mapping: %w", unmarshalErr)
		}
	}
	store.setMemory(initial)
	if err == sql.ErrNoRows {
		if persistErr := store.persist(ctx, store.Get()); persistErr != nil {
			return nil, persistErr
		}
	}
	return store, nil
}

func (s *LiveConfigStore) Get() LiveConfig { s.mu.RLock(); defer s.mu.RUnlock(); return s.config }
func (s *LiveConfigStore) Set(config LiveConfig) {
	s.mu.Lock()
	s.config = normalizedConfig(config)
	s.mu.Unlock()
}

func (s *LiveConfigStore) setMemory(config LiveConfig) {
	s.mu.Lock()
	s.config = normalizedConfig(config)
	s.mu.Unlock()
}

// Apply changes the live value and persists it before returning.
func (s *LiveConfigStore) Apply(ctx context.Context, config LiveConfig) error {
	config = normalizedConfig(config)
	if s.db != nil {
		if err := s.persist(ctx, config); err != nil {
			return err
		}
	}
	s.setMemory(config)
	return nil
}

func (s *LiveConfigStore) persist(ctx context.Context, config LiveConfig) error {
	templates, err := json.Marshal(config.Templates)
	if err != nil {
		return fmt.Errorf("encode event integration templates: %w", err)
	}
	sensitivity, err := json.Marshal(config.SensitivityBySeverity)
	if err != nil {
		return fmt.Errorf("encode event sensitivity mapping: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO event_integration_config (id, events_api_base, webhook_url, pattern, templates_json, sensitivity_by_severity_json, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET events_api_base = excluded.events_api_base,
		webhook_url = excluded.webhook_url, pattern = excluded.pattern,
		templates_json = excluded.templates_json, sensitivity_by_severity_json = excluded.sensitivity_by_severity_json, updated_at = CURRENT_TIMESTAMP`,
		config.EventsAPIBase, config.WebhookURL, config.Pattern, string(templates), string(sensitivity))
	if err != nil {
		return fmt.Errorf("persist event integration config: %w", err)
	}
	return nil
}

func normalizedConfig(config LiveConfig) LiveConfig {
	pattern := strings.TrimSpace(config.Pattern)
	if pattern == "incident.*" {
		// Lifecycle types have two segments after the incident namespace, for
		// example incident.opened.v1. Keep the legacy intent from becoming a
		// no-op subscription.
		pattern = "incident.**"
	}
	templates := make(map[string]EventTemplate, len(config.Templates))
	for eventType, template := range config.Templates {
		eventType = strings.TrimSpace(eventType)
		if eventType == "" {
			continue
		}
		templates[eventType] = EventTemplate{Title: strings.TrimSpace(template.Title), Body: strings.TrimSpace(template.Body)}
	}
	sensitivity := map[string]string{"critical": "critical", "warning": "sensitive", "informational": "public"}
	for severity, label := range config.SensitivityBySeverity {
		severity = strings.ToLower(strings.TrimSpace(severity))
		label = strings.TrimSpace(label)
		if severity != "" && label != "" {
			sensitivity[severity] = label
		}
	}
	return LiveConfig{EventsAPIBase: strings.TrimSpace(config.EventsAPIBase), WebhookURL: strings.TrimSpace(config.WebhookURL), Pattern: pattern, Templates: templates, SensitivityBySeverity: sensitivity}
}

func Handler(store *LiveConfigStore, onChange func(LiveConfig) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(store.Get())
			return
		}
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var config LiveConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "invalid event integration config", http.StatusBadRequest)
			return
		}
		if err := store.Apply(r.Context(), config); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if onChange != nil {
			if err := onChange(store.Get()); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(store.Get())
	})
}
