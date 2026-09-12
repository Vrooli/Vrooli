package hub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type switchboardDelivery struct {
	base   string
	client *http.Client
}

func NewSwitchboardDelivery(base string) ChannelDelivery {
	return &switchboardDelivery{base: strings.TrimRight(strings.TrimSpace(base), "/"), client: http.DefaultClient}
}

func (s *switchboardDelivery) Send(ctx context.Context, channel, address, title, body string) (string, error) {
	if s.base == "" {
		return "", fmt.Errorf("switchboard API URL is empty")
	}
	payload, err := json.Marshal(map[string]string{
		"channel_id": channel, "thread_key": address, "text": title + "\n\n" + body,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/api/v1/channels/send", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("switchboard delivery: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("switchboard delivery returned %s", resp.Status)
	}
	return "switchboard:" + channel + ":" + address, nil
}
