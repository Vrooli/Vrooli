// Package autofiler owns governed automatic backlog filing for maintenance
// findings. It is source-neutral: targeting strategies and finding producers
// feed stable Findings into the same policy and filing core.
package autofiler

import (
	"strings"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/settings"
)

const (
	OriginPrefix = "auto-filer"
	DefaultTag   = "auto-filer"
)

type Mode string

const (
	ModeSuggest Mode = settings.AutoFilerModeSuggest
	ModeAutoAdd Mode = settings.AutoFilerModeAutoAdd
)

type Strategy string

const (
	StrategyFeaturePending Strategy = settings.AutoFilerStrategyFeaturePending
	StrategyImportance     Strategy = settings.AutoFilerStrategyImportance
)

type Severity string

const (
	SeverityRed    Severity = "red"
	SeverityYellow Severity = "yellow"
)

type Finding struct {
	ID          string
	Scenario    string
	Dimension   string
	Severity    Severity
	Title       string
	Description string
	Details     string
}

func (f Finding) StableID() string {
	return strings.TrimSpace(f.ID)
}

func Origin(strategy Strategy, findingID string) string {
	strategyPart := strings.TrimSpace(string(strategy))
	if strategyPart == "" {
		strategyPart = "unknown"
	}
	return OriginPrefix + "/" + strategyPart + "/" + strings.TrimSpace(findingID)
}

func IsAutoFiled(item backlog.BacklogItem) bool {
	return item.CreatedBy != nil && strings.HasPrefix(strings.TrimSpace(item.CreatedBy.Source), OriginPrefix+"/")
}

func OpenAutoFiled(items []backlog.BacklogItem) []backlog.BacklogItem {
	out := make([]backlog.BacklogItem, 0, len(items))
	for _, item := range items {
		if !IsAutoFiled(item) || backlog.IsArchived(item) || backlog.IsTerminalStatus(item.Status) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func ItemRef(item backlog.BacklogItem) string {
	return string(item.Kind) + "/" + item.Name
}
