package dbdetect_test

import (
	"context"
	"testing"

	"test-genie/internal/playbooks/dbdetect"
	"test-genie/internal/playbooks/dbdetect/mocks"
)

func TestSourceCollector(t *testing.T) {
	dbdetect.SetSourceTokens([]string{"SQLITE_PATH", `sql.Open("sqlite"`, "POSTGRES_URL"})

	t.Run("single hit", func(t *testing.T) {
		fs := &mocks.FakeFilesystem{Files: map[string][]byte{
			"/s/main.go": []byte(`package main\nconst x = "SQLITE_PATH"`),
		}}
		obs, err := dbdetect.SourceCollector{}.Collect(context.Background(), dbdetect.ScenarioInputs{ScenarioDir: "/s", Filesystem: fs})
		if err != nil {
			t.Fatalf("collect: %v", err)
		}
		if len(obs) != 1 || obs[0].Value != "SQLITE_PATH" || obs[0].Count != 1 {
			t.Fatalf("got %v", obs)
		}
	})

	t.Run("multiple hits counted per token", func(t *testing.T) {
		fs := &mocks.FakeFilesystem{Files: map[string][]byte{
			"/s/a.go": []byte("SQLITE_PATH"),
			"/s/b.go": []byte("SQLITE_PATH"),
			"/s/c.go": []byte(`sql.Open("sqlite", "")`),
		}}
		obs, err := dbdetect.SourceCollector{}.Collect(context.Background(), dbdetect.ScenarioInputs{ScenarioDir: "/s", Filesystem: fs})
		if err != nil {
			t.Fatalf("collect: %v", err)
		}
		gotCounts := map[string]int{}
		for _, o := range obs {
			gotCounts[o.Value] = o.Count
		}
		if gotCounts["SQLITE_PATH"] != 2 {
			t.Fatalf("expected SQLITE_PATH count 2, got %d", gotCounts["SQLITE_PATH"])
		}
		if gotCounts[`sql.Open("sqlite"`] != 1 {
			t.Fatalf("expected sql.Open count 1, got %d", gotCounts[`sql.Open("sqlite"`])
		}
	})

	t.Run("non-go files ignored", func(t *testing.T) {
		fs := &mocks.FakeFilesystem{Files: map[string][]byte{
			"/s/README.md": []byte("SQLITE_PATH everywhere"),
		}}
		obs, err := dbdetect.SourceCollector{}.Collect(context.Background(), dbdetect.ScenarioInputs{ScenarioDir: "/s", Filesystem: fs})
		if err != nil {
			t.Fatalf("collect: %v", err)
		}
		if len(obs) != 0 {
			t.Fatalf("expected no observations, got %v", obs)
		}
	})

	t.Run("test files ignored", func(t *testing.T) {
		fs := &mocks.FakeFilesystem{Files: map[string][]byte{
			"/s/main_test.go": []byte("SQLITE_PATH"),
		}}
		obs, err := dbdetect.SourceCollector{}.Collect(context.Background(), dbdetect.ScenarioInputs{ScenarioDir: "/s", Filesystem: fs})
		if err != nil {
			t.Fatalf("collect: %v", err)
		}
		if len(obs) != 0 {
			t.Fatalf("expected no observations, got %v", obs)
		}
	})

	t.Run("no tokens registered yields nothing", func(t *testing.T) {
		dbdetect.SetSourceTokens(nil)
		t.Cleanup(func() { dbdetect.SetSourceTokens([]string{"SQLITE_PATH"}) })
		fs := &mocks.FakeFilesystem{Files: map[string][]byte{"/s/x.go": []byte("SQLITE_PATH")}}
		obs, err := dbdetect.SourceCollector{}.Collect(context.Background(), dbdetect.ScenarioInputs{ScenarioDir: "/s", Filesystem: fs})
		if err != nil {
			t.Fatalf("collect: %v", err)
		}
		if len(obs) != 0 {
			t.Fatalf("expected no observations, got %v", obs)
		}
	})
}
