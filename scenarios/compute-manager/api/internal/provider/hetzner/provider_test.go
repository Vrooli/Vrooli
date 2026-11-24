package hetzner

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"compute-manager/internal/provider"
)

func TestCreateMapsServerAndResolvesTokenAtCallTime(t *testing.T) {
	var token string
	var publicNet struct {
		EnableIPv4 bool `json:"enable_ipv4"`
		EnableIPv6 bool `json:"enable_ipv6"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token = r.Header.Get("Authorization")
		var body struct {
			PublicNet struct {
				EnableIPv4 bool `json:"enable_ipv4"`
				EnableIPv6 bool `json:"enable_ipv6"`
			} `json:"public_net"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode create body: %v", err)
		}
		publicNet = body.PublicNet
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"server":{"id":7,"created":"2026-09-03T20:00:00Z","public_net":{"ipv4":{"ip":"203.0.113.7"}},"datacenter":{"location":{"name":"fsn1"}},"server_type":{"name":"cx22"},"image":{"name":"ubuntu"}}}`))
	}))
	defer server.Close()
	p := &Provider{BaseURL: server.URL, Token: func(context.Context) (string, error) { return "secret", nil }}
	got, err := p.Create(context.Background(), provider.Spec{Region: "fsn1", Size: "cx22", IPv4: true, Image: "ubuntu"})
	if err != nil || got.ID != "7" || got.Address != "203.0.113.7" || token != "Bearer secret" || !publicNet.EnableIPv4 || publicNet.EnableIPv6 {
		t.Fatalf("got=%+v err=%v token=%q public_net=%+v", got, err, token, publicNet)
	}
}

func TestUnavailableCredentialIsTyped(t *testing.T) {
	p := &Provider{Token: func(context.Context) (string, error) { return "", errors.New("missing") }}
	_, err := p.Describe(context.Background(), "7")
	if !errors.Is(err, provider.ErrProviderUnavailable) {
		t.Fatalf("err=%v", err)
	}
}
