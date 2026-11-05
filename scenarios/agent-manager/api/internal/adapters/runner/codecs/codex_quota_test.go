package codecs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCodexLiveQuotaSeparatesBucketsAndWindows(t *testing.T) {
	data := []byte(`{"accountId":"sensitive","rateLimits":{"primary":{"usedPercent":99,"windowDurationMins":300}},"rateLimitsByLimitId":{"codex":{"primary":{"usedPercent":0,"windowDurationMins":300,"resetsAt":2000000000},"secondary":{"usedPercent":35,"windowDurationMins":10080}},"other":{"primary":{"usedPercent":14,"windowDurationMins":10080}}}}`)
	windows, err := parseCodexLiveQuota(data)
	if err != nil || len(windows) != 3 {
		t.Fatalf("windows=%+v err=%v", windows, err)
	}
	if windows[0].Pool != "codex" || windows[0].UsedPercent != 0 || windows[0].WindowMinutes != 300 || windows[0].ResetsAt == nil || windows[1].WindowMinutes != 10080 || windows[2].Pool != "other" {
		t.Fatalf("windows=%+v", windows)
	}
	missing, err := parseCodexLiveQuota([]byte(`{"rateLimits":{"primary":{"usedPercent":3}}}`))
	if err != nil || len(missing) != 0 {
		t.Fatalf("missing duration must not create a bar: %v %v", missing, err)
	}
}

func TestCodexLiveQuotaReadsAppServerAndHonorsDeadline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "codex")
	script := `#!/bin/sh
while IFS= read -r line; do
 case "$line" in
  *'"method":"initialize"'*) echo '{"id":1,"result":{}}' ;;
  *'"method":"account/rateLimits/read"'*) echo '{"id":2,"result":{"accountId":"secret-account","rateLimits":{"primary":{"usedPercent":12,"windowDurationMins":10080}}}}' ;;
 esac
done
`
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	windows, err := NewCodexForTestWithBinary(path).ReadLiveQuota(ctx)
	if err != nil || len(windows) != 1 || windows[0].WindowMinutes != 10080 || windows[0].UsedPercent != 12 {
		t.Fatalf("windows=%+v err=%v", windows, err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexec sleep 30\n"), 0700); err != nil {
		t.Fatal(err)
	}
	short, stop := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer stop()
	start := time.Now()
	_, err = NewCodexForTestWithBinary(path).ReadLiveQuota(short)
	if err == nil || time.Since(start) > 3*time.Second {
		t.Fatalf("deadline err=%v elapsed=%s", err, time.Since(start))
	}
}

// Explicit opt-in live read checks provider availability without logging any
// provider account identifiers or quota values.
func TestCodexLiveQuotaProviderOptIn(t *testing.T) {
	if os.Getenv("CODEX_QUOTA_LIVE_TEST") != "1" {
		t.Skip("live provider read not requested")
	}
	c, err := NewCodex()
	if err != nil {
		t.Fatal("cannot initialize Codex")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	windows, err := c.ReadLiveQuota(ctx)
	if err != nil {
		t.Fatal("bounded provider read failed")
	}
	if len(windows) == 0 {
		t.Fatal("provider returned no displayable quota windows")
	}
}
