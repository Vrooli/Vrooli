package livedesktop

import (
	"net"
	"slices"
	"testing"
	"time"
)

func TestRemoteAccessCommandsBindToLoopback(t *testing.T) {
	vnc := x11vncArgs(":99", 5901)
	if !slices.Contains(vnc, "-localhost") {
		t.Fatalf("x11vnc args must include -localhost: %v", vnc)
	}
	websockify := websockifyArgs(6081, 5901)
	if websockify[0] != "127.0.0.1:6081" || websockify[1] != "127.0.0.1:5901" {
		t.Fatalf("websockify must use loopback endpoints: %v", websockify)
	}
}

func TestWaitForLoopbackListenerRequiresReachableEndpoint(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	if err := waitForLoopbackListener(port, time.Second); err != nil {
		t.Fatalf("waitForLoopbackListener() error = %v", err)
	}
}
