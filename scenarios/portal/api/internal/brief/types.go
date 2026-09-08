package brief

import (
	"context"
	"time"

	agentbrief "github.com/vrooli/agentbrief-go"
)

type Record struct {
	ID               string
	Consumer         agentbrief.Consumer
	Verdict          agentbrief.Verdict
	Reason           string
	ChatID           string
	MessageID        string
	Harness          string
	SessionRef       string
	PromptDigest     string
	EffectiveQuery   string
	Rendered         string
	MaxTrustClass    agentbrief.Class
	Degraded         bool
	LatencyMS        int64
	CreatedAt        time.Time
	Items            []agentbrief.Item
	QueriedProviders []string
}

type BuildInput struct {
	Prompt     string
	Consumer   agentbrief.Consumer
	ChatID     string
	MessageID  string
	Harness    string
	SessionRef string
	BudgetMS   int
}

type ListInput struct {
	Consumer   agentbrief.Consumer
	ChatID     string
	SessionRef string
	Limit      int
}

type UseInput struct {
	BriefID   string
	ItemIndex int
	Kind      string
}

type StatsInput struct {
	WindowDays int
	Consumer   agentbrief.Consumer
}

type StatsRow struct {
	Consumer          agentbrief.Consumer
	BriefsBuilt       int64
	BriefsDelivered   int64
	ItemsDelivered    int64
	ItemsUsed         int64
	UsageRate         float64
	WithheldRate      float64
	WithheldByVerdict map[agentbrief.Verdict]int64
}

type Repository interface {
	Save(context.Context, Record) error
	Get(context.Context, string) (Record, error)
	List(context.Context, ListInput) ([]Record, error)
	RecordUse(context.Context, UseInput, time.Time) (bool, error)
	Stats(context.Context, StatsInput, time.Time) ([]StatsRow, error)
	DeleteBefore(context.Context, time.Time) error
}
