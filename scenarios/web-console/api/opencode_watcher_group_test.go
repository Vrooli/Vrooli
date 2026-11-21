package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"web-console/backends/opencode"
	"web-console/internal/sessionstore"
)

func TestDiscoverOpenCodeCaptureTargetsFindsBothMajors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("OPENCODE_DATA_DIR", home)
	arch := map[string]string{"amd64": "x64", "arm64": "arm64", "386": "ia32"}[runtime.GOARCH]
	if arch == "" {
		t.Skip("unsupported fixture architecture")
	}
	write := func(version, packagePath string) {
		path := filepath.Join(home, "cli", version+"-"+runtime.GOOS+"-"+runtime.GOARCH, "node_modules", packagePath, "bin", "opencode")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write("1.19.5", "opencode-"+runtime.GOOS+"-"+arch+"-baseline")
	write("2.3.2", "@opencode/cli-"+runtime.GOOS+"-"+arch+"-baseline")
	targets := discoverOpenCodeCaptureTargets()
	if len(targets) != 2 || targets[0].Major != 1 || targets[1].Major != 2 {
		t.Fatalf("targets = %#v", targets)
	}
	if targets[0].Version != "1.19.5" || targets[1].Version != "2.3.2" {
		t.Fatalf("versions = %#v", targets)
	}
}

func TestOpenCodeWatcherGroupRoutesBySession(t *testing.T) {
	srv, _, _ := newOpenCodeWatcherTest(t)
	ctx := context.Background()
	v1Client := &fakeOpenCodeClient{messages: map[string][]opencode.MessageWithParts{"ses_v1": {ocUser(10, "v1")}}}
	v2Client := &fakeOpenCodeClient{messages: map[string][]opencode.MessageWithParts{"ses_v2": {ocUser(20, "v2")}}}
	v1 := &OpenCodeWatcher{server: srv, claimed: map[string]string{"ses_v1": "pane_v1"}, client: v1Client, major: 1}
	v2 := &OpenCodeWatcher{server: srv, claimed: map[string]string{"ses_v2": "pane_v2"}, client: v2Client, major: 2, pendingPermission: map[string]string{"pane_v2": "req_2"}}
	group := &OpenCodeWatcherGroup{captures: []*OpenCodeWatcher{v1, v2}}

	messages, err := group.nativeClient().SessionMessages(ctx, "ses_v2")
	if err != nil || len(messages) != 1 {
		t.Fatalf("v2 messages = %#v, %v", messages, err)
	}
	if len(v2Client.messageCalls) != 1 || len(v1Client.messageCalls) != 0 {
		t.Fatalf("routing calls v1=%v v2=%v", v1Client.messageCalls, v2Client.messageCalls)
	}
	if err := group.ReplyPermission(ctx, "pane_v2", "once"); err != nil {
		t.Fatal(err)
	}
	if len(v2Client.replies) != 1 || v2Client.replies[0] != "req_2:once" {
		t.Fatalf("v2 replies = %#v", v2Client.replies)
	}
}

func TestVersionGreaterHandlesMultiDigitComponents(t *testing.T) {
	if !versionGreater("1.19.10", "1.19.9") {
		t.Fatal("1.19.10 should be greater than 1.19.9")
	}
	if versionGreater("1.19.9", "1.19.10") {
		t.Fatal("1.19.9 should not be greater than 1.19.10")
	}
}

func TestLoadExistingClaimsForClientOnlyClaimsServedSessions(t *testing.T) {
	srv, w, paneID := newOpenCodeWatcherTest(t)
	ctx := context.Background()
	if err := srv.sessionStore.UpdateAgentInfo(ctx, paneID, sessionstore.AgentInfo{AgentSessionID: "ses_v1"}); err != nil {
		t.Fatal(err)
	}
	if err := srv.sessionStore.Save(ctx, sessionstore.Metadata{ID: "pane_v2", AgentType: sessionstore.AgentOpenCode, AgentSessionID: "ses_v2", CWD: "/work", Created: time.Now().Add(-time.Minute)}); err != nil {
		t.Fatal(err)
	}
	client := &fakeOpenCodeClient{
		messages:      map[string][]opencode.MessageWithParts{"ses_v1": {ocUser(10, "v1")}},
		messageErrors: map[string]error{"ses_v2": errors.New("session not found")},
	}
	w.loadExistingClaimsForClient(client)
	if w.claimed["ses_v1"] != paneID {
		t.Fatalf("served session was not claimed: %#v", w.claimed)
	}
	if _, ok := w.claimed["ses_v2"]; ok {
		t.Fatalf("unserved session was claimed: %#v", w.claimed)
	}
}

func TestOpenCodeSessionMatchesResumedPane(t *testing.T) {
	now := time.Now().UTC()
	pane := sessionstore.Metadata{Created: now, CWD: "/work"}
	session := opencode.Session{Directory: "/work"}
	session.Time.Created = now.Add(-time.Hour).UnixMilli()
	session.Time.Updated = now.Add(time.Minute).UnixMilli()
	if !opencodeSessionMatchesPane(session, pane) {
		t.Fatal("a session resumed by the pane should match")
	}
	session.Time.Updated = now.Add(-time.Hour).UnixMilli()
	if opencodeSessionMatchesPane(session, pane) {
		t.Fatal("an old idle session must not match a fresh pane")
	}
}
