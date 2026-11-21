package hub

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
)

// A section_review question can follow a Markdown file of the packet
// (AskQuestion.SectionsFrom): its sections are the file's ## and ### headings.
// The Hub reads them, and reads them again on every revision, so replacing the
// file changes the review with it. A form whose sections were frozen at the
// first send would ask the operator to review headings the document no longer
// has, and their verdicts would answer a plan nobody wrote.

// maxSectionSourceBytes is the largest Markdown file the Hub reads headings from.
const maxSectionSourceBytes = 1 << 20

// BlobReader reads the stored bytes of one blob, at most max of them.
type BlobReader func(ctx context.Context, sha256 string, max int64) ([]byte, error)

// followSections gives every section_review question that follows a file the
// sections of that file as the packet holds it now. A question that cannot follow
// its file (the file is not attached, has no headings, or cannot be read and the
// question has no earlier sections to fall back on) is left out and reported;
// never the whole ask.
func followSections(ctx context.Context, questions []AskQuestion, attachments []AskAttachment, read BlobReader) ([]AskQuestion, []PacketWarning) {
	var warnings []PacketWarning
	out := make([]AskQuestion, 0, len(questions))
	for index, question := range questions {
		reference := strings.TrimSpace(question.SectionsFrom)
		if question.Type != QuestionSectionReview || reference == "" {
			out = append(out, question)
			continue
		}
		field := fmt.Sprintf("questions[%d]", index)
		sections, notes, err := sectionsOf(ctx, reference, attachments, read)
		switch {
		case err == nil:
			question.SectionsFrom, question.Sections = reference, sections
			warnings = append(warnings, withField(notes, field)...)
		case errors.Is(err, errSectionSourceUnreadable) && len(question.Sections) > 0:
			question.SectionsFrom = reference
			warnings = append(warnings, PacketWarning{
				Code: WarnQuestionInvalid, Field: field, Severity: WarningAdvice,
				Message: fmt.Sprintf("question %q: %s; its earlier sections are kept", question.ID, err.Error()),
			})
		default:
			warnings = append(warnings, droppedQuestion(field, fmt.Sprintf("question %q: %s; the question was left out", question.ID, err.Error())))
			continue
		}
		out = append(out, question)
	}
	return out, warnings
}

func withField(warnings []PacketWarning, field string) []PacketWarning {
	for index := range warnings {
		warnings[index].Field = field
	}
	return warnings
}

var errSectionSourceUnreadable = errors.New("unreadable")

// sectionsOf reads the headings of the file reference names: an attachment name,
// or "name/path" for a file inside a folder attachment. The sections point back
// at the reference, so the operator opens the file at the heading.
func sectionsOf(ctx context.Context, reference string, attachments []AskAttachment, read BlobReader) ([]AskQuestionSection, []PacketWarning, error) {
	name, entryPath, _ := strings.Cut(reference, "/")
	var target *AskAttachment
	for index := range attachments {
		if attachments[index].Name == name {
			target = &attachments[index]
			break
		}
	}
	if target == nil {
		return nil, nil, fmt.Errorf("sections_from %q is not attached to this ask (attach the file, or fix the name)", reference)
	}
	var sha string
	var size int64
	switch {
	case target.Kind == AttachmentFile && entryPath != "":
		return nil, nil, fmt.Errorf("sections_from %q: %q is a file, so its name alone says where to read", reference, name)
	case target.Kind == AttachmentFile:
		sha, size = target.SHA256, target.SizeBytes
	case entryPath == "":
		return nil, nil, fmt.Errorf("sections_from %q is a folder; name the file inside it as %s/PATH", reference, name)
	default:
		for _, entry := range target.Entries {
			if entry.Path == entryPath {
				sha, size = entry.SHA256, entry.SizeBytes
				break
			}
		}
		if sha == "" {
			return nil, nil, fmt.Errorf("sections_from %q: the folder %q has no file %q", reference, name, entryPath)
		}
	}
	if size > maxSectionSourceBytes {
		return nil, nil, fmt.Errorf("sections_from %q is %d bytes, over the %d MiB the Hub reads headings from", reference, size, maxSectionSourceBytes>>20)
	}
	if read == nil {
		return nil, nil, fmt.Errorf("sections_from %q cannot be read here: %w", reference, errSectionSourceUnreadable)
	}
	data, err := read(ctx, sha, maxSectionSourceBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("sections_from %q cannot be read (%v): %w", reference, err, errSectionSourceUnreadable)
	}
	sections, total, err := markdownSections(bytes.NewReader(data), reference)
	if err != nil {
		return nil, nil, fmt.Errorf("sections_from %q has no ## or ### headings to review", reference)
	}
	var notes []PacketWarning
	if total > len(sections) {
		notes = append(notes, PacketWarning{
			Code: WarnQuestionInvalid, Severity: WarningAdvice,
			Message: fmt.Sprintf("%s has %d headings; the review shows the first %d", reference, total, len(sections)),
		})
	}
	return sections, notes, nil
}
