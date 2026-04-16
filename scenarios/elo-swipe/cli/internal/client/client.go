package client

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type Client struct {
	core *cliapp.ScenarioApp
}

func New(core *cliapp.ScenarioApp) *Client {
	return &Client{core: core}
}

type List struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ItemCount   int    `json:"item_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CreateListResponse struct {
	ListID        string `json:"list_id"`
	ItemCount     int    `json:"item_count"`
	ComparisonURL string `json:"comparison_url"`
}

type Item struct {
	ID      string          `json:"id"`
	Content json.RawMessage `json:"content"`
}

type Progress struct {
	Completed int `json:"completed"`
	Total     int `json:"total"`
}

type NextComparison struct {
	ItemA    *Item    `json:"item_a"`
	ItemB    *Item    `json:"item_b"`
	Progress Progress `json:"progress"`
}

type Comparison struct {
	ID                 string  `json:"id"`
	ListID             string  `json:"list_id"`
	WinnerID           string  `json:"winner_id"`
	LoserID            string  `json:"loser_id"`
	WinnerRatingBefore float64 `json:"winner_rating_before"`
	LoserRatingBefore  float64 `json:"loser_rating_before"`
	WinnerRatingAfter  float64 `json:"winner_rating_after"`
	LoserRatingAfter   float64 `json:"loser_rating_after"`
	Timestamp          string  `json:"timestamp"`
}

type RankedItem struct {
	Rank       int             `json:"rank"`
	Item       json.RawMessage `json:"item"`
	EloRating  float64         `json:"elo_rating"`
	Confidence float64         `json:"confidence"`
}

type RankingsResponse struct {
	Rankings []RankedItem `json:"rankings"`
}

func (c *Client) ListLists() ([]List, error) {
	body, err := c.core.Get("/lists", nil)
	if err != nil {
		return nil, err
	}
	var resp []List
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return resp, nil
}

func (c *Client) GetList(id string) (*List, error) {
	body, err := c.core.Get("/lists/"+id, nil)
	if err != nil {
		return nil, err
	}
	var resp List
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &resp, nil
}

func (c *Client) CreateList(name, description string, items []json.RawMessage) (*CreateListResponse, error) {
	type itemInput struct {
		Content json.RawMessage `json:"content"`
	}
	payload := struct {
		Name        string      `json:"name"`
		Description string      `json:"description"`
		Items       []itemInput `json:"items"`
	}{
		Name:        name,
		Description: description,
		Items:       make([]itemInput, 0, len(items)),
	}
	for _, item := range items {
		payload.Items = append(payload.Items, itemInput{Content: item})
	}
	body, err := c.core.Request("POST", "/lists", nil, payload)
	if err != nil {
		return nil, err
	}
	var resp CreateListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &resp, nil
}

func (c *Client) NextComparison(listID string) (*NextComparison, error) {
	body, err := c.core.Get("/lists/"+listID+"/next-comparison", nil)
	if err != nil {
		if isAPIStatus(err, 204) {
			return nil, nil
		}
		return nil, err
	}
	var resp NextComparison
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &resp, nil
}

func (c *Client) CreateComparison(listID, winnerID, loserID string) (*Comparison, error) {
	payload := struct {
		ListID   string `json:"list_id"`
		WinnerID string `json:"winner_id"`
		LoserID  string `json:"loser_id"`
	}{
		ListID:   listID,
		WinnerID: winnerID,
		LoserID:  loserID,
	}
	body, err := c.core.Request("POST", "/comparisons", nil, payload)
	if err != nil {
		return nil, err
	}
	var resp Comparison
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &resp, nil
}

func (c *Client) DeleteComparison(id string) error {
	_, err := c.core.Request("DELETE", "/comparisons/"+id, nil, nil)
	if err != nil && !isAPIStatus(err, 204) {
		return err
	}
	return nil
}

func (c *Client) Rankings(listID string) ([]RankedItem, error) {
	body, err := c.core.Get("/lists/"+listID+"/rankings", nil)
	if err != nil {
		return nil, err
	}
	var resp RankingsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return resp.Rankings, nil
}

func isAPIStatus(err error, status int) bool {
	var apiErr *cliutil.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == status
}
