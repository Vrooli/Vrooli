package channelmanager

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// PreviewInput is metadata only. Media bytes stay with the asset owner; the
// release gate needs only dimensions and disclosure placement evidence.
type PreviewInput struct {
	PlatformID                string
	Caption                   string
	Title                     string
	PostType                  string
	FormatKind                string
	MediaWidth                int
	MediaHeight               int
	DisclosureVisible         bool
	DisclosureInVisibleRegion bool
	FirstComment              string
}

type PostPreview struct {
	Caption            string
	Title              string
	TitleTruncated     bool
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
	title, titleTruncated := truncateCaption(input.Title, platform.TitleLimit)
	preview := PostPreview{Caption: caption, Title: title, CaptionTruncated: truncated, TitleTruncated: titleTruncated, DisclosureRequired: platform.DisclosureRequired, ReleaseAllowed: true, FirstComment: input.FirstComment, MediaPresentation: "native"}
	if input.PostType != "" && !slices.ContainsFunc(platform.PostTypes, func(postType PostType) bool {
		return postType.ID == input.PostType && postType.FormatKind == input.FormatKind
	}) {
		preview.BlockingErrors = append(preview.BlockingErrors, "post type is not supported by this platform format")
	}
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
	} else if target, ok := parseAspectRatio(format.PreferredAspectRatio); ok && input.MediaWidth > 0 && input.MediaHeight > 0 {
		actual := float64(input.MediaWidth) / float64(input.MediaHeight)
		if actual != target && format.PreviewFit != "" {
			preview.MediaPresentation = format.PreviewFit
		}
	}
	if platform.DisclosureRequired && (!input.DisclosureVisible || !input.DisclosureInVisibleRegion && input.DisclosureVisible) {
		preview.BlockingErrors = append(preview.BlockingErrors, "required disclosure is not visible in the rendered post")
	}
	if input.FirstComment != "" && !platform.FirstCommentSupported {
		preview.BlockingErrors = append(preview.BlockingErrors, "first comments are not supported by this platform descriptor")
	}
	preview.ReleaseAllowed = len(preview.BlockingErrors) == 0
	return preview, nil
}

func parseAspectRatio(value string) (float64, bool) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, false
	}
	width, widthErr := strconv.ParseFloat(parts[0], 64)
	height, heightErr := strconv.ParseFloat(parts[1], 64)
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return 0, false
	}
	return width / height, true
}

func truncateCaption(caption string, limit int) (string, bool) {
	if limit < 1 || len([]rune(caption)) <= limit {
		return caption, false
	}
	runes := []rune(strings.TrimSpace(caption))
	return string(runes[:limit]), true
}
