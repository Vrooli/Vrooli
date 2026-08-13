package notes

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/notes"

	"github.com/vrooli/cli-core/cliapp"
)

// attachCall uploads opaque file bytes through the notes REST multipart
// exception. The response body is still proto-typed metadata; cli-core's
// DecodeUploadResponse handles the protojson decode so this operation stays
// transport-aware only at the boundary (UploadFile + DecodeUploadResponse), not
// the wire-format layer.
func (h *handlers) attachCall(ctx cliapp.OperationContext) (*notesv1.UploadAttachmentResponse, error) {
	id := ctx.Positional("id")
	path := ctx.Flag("file")
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open attachment file: %w", err)
	}
	defer file.Close()

	respBody, err := cliapp.UploadFile(h.core, "/notes/"+url.PathEscape(id)+"/attachments", nil, cliapp.UploadedFile{
		Name:   filepath.Base(path),
		Reader: file,
	})
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("attach file to note %q", id), err, nil)
	}
	resp, err := cliapp.DecodeUploadResponse[*notesv1.UploadAttachmentResponse](respBody)
	if err != nil {
		return nil, err
	}
	if resp.Attachment == nil {
		return nil, fmt.Errorf("server returned no attachment")
	}
	return resp, nil
}

// attachReport renders the upload metadata as the human mutation report.
func (h *handlers) attachReport(_ cliapp.OperationContext, resp *notesv1.UploadAttachmentResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Attached file to note %s.", resp.Attachment.NoteId)},
		Changes: []string{
			fmt.Sprintf("%s — %s (%d bytes)", resp.Attachment.Key, resp.Attachment.MimeType, resp.Attachment.SizeBytes),
		},
		NextCommand: []string{
			fmt.Sprintf("`notes get %s` — show this note", resp.Attachment.NoteId),
		},
	}
}

// attach uploads opaque file bytes through the notes REST multipart exception.
// The response body is still proto-typed metadata; cli-core's
// DecodeUploadResponse handles the protojson decode so this handler stays
// transport-aware only at the boundary (UploadFile + DecodeUploadResponse),
// not the wire-format layer.
func (h *handlers) attach(ctx cliapp.RunContext) error {
	return cliapp.Upload(h.attachCall, h.attachReport).Run(ctx)
}
