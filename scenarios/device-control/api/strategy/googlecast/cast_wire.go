package googlecast

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"device-control/strategy"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	connectionNS            = "urn:x-cast:com.google.cast.tp.connection"
	receiverNS              = "urn:x-cast:com.google.cast.receiver"
	mediaNS                 = "urn:x-cast:com.google.cast.media"
	heartbeatNS             = "urn:x-cast:com.google.cast.tp.heartbeat"
	frameLimit              = 8 << 20
	defaultHeartbeatTimeout = 15 * time.Second
)

type castMessage struct {
	ProtocolVersion int    `json:"protocol_version"`
	SourceID        string `json:"source_id"`
	DestinationID   string `json:"destination_id"`
	Namespace       string `json:"namespace"`
	PayloadUTF8     string `json:"payload_utf8,omitempty"`
}
type wireClient struct {
	endpoint          string
	mu                sync.Mutex
	dialer            func(context.Context) (net.Conn, error)
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
}

func newWireClient(endpoint string) (Client, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("Google Cast endpoint is empty")
	}
	return &wireClient{endpoint: endpoint}, nil
}

func (c *wireClient) dial(ctx context.Context) (net.Conn, error) {
	if c.dialer != nil {
		return c.dialer(ctx)
	}
	dialer := &net.Dialer{}
	raw, err := dialer.DialContext(ctx, "tcp", c.endpoint)
	if err != nil {
		return nil, err
	}
	tlsConn := tls.Client(raw, &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}) // #nosec G402 -- Cast receivers use trusted-LAN self-signed certificates.
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		raw.Close()
		return nil, err
	}
	return tlsConn, nil
}

func (c *wireClient) connect(ctx context.Context) (net.Conn, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	if err := writeCast(conn, castMessage{ProtocolVersion: 0, SourceID: "sender-0", DestinationID: "receiver-0", Namespace: connectionNS, PayloadUTF8: `{"type":"CONNECT","origin":{}}`}); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func (c *wireClient) request(ctx context.Context, namespace string, payload map[string]any, destination string) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("Cast %s connect: %w", namespace, err)
	}
	defer conn.Close()
	if destination != "receiver-0" {
		if err := writeCast(conn, castMessage{ProtocolVersion: 0, SourceID: "sender-0", DestinationID: destination, Namespace: connectionNS, PayloadUTF8: `{"type":"CONNECT","origin":{}}`}); err != nil {
			return nil, err
		}
	}
	body, _ := json.Marshal(payload)
	if err := writeCast(conn, castMessage{ProtocolVersion: 0, SourceID: "sender-0", DestinationID: destination, Namespace: namespace, PayloadUTF8: string(body)}); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(5 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetReadDeadline(deadline)
	for {
		msg, err := readCast(conn)
		if err != nil {
			return nil, fmt.Errorf("Cast %s response: %w", namespace, err)
		}
		if msg.PayloadUTF8 == "" {
			continue
		}
		if msg.Namespace == heartbeatNS {
			var heartbeat map[string]any
			if json.Unmarshal([]byte(msg.PayloadUTF8), &heartbeat) == nil && heartbeat["type"] == "PING" {
				if err := writeCast(conn, castMessage{ProtocolVersion: 0, SourceID: "sender-0", DestinationID: "receiver-0", Namespace: heartbeatNS, PayloadUTF8: `{"type":"PONG"}`}); err != nil {
					return nil, err
				}
			}
			continue
		}
		var result map[string]any
		if json.Unmarshal([]byte(msg.PayloadUTF8), &result) == nil {
			responseType, _ := result["type"].(string)
			if responseType == "CONNECTED" || responseType == "PONG" {
				continue
			}
			if responseType == "LAUNCH_ERROR" || responseType == "INVALID_REQUEST" || responseType == "ERROR" {
				return nil, fmt.Errorf("Cast receiver returned %s: %s", responseType, msg.PayloadUTF8)
			}
			return result, nil
		}
	}
}

func (c *wireClient) Status(ctx context.Context) (ReceiverStatus, error) {
	result, err := c.request(ctx, receiverNS, map[string]any{"type": "GET_STATUS", "requestId": nextRequestID()}, "receiver-0")
	if err != nil {
		return ReceiverStatus{}, err
	}
	status := parseStatus(result)
	if status.Application != "" && status.TransportID != "" {
		mediaResult, mediaErr := c.request(ctx, mediaNS, map[string]any{"type": "GET_STATUS", "requestId": nextRequestID()}, status.TransportID)
		if mediaErr == nil {
			media := parseStatus(mediaResult)
			if media.PlayerState != "" {
				status.PlayerState = media.PlayerState
			}
			if media.MediaTitle != "" {
				status.MediaTitle = media.MediaTitle
			}
			if media.MediaArtist != "" {
				status.MediaArtist = media.MediaArtist
			}
		}
	}
	return status, nil
}

func (c *wireClient) SetVolume(ctx context.Context, value float64) error {
	_, err := c.request(ctx, receiverNS, map[string]any{"type": "SET_VOLUME", "requestId": nextRequestID(), "volume": map[string]any{"level": value}}, "receiver-0")
	return err
}

func (c *wireClient) SetMuted(ctx context.Context, muted bool) error {
	_, err := c.request(ctx, receiverNS, map[string]any{"type": "SET_VOLUME", "requestId": nextRequestID(), "volume": map[string]any{"muted": muted}}, "receiver-0")
	return err
}

func (c *wireClient) Launch(ctx context.Context, app string) error {
	_, err := c.request(ctx, receiverNS, map[string]any{"type": "LAUNCH", "requestId": nextRequestID(), "appId": app}, "receiver-0")
	return err
}

func (c *wireClient) Media(ctx context.Context, action string) error {
	status, err := c.Status(ctx)
	if err != nil {
		return err
	}
	payload := map[string]any{"type": "MEDIA_COMMAND", "requestId": nextRequestID(), "mediaSessionId": 1, "command": map[string]any{"type": stringsToCastType(action)}}
	destination := status.TransportID
	if destination == "" {
		destination = "receiver-0"
	}
	_, err = c.request(ctx, mediaNS, payload, destination)
	return err
}

func (c *wireClient) Observe(ctx context.Context, sink Observer) error {
	conn, err := c.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := writeCast(conn, castMessage{ProtocolVersion: 0, SourceID: "sender-0", DestinationID: "receiver-0", Namespace: receiverNS, PayloadUTF8: `{"type":"GET_STATUS","requestId":1}`}); err != nil {
		return err
	}
	heartbeatInterval := c.heartbeatInterval
	if heartbeatInterval <= 0 {
		heartbeatInterval = 5 * time.Second
	}
	heartbeatTimeout := c.heartbeatTimeout
	if heartbeatTimeout <= 0 {
		heartbeatTimeout = defaultHeartbeatTimeout
	}
	var previous *ReceiverStatus
	lastPeerHeartbeat := time.Now()
	nextHeartbeat := time.Now().Add(heartbeatInterval)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		now := time.Now()
		if !now.Before(nextHeartbeat) {
			if err := writeCast(conn, castMessage{ProtocolVersion: 0, SourceID: "sender-0", DestinationID: "receiver-0", Namespace: heartbeatNS, PayloadUTF8: `{"type":"PING"}`}); err != nil {
				return err
			}
			nextHeartbeat = now.Add(heartbeatInterval)
		}
		readDeadline := now.Add(250 * time.Millisecond)
		if nextHeartbeat.Before(readDeadline) {
			readDeadline = nextHeartbeat
		}
		_ = conn.SetReadDeadline(readDeadline)
		msg, readErr := readCast(conn)
		if readErr != nil {
			if e, ok := readErr.(net.Error); ok && e.Timeout() {
				if time.Since(lastPeerHeartbeat) > heartbeatTimeout {
					return fmt.Errorf("Cast heartbeat timed out")
				}
				continue
			}
			return readErr
		}
		if msg.Namespace == heartbeatNS {
			var heartbeat map[string]any
			if json.Unmarshal([]byte(msg.PayloadUTF8), &heartbeat) == nil {
				switch heartbeat["type"] {
				case "PING":
					lastPeerHeartbeat = time.Now()
					if err := writeCast(conn, castMessage{ProtocolVersion: 0, SourceID: "sender-0", DestinationID: "receiver-0", Namespace: heartbeatNS, PayloadUTF8: `{"type":"PONG"}`}); err != nil {
						return err
					}
				case "PONG":
					// The observer sends PING frames as well as answering peer
					// PINGs. A receiver's PONG is proof that the long-lived
					// connection is healthy and must refresh the deadline.
					lastPeerHeartbeat = time.Now()
				}
			}
			continue
		}
		if msg.Namespace != receiverNS && msg.Namespace != mediaNS {
			continue
		}
		var payload map[string]any
		if json.Unmarshal([]byte(msg.PayloadUTF8), &payload) != nil {
			continue
		}
		current := parseStatus(payload)
		if previous != nil {
			emitStatusDiff(*previous, current, sink)
		}
		previous = &current
	}
}
func (c *wireClient) Close() error { return nil }

var requestCounter uint64

func nextRequestID() uint64 { return atomic.AddUint64(&requestCounter, 1) }
func writeCast(w io.Writer, message castMessage) error {
	payload := marshalCastMessage(message)
	if len(payload) > frameLimit {
		return fmt.Errorf("Cast frame exceeds %d bytes", frameLimit)
	}
	var prefix [4]byte
	binary.BigEndian.PutUint32(prefix[:], uint32(len(payload)))
	if err := writeAll(w, prefix[:]); err != nil {
		return err
	}
	return writeAll(w, payload)
}

func writeAll(w io.Writer, payload []byte) error {
	for len(payload) > 0 {
		written, err := w.Write(payload)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		payload = payload[written:]
	}
	return nil
}

func readCast(r io.Reader) (castMessage, error) {
	var prefix [4]byte
	if _, err := io.ReadFull(r, prefix[:]); err != nil {
		return castMessage{}, err
	}
	length := binary.BigEndian.Uint32(prefix[:])
	if length > frameLimit {
		return castMessage{}, fmt.Errorf("Cast frame exceeds %d bytes", frameLimit)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return castMessage{}, err
	}
	return unmarshalCastMessage(payload)
}

func marshalCastMessage(message castMessage) []byte {
	var payload []byte
	payload = protowire.AppendTag(payload, 1, protowire.VarintType)
	payload = protowire.AppendVarint(payload, uint64(message.ProtocolVersion))
	payload = protowire.AppendTag(payload, 2, protowire.BytesType)
	payload = protowire.AppendString(payload, message.SourceID)
	payload = protowire.AppendTag(payload, 3, protowire.BytesType)
	payload = protowire.AppendString(payload, message.DestinationID)
	payload = protowire.AppendTag(payload, 4, protowire.BytesType)
	payload = protowire.AppendString(payload, message.Namespace)
	// CastMessage is a proto2 envelope and requires payload_type even when the
	// payload is the default UTF-8 JSON form (enum value STRING=0).
	payload = protowire.AppendTag(payload, 5, protowire.VarintType)
	payload = protowire.AppendVarint(payload, 0)
	payload = protowire.AppendTag(payload, 6, protowire.BytesType)
	payload = protowire.AppendBytes(payload, []byte(message.PayloadUTF8))
	return payload
}

func unmarshalCastMessage(payload []byte) (castMessage, error) {
	var message castMessage
	for len(payload) > 0 {
		number, typ, n := protowire.ConsumeTag(payload)
		if n < 0 {
			return castMessage{}, protowire.ParseError(n)
		}
		payload = payload[n:]
		switch number {
		case 1:
			value, consumed := protowire.ConsumeVarint(payload)
			if consumed < 0 {
				return castMessage{}, protowire.ParseError(consumed)
			}
			message.ProtocolVersion = int(value)
			payload = payload[consumed:]
		case 2, 3, 4:
			value, consumed := protowire.ConsumeString(payload)
			if consumed < 0 {
				return castMessage{}, protowire.ParseError(consumed)
			}
			switch number {
			case 2:
				message.SourceID = value
			case 3:
				message.DestinationID = value
			case 4:
				message.Namespace = value
			}
			payload = payload[consumed:]
		case 6:
			value, consumed := protowire.ConsumeBytes(payload)
			if consumed < 0 {
				return castMessage{}, protowire.ParseError(consumed)
			}
			message.PayloadUTF8 = string(value)
			payload = payload[consumed:]
		default:
			consumed := protowire.ConsumeFieldValue(number, typ, payload)
			if consumed < 0 {
				return castMessage{}, protowire.ParseError(consumed)
			}
			payload = payload[consumed:]
		}
	}
	return message, nil
}

func parseStatus(payload map[string]any) ReceiverStatus {
	status, _ := payload["status"].(map[string]any)
	volume, _ := status["volume"].(map[string]any)
	receiver := ReceiverStatus{Application: stringValue(status, "applications", 0, "appId"), TransportID: stringValue(status, "applications", 0, "transportId")}
	if receiver.TransportID == "" {
		// Receiver status from current Cast TV firmware exposes the application
		// route as sessionId rather than transportId.
		receiver.TransportID = stringValue(status, "applications", 0, "sessionId")
	}
	if value, ok := volume["level"].(float64); ok {
		receiver.Volume = value
	}
	if muted, ok := volume["muted"].(bool); ok {
		receiver.Muted = muted
	}
	if sessions, ok := status["applications"].([]any); ok && len(sessions) > 0 {
		if app, ok := sessions[0].(map[string]any); ok {
			receiver.Application, _ = app["appId"].(string)
			receiver.TransportID, _ = app["transportId"].(string)
			if receiver.TransportID == "" {
				receiver.TransportID, _ = app["sessionId"].(string)
			}
		}
	}
	if player, ok := payload["playerState"].(string); ok {
		receiver.PlayerState = player
	}
	if player, ok := status["playerState"].(string); ok {
		receiver.PlayerState = player
	}
	if statuses, ok := payload["status"].([]any); ok && len(statuses) > 0 {
		if mediaStatus, ok := statuses[0].(map[string]any); ok {
			if player, ok := mediaStatus["playerState"].(string); ok {
				receiver.PlayerState = player
			}
			if media, ok := mediaStatus["media"].(map[string]any); ok {
				if metadata, ok := media["metadata"].(map[string]any); ok {
					receiver.MediaTitle, _ = metadata["title"].(string)
					receiver.MediaArtist, _ = metadata["artist"].(string)
				}
			}
		}
	}
	if media, ok := payload["mediaInformation"].(map[string]any); ok {
		if metadata, ok := media["metadata"].(map[string]any); ok {
			receiver.MediaTitle, _ = metadata["title"].(string)
			receiver.MediaArtist, _ = metadata["artist"].(string)
		}
	}
	return receiver
}

func stringValue(root map[string]any, array string, index int, key string) string {
	items, ok := root[array].([]any)
	if !ok || len(items) <= index {
		return ""
	}
	item, ok := items[index].(map[string]any)
	if !ok {
		return ""
	}
	value, _ := item[key].(string)
	return value
}

func emitStatusDiff(old, current ReceiverStatus, sink Observer) {
	if old.Volume != current.Volume {
		sink(strategy.StateChangeEvent{Transport: "google-cast", Attribute: "volume", OldValue: old.Volume, NewValue: current.Volume, CausationID: "", StateClass: strategy.StateBearing})
	}
	if old.Muted != current.Muted {
		sink(strategy.StateChangeEvent{Transport: "google-cast", Attribute: "muted", OldValue: old.Muted, NewValue: current.Muted, StateClass: strategy.StateBearing})
	}
	if old.Application != current.Application {
		sink(strategy.StateChangeEvent{Transport: "google-cast", Attribute: "application", OldValue: old.Application, NewValue: current.Application, StateClass: strategy.StateBearing})
	}
	if old.PlayerState != current.PlayerState {
		sink(strategy.StateChangeEvent{Transport: "google-cast", Attribute: "player_state", OldValue: old.PlayerState, NewValue: current.PlayerState, StateClass: strategy.StateBearing})
	}
}

func stringsToCastType(action string) string {
	switch action {
	case "play":
		return "PLAY"
	case "pause":
		return "PAUSE"
	case "stop":
		return "STOP"
	case "next":
		return "QUEUE_NEXT"
	case "previous":
		return "QUEUE_PREV"
	}
	return action
}
