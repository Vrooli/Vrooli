// Package subscription implements persistent subscription management for
// event-driven consumers. Subscriptions define which events to forward and
// how (SSE or webhook).
//
// DOC: docs/guides/creating-subscriptions.md
package subscription

import (
	"context"
	"time"
)

// DeliveryType defines how events are delivered to subscribers.
type DeliveryType string

const (
	DeliverySSE     DeliveryType = "sse"
	DeliveryWebhook DeliveryType = "webhook"
)

// Subscription represents a persistent event subscription.
type Subscription struct {
	ID             int64        `json:"id"`
	Name           string       `json:"name"`
	OwnerScenario  string       `json:"owner_scenario"`
	EventPattern   string       `json:"event_pattern"`
	SourceFilter   string       `json:"source_filter,omitempty"`
	DeliveryType   DeliveryType `json:"delivery_type"`
	DeliveryTarget string       `json:"delivery_target"`
	Enabled        bool         `json:"enabled"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// Health tracks delivery statistics for a subscription.
type Health struct {
	SubscriptionID      int64  `json:"subscription_id"`
	TotalDelivered      int64  `json:"total_delivered"`
	TotalFailed         int64  `json:"total_failed"`
	ConsecutiveFailures int64  `json:"consecutive_failures"`
	LastDeliveredAt     string `json:"last_delivered_at,omitempty"`
	LastFailedAt        string `json:"last_failed_at,omitempty"`
	Status              string `json:"status"` // "active", "circuit_broken"
}

// QueuedDelivery is a durable attempt to fan an event out to one webhook.
// PayloadJSON is stored with the queue row so retries do not depend on event
// retention or a second lookup path.
type QueuedDelivery struct {
	ID             int64
	SubscriptionID int64
	EventID        string
	PayloadJSON    string
	Attempts       int
	NextAttemptAt  time.Time
}

// ListFilters defines filters for listing subscriptions.
type ListFilters struct {
	Owner   string
	Pattern string
	Enabled *bool
}

// Store defines the subscription storage interface.
type Store interface {
	Create(ctx context.Context, s Subscription) (int64, error)
	Get(ctx context.Context, id int64) (Subscription, error)
	List(ctx context.Context, f ListFilters) ([]Subscription, error)
	Update(ctx context.Context, s Subscription) error
	Delete(ctx context.Context, id int64) error
	GetHealth(ctx context.Context, id int64) (Health, error)
	Close() error
}

type QueueStore interface {
	Store
	Enqueue(ctx context.Context, subscriptionID int64, eventID, payload string) error
	Due(ctx context.Context, limit int) ([]QueuedDelivery, error)
	MarkDelivered(ctx context.Context, deliveryID, subscriptionID int64, at time.Time) error
	MarkFailed(ctx context.Context, deliveryID, subscriptionID int64, reason string, next time.Time, deadLetter bool) error
}
