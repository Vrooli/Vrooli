package codex

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// Explicit Tier 3 probe: owns only a new temporary thread, never the operator's
// active task. The default deterministic suite does not request inference.
func TestVoiceDaemonLive(t *testing.T) {
	socket := os.Getenv("WC_CODEX_VOICE_SOCKET")
	if socket == "" {
		t.Skip("set WC_CODEX_VOICE_SOCKET for the bounded live voice probe")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	c, err := DialSessionDaemon(ctx, socket)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var thread ThreadStartResult
	if err = c.Call(ctx, "thread/start", map[string]any{"cwd": t.TempDir(), "approvalPolicy": "never", "sandbox": "read-only"}, &thread); err != nil {
		t.Fatal(err)
	}
	id := thread.ID()
	if id == "" {
		t.Fatal("missing test thread")
	}
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = c.Call(cleanup, "thread/realtime/stop", map[string]string{"threadId": id}, nil)
		_ = c.Call(cleanup, "thread/delete", map[string]string{"threadId": id}, nil)
	}()
	ch, unsubscribe := c.SubscribeNotifications()
	defer unsubscribe()
	if err = c.Call(ctx, "thread/realtime/start", map[string]any{"threadId": id, "transport": map[string]string{"type": "websocket"}, "outputModality": "audio", "version": "v3", "codexResponsesAsItems": true}, nil); err != nil {
		t.Fatal(err)
	}
	t.Log("ASSERT native voice start accepted")
	count := 2400
	if err = c.Call(ctx, "thread/realtime/appendAudio", map[string]any{"threadId": id, "audio": RealtimeAudio{Data: base64.StdEncoding.EncodeToString(make([]byte, count*2)), SampleRate: 24000, NumChannels: 1, SamplesPerChannel: &count}}, nil); err != nil {
		t.Fatal(err)
	}
	t.Log("ASSERT native PCM input accepted")
	if err = c.Call(ctx, "thread/realtime/appendText", map[string]string{"threadId": id, "text": "Please say only: Voice connection verified. Do not delegate any coding work."}, nil); err != nil {
		t.Fatal(err)
	}
	for {
		select {
		case <-ctx.Done():
			t.Fatal("no output audio before voice deadline")
		case message, ok := <-ch:
			if !ok {
				t.Fatal("voice observer disconnected")
			}
			n, ok := ParseRealtime(message)
			if !ok || n.ThreadID != id {
				continue
			}
			if message.Method == "thread/realtime/error" {
				t.Fatalf("native voice: %s", n.Message)
			}
			if message.Method == "thread/realtime/outputAudio/delta" && n.Audio != nil && n.Audio.Data != "" {
				t.Log("ASSERT native output audio received")
				return
			}
		}
	}
}

func TestVoiceObserversDoNotStealOwnerEvents(t *testing.T) {
	c := NewClient(writeCloser{io.Discard}, strings.NewReader(""))
	a, stopA := c.SubscribeNotifications()
	defer stopA()
	b, stopB := c.SubscribeNotifications()
	defer stopB()
	message := Message{Method: "thread/realtime/started", Params: json.RawMessage(`{"threadId":"thread-1"}`)}
	c.broadcastNotification(message)
	for _, ch := range []<-chan Message{a, b} {
		if got := <-ch; got.Method != message.Method {
			t.Fatalf("event=%+v", got)
		}
	}
	stopA()
	stopA()
	c.broadcastNotification(message)
	if got := <-b; got.Method != message.Method {
		t.Fatal("other observer lost event")
	}
}
func TestVoiceObserverOverflowIsExplicit(t *testing.T) {
	c := NewClient(writeCloser{io.Discard}, strings.NewReader(""))
	ch, stop := c.SubscribeNotifications()
	defer stop()
	for i := 0; i < 129; i++ {
		c.broadcastNotification(Message{Method: "audio"})
	}
	count := 0
	for range ch {
		count++
	}
	if count != 128 {
		t.Fatalf("buffer=%d", count)
	}
}
func TestVoiceObserverAfterEOFIsClosed(t *testing.T) {
	c := NewClient(writeCloser{io.Discard}, strings.NewReader(""))
	_ = c.Close()
	ch, stop := c.SubscribeNotifications()
	defer stop()
	if _, ok := <-ch; ok {
		t.Fatal("closed owner accepted observer")
	}
}
func TestParseRealtimeRequiresNativeThread(t *testing.T) {
	for _, message := range []Message{{Method: "turn/started", Params: json.RawMessage(`{"threadId":"t"}`)}, {Method: "thread/realtime/started", Params: json.RawMessage(`{}`)}, {Method: "thread/realtime/started", Params: json.RawMessage(`null`)}} {
		if _, ok := ParseRealtime(message); ok {
			t.Fatalf("accepted %+v", message)
		}
	}
	n, ok := ParseRealtime(Message{Method: "thread/realtime/item/completed", Params: json.RawMessage(`{"threadId":"t","item":{"id":"i","realtimeSessionId":"v","type":"transcriptSegment","role":"user","text":"hello"}}`)})
	if !ok || n.Item == nil || n.Item.ID != "i" {
		t.Fatalf("notification=%+v", n)
	}
}

func TestBrowserVoiceVersionQualification(t *testing.T) {
	for _, version := range []string{"0.161.0", "web-console/0.161.0 (Ubuntu 24.4.0; x86_64) dumb (web-console; 1.0.0)", "codex/0.161.2"} {
		if !SupportsBrowserVoice(version) {
			t.Fatalf("qualified version rejected: %s", version)
		}
	}
	for _, version := range []string{"", "0.160.0", "0.162.0", "0.161.0-unknown", "0.161.0evil", "10.161.0"} {
		if SupportsBrowserVoice(version) {
			t.Fatalf("unqualified version accepted: %s", version)
		}
	}
}

// Explicit Tier 3 probe used only with the disposable browser validation
// session. The browser must already own a connected WebRTC conversation.
func TestVoiceBrowserPlaybackPromptLive(t *testing.T) {
	socket, threadID := os.Getenv("WC_CODEX_VOICE_TEST_SOCKET"), os.Getenv("WC_CODEX_VOICE_TEST_THREAD")
	if socket == "" || threadID == "" {
		t.Skip("set the disposable browser voice test socket and thread")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client, err := DialSessionDaemon(ctx, socket)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	var thread struct {
		Thread struct {
			ID  string `json:"id"`
			CWD string `json:"cwd"`
		} `json:"thread"`
	}
	if err := client.Call(ctx, "thread/read", map[string]string{"threadId": threadID}, &thread); err != nil {
		t.Fatal(err)
	}
	if thread.Thread.ID != threadID || thread.Thread.CWD != "/tmp/wc-native-voice-live-workspace" {
		t.Fatal("test thread identity/workspace mismatch")
	}
	var loaded struct {
		Data []string `json:"data"`
	}
	if err := client.Call(ctx, "thread/loaded/list", map[string]any{}, &loaded); err != nil {
		t.Fatal(err)
	}
	if len(loaded.Data) != 1 || loaded.Data[0] != threadID {
		t.Fatal("test daemon must have only the specified loaded thread")
	}
	if err := client.Call(ctx, "thread/realtime/appendText", map[string]string{"threadId": threadID, "text": "Please say the words Browser audio verified aloud, and do not call any tools."}, nil); err != nil {
		t.Fatal(err)
	}
}
