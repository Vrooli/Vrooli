package channelmanager

import "testing"

// [REQ:CHANMGR-P1-009] Preview rules come from the selected descriptor and a
// disclosure cropped out of the rendered post blocks release.
func TestPreviewReleaseBlocksInvisibleDisclosureAndInvalidMedia(t *testing.T) {
	s, err := New([]Platform{{ID: "short-video", DailyCeiling: 1, CaptionLimit: 4, ActionKinds: []string{"publish"}, DisclosureRequired: true, Formats: []Format{{Kind: "video", MIMETypes: []string{"video/mp4"}, MaxBytes: 1, DurationResolved: true, MinWidth: 360, MinHeight: 360, MaxWidth: 1080, MaxHeight: 1920}}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := s.PreviewRelease(PreviewInput{PlatformID: "short-video", Caption: "hello", FormatKind: "video", MediaWidth: 200, MediaHeight: 200, DisclosureVisible: false})
	if err != nil || preview.ReleaseAllowed || len(preview.BlockingErrors) != 2 || preview.Caption != "hell" || !preview.CaptionTruncated {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
}

func TestPreviewReleaseModelsDescriptorPresentationAndVisibleDisclosure(t *testing.T) {
	s, err := New([]Platform{{ID: "video", DailyCeiling: 1, CaptionLimit: 12, TitleLimit: 4, ActionKinds: []string{"publish"}, PostTypes: []PostType{{ID: "video", FormatKind: "video", TitleRequired: true}}, DisclosureRequired: true, FirstCommentSupported: true, Formats: []Format{{Kind: "video", MIMETypes: []string{"video/mp4"}, MaxBytes: 1, DurationResolved: true, MinWidth: 360, MinHeight: 360, MaxWidth: 1920, MaxHeight: 1920, PreferredAspectRatio: "9:16", PreviewFit: "letterbox"}}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := s.PreviewRelease(PreviewInput{PlatformID: "video", PostType: "video", Title: "title", Caption: "caption", FormatKind: "video", MediaWidth: 1920, MediaHeight: 1080, DisclosureVisible: true, DisclosureInVisibleRegion: false})
	if err != nil || preview.ReleaseAllowed || preview.MediaPresentation != "letterbox" || !preview.TitleTruncated || len(preview.BlockingErrors) != 1 {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
}
