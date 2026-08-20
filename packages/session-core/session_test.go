package sessioncore

import (
	"context"
	"io"
	"testing"
)

type fakeAgent struct {
	in  []AgentFrame
	out []AgentFrame
}

func (f *fakeAgent) Receive(context.Context) (AgentFrame, error) {
	if len(f.in) == 0 {
		return AgentFrame{}, io.EOF
	}
	v := f.in[0]
	f.in = f.in[1:]
	return v, nil
}

func (f *fakeAgent) Send(_ context.Context, frame AgentFrame) error {
	f.out = append(f.out, frame)
	return nil
}
func (f *fakeAgent) Close() error { return nil }

func TestAgentBackendPreservesBytesAndUsesTypedResize(t *testing.T) {
	stream := &fakeAgent{in: []AgentFrame{{Data: []byte("hello ")}, {Data: []byte("world")}}}
	b := NewAgentBackend(func(context.Context, LaunchSpec) (AgentStream, error) { return stream, nil })
	p, err := b.Open(context.Background(), LaunchSpec{})
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 32)
	n, err := p.Read(buf)
	if err != nil || string(buf[:n]) != "hello " {
		t.Fatalf("read=%q err=%v", buf[:n], err)
	}
	if err := p.WriteInput([]byte("echo $(unsafe)"), KindKeystroke); err != nil {
		t.Fatal(err)
	}
	if err := p.SetSize(120, 40); err != nil {
		t.Fatal(err)
	}
	if len(stream.out) != 2 || string(stream.out[0].Data) != "echo $(unsafe)" || !stream.out[1].Resize {
		t.Fatalf("typed frames not preserved: %#v", stream.out)
	}
}

type fakeSSH struct {
	data    []byte
	writes  []byte
	resized bool
	started bool
}

func (s *fakeSSH) Read(p []byte) (int, error) {
	if len(s.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, s.data)
	s.data = s.data[n:]
	return n, nil
}
func (s *fakeSSH) Write(p []byte) (int, error)             { s.writes = append(s.writes, p...); return len(p), nil }
func (s *fakeSSH) RequestPTY(string, uint16, uint16) error { return nil }
func (s *fakeSSH) Resize(uint16, uint16) error             { s.resized = true; return nil }
func (s *fakeSSH) Start() error                            { s.started = true; return nil }
func (s *fakeSSH) Wait() error                             { return nil }
func (s *fakeSSH) Close() error                            { return nil }

func TestSSHBackendHasSameObservableContract(t *testing.T) {
	ssh := &fakeSSH{data: []byte("hello")}
	b := NewSSHBackend(func(context.Context, LaunchSpec) (SSHSession, error) { return ssh, nil })
	p, err := b.Open(context.Background(), LaunchSpec{Cols: 80, Rows: 24})
	if err != nil || !ssh.started {
		t.Fatalf("open err=%v started=%v", err, ssh.started)
	}
	buf := make([]byte, 8)
	n, err := p.Read(buf)
	if err != nil || string(buf[:n]) != "hello" {
		t.Fatalf("read=%q err=%v", buf[:n], err)
	}
	if err := p.WriteInput([]byte("x"), KindPaste); err != nil {
		t.Fatal(err)
	}
	if err := p.SetSize(100, 30); err != nil || !ssh.resized {
		t.Fatalf("resize err=%v resized=%v", err, ssh.resized)
	}
}

func TestOriginRejectsLookalikesAndAcceptsSameAuthority(t *testing.T) {
	for _, tc := range []struct {
		origin, host string
		want         bool
	}{
		{"https://desktop.example:8443", "desktop.example:8443", true},
		{"https://desktop.example.evil", "desktop.example", false},
		{"javascript://desktop.example", "desktop.example", false},
		{"", "desktop.example", false},
	} {
		if got := SameOrigin(tc.origin, tc.host); got != tc.want {
			t.Errorf("SameOrigin(%q,%q)=%v want %v", tc.origin, tc.host, got, tc.want)
		}
	}
}
