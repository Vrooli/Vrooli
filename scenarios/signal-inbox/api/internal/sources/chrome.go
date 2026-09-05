package sources

import (
	"context"
	"fmt"
	"io"
	"strings"

	"signal-inbox/internal/signals"

	"golang.org/x/net/html"
)

const ChromeBookmarksAdapterID = "chrome-bookmarks-html"

type ChromeBookmarksAdapter struct{}

func (ChromeBookmarksAdapter) Descriptor() Descriptor {
	return Descriptor{ID: ChromeBookmarksAdapterID, Kind: "archive_import", RiskTier: RiskTier0}
}

func (ChromeBookmarksAdapter) Parse(_ context.Context, input io.Reader) ([]signals.CaptureInput, error) {
	doc, err := html.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("parse Chrome bookmarks HTML: %w", err)
	}
	var captures []signals.CaptureInput
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && strings.EqualFold(node.Data, "a") {
			for _, attr := range node.Attr {
				if strings.EqualFold(attr.Key, "href") && strings.TrimSpace(attr.Val) != "" {
					captures = append(captures, signals.CaptureInput{URL: strings.TrimSpace(attr.Val), CaptureNote: strings.TrimSpace(nodeText(node)), Tags: folderTags(node)})
					break
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	if len(captures) == 0 {
		return nil, fmt.Errorf("not a measured Chrome bookmark HTML export: no bookmark anchors found")
	}
	return captures, nil
}

// folderTags preserves the bookmark-folder context carried by the measured
// Netscape export. It walks ancestor DL nodes and reads the H3 in the
// immediately preceding DT sibling; a bookmark stays one immutable signal,
// while its folder path becomes searchable facet metadata.
func folderTags(node *html.Node) []string {
	var reverse []string
	for current := node.Parent; current != nil; current = current.Parent {
		if current.Type != html.ElementNode || !strings.EqualFold(current.Data, "dl") {
			continue
		}
		for sibling := current.PrevSibling; sibling != nil; sibling = sibling.PrevSibling {
			if title := firstElementText(sibling, "h3"); title != "" {
				reverse = append(reverse, title)
				break
			}
		}
	}
	for left, right := 0, len(reverse)-1; left < right; left, right = left+1, right-1 {
		reverse[left], reverse[right] = reverse[right], reverse[left]
	}
	return reverse
}

func firstElementText(node *html.Node, name string) string {
	if node.Type == html.ElementNode && strings.EqualFold(node.Data, name) {
		return strings.TrimSpace(nodeText(node))
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if text := firstElementText(child, name); text != "" {
			return text
		}
	}
	return ""
}

func nodeText(node *html.Node) string {
	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			builder.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return builder.String()
}
