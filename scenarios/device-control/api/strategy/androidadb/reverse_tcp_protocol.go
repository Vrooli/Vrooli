package androidadb

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

// The ADB CLI accepts only tcp:<integer> in forwarding commands, while adbd
// supports explicit localhost listeners. Keep this narrowly typed protocol
// operation inside the ADB owner; never fall back to an all-interface listener.
type reverseTCPCreator interface {
	createReverseTCP(context.Context, string, int, int) error
}

func (commandRunner) createReverseTCP(ctx context.Context, selector string, devicePort, hostPort int) error {
	// Do not silently select a different daemon from the CLI operations used for
	// inspection and removal. Nondefault daemon configurations need qualification.
	for _, key := range []string{"ADB_SERVER_SOCKET", "ANDROID_ADB_SERVER_ADDRESS", "ANDROID_ADB_SERVER_PORT"} {
		if os.Getenv(key) != "" {
			return fmt.Errorf("reverse TCP protocol requires the default local ADB server")
		}
	}
	c, err := (&net.Dialer{}).DialContext(ctx, "tcp", "127.0.0.1:5037")
	if err != nil {
		return fmt.Errorf("connect to local ADB server: %w", err)
	}
	defer c.Close()
	return createReverseTCPProtocol(ctx, c, selector, devicePort, hostPort)
}

func createReverseTCPProtocol(ctx context.Context, c net.Conn, selector string, devicePort, hostPort int) error {
	if selector == "" || len(selector) > 512 || strings.ContainsAny(selector, ";\r\n\x00") {
		return fmt.Errorf("reverse TCP requires an exact bounded device selector")
	}
	if devicePort < 1024 || devicePort > 65535 || hostPort < 1024 || hostPort > 65535 {
		return fmt.Errorf("invalid reverse TCP ports")
	}
	deadline := time.Now().Add(5 * time.Second)
	if end, ok := ctx.Deadline(); ok && end.Before(deadline) {
		deadline = end
	}
	if err := c.SetDeadline(deadline); err != nil {
		return err
	}
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()
	send := func(request string) error {
		_, err := io.WriteString(c, fmt.Sprintf("%04x%s", len(request), request))
		return err
	}
	status := func() error {
		var status [4]byte
		if _, err := io.ReadFull(c, status[:]); err != nil {
			return err
		}
		if string(status[:]) == "FAIL" {
			return fmt.Errorf("ADB rejected the loopback reverse request")
		}
		if string(status[:]) != "OKAY" {
			return fmt.Errorf("invalid ADB protocol status")
		}
		return nil
	}
	if err := send("host:transport:" + selector); err != nil {
		return err
	}
	if err := status(); err != nil {
		return err
	}
	request := fmt.Sprintf("reverse:forward:norebind:tcp:localhost:%d;tcp:localhost:%d", devicePort, hostPort)
	if err := send(request); err != nil {
		return err
	}
	// Acknowledge service connection, then the device's install result.
	if err := status(); err != nil {
		return err
	}
	return status()
}
