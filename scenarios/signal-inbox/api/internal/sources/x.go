package sources

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"signal-inbox/internal/signals"
)

// XAuthoredArchiveAdapter and XLikesArchiveAdapter deliberately split the same
// tier-zero export into independently enabled streams. The archive contains no
// bookmark records, so this code must never infer one.
const (
	XAuthoredArchiveAdapterID = "x-authored-archive"
	XLikesArchiveAdapterID    = "x-liked-archive"
)

type (
	XAuthoredArchiveAdapter struct{}
	XLikesArchiveAdapter    struct{}
)

func (XAuthoredArchiveAdapter) Descriptor() Descriptor {
	return Descriptor{ID: XAuthoredArchiveAdapterID, Kind: "archive_import", RiskTier: RiskTier0}
}

func (XLikesArchiveAdapter) Descriptor() Descriptor {
	return Descriptor{ID: XLikesArchiveAdapterID, Kind: "archive_import", RiskTier: RiskTier0}
}

func (XAuthoredArchiveAdapter) Parse(_ context.Context, input io.Reader) ([]signals.CaptureInput, error) {
	return parseXArchive(input, "data/tweets.js")
}

func (XLikesArchiveAdapter) Parse(_ context.Context, input io.Reader) ([]signals.CaptureInput, error) {
	return parseXArchive(input, "data/like.js")
}

const (
	maxXArchiveBytes      int64 = 512 << 20
	maxXArchiveEntryBytes int64 = 256 << 20
)

func parseXArchive(input io.Reader, wanted string) ([]signals.CaptureInput, error) {
	data, err := io.ReadAll(io.LimitReader(input, maxXArchiveBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read X archive ZIP: %w", err)
	}
	if int64(len(data)) > maxXArchiveBytes {
		return nil, fmt.Errorf("read X archive ZIP: archive exceeds %d byte limit", maxXArchiveBytes)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("parse X archive ZIP: %w", err)
	}
	for _, file := range archive.File {
		if strings.EqualFold(strings.TrimSpace(file.Name), wanted) {
			if file.UncompressedSize64 > uint64(maxXArchiveEntryBytes) {
				return nil, fmt.Errorf("read X archive %s: entry exceeds %d byte limit", wanted, maxXArchiveEntryBytes)
			}
			reader, openErr := file.Open()
			if openErr != nil {
				return nil, fmt.Errorf("open X archive %s: %w", wanted, openErr)
			}
			defer reader.Close()
			payload, readErr := io.ReadAll(io.LimitReader(reader, maxXArchiveEntryBytes+1))
			if readErr != nil {
				return nil, fmt.Errorf("read X archive %s: %w", wanted, readErr)
			}
			if int64(len(payload)) > maxXArchiveEntryBytes {
				return nil, fmt.Errorf("read X archive %s: entry exceeds %d byte limit", wanted, maxXArchiveEntryBytes)
			}
			return parseXAssignment(payload, wanted)
		}
	}
	return nil, fmt.Errorf("not a measured X archive: %s is absent", wanted)
}

func parseXAssignment(payload []byte, name string) ([]signals.CaptureInput, error) {
	start := bytes.IndexByte(payload, '[')
	if start < 0 {
		return nil, fmt.Errorf("parse X archive %s: expected JavaScript array assignment", name)
	}
	if name == "data/like.js" {
		var rows []struct {
			Like struct {
				TweetID     string `json:"tweetId"`
				FullText    string `json:"fullText"`
				ExpandedURL string `json:"expandedUrl"`
			} `json:"like"`
		}
		if err := json.Unmarshal(payload[start:], &rows); err != nil {
			return nil, fmt.Errorf("parse X likes: %w", err)
		}
		captures := make([]signals.CaptureInput, 0, len(rows))
		for _, row := range rows {
			id := strings.TrimSpace(row.Like.TweetID)
			text := strings.TrimSpace(row.Like.FullText)
			if id == "" {
				continue
			}
			url := strings.TrimSpace(row.Like.ExpandedURL)
			if url == "" {
				url = xStatusURL(id)
			}
			capture := signals.CaptureInput{URL: url, Tags: []string{"x", "liked"}}
			if text != "" {
				capture.Text = text
			}
			captures = append(captures, capture)
		}
		if len(captures) == 0 {
			return nil, fmt.Errorf("parse X likes: no valid like records")
		}
		return captures, nil
	}
	var rows []struct {
		Tweet struct {
			ID       string `json:"id_str"`
			FullText string `json:"full_text"`
			Text     string `json:"text"`
		} `json:"tweet"`
	}
	if err := json.Unmarshal(payload[start:], &rows); err != nil {
		return nil, fmt.Errorf("parse X authored posts: %w", err)
	}
	captures := make([]signals.CaptureInput, 0, len(rows))
	for _, row := range rows {
		id := strings.TrimSpace(row.Tweet.ID)
		text := strings.TrimSpace(row.Tweet.FullText)
		if text == "" {
			text = strings.TrimSpace(row.Tweet.Text)
		}
		if id == "" {
			continue
		}
		capture := signals.CaptureInput{URL: xStatusURL(id), Tags: []string{"x", "authored"}}
		if text != "" {
			capture.Text = text
		}
		captures = append(captures, capture)
	}
	if len(captures) == 0 {
		return nil, fmt.Errorf("parse X authored posts: no valid tweet records")
	}
	return captures, nil
}

func xStatusURL(id string) string { return "https://x.com/i/web/status/" + id }
