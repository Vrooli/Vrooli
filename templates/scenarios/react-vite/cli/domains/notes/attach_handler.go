package notes

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/notes"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vrooli/cli-core/cliapp"
)

// attach uploads opaque file bytes through the notes REST multipart exception.
// The response body is still proto-typed metadata, so the CLI decodes it with
// protojson after cli-core's UploadFile helper returns the raw response body.
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
	var resp notesv1.UploadAttachmentResponse
	if err := protojson.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("decode attachment response: %w", err)
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
