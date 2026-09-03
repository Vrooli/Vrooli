package main

import (
	"context"
	"strconv"
	"testing"
)

// The 2026-09-01 orphaned-mock-api incident: a stranger answering 200 must
// not become the loop's target through the one adoption path.
func TestAdoptPortRejectsForeignAPI(t *testing.T) {
	isolatedHome(t)
	foreign := fakeAPI(t, "mock-api")
	autoheal := fakeAPI(t, "Vrooli Autoheal API")
	config := testConfig(t)

	if config.adoptPort(context.Background(), foreign) {
		t.Fatal("a foreign service was adopted")
	}
	if config.APIPort != "" || config.HealthEndpoint != "" || config.TickEndpoint != "" {
		t.Fatalf("rejected adoption must leave the config untouched: %+v", config)
	}
	if !config.adoptPort(context.Background(), autoheal) {
		t.Fatal("the autoheal API was not adopted")
	}
	if config.APIPort != autoheal || config.LastKnownPort != autoheal {
		t.Fatalf("APIPort=%q LastKnownPort=%q, want %q", config.APIPort, config.LastKnownPort, autoheal)
	}
	if config.TickEndpoint != "http://localhost:"+autoheal+"/api/v1/tick" {
		t.Fatalf("tick endpoint = %q", config.TickEndpoint)
	}
	if config.adoptPort(context.Background(), closedPort(t)) {
		t.Fatal("a closed port was adopted")
	}
}

// Every strategy, fed a port where a stranger answers, must yield no verified
// port; fed the autoheal port, it must verify it. Strategy 2 alone may name
// the stranger's port as pending.
func TestDetectAPIPortVerifiesIdentityOnEveryStrategy(t *testing.T) {
	isolatedHome(t)
	foreign := fakeAPI(t, "mock-api")
	autoheal := fakeAPI(t, "Vrooli Autoheal API")

	strategies := []struct {
		name  string
		feed  func(t *testing.T, config *Config, port string)
		wantP bool
	}{
		{"1 environment", func(t *testing.T, _ *Config, port string) { t.Setenv("API_PORT", port) }, false},
		{"2 registry port file", func(t *testing.T, c *Config, port string) { writeRegistryFile(t, c, "port", port+"\n") }, true},
		{"3 cli status", func(t *testing.T, c *Config, port string) { c.VrooliCmdPath = fakeVrooli(t, contractBody(port)) }, false},
		{"4 registry metadata", func(t *testing.T, c *Config, port string) {
			writeRegistryFile(t, c, "metadata.json", `{"api_port":"`+port+`"}`)
		}, false},
		{"5 last known port", func(_ *testing.T, c *Config, port string) { c.LastKnownPort = port }, false},
		{"6 probe list", func(_ *testing.T, c *Config, port string) {
			p, _ := strconv.Atoi(port)
			c.ProbePorts = []int{p}
		}, false},
	}
	for _, strategy := range strategies {
		t.Run(strategy.name, func(t *testing.T) {
			config := testConfig(t)
			strategy.feed(t, config, foreign)
			found := detectAPIPort(context.Background(), config)
			if found.Verified != "" {
				t.Fatalf("strategy verified the foreign port %q", found.Verified)
			}
			if strategy.wantP && found.Pending != foreign {
				t.Fatalf("pending = %q, want the registry port %q", found.Pending, foreign)
			}
			if !strategy.wantP && found.Pending != "" {
				t.Fatalf("only the registry strategy may surface a pending port, got %q", found.Pending)
			}

			config = testConfig(t)
			strategy.feed(t, config, autoheal)
			if found := detectAPIPort(context.Background(), config); found.Verified != autoheal {
				t.Fatalf("strategy did not verify the autoheal port: %+v", found)
			}
		})
	}
}
