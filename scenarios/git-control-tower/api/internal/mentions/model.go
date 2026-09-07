// Package mentions normalizes authenticated host comments before advisory
// dispatch. It contains no provider webhook transport or publication client.
package mentions

import (
	"fmt"
	"strings"
)

type Delivery struct {
	Provider       string `json:"provider"`
	EventID        string `json:"eventId"`
	EventVersion   string `json:"eventVersion,omitempty"`
	ActorID        string `json:"actorId"`
	ThreadID       string `json:"threadId"`
	RepositoryID   string `json:"repositoryId"`
	HeadRevision   string `json:"headRevision"`
	Body           string `json:"body"`
	Authenticated  bool   `json:"authenticated"`
	ActorCanRead   bool   `json:"actorCanRead"`
	ReplyPermitted bool   `json:"replyPermitted"`
	BotActor       bool   `json:"botActor"`
}

type Command string

const (
	CommandSummarize Command = "summarize"
	CommandReview    Command = "review"
	CommandExplain   Command = "explain"
)

type Request struct {
	Delivery Delivery `json:"delivery"`
	Command  Command  `json:"command"`
	Key      string   `json:"key"`
}

func Parse(d Delivery) (Request, error) {
	if strings.TrimSpace(d.Provider) == "" || strings.TrimSpace(d.EventID) == "" || strings.TrimSpace(d.ActorID) == "" || strings.TrimSpace(d.RepositoryID) == "" || strings.TrimSpace(d.HeadRevision) == "" {
		return Request{}, fmt.Errorf("delivery identity is incomplete")
	}
	if !d.Authenticated || !d.ActorCanRead || d.BotActor {
		return Request{}, fmt.Errorf("delivery is not an authenticated human-readable event")
	}
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(d.Body)))
	if len(fields) < 2 || fields[0] != "@vrooli" {
		return Request{}, fmt.Errorf("comment is not an addressed advisory command")
	}
	var command Command
	switch fields[1] {
	case string(CommandSummarize):
		command = CommandSummarize
	case string(CommandReview):
		command = CommandReview
	case string(CommandExplain):
		command = CommandExplain
	default:
		return Request{}, fmt.Errorf("unsupported advisory command %q", fields[1])
	}
	key := d.Provider + ":" + d.EventID
	if version := strings.TrimSpace(d.EventVersion); version != "" {
		key += ":" + version
	}
	return Request{Delivery: d, Command: command, Key: key}, nil
}

func PublicText(summary string, privateRefs []string) string {
	for _, ref := range privateRefs {
		if strings.TrimSpace(ref) != "" {
			summary = strings.ReplaceAll(summary, ref, "[private reference omitted]")
		}
	}
	return strings.TrimSpace(summary)
}
