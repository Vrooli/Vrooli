package backup

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestArchiveCodec(t *testing.T) {
	a := Archive{Version: 1, Prefix: "p:", Entries: []Entry{{Key: "p:x", Value: "dmFsdWU=", TTL: 12}}}
	b, e := Encode(a)
	if e != nil {
		t.Fatal(e)
	}
	got, e := Decode(b)
	if e != nil || got.Entries[0].Key != "p:x" {
		t.Fatalf("Decode = %+v, %v", got, e)
	}
}

func TestDumpRequiresAddress(t *testing.T) {
	if _, e := (Client{}).Dump(context.Background(), "p:"); e == nil {
		t.Fatal("expected error")
	}
}

func TestClientRejectsUnknownRESP(t *testing.T) {
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer ln.Close()
	go func() {
		c, _ := ln.Accept()
		defer c.Close()
		buf := make([]byte, 128)
		_, _ = c.Read(buf)
		_, _ = c.Write([]byte("!x\r\n"))
	}()
	_, e = (Client{Address: ln.Addr().String(), Timeout: time.Second}).call(context.Background(), "PING")
	if e == nil {
		t.Fatal("expected error")
	}
}
