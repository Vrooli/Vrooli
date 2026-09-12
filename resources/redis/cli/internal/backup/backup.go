// Package backup provides a typed Redis prefix backup format for resource users.
package backup

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/tuning"
)

// Entry is one Redis DUMP payload and its remaining lifetime in milliseconds.
type Entry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	TTL   int64  `json:"ttl_ms"`
}
type Archive struct {
	Version int     `json:"version"`
	Prefix  string  `json:"prefix"`
	Entries []Entry `json:"entries"`
}

// Client issues the subset of RESP commands needed for prefix backups.
type Client struct {
	Address string
	Timeout time.Duration
}

func (c Client) timeout() time.Duration {
	if c.Timeout == 0 {
		return tuning.CredentialServiceTimeout()
	}
	return c.Timeout
}

func (c Client) call(ctx context.Context, args ...string) (any, error) {
	d := net.Dialer{Timeout: c.timeout()}
	conn, err := d.DialContext(ctx, "tcp", c.Address)
	if err != nil {
		return nil, fmt.Errorf("connect Redis: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(c.timeout()))
	}
	w := bufio.NewWriter(conn)
	fmt.Fprintf(w, "*%d\r\n", len(args))
	for _, a := range args {
		fmt.Fprintf(w, "$%d\r\n%s\r\n", len(a), a)
	}
	if err := w.Flush(); err != nil {
		return nil, err
	}
	return readRESP(bufio.NewReader(conn))
}

func readRESP(r *bufio.Reader) (any, error) {
	p, e := r.ReadByte()
	if e != nil {
		return nil, e
	}
	line, e := r.ReadString('\n')
	if e != nil {
		return nil, e
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	switch p {
	case '+':
		return line, nil
	case '-':
		return nil, fmt.Errorf("Redis: %s", line)
	case ':':
		return strconv.ParseInt(line, 10, 64)
	case '$':
		n, e := strconv.Atoi(line)
		if e != nil {
			return nil, e
		}
		if n < 0 {
			return nil, nil
		}
		b := make([]byte, n+2)
		if _, e = r.Read(b); e != nil {
			return nil, e
		}
		return b[:n], nil
	case '*':
		n, e := strconv.Atoi(line)
		if e != nil {
			return nil, e
		}
		a := make([]any, n)
		for i := range a {
			if a[i], e = readRESP(r); e != nil {
				return nil, e
			}
		}
		return a, nil
	}
	return nil, fmt.Errorf("unknown RESP prefix %q", p)
}

func stringReply(v any) (string, error) {
	b, ok := v.([]byte)
	if ok {
		return string(b), nil
	}
	s, ok := v.(string)
	if ok {
		return s, nil
	}
	return "", fmt.Errorf("unexpected Redis reply %T", v)
}

func (c Client) Dump(ctx context.Context, prefix string) (Archive, error) {
	if c.Address == "" {
		return Archive{}, fmt.Errorf("Redis address is required")
	}
	cursor := "0"
	out := Archive{Version: 1, Prefix: prefix}
	for {
		v, e := c.call(ctx, "SCAN", cursor, "MATCH", prefix+"*", "COUNT", "100")
		if e != nil {
			return Archive{}, e
		}
		a, ok := v.([]any)
		if !ok || len(a) != 2 {
			return Archive{}, fmt.Errorf("invalid SCAN response")
		}
		cursor, e = stringReply(a[0])
		if e != nil {
			return Archive{}, e
		}
		keys, ok := a[1].([]any)
		if !ok {
			return Archive{}, fmt.Errorf("invalid SCAN keys")
		}
		for _, raw := range keys {
			k, e := stringReply(raw)
			if e != nil {
				return Archive{}, e
			}
			dump, e := c.call(ctx, "DUMP", k)
			if e != nil {
				return Archive{}, e
			}
			if dump == nil {
				continue
			}
			payload, e := stringReply(dump)
			if e != nil {
				return Archive{}, e
			}
			ttl, e := c.call(ctx, "PTTL", k)
			if e != nil {
				return Archive{}, e
			}
			milliseconds, ok := ttl.(int64)
			if !ok {
				return Archive{}, fmt.Errorf("invalid PTTL response")
			}
			out.Entries = append(out.Entries, Entry{Key: k, Value: base64.StdEncoding.EncodeToString([]byte(payload)), TTL: milliseconds})
		}
		if cursor == "0" {
			break
		}
	}
	sort.Slice(out.Entries, func(i, j int) bool { return out.Entries[i].Key < out.Entries[j].Key })
	return out, nil
}

func (c Client) Restore(ctx context.Context, archive Archive, prefix string) error {
	if archive.Version != 1 {
		return fmt.Errorf("unsupported Redis archive version %d", archive.Version)
	}
	for _, entry := range archive.Entries {
		if prefix != "" && !strings.HasPrefix(entry.Key, prefix) {
			continue
		}
		payload, e := base64.StdEncoding.DecodeString(entry.Value)
		if e != nil {
			return fmt.Errorf("decode %s: %w", entry.Key, e)
		}
		ttl := entry.TTL
		if ttl < 0 {
			ttl = 0
		}
		v, e := c.call(ctx, "RESTORE", entry.Key, strconv.FormatInt(ttl, 10), string(payload), "REPLACE")
		if e != nil {
			return fmt.Errorf("restore %s: %w", entry.Key, e)
		}
		if _, e = stringReply(v); e != nil {
			return e
		}
	}
	return nil
}
func Encode(a Archive) ([]byte, error) { return cliout.MarshalIndent(a) }
func Decode(b []byte) (Archive, error) { var a Archive; err := json.Unmarshal(b, &a); return a, err }
