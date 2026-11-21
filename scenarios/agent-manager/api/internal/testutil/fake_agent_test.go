package testutil

import (
	"os"
	"testing"
)

func TestBuildFakeAgentBuildsReusableExecutable(t *testing.T) {
	path := BuildFakeAgent(t)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("fake agent stat: %v", err)
	}
	if info.IsDir() || info.Mode()&0o111 == 0 {
		t.Fatalf("fake agent is not executable: mode=%v", info.Mode())
	}
	if again := BuildFakeAgent(t); again != path {
		t.Fatalf("BuildFakeAgent path = %q, want cached %q", again, path)
	}
}
