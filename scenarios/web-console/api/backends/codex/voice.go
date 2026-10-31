package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Realtime is experimental. Qualify a protocol family explicitly rather than
// advertising audio against arbitrary installed Codex versions.
var browserVoiceVersion = regexp.MustCompile(`(?:^|/)0\.161\.[0-9]+(?:\s|$)`)

func SupportsBrowserVoice(version string) bool { return browserVoiceVersion.MatchString(version) }

// SubscribeNotifications observes the existing owner without competing with its
// conversation reader. Overflow closes the subscription: audio loss must be
// visible to the caller rather than silently producing a broken voice stream.
func (c *Client) SubscribeNotifications() (<-chan Message, func()) {
	c.subscribersMu.Lock()
	defer c.subscribersMu.Unlock()
	ch := make(chan Message, 128)
	if c.notificationsClosed {
		close(ch)
		return ch, func() {}
	}
	select {
	case <-c.readDone:
		close(ch)
		return ch, func() {}
	default:
	}
	if c.subscribers == nil {
		c.subscribers = make(map[chan Message]struct{})
	}
	c.subscribers[ch] = struct{}{}
	return ch, func() {
		c.subscribersMu.Lock()
		defer c.subscribersMu.Unlock()
		if _, ok := c.subscribers[ch]; ok {
			delete(c.subscribers, ch)
			close(ch)
		}
	}
}

func (c *Client) broadcastNotification(message Message) {
	c.subscribersMu.Lock()
	defer c.subscribersMu.Unlock()
	for ch := range c.subscribers {
		select {
		case ch <- message:
		default:
			delete(c.subscribers, ch)
			close(ch)
		}
	}
}

func (o *ManagedOwner) VoiceClient() *Client { return o.client }

// wsStream adapts Codex's Unix WebSocket transport to the same JSONL client
// used by managed stdio owners. Closing it disconnects this observer only.
type wsStream struct {
	conn   *websocket.Conn
	reader io.Reader
	mu     sync.Mutex
}

func (s *wsStream) Read(p []byte) (int, error) {
	for {
		if s.reader == nil {
			_, r, err := s.conn.NextReader()
			if err != nil {
				return 0, err
			}
			s.reader = io.MultiReader(r, strings.NewReader("\n"))
		}
		n, err := s.reader.Read(p)
		if err == io.EOF {
			s.reader = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}
func (s *wsStream) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := s.conn.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}
func (s *wsStream) Close() error { return s.conn.Close() }

// DialSessionDaemon never launches an agent. Its socket must come from the
// caller's verified per-session home, never from a browser-supplied path.
func DialSessionDaemon(ctx context.Context, socketPath string) (*Client, error) {
	var directory *os.File
	address := socketPath
	if runtime.GOOS == "linux" && len(address) > 100 {
		var err error
		directory, err = os.Open(filepath.Dir(address))
		if err != nil {
			return nil, err
		}
		defer directory.Close()
		address = fmt.Sprintf("/proc/self/fd/%d/%s", directory.Fd(), filepath.Base(address))
	}
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second, NetDialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", address)
	}}
	conn, resp, err := dialer.DialContext(ctx, "ws://localhost/", http.Header{})
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	conn.SetReadLimit(16 * 1024 * 1024)
	stream := &wsStream{conn: conn}
	client := NewClient(stream, stream)
	if err := client.Initialize(ctx, "web-console", "1.0.0"); err != nil {
		_ = client.Close()
		return nil, err
	}
	// The legacy notification channel has no owner on this dedicated connection.
	go func() {
		for range client.Notifications() {
		}
	}()
	// An observer must not decide approvals owned by the attached terminal.
	go func() {
		for range client.Requests() {
		}
	}()
	return client, nil
}

type RealtimeAudio struct {
	Data              string  `json:"data"`
	SampleRate        int     `json:"sampleRate"`
	NumChannels       int     `json:"numChannels"`
	SamplesPerChannel *int    `json:"samplesPerChannel"`
	ItemID            *string `json:"itemId"`
}

// RealtimeItem is the committed, identity-bearing transcript surface. Flat
// legacy transcript notifications have no stable item identity and stay live UI.
type RealtimeItem struct {
	ID                string `json:"id"`
	RealtimeSessionID string `json:"realtimeSessionId"`
	Type              string `json:"type"`
	Role              string `json:"role"`
	Text              string `json:"text"`
}
type RealtimeNotification struct {
	ThreadID          string         `json:"threadId"`
	RealtimeSessionID string         `json:"realtimeSessionId"`
	SDP               string         `json:"sdp"`
	Audio             *RealtimeAudio `json:"audio"`
	Item              *RealtimeItem  `json:"item"`
	ItemID            string         `json:"itemId"`
	Delta             string         `json:"delta"`
	Text              string         `json:"text"`
	Role              string         `json:"role"`
	Message           string         `json:"message"`
	Reason            string         `json:"reason"`
}

func ParseRealtime(message Message) (RealtimeNotification, bool) {
	var out RealtimeNotification
	if !strings.HasPrefix(message.Method, "thread/realtime/") || json.Unmarshal(message.Params, &out) != nil || out.ThreadID == "" {
		return out, false
	}
	return out, true
}
