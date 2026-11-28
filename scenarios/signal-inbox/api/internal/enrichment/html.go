package enrichment

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"signal-inbox/internal/signals"

	"golang.org/x/net/html"
)

const maxHTMLBytes = 2 << 20

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type HTMLExtractor struct{ client HTTPDoer }

func NewHTMLExtractor(client HTTPDoer) *HTMLExtractor { return &HTMLExtractor{client: client} }

func (e *HTMLExtractor) Supports(kind signals.SourceKind) bool { return kind == signals.SourceKindURL }

func (e *HTMLExtractor) Extract(ctx context.Context, signal signals.Signal) (ExtractionResult, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, signal.SourceURL, nil)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("create URL extraction request: %w", err)
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; VrooliSignalInbox/1.0; +https://vrooli.local)")
	response, err := e.client.Do(request)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("fetch URL: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ExtractionResult{}, fmt.Errorf("fetch URL: unexpected HTTP status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxHTMLBytes+1))
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("read URL body: %w", err)
	}
	if len(body) > maxHTMLBytes {
		return ExtractionResult{}, fmt.Errorf("read URL body: exceeds %d byte limit", maxHTMLBytes)
	}
	content, units := readableHTMLText(strings.NewReader(string(body)))
	return ExtractionResult{Content: content, ContentUnits: units}, nil
}

func readableHTMLText(reader io.Reader) (string, int) {
	tokenizer := html.NewTokenizer(reader)
	var parts []string
	skipDepth := 0
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}
		token := tokenizer.Token()
		switch tokenType {
		case html.StartTagToken:
			if skippedElement(token.Data) {
				skipDepth++
			}
		case html.EndTagToken:
			if skippedElement(token.Data) && skipDepth > 0 {
				skipDepth--
			}
		case html.TextToken:
			if skipDepth == 0 {
				if text := normalizeContent(token.Data); text != "" {
					parts = append(parts, text)
				}
			}
		}
	}
	return strings.Join(parts, "\n"), len(parts)
}

func skippedElement(name string) bool {
	switch strings.ToLower(name) {
	case "script", "style", "noscript", "svg", "head", "nav", "header", "footer", "aside", "form":
		return true
	default:
		return false
	}
}
