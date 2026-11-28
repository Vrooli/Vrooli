package hub

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
)

// InlineExplainerName is the attachment name of a page sent inline by an event
// producer. A re-send under the same subject key replaces it by this name.
const InlineExplainerName = "decision.html"

// storeInlineExplainer turns an inline page into the ask's explainer
// attachment: the bytes go to the blob store like an uploaded file, and the
// rest of the packet path (limits, the explainer rules, revisions) treats it
// as any other explainer. A page the store refuses is left out with a warning;
// it never fails the ask.
func (s *Service) storeInlineExplainer(ctx context.Context, spec AskSpec, check bool) (AskSpec, []PacketWarning) {
	page := spec.ExplainerHTML
	spec.ExplainerHTML = ""
	if page == "" {
		return spec, nil
	}
	sum := sha256.Sum256([]byte(page))
	sha := hex.EncodeToString(sum[:])
	if !check {
		if _, err := s.PutAttachmentBlob(ctx, sha, "text/html", []byte(page)); err != nil {
			return spec, []PacketWarning{attachmentWarning(WarnAttachmentBlobMissing, "explainer_html",
				"the Hub could not store the decision's page ("+err.Error()+"), so it was left out; the ask was sent without it.", WarningDropped)}
		}
	}
	spec.Attachments = append([]AskAttachment{{
		Name:      InlineExplainerName,
		Kind:      AttachmentFile,
		Role:      AttachmentRoleExplainer,
		SHA256:    sha,
		SizeBytes: int64(len(page)),
		MediaType: "text/html",
	}}, spec.Attachments...)
	return spec, nil
}
