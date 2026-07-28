package sources

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"signal-inbox/internal/signals"
)

// RedditSavedArchiveAdapter imports the operator's Reddit GDPR export without
// making a platform request. The measured export carries deliberately saved
// material in saved_posts.csv and saved_comments.csv; authored posts, votes,
// messages, and account data are intentionally outside this intake path.
const RedditSavedArchiveAdapterID = "reddit-saved-archive"

type RedditSavedArchiveAdapter struct{}

// redditArchiveLimits bound the untrusted local export before it is expanded
// or accumulated as capture inputs. They are deliberately generous for an
// operator-owned account export, while preventing a malformed ZIP from using
// unbounded process memory.
type redditArchiveLimits struct {
	maxArchiveBytes  int64
	maxSavedCSVBytes uint64
	maxSavedEntries  int
}

var defaultRedditArchiveLimits = redditArchiveLimits{
	maxArchiveBytes:  128 << 20,
	maxSavedCSVBytes: 64 << 20,
	maxSavedEntries:  250_000,
}

func (RedditSavedArchiveAdapter) Descriptor() Descriptor {
	return Descriptor{ID: RedditSavedArchiveAdapterID, Kind: "archive_import", RiskTier: RiskTier0}
}

func (RedditSavedArchiveAdapter) Parse(_ context.Context, input io.Reader) ([]signals.CaptureInput, error) {
	return parseRedditSavedArchive(input, defaultRedditArchiveLimits)
}

func parseRedditSavedArchive(input io.Reader, limits redditArchiveLimits) ([]signals.CaptureInput, error) {
	data, err := io.ReadAll(io.LimitReader(input, limits.maxArchiveBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read Reddit GDPR export ZIP: %w", err)
	}
	if int64(len(data)) > limits.maxArchiveBytes {
		return nil, fmt.Errorf("read Reddit GDPR export ZIP: archive exceeds %d byte limit", limits.maxArchiveBytes)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("parse Reddit GDPR export ZIP: %w", err)
	}

	var captures []signals.CaptureInput
	found := false
	for _, file := range archive.File {
		name := strings.ToLower(strings.TrimSpace(file.Name))
		if name != "saved_posts.csv" && name != "saved_comments.csv" {
			continue
		}
		found = true
		entries, err := parseRedditSavedCSV(file, limits)
		if err != nil {
			return nil, err
		}
		captures = append(captures, entries...)
	}
	if !found {
		return nil, fmt.Errorf("not a measured Reddit GDPR export: saved_posts.csv and saved_comments.csv are absent")
	}
	return captures, nil
}

func parseRedditSavedCSV(file *zip.File, limits redditArchiveLimits) ([]signals.CaptureInput, error) {
	if file.UncompressedSize64 > limits.maxSavedCSVBytes {
		return nil, fmt.Errorf("read Reddit export %s: saved CSV exceeds %d byte limit", file.Name, limits.maxSavedCSVBytes)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open Reddit export %s: %w", file.Name, err)
	}
	defer reader.Close()

	limitedReader := &io.LimitedReader{R: reader, N: int64(limits.maxSavedCSVBytes) + 1}
	csvReader := csv.NewReader(limitedReader)
	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("read Reddit export %s header: %w", file.Name, err)
	}
	permalinkColumn := -1
	for index, value := range header {
		if strings.EqualFold(strings.TrimSpace(value), "permalink") {
			permalinkColumn = index
			break
		}
	}
	if permalinkColumn < 0 {
		return nil, fmt.Errorf("read Reddit export %s: permalink column is required", file.Name)
	}

	entries := []signals.CaptureInput{}
	for {
		record, readErr := csvReader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("read Reddit export %s: %w", file.Name, readErr)
		}
		if permalinkColumn >= len(record) {
			return nil, fmt.Errorf("read Reddit export %s: malformed permalink row", file.Name)
		}
		url := strings.TrimSpace(record[permalinkColumn])
		if url == "" {
			continue
		}
		if len(entries) >= limits.maxSavedEntries {
			return nil, fmt.Errorf("read Reddit export %s: saved entry count exceeds %d limit", file.Name, limits.maxSavedEntries)
		}
		entries = append(entries, signals.CaptureInput{URL: url, Tags: []string{"reddit", "saved"}})
	}
	if limitedReader.N == 0 {
		return nil, fmt.Errorf("read Reddit export %s: saved CSV exceeds %d byte limit", file.Name, limits.maxSavedCSVBytes)
	}
	return entries, nil
}
