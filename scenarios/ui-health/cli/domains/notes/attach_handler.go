package notes

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/notes"

	"github.com/vrooli/cli-core/cliapp"
)

// attach uploads opaque file bytes through the notes REST multipart exception.
// The response body is still proto-typed metadata; cli-core's
// DecodeUploadResponse handles the protojson decode so this handler stays
// transport-aware only at the boundary (UploadFile + DecodeUploadResponse),
// not the wire-format layer.
func (h *handlers) attach(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	path := ctx.Flag("file")
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open attachment file: %w", err)
	}
	defer file.Close()

	respBody, err := cliapp.UploadFile(h.core, "/notes/"+url.PathEscape(id)+"/attachments", nil, cliapp.UploadedFile{
		Name:   filepath.Base(path),
		Reader: file,
	})
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("attach file to note %q", id), err, nil)
	}
	resp, err := cliapp.DecodeUploadResponse[*notesv1.UploadAttachmentResponse](respBody)
	if err != nil {
		return err
	}
	if resp.Attachment == nil {
		return fmt.Errorf("server returned no attachment")
	}

	return ctx.RenderMutation(cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Attached file to note %s.", resp.Attachment.NoteId)},
		Changes: []string{
			fmt.Sprintf("%s — %s (%d bytes)", resp.Attachment.Key, resp.Attachment.MimeType, resp.Attachment.SizeBytes),
		},
		NextCommand: []string{
			fmt.Sprintf("`notes get %s` — show this note", resp.Attachment.NoteId),
		},
	})
}
