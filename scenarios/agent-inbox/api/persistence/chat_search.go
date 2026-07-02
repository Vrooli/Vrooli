// Package persistence provides database operations for the Agent Inbox scenario.
// This file contains search operations for chats and messages.
package persistence

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"agent-inbox/domain"
)

// SearchResult represents a single search match.
type SearchResult struct {
	Chat       domain.Chat `json:"chat"`
	MessageID  string      `json:"message_id,omitempty"`
	Snippet    string      `json:"snippet,omitempty"`
	MatchStart int         `json:"match_start"`
	MatchEnd   int         `json:"match_end"`
	Rank       float64     `json:"rank"`
	MatchType  string      `json:"match_type"` // "chat_name" or "message_content"
}

// compileSearchPattern compiles a search query into a regexp based on the options.
// Follows prompt-manager's compileContentPattern logic.
func compileSearchPattern(query string, caseSensitive, wholeWord, useRegex bool) (*regexp.Regexp, error) {
	pattern := query
	if !useRegex {
		pattern = regexp.QuoteMeta(pattern)
	}
	if wholeWord {
		pattern = `\b` + pattern + `\b`
	}
	if !caseSensitive {
		pattern = `(?i)` + pattern
	}
	return regexp.Compile(pattern)
}

// ExtractSnippet returns a ~50-char window from line centered on the match,
// with "..." prefix/suffix if truncated. Also returns the byte offset of the
// match within the returned snippet. The window is sized to fit the sidebar
// without CSS truncation hiding the highlighted match.
func ExtractSnippet(line string, matchStart, matchEnd int) (snippet string, snipStart, snipEnd int) {
	const windowSize = 50
	matchLen := matchEnd - matchStart

	// If the line is short enough, return it as-is
	if len(line) <= windowSize {
		return line, matchStart, matchEnd
	}

	// Center window around match
	center := matchStart + matchLen/2
	halfWindow := windowSize / 2
	start := center - halfWindow
	end := center + halfWindow

	if start < 0 {
		end -= start // shift end right
		start = 0
	}
	if end > len(line) {
		start -= end - len(line) // shift start left
		end = len(line)
	}
	if start < 0 {
		start = 0
	}

	snippet = line[start:end]
	snipStart = matchStart - start
	snipEnd = matchEnd - start

	// Clamp to snippet bounds
	if snipStart < 0 {
		snipStart = 0
	}
	if snipEnd > len(snippet) {
		snipEnd = len(snippet)
	}

	prefix := ""
	suffix := ""
	if start > 0 {
		prefix = "..."
	}
	if end < len(line) {
		suffix = "..."
	}

	snippet = prefix + snippet + suffix
	snipStart += len(prefix)
	snipEnd += len(prefix)

	// Convert byte offsets to character (rune) offsets for JavaScript String.slice()
	// which works on characters, not bytes. For ASCII-only content this is a no-op.
	snipStart = utf8.RuneCountInString(snippet[:snipStart])
	snipEnd = utf8.RuneCountInString(snippet[:snipEnd])
	return snippet, snipStart, snipEnd
}

// SearchChats performs regex-based search across chat names and message content.
// Returns results with match byte ranges for client-side highlighting.
// perChat controls how many message matches are returned per chat (1–10, default 1).
func (r *Repository) SearchChats(ctx context.Context, query string, limit, perChat int, caseSensitive, wholeWord, useRegex bool) ([]SearchResult, error) {
	if query == "" {
		return []SearchResult{}, nil
	}

	pattern, err := compileSearchPattern(query, caseSensitive, wholeWord, useRegex)
	if err != nil {
		return nil, fmt.Errorf("invalid search pattern: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}
	if perChat <= 0 {
		perChat = 1
	}
	if perChat > 10 {
		perChat = 10
	}

	results := make([]SearchResult, 0)

	// 1. Search chat names
	chatRows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.name, c.preview, c.model, c.view_mode, c.is_read, c.is_archived, c.is_starred, c.created_at, c.updated_at,
			COALESCE(GROUP_CONCAT(cl.label_id), '') as label_ids
		FROM chats c
		LEFT JOIN chat_labels cl ON c.id = cl.chat_id
		GROUP BY c.id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query chats: %w", err)
	}
	defer chatRows.Close()

	for chatRows.Next() {
		var c domain.Chat
		var labelIDs string
		if err := chatRows.Scan(&c.ID, &c.Name, &c.Preview, &c.Model, &c.ViewMode,
			&c.IsRead, &c.IsArchived, &c.IsStarred, scanTime(&c.CreatedAt), scanTime(&c.UpdatedAt),
			&labelIDs); err != nil {
			continue
		}
		c.LabelIDs = parseArrayString(labelIDs)

		loc := pattern.FindStringIndex(c.Name)
		if loc != nil {
			results = append(results, SearchResult{
				Chat:       c,
				Snippet:    c.Name,
				MatchStart: utf8.RuneCountInString(c.Name[:loc[0]]),
				MatchEnd:   utf8.RuneCountInString(c.Name[:loc[1]]),
				Rank:       2.0, // Chat name matches rank higher
				MatchType:  "chat_name",
			})
		}
	}

	// 2. Search message content
	msgRows, err := r.db.QueryContext(ctx, `
		SELECT m.id, m.chat_id, m.content,
			c.id, c.name, c.preview, c.model, c.view_mode, c.is_read, c.is_archived, c.is_starred, c.created_at, c.updated_at,
			COALESCE(GROUP_CONCAT(cl.label_id), '') as label_ids
		FROM messages m
		JOIN chats c ON c.id = m.chat_id
		LEFT JOIN chat_labels cl ON c.id = cl.chat_id
		WHERE m.role IN ('user', 'assistant')
		GROUP BY m.id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer msgRows.Close()

	// Track matches per chat to enforce perChat limit
	chatMatchCount := make(map[string]int)

	for msgRows.Next() {
		var msgID, chatID, content string
		var c domain.Chat
		var labelIDs string
		if err := msgRows.Scan(&msgID, &chatID, &content,
			&c.ID, &c.Name, &c.Preview, &c.Model, &c.ViewMode,
			&c.IsRead, &c.IsArchived, &c.IsStarred, scanTime(&c.CreatedAt), scanTime(&c.UpdatedAt),
			&labelIDs); err != nil {
			continue
		}
		c.LabelIDs = parseArrayString(labelIDs)

		if chatMatchCount[chatID] >= perChat {
			continue
		}

		// Search line by line for distinct matches.
		// For each line, iterate ALL match occurrences (not just the first)
		// so that long single-line messages return multiple results up to perChat.
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if chatMatchCount[chatID] >= perChat {
				break
			}

			locs := pattern.FindAllStringIndex(line, -1)
			for _, loc := range locs {
				if chatMatchCount[chatID] >= perChat {
					break
				}

				snippet, snipStart, snipEnd := ExtractSnippet(line, loc[0], loc[1])

				results = append(results, SearchResult{
					Chat:       c,
					MessageID:  msgID,
					Snippet:    snippet,
					MatchStart: snipStart,
					MatchEnd:   snipEnd,
					Rank:       1.0,
					MatchType:  "message_content",
				})
				chatMatchCount[chatID]++
			}
		}
	}

	// Sort: chat_name matches first, then message_content
	// Already naturally ordered by query order, but enforce limit
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}
