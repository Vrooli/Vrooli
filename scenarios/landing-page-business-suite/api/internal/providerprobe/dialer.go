package providerprobe

import (
	"context"
	"net"
	"net/smtp"
	"time"
)

// smtpDialer opens a bounded SMTP session. It is separate from the probe so
// the probe can be tested without a network.
type smtpDialer struct {
	timeout time.Duration
}

func (d *smtpDialer) dial(ctx context.Context, addr string) (SMTPSession, error) {
	timeout := d.timeout
	if timeout <= 0 {
		timeout = defaultProbeTimeout
	}
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return nil, err
	}
	host, _, splitErr := net.SplitHostPort(addr)
	if splitErr != nil {
		host = addr
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}
