package channelmanager

import (
	"fmt"
	"strings"
)

// PreviewInput is metadata only. Media bytes stay with the asset owner; the
// release gate needs only dimensions and disclosure placement evidence.
type PreviewInput struct {
	PlatformID        string
	Caption           string
	FormatKind        string
	MediaWidth        int
	MediaHeight       int
	DisclosureVisible bool
	FirstComment      string
}

type PostPreview struct {
	Caption            string
	CaptionTruncated   bool
	MediaPresentation  string
	DisclosureRequired bool
	ReleaseAllowed     bool
	BlockingErrors     []string
	FirstComment       string
}

// PreviewRelease is descriptor-driven: it contains no per-platform branch.
// A platform can opt into a caption limit or preferred media dimensions by
// extending its descriptor; absent limits preserve the complete caption.
func (s *Service) PreviewRelease(input PreviewInput) (PostPreview, error) {
	platform, ok := s.Platforms[input.PlatformID]
	if !ok {
		return PostPreview{}, fmt.Errorf("unknown platform %q", input.PlatformID)
	}
	caption, truncated := truncateCaption(input.Caption, platform.CaptionLimit)
	preview := PostPreview{Caption: caption, CaptionTruncated: truncated, DisclosureRequired: platform.DisclosureRequired, ReleaseAllowed: true, FirstComment: input.FirstComment, MediaPresentation: "native"}
	if input.FormatKind == "" {
		preview.BlockingErrors = append(preview.BlockingErrors, "media format is required")
	}
	var format *Format
	for i := range platform.Formats {
		if platform.Formats[i].Kind == input.FormatKind {
			format = &platform.Formats[i]
			break
		}
	}
	if format == nil {
		preview.BlockingErrors = append(preview.BlockingErrors, "media format is not supported by this platform")
	} else if input.MediaWidth < format.MinWidth || input.MediaHeight < format.MinHeight || input.MediaWidth > format.MaxWidth || input.MediaHeight > format.MaxHeight {
		preview.BlockingErrors = append(preview.BlockingErrors, "media dimensions are outside platform descriptor limits")
	}
	if platform.DisclosureRequired && !input.DisclosureVisible {
		preview.BlockingErrors = append(preview.BlockingErrors, "required disclosure is not visible in the rendered post")
	}
	preview.ReleaseAllowed = len(preview.BlockingErrors) == 0
	return preview, nil
}

func truncateCaption(caption string, limit int) (string, bool) {
	if limit < 1 || len([]rune(caption)) <= limit {
		return caption, false
	}
	runes := []rune(strings.TrimSpace(caption))
	return string(runes[:limit]), true
}
