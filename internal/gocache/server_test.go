package gocache

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// progClient writes requests exactly as cmd/go's ProgCache.writeToChild does:
// the JSON object and its newline, an extra newline, then for a put the body
// as one base64 JSON string on its own line.
type progClient struct {
	t      *testing.T
	in     *io.PipeWriter
	out    *json.Decoder
	nextID int64
	done   chan error
}

func startSession(t *testing.T, cache string) *progClient {
	t.Helper()
	inReader, inWriter := io.Pipe()
	outReader, outWriter := io.Pipe()
	client := &progClient{t: t, in: inWriter, out: json.NewDecoder(bufio.NewReader(outReader)), done: make(chan error, 1)}
	go func() {
		err := Serve(ServeOptions{Stdin: inReader, Stdout: outWriter, Stderr: io.Discard, Getenv: func(key string) string {
			if key == "GOCACHE" {
				return cache
			}
			return ""
		}})
		outWriter.Close()
		client.done <- err
	}()
	var hello progResponse
	if err := client.out.Decode(&hello); err != nil {
		t.Fatalf("read capabilities: %v", err)
	}
	if hello.ID != 0 || strings.Join(hello.KnownCommands, ",") != "get,put,close" {
		t.Fatalf("capabilities = %+v", hello)
	}
	return client
}

func (c *progClient) send(request progRequest, body []byte) progResponse {
	c.t.Helper()
	c.nextID++
	request.ID = c.nextID
	request.BodySize = int64(len(body))
	header, err := json.Marshal(request)
	if err != nil {
		c.t.Fatal(err)
	}
	var wire bytes.Buffer
	wire.Write(header)
	wire.WriteString("\n\n")
	if len(body) > 0 {
		wire.WriteByte('"')
		wire.WriteString(base64.StdEncoding.EncodeToString(body))
		wire.WriteString("\"\n")
	}
	if _, err := c.in.Write(wire.Bytes()); err != nil {
		c.t.Fatalf("write request: %v", err)
	}
	var response progResponse
	if err := c.out.Decode(&response); err != nil {
		c.t.Fatalf("read response: %v", err)
	}
	if response.ID != request.ID {
		c.t.Fatalf("response ID %d, want %d", response.ID, request.ID)
	}
	return response
}

func (c *progClient) put(action ID, body []byte) progResponse {
	output := sha256.Sum256(body)
	return c.send(progRequest{Command: commandPut, ActionID: action[:], OutputID: output[:]}, body)
}

func (c *progClient) get(action ID) progResponse {
	return c.send(progRequest{Command: commandGet, ActionID: action[:]}, nil)
}

func (c *progClient) close() {
	c.t.Helper()
	c.send(progRequest{Command: commandClose}, nil)
	if err := <-c.done; err != nil {
		c.t.Fatalf("session ended with %v", err)
	}
}

func newStoreDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for i := range shardCount {
		if err := os.Mkdir(filepath.Join(root, shardName(byte(i))), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func actionID(name string) ID { return sha256.Sum256([]byte("action:" + name)) }

func TestSessionServesPutOutputsAndLaterHits(t *testing.T) {
	cache := newStoreDir(t)
	client := startSession(t, cache)
	body := bytes.Repeat([]byte("compiled package archive\n"), 1000)
	put := client.put(actionID("pkg"), body)
	if put.Err != "" || put.DiskPath == "" {
		t.Fatalf("put = %+v", put)
	}
	if got, err := os.ReadFile(put.DiskPath); err != nil || !bytes.Equal(got, body) {
		t.Fatalf("put output on disk = %d bytes, %v", len(got), err)
	}
	get := client.get(actionID("pkg"))
	output := sha256.Sum256(body)
	if get.Miss || get.DiskPath != put.DiskPath || !bytes.Equal(get.OutputID, output[:]) || get.Size != int64(len(body)) || get.Time == nil {
		t.Fatalf("get = %+v", get)
	}
	if miss := client.get(actionID("absent")); !miss.Miss || miss.DiskPath != "" {
		t.Fatalf("absent get = %+v", miss)
	}
	client.close()
	journals, _ := filepath.Glob(filepath.Join(cache, StateDirName, sessionsDirName, "*"+pinJournalSuffix))
	if len(journals) != 0 {
		t.Fatalf("closed session left journals %v", journals)
	}
}

func TestSessionWritesEntriesCmdGoAccepts(t *testing.T) {
	cache := newStoreDir(t)
	client := startSession(t, cache)
	action := actionID("format")
	client.put(action, []byte("x"))
	client.close()
	data, err := os.ReadFile(Store{Root: cache}.entryPath(action, kindAction))
	if err != nil {
		t.Fatal(err)
	}
	// The exact layout cmd/go's DiskCache.putIndexEntry writes.
	if len(data) != entrySize || !strings.HasPrefix(string(data), "v1 "+action.String()+" ") {
		t.Fatalf("entry = %q", data)
	}
	entry, err := decodeEntry(action, data)
	output := sha256.Sum256([]byte("x"))
	if err != nil || entry.Output != output || entry.Size != 1 {
		t.Fatalf("decoded entry = %+v, %v", entry, err)
	}
}

func TestSessionRejectsBodyThatDoesNotMatchOutputID(t *testing.T) {
	cache := newStoreDir(t)
	client := startSession(t, cache)
	wrong := sha256.Sum256([]byte("other"))
	action := actionID("corrupt")
	response := client.send(progRequest{Command: commandPut, ActionID: action[:], OutputID: wrong[:]}, []byte("body"))
	if response.Err == "" || response.DiskPath != "" {
		t.Fatalf("mismatched put = %+v", response)
	}
	// The stream stays aligned after a refused body.
	if ok := client.put(actionID("next"), []byte("fine")); ok.Err != "" {
		t.Fatalf("put after refusal = %+v", ok)
	}
	if miss := client.get(action); !miss.Miss {
		t.Fatalf("refused body became servable: %+v", miss)
	}
	staged, _ := filepath.Glob(filepath.Join(cache, "*", "*"+tempMarker+"*"))
	if len(staged) != 0 {
		t.Fatalf("refused body left staged files %v", staged)
	}
	client.close()
}

func TestSessionStreamsBodiesLargerThanItsReadBuffer(t *testing.T) {
	cache := newStoreDir(t)
	client := startSession(t, cache)
	body := make([]byte, 3<<20)
	for i := range body {
		body[i] = byte(i * 7)
	}
	put := client.put(actionID("large"), body)
	if put.Err != "" {
		t.Fatalf("large put = %+v", put)
	}
	if got, _ := os.ReadFile(put.DiskPath); !bytes.Equal(got, body) {
		t.Fatal("large body corrupted")
	}
	empty := client.put(actionID("empty"), nil)
	if empty.Err != "" {
		t.Fatalf("empty put = %+v", empty)
	}
	if info, err := os.Stat(empty.DiskPath); err != nil || info.Size() != 0 {
		t.Fatalf("empty output = %v, %v", info, err)
	}
	client.close()
}

func TestSessionNeverServesNativeExecutableDirectories(t *testing.T) {
	cache := newStoreDir(t)
	store := Store{Root: cache}
	action := actionID("link")
	output := sha256.Sum256([]byte("exe"))
	if err := os.MkdirAll(filepath.Join(store.entryPath(output, kindOutput)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.entryPath(action, kindAction), encodeEntry(action, Entry{Output: output, Size: 3, Time: time.Now()}), 0o644); err != nil {
		t.Fatal(err)
	}
	client := startSession(t, cache)
	if response := client.get(action); !response.Miss {
		t.Fatalf("executable directory served as an output file: %+v", response)
	}
	client.close()
}

func TestSessionEndsCleanlyWhenGoCommandDisappears(t *testing.T) {
	cache := newStoreDir(t)
	client := startSession(t, cache)
	client.put(actionID("a"), []byte("a"))
	client.in.Close()
	if err := <-client.done; err != nil {
		t.Fatalf("session after EOF: %v", err)
	}
	journals, _ := filepath.Glob(filepath.Join(cache, StateDirName, sessionsDirName, "*"+pinJournalSuffix))
	if len(journals) != 0 {
		t.Fatalf("abandoned session left journals %v", journals)
	}
}

func TestSessionAnswersUnknownCommandsWithoutLosingTheStream(t *testing.T) {
	client := startSession(t, newStoreDir(t))
	if response := client.send(progRequest{Command: "get2"}, []byte("ignored")); response.Err == "" {
		t.Fatalf("unknown command = %+v", response)
	}
	if response := client.put(actionID("after"), []byte("ok")); response.Err != "" {
		t.Fatalf("put after unknown command = %+v", response)
	}
	client.close()
}
