package homeintegration

import (
	"context"
	"errors"
	"time"
)

const TimeFormat = time.RFC3339Nano

var ErrUnknownAction = errors.New("unknown home automation action")

type Action struct {
	Name             string
	Description      string
	Effect           string
	ApprovalRequired bool
}

type Event struct {
	ID            string
	Type          string
	Summary       string
	OccurredAt    time.Time
	PublishStatus string
	PublishError  string
}

type Invocation struct {
	ID         string
	ActionName string
	Status     string
	Approved   bool
	Message    string
	Params     map[string]string
	EventID    string
	CreatedAt  time.Time
}

type Repository interface {
	SaveEvent(ctx context.Context, event Event) (Event, error)
	UpdateEventPublish(ctx context.Context, id, status, publishError string) (Event, error)
	ListEvents(ctx context.Context, limit int) ([]Event, error)
	SaveInvocation(ctx context.Context, invocation Invocation) (Invocation, error)
}

type Publisher interface {
	Publish(ctx context.Context, event Event) error
}
