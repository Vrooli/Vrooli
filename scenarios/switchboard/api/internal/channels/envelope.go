package channels

import "time"

// AuthorKind is deliberately closed: only humans can initiate turns.
type AuthorKind string

const (
	AuthorHuman AuthorKind = "human"
	AuthorAgent AuthorKind = "agent"
)

type Media struct {
	Name string `json:"name"`
	MIME string `json:"mime"`
	URL  string `json:"url"`
	Size int64  `json:"size"`
}

// Envelope is the only message shape visible above an adapter.
type Envelope struct {
	ChannelID       string     `json:"channel_id"`
	RemoteMessageID string     `json:"remote_message_id"`
	ThreadKey       string     `json:"thread_key"`
	Group           bool       `json:"group,omitempty"`
	SenderAddress   string     `json:"sender_address"`
	AuthorKind      AuthorKind `json:"author_kind"`
	Text            string     `json:"text"`
	Media           []Media    `json:"media,omitempty"`
	ReplyToRemoteID string     `json:"reply_to_remote_id,omitempty"`
	Mentions        []string   `json:"mentions,omitempty"`
	ReceivedAt      time.Time  `json:"received_at"`
}

type Outbound struct {
	ChannelID       string  `json:"channel_id"`
	ThreadKey       string  `json:"thread_key"`
	Text            string  `json:"text"`
	Media           []Media `json:"media,omitempty"`
	ReplyToRemoteID string  `json:"reply_to_remote_id,omitempty"`
}
