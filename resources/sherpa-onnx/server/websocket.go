package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

const maxWebSocketMessageBytes = 1 << 20

type wsConn struct {
	conn  net.Conn
	read  *bufio.Reader
	write *bufio.Writer
	mu    sync.Mutex
}

func upgradeWebSocket(w http.ResponseWriter, r *http.Request) (*wsConn, error) {
	if !headerHasToken(r.Header.Get("Connection"), "upgrade") || !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return nil, fmt.Errorf("websocket upgrade headers are missing")
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		return nil, fmt.Errorf("unsupported websocket version")
	}
	key := strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key"))
	if key == "" {
		return nil, fmt.Errorf("websocket key is missing")
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("http server does not support websocket hijacking")
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("hijack websocket connection: %w", err)
	}
	acceptHash := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	if _, err := fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nSec-WebSocket-Accept: %s\r\n\r\n", base64.StdEncoding.EncodeToString(acceptHash[:])); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("write websocket handshake: %w", err)
	}
	if err := rw.Flush(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("flush websocket handshake: %w", err)
	}
	return &wsConn{conn: conn, read: rw.Reader, write: rw.Writer}, nil
}

func headerHasToken(value, token string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(part), token) {
			return true
		}
	}
	return false
}

func (c *wsConn) readMessage() (byte, []byte, error) {
	var firstOpcode byte
	var message []byte
	for {
		fin, opcode, payload, err := c.readFrame()
		if err != nil {
			return 0, nil, err
		}
		switch opcode {
		case 0x8: // close
			_ = c.writeFrame(0x8, payload)
			return 0, nil, io.EOF
		case 0x9: // ping
			if err := c.writeFrame(0xa, payload); err != nil {
				return 0, nil, err
			}
			continue
		case 0xa: // pong
			continue
		case 0x1, 0x2:
			if firstOpcode != 0 {
				return 0, nil, errors.New("websocket received a new data frame before a fragmented message ended")
			}
			firstOpcode = opcode
		case 0x0:
			if firstOpcode == 0 {
				return 0, nil, errors.New("websocket continuation has no initial data frame")
			}
		default:
			return 0, nil, fmt.Errorf("websocket opcode 0x%x is unsupported", opcode)
		}
		if len(message)+len(payload) > maxWebSocketMessageBytes {
			return 0, nil, fmt.Errorf("websocket message exceeds %d-byte limit", maxWebSocketMessageBytes)
		}
		message = append(message, payload...)
		if fin {
			return firstOpcode, message, nil
		}
	}
}

func (c *wsConn) readFrame() (bool, byte, []byte, error) {
	first, err := c.read.ReadByte()
	if err != nil {
		return false, 0, nil, err
	}
	second, err := c.read.ReadByte()
	if err != nil {
		return false, 0, nil, err
	}
	fin := first&0x80 != 0
	opcode := first & 0x0f
	masked := second&0x80 != 0
	length := int64(second & 0x7f)
	if length == 126 {
		var extended [2]byte
		if _, err := io.ReadFull(c.read, extended[:]); err != nil {
			return false, 0, nil, err
		}
		length = int64(binary.BigEndian.Uint16(extended[:]))
	} else if length == 127 {
		var extended [8]byte
		if _, err := io.ReadFull(c.read, extended[:]); err != nil {
			return false, 0, nil, err
		}
		if extended[0]&0x80 != 0 {
			return false, 0, nil, errors.New("websocket frame length overflows int64")
		}
		length = int64(binary.BigEndian.Uint64(extended[:]))
	}
	if length > maxWebSocketMessageBytes {
		return false, 0, nil, fmt.Errorf("websocket frame exceeds %d-byte limit", maxWebSocketMessageBytes)
	}
	if opcode >= 0x8 && (!fin || length > 125) {
		return false, 0, nil, errors.New("invalid websocket control frame")
	}
	if !masked {
		return false, 0, nil, errors.New("client websocket frame is not masked")
	}
	var mask [4]byte
	if _, err := io.ReadFull(c.read, mask[:]); err != nil {
		return false, 0, nil, err
	}
	payload := make([]byte, int(length))
	if _, err := io.ReadFull(c.read, payload); err != nil {
		return false, 0, nil, err
	}
	for i := range payload {
		payload[i] ^= mask[i%4]
	}
	return fin, opcode, payload, nil
}

func (c *wsConn) writeJSON(payload []byte) error { return c.writeFrame(0x1, payload) }

func (c *wsConn) writeFrame(opcode byte, payload []byte) error {
	if len(payload) > maxWebSocketMessageBytes {
		return fmt.Errorf("websocket response exceeds %d-byte limit", maxWebSocketMessageBytes)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	first := byte(0x80 | (opcode & 0x0f))
	if err := c.write.WriteByte(first); err != nil {
		return err
	}
	switch {
	case len(payload) < 126:
		if err := c.write.WriteByte(byte(len(payload))); err != nil {
			return err
		}
	case len(payload) <= 0xffff:
		if err := c.write.WriteByte(126); err != nil {
			return err
		}
		var extended [2]byte
		binary.BigEndian.PutUint16(extended[:], uint16(len(payload)))
		if _, err := c.write.Write(extended[:]); err != nil {
			return err
		}
	default:
		if err := c.write.WriteByte(127); err != nil {
			return err
		}
		var extended [8]byte
		binary.BigEndian.PutUint64(extended[:], uint64(len(payload)))
		if _, err := c.write.Write(extended[:]); err != nil {
			return err
		}
	}
	if _, err := c.write.Write(payload); err != nil {
		return err
	}
	return c.write.Flush()
}

func (c *wsConn) close() error { return c.conn.Close() }
