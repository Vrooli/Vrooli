package dbdetect_test

import (
	"context"
	"testing"

	"test-genie/internal/playbooks/dbdetect"
	"test-genie/internal/playbooks/dbdetect/mocks"
)

func TestGodepsCollector(t *testing.T) {
	t.Run("single driver", func(t *testing.T) {
		fs := &mocks.FakeFilesystem{Files: map[string][]byte{
			"/s/api/go.mod": []byte("module x\n\ngo 1.24\n\nrequire modernc.org/sqlite v1.40.1\n"),
		}}
		obs, err := dbdetect.GodepsCollector{}.Collect(context.Background(), dbdetect.ScenarioInputs{ScenarioDir: "/s", Filesystem: fs})
		if err != nil {
			t.Fatalf("collect: %v", err)
		}
		if len(obs) != 1 || obs[0].Value != "modernc.org/sqlite" {
			t.Fatalf("got %v", obs)
		}
	})

	t.Run("multiple modfiles aggregated", func(t *testing.T) {
		fs := &mocks.FakeFilesystem{Files: map[string][]byte{
			"/s/api/go.mod": []byte("module x\n\ngo 1.24\n\nrequire (\n\tmodernc.org/sqlite v1.40.1\n\tgithub.com/lib/pq v1.10.0\n)\n"),
			"/s/cli/go.mod": []byte("module y\n\ngo 1.24\n\nrequire modernc.org/sqlite v1.40.1\n"),
		}}
		obs, err := dbdetect.GodepsCollector{}.Collect(context.Background(), dbdetect.ScenarioInputs{ScenarioDir: "/s", Filesystem: fs})
		if err != nil {
			t.Fatalf("collect: %v", err)
		}
		var sqliteCount int
		for _, o := range obs {
			if o.Value == "modernc.org/sqlite" {
				sqliteCount = o.Count
			}
		}
		if sqliteCount != 2 {
			t.Fatalf("expected sqlite count 2, got %d (obs=%v)", sqliteCount, obs)
		}
	})

	t.Run("no go.mod returns nothing without error", func(t *testing.T) {
		fs := &mocks.FakeFilesystem{Files: map[string][]byte{}}
		obs, err := dbdetect.GodepsCollector{}.Collect(context.Background(), dbdetect.ScenarioInputs{ScenarioDir: "/s", Filesystem: fs})
		if err != nil {
			t.Fatalf("collect: %v", err)
		}
		if len(obs) != 0 {
			t.Fatalf("expected no observations, got %v", obs)
		}
	})

	t.Run("malformed go.mod returns error", func(t *testing.T) {
		fs := &mocks.FakeFilesystem{Files: map[string][]byte{
			"/s/go.mod": []byte("this is not valid go.mod content\n!@#$"),
		}}
		_, err := dbdetect.GodepsCollector{}.Collect(context.Background(), dbdetect.ScenarioInputs{ScenarioDir: "/s", Filesystem: fs})
		if err == nil {
			t.Fatalf("expected parse error")
		}
	})
}
