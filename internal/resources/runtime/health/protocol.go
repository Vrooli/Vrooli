package health

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
)

const (
	postgresProtocolVersion = 196608
	maxPostgresMessage      = 1 << 20
	maxPostgresStartupReads = 16
	maxRedisReplyBytes      = 4096
)

// runPostgreSQLReadiness follows the PostgreSQL startup protocol through the
// server's first authentication request. This matches pg_isready's purpose:
// it proves PostgreSQL is accepting protocol connections, without logging or
// needing a password. It does not claim that application credentials or a
// query succeed.
func runPostgreSQLReadiness(ctx context.Context, target string, env []string) Result {
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", target)
	if err != nil {
		return Result{Message: "PostgreSQL is not accepting connections"}
	}
	defer conn.Close()
	setConnDeadline(ctx, conn)

	values := environmentValues(env)
	user := strings.TrimSpace(values["POSTGRES_USER"])
	database := strings.TrimSpace(values["POSTGRES_DB"])
	if user == "" {
		user = "vrooli-health-check"
	}
	if database == "" {
		database = "postgres"
	}
	startup := postgresStartupPacket(user, database)
	if err := writeFull(conn, startup); err != nil {
		return Result{Message: "PostgreSQL startup handshake failed"}
	}

	reader := bufio.NewReader(conn)
	for i := 0; i < maxPostgresStartupReads; i++ {
		messageType, payload, readErr := readPostgresMessage(reader)
		if readErr != nil {
			return Result{Message: "PostgreSQL startup handshake failed"}
		}
		switch messageType {
		case 'R': // Authentication request: the server is ready for clients.
			return Result{Healthy: true, Message: "healthy"}
		case 'Z': // ReadyForQuery can occur with trust authentication.
			return Result{Healthy: true, Message: "healthy"}
		case 'E':
			if postgresErrorCode(payload) == "57P03" {
				return Result{Message: "PostgreSQL is starting and cannot accept connections"}
			}
			return Result{Message: "PostgreSQL rejected the startup handshake"}
		}
	}
	return Result{Message: "PostgreSQL startup handshake returned no readiness response"}
}

func postgresStartupPacket(user, database string) []byte {
	parameters := "user\x00" + user + "\x00database\x00" + database + "\x00application_name\x00vrooli-resource-health\x00client_encoding\x00UTF8\x00\x00"
	packet := make([]byte, 8+len(parameters))
	binary.BigEndian.PutUint32(packet[:4], uint32(len(packet)))
	binary.BigEndian.PutUint32(packet[4:8], postgresProtocolVersion)
	copy(packet[8:], parameters)
	return packet
}

func readPostgresMessage(reader io.Reader) (byte, []byte, error) {
	var messageType [1]byte
	if _, err := io.ReadFull(reader, messageType[:]); err != nil {
		return 0, nil, err
	}
	var rawLength [4]byte
	if _, err := io.ReadFull(reader, rawLength[:]); err != nil {
		return 0, nil, err
	}
	length := binary.BigEndian.Uint32(rawLength[:])
	if length < 4 || length > maxPostgresMessage {
		return 0, nil, fmt.Errorf("invalid PostgreSQL message length %d", length)
	}
	payload := make([]byte, int(length)-4)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return 0, nil, err
	}
	return messageType[0], payload, nil
}

func postgresErrorCode(payload []byte) string {
	for len(payload) > 0 {
		field := payload[0]
		payload = payload[1:]
		if field == 0 {
			return ""
		}
		end := bytes.IndexByte(payload, 0)
		if end < 0 {
			return ""
		}
		if field == 'C' {
			return string(payload[:end])
		}
		payload = payload[end+1:]
	}
	return ""
}

func runRedisReadiness(ctx context.Context, target string) Result {
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", target)
	if err != nil {
		return Result{Message: "Redis is not accepting commands"}
	}
	defer conn.Close()
	setConnDeadline(ctx, conn)
	if err := writeFull(conn, []byte("*1\r\n$4\r\nPING\r\n")); err != nil {
		return Result{Message: "Redis PING failed"}
	}
	reply, err := bufio.NewReaderSize(conn, maxRedisReplyBytes).ReadSlice('\n')
	if err != nil {
		return Result{Message: "Redis PING failed"}
	}
	if len(reply) > maxRedisReplyBytes || !bytes.Equal(reply, []byte("+PONG\r\n")) {
		return Result{Message: "Redis did not return PONG"}
	}
	return Result{Healthy: true, Message: "healthy"}
}

func writeFull(writer io.Writer, payload []byte) error {
	for len(payload) > 0 {
		written, err := writer.Write(payload)
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

func environmentValues(env []string) map[string]string {
	values := make(map[string]string, len(env))
	for _, entry := range env {
		if key, value, ok := strings.Cut(entry, "="); ok {
			values[key] = value
		}
	}
	return values
}

func setConnDeadline(ctx context.Context, conn net.Conn) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
}
