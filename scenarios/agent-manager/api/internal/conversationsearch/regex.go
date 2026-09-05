package conversationsearch

import (
	"context"
	"errors"
	"regexp"
	"regexp/syntax"
	"sort"
	"strings"
	"time"
	"unicode"
)

type RegexPolicy struct {
	MaxCandidates          int
	NoLiteralMaxCandidates int
	MaxCandidateBytes      int
	NoLiteralMaxBytes      int
	MaxResults             int
	MaxDuration            time.Duration
}

func DefaultRegexPolicy() RegexPolicy {
	return RegexPolicy{
		MaxCandidates: 2000, NoLiteralMaxCandidates: 500,
		MaxCandidateBytes: 16 << 20, NoLiteralMaxBytes: 4 << 20,
		MaxResults: 1000, MaxDuration: 750 * time.Millisecond,
	}
}

type RegexSearchRequest struct {
	Pattern  string
	Sort     SearchSort
	Filters  SearchFilters
	PageSize int
	Cursor   string
}

func (s *Service) SearchRegex(ctx context.Context, request RegexSearchRequest) (TextSearchResponse, error) {
	if s.regex == nil {
		return TextSearchResponse{}, errors.New("regex candidate repository is unavailable")
	}
	request.Pattern = strings.TrimSpace(request.Pattern)
	if request.Pattern == "" {
		return TextSearchResponse{}, invalidRequest("query is required for regex search")
	}
	compiled, err := regexp.Compile(request.Pattern)
	if err != nil {
		return TextSearchResponse{}, errors.Join(ErrInvalidRequest, err)
	}
	if request.Sort == 0 {
		request.Sort = SearchSortRelevance
	}
	if request.PageSize == 0 {
		request.PageSize = defaultSearchPageSize
	}
	if request.PageSize < 1 || request.PageSize > maximumSearchPageSize {
		return TextSearchResponse{}, invalidRequest("page_size must be between 1 and 100")
	}

	fingerprintRequest := TextSearchRequest{
		Query: request.Pattern, Sort: request.Sort, Filters: request.Filters,
		PageSize: request.PageSize, fingerprintMode: "regex",
	}
	fingerprint, err := searchFingerprint(fingerprintRequest)
	if err != nil {
		return TextSearchResponse{}, err
	}
	var after *CandidateCursor
	if request.Cursor != "" {
		payload, decodeErr := s.cursors.decode(request.Cursor, fingerprint, request.Sort)
		if decodeErr != nil {
			return TextSearchResponse{}, errors.Join(ErrInvalidRequest, decodeErr)
		}
		after = &CandidateCursor{Score: payload.Score, OccurredAt: payload.OccurredAt, DocumentID: payload.DocumentID}
	}

	literal := mandatoryRegexLiteral(request.Pattern)
	candidateLimit, byteLimit := s.regexPolicy.MaxCandidates, s.regexPolicy.MaxCandidateBytes
	if literal == "" {
		candidateLimit, byteLimit = s.regexPolicy.NoLiteralMaxCandidates, s.regexPolicy.NoLiteralMaxBytes
	}
	started := time.Now()
	page, err := s.regex.RegexCandidates(ctx, candidateQueryFromRegex(request, literal, candidateLimit, byteLimit))
	if err != nil {
		return TextSearchResponse{}, err
	}
	matches := make([]SearchHit, 0, request.PageSize+1)
	evaluatedCandidates := 0
	for candidateIndex, document := range page.Documents {
		if err := ctx.Err(); err != nil {
			return TextSearchResponse{}, err
		}
		if time.Since(started) >= s.regexPolicy.MaxDuration {
			page.HasMore = true
			page.LimitReason = RegexLimitDeadline
			break
		}
		evaluatedCandidates++
		content := strings.ToValidUTF8(document.Content, "\uFFFD")
		locations := compiled.FindAllStringIndex(content, 32)
		if len(locations) == 0 {
			if time.Since(started) >= s.regexPolicy.MaxDuration {
				page.LimitReason = RegexLimitDeadline
				break
			}
			continue
		}
		document.Content = content
		snippet, highlights := boundedRegexSnippet(content, locations, maximumSnippetBytes)
		matchBytes := locations[0][1] - locations[0][0]
		score := 1 + float64(matchBytes)/float64(maximum(1, len(content))) + 1/float64(1+locations[0][0])
		matches = append(matches, SearchHit{
			Document: document, Snippet: snippet, Highlights: highlights, Score: score,
			Leg: SearchLegRegex, DeepLink: "/runs/" + document.SourceRunID + "?event=" + document.SourceEventID,
		})
		if len(matches) >= s.regexPolicy.MaxResults && (candidateIndex+1 < len(page.Documents) || page.HasMore) {
			page.LimitReason = RegexLimitCandidates
			break
		}
		if time.Since(started) >= s.regexPolicy.MaxDuration {
			page.LimitReason = RegexLimitDeadline
			break
		}
	}
	sortRegexHits(matches, request.Sort)
	if after != nil {
		matches = regexHitsAfter(matches, request.Sort, *after)
	}
	response := TextSearchResponse{
		PartialReason: page.LimitReason, ScannedCandidates: evaluatedCandidates, ScannedBytes: page.ScannedBytes,
	}
	if len(matches) > request.PageSize {
		matches = matches[:request.PageSize]
		last := matches[len(matches)-1]
		response.NextCursor, err = s.cursors.encode(cursorPayload{
			Fingerprint: fingerprint, Sort: request.Sort, Score: last.Score,
			OccurredAt: last.Document.OccurredAt, DocumentID: last.Document.DocumentID,
		})
		if err != nil {
			return TextSearchResponse{}, err
		}
	}
	for index := range matches {
		matches[index].Rank = index + 1
	}
	response.Hits = matches
	response.CanonicalVisibleMessages, response.CatalogDocuments, response.LexicalDocuments, err = s.status.CountCoverage(ctx)
	if err != nil {
		return TextSearchResponse{}, err
	}
	return response, nil
}

func candidateQueryFromRegex(request RegexSearchRequest, literal string, candidateLimit, byteLimit int) CandidateQuery {
	return CandidateQuery{
		PrefilterLiteral: literal, Limit: candidateLimit, ByteLimit: byteLimit,
		OccurredAfter: request.Filters.OccurredAfter, OccurredBefore: request.Filters.OccurredBefore,
		Roles: request.Filters.Roles, Harnesses: request.Filters.Harnesses,
		ProviderOrigins: request.Filters.ProviderOrigins, ProjectScopes: request.Filters.ProjectScopes,
		CWDScopes: request.Filters.CWDScopes, Runners: request.Filters.Runners, Models: request.Filters.Models,
		Profiles: request.Filters.Profiles, RunStatuses: request.Filters.RunStatuses,
		Tags: request.Filters.Tags, Workloads: request.Filters.Workloads, ContentClasses: request.Filters.ContentClasses,
	}
}

func mandatoryRegexLiteral(pattern string) string {
	parsed, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return ""
	}
	literal := longestMandatoryLiteral(parsed.Simplify())
	if literal == "" {
		return ""
	}
	for _, character := range literal {
		if character > unicode.MaxASCII {
			return ""
		}
	}
	if !strings.ContainsFunc(literal, func(character rune) bool { return unicode.IsLetter(character) || unicode.IsDigit(character) }) {
		return ""
	}
	return literal
}

func longestMandatoryLiteral(expression *syntax.Regexp) string {
	switch expression.Op {
	case syntax.OpLiteral:
		return string(expression.Rune)
	case syntax.OpCapture, syntax.OpPlus:
		return longestMandatoryLiteral(expression.Sub[0])
	case syntax.OpRepeat:
		if expression.Min > 0 {
			return longestMandatoryLiteral(expression.Sub[0])
		}
	case syntax.OpConcat:
		longest := ""
		for _, child := range expression.Sub {
			if candidate := longestMandatoryLiteral(child); len(candidate) > len(longest) {
				longest = candidate
			}
		}
		return longest
	}
	return ""
}

func sortRegexHits(hits []SearchHit, order SearchSort) {
	sort.SliceStable(hits, func(left, right int) bool {
		a, b := hits[left], hits[right]
		switch order {
		case SearchSortOldest:
			if !a.Document.OccurredAt.Equal(b.Document.OccurredAt) {
				return a.Document.OccurredAt.Before(b.Document.OccurredAt)
			}
		case SearchSortNewest:
			if !a.Document.OccurredAt.Equal(b.Document.OccurredAt) {
				return a.Document.OccurredAt.After(b.Document.OccurredAt)
			}
		default:
			if a.Score != b.Score {
				return a.Score > b.Score
			}
			if !a.Document.OccurredAt.Equal(b.Document.OccurredAt) {
				return a.Document.OccurredAt.After(b.Document.OccurredAt)
			}
		}
		return a.Document.DocumentID < b.Document.DocumentID
	})
}

func regexHitsAfter(hits []SearchHit, order SearchSort, cursor CandidateCursor) []SearchHit {
	filtered := hits[:0]
	for _, hit := range hits {
		after := false
		switch order {
		case SearchSortOldest:
			after = hit.Document.OccurredAt.After(cursor.OccurredAt) ||
				(hit.Document.OccurredAt.Equal(cursor.OccurredAt) && hit.Document.DocumentID > cursor.DocumentID)
		case SearchSortNewest:
			after = hit.Document.OccurredAt.Before(cursor.OccurredAt) ||
				(hit.Document.OccurredAt.Equal(cursor.OccurredAt) && hit.Document.DocumentID > cursor.DocumentID)
		default:
			after = hit.Score < cursor.Score || (hit.Score == cursor.Score &&
				(hit.Document.OccurredAt.Before(cursor.OccurredAt) ||
					(hit.Document.OccurredAt.Equal(cursor.OccurredAt) && hit.Document.DocumentID > cursor.DocumentID)))
		}
		if after {
			filtered = append(filtered, hit)
		}
	}
	return filtered
}

func boundedRegexSnippet(content string, locations [][]int, maximumBytes int) (string, []Highlight) {
	first := locations[0]
	start := first[0] - maximumBytes/3
	if start < 0 {
		start = 0
	}
	end := start + maximumBytes
	if end > len(content) {
		end = len(content)
		start = maximum(0, end-maximumBytes)
	}
	for start > 0 && !isRuneStart(content[start]) {
		start--
	}
	for end < len(content) && !isRuneStart(content[end]) {
		end--
	}
	highlights := make([]Highlight, 0, len(locations))
	for _, location := range locations {
		if location[0] < start || location[1] > end {
			continue
		}
		highlights = append(highlights, Highlight{
			StartRune: runeCount(content[start:location[0]]),
			EndRune:   runeCount(content[start:location[1]]),
		})
	}
	return content[start:end], highlights
}

func maximum(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func isRuneStart(value byte) bool { return value&0xC0 != 0x80 }

func runeCount(value string) int {
	count := 0
	for range value {
		count++
	}
	return count
}
