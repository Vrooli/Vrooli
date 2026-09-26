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
