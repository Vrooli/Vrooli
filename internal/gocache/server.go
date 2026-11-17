package gocache

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// The cmd/go GOCACHEPROG protocol (cmd/go/internal/cacheprog). Requests are
// one JSON object per line; a put with a body is followed by the body as one
// base64 JSON string. Responses may be sent in any order; this server answers
// in order because a body must be consumed before the next request is read.
const (
	commandGet   = "get"
	commandPut   = "put"
	commandClose = "close"
)

// maxRequestLine bounds a request header. Bodies are streamed separately.
const maxRequestLine = 1 << 20

type progRequest struct {
	ID       int64
	Command  string
	ActionID []byte `json:",omitempty"`
	OutputID []byte `json:",omitempty"`
	BodySize int64  `json:",omitempty"`
}

type progResponse struct {
	ID            int64
	Err           string     `json:",omitempty"`
	KnownCommands []string   `json:",omitempty"`
	Miss          bool       `json:",omitempty"`
	OutputID      []byte     `json:",omitempty"`
	Size          int64      `json:",omitempty"`
	Time          *time.Time `json:",omitempty"`
	DiskPath      string     `json:",omitempty"`
}

// ServeOptions connects one GOCACHEPROG session to its go command.
type ServeOptions struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// Getenv reads the environment cmd/go started the helper with. cmd/go
	// exports its effective GOCACHE, which binds the session to exactly the
	// store the go command would otherwise use itself.
	Getenv func(string) string
	Now    func() time.Time
}

// Serve runs one session until cmd/go sends close or goes away.
func Serve(opts ServeOptions) (err error) {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	store, err := OpenStore(opts.Getenv("GOCACHE"))
	if err != nil {
		return err
	}
	s := newSession(store, opts.Now, opts.Stderr)
	defer func() { err = errors.Join(err, s.close()) }()

	in := bufio.NewReaderSize(opts.Stdin, 256<<10)
	out := bufio.NewWriter(opts.Stdout)
	send := func(response progResponse) error {
		data, err := json.Marshal(response)
		if err != nil {
			return err
		}
		if _, err := out.Write(append(data, '\n')); err != nil {
			return err
		}
		return out.Flush()
	}
	if err := send(progResponse{KnownCommands: []string{commandGet, commandPut, commandClose}}); err != nil {
		return err
	}
	for {
		request, err := readRequest(in)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		var body *putBody
		if request.BodySize > 0 {
			if body, err = openPutBody(in); err != nil {
				return err
			}
		}
		response := progResponse{ID: request.ID}
		switch request.Command {
		case commandGet:
			s.answerGet(request, &response)
		case commandPut:
			s.answerPut(request, body, &response)
		case commandClose:
			if err := body.finish(); err != nil {
				return err
			}
			return send(response)
		default:
			response.Err = fmt.Sprintf("unknown command %q", request.Command)
		}
		// Every body is consumed completely before the next request is read,
		// whatever the handler did with it; otherwise the stream desynchronizes.
		if err := body.finish(); err != nil {
			return err
		}
		if err := send(response); err != nil {
			return err
		}
	}
}

func (s *session) answerGet(request progRequest, response *progResponse) {
	action, ok := idFromBytes(request.ActionID)
	if !ok {
		response.Err = "get requires a 32-byte ActionID"
		return
	}
	result, err := s.get(action)
	if err != nil {
		response.Miss = true
		return
	}
	stored := result.entry.Time
	response.OutputID = result.entry.Output[:]
	response.Size = result.entry.Size
	response.Time = &stored
	response.DiskPath = result.diskPath
}

func (s *session) answerPut(request progRequest, body *putBody, response *progResponse) {
	action, okAction := idFromBytes(request.ActionID)
	output, okOutput := idFromBytes(request.OutputID)
	if !okAction || !okOutput {
		response.Err = "put requires 32-byte ActionID and OutputID"
		return
	}
	var reader io.Reader = bytes.NewReader(nil)
	if body != nil {
		reader = body
	}
	path, err := s.put(action, output, request.BodySize, reader)
	if err != nil {
		response.Err = err.Error()
		return
	}
	response.DiskPath = path
}

func readRequest(in *bufio.Reader) (progRequest, error) {
	if err := skipSpace(in); err != nil {
		return progRequest{}, err
	}
	line, err := readLine(in)
	if err != nil {
		return progRequest{}, err
	}
	var request progRequest
	if err := json.Unmarshal(line, &request); err != nil {
		return progRequest{}, fmt.Errorf("decode GOCACHEPROG request: %w", err)
	}
	if request.BodySize < 0 {
		return progRequest{}, fmt.Errorf("GOCACHEPROG request %d has a negative body size", request.ID)
	}
	return request, nil
}

func readLine(in *bufio.Reader) ([]byte, error) {
	var line []byte
	for {
		chunk, err := in.ReadSlice('\n')
		line = append(line, chunk...)
		if len(line) > maxRequestLine {
			return nil, errors.New("GOCACHEPROG request line is too long")
		}
		switch {
		case err == nil:
			return line, nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			if len(bytes.TrimSpace(line)) == 0 {
				return nil, io.EOF
			}
			return line, nil
		default:
			return nil, err
		}
	}
}

func skipSpace(in *bufio.Reader) error {
	for {
		b, err := in.ReadByte()
		if err != nil {
			return err
		}
		if b != ' ' && b != '\t' && b != '\r' && b != '\n' {
			return in.UnreadByte()
		}
	}
}

// putBody streams a put's base64 body without buffering it in memory.
type putBody struct {
	quoted  *quotedReader
	decoder io.Reader
}

func openPutBody(in *bufio.Reader) (*putBody, error) {
	if err := skipSpace(in); err != nil {
		return nil, fmt.Errorf("read GOCACHEPROG body: %w", unexpectedEOF(err))
	}
	quote, err := in.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("read GOCACHEPROG body: %w", unexpectedEOF(err))
	}
	if quote != '"' {
		return nil, fmt.Errorf("GOCACHEPROG body does not start with a JSON string")
	}
	quoted := &quotedReader{in: in}
	return &putBody{quoted: quoted, decoder: base64.NewDecoder(base64.StdEncoding, quoted)}, nil
}

func (b *putBody) Read(p []byte) (int, error) { return b.decoder.Read(p) }

// finish discards whatever the handler left unread and requires the closing
// quote, keeping the request stream aligned.
func (b *putBody) finish() error {
	if b == nil {
		return nil
	}
	if _, err := io.Copy(io.Discard, b.decoder); err != nil {
		return fmt.Errorf("read GOCACHEPROG body: %w", err)
	}
	if !b.quoted.done {
		return errors.New("GOCACHEPROG body was not terminated")
	}
	return nil
}

// quotedReader yields bytes up to, not including, the closing quote. The
// base64 alphabet contains no quote, so the first one ends the body.
type quotedReader struct {
	in      *bufio.Reader
	pending []byte
	done    bool
}

func (q *quotedReader) Read(p []byte) (int, error) {
	for len(q.pending) == 0 {
		if q.done {
			return 0, io.EOF
		}
		chunk, err := q.in.ReadSlice('"')
		switch {
		case err == nil:
			q.pending, q.done = chunk[:len(chunk)-1], true
		case errors.Is(err, bufio.ErrBufferFull):
			q.pending = chunk
		default:
			return 0, unexpectedEOF(err)
		}
	}
	n := copy(p, q.pending)
	q.pending = q.pending[n:]
	return n, nil
}

func unexpectedEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
