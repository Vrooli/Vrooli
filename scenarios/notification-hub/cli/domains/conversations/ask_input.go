package conversations

// What an `ask` is made of before it is sent: the text from flags or a body
// file, the options, and the idempotency key derived from them.

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/protobuf/proto"

	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

// askText is the cover text of an ask.
type askText struct {
	Question         string
	BigPicture       string
	TechnicalDetails string
	// Ignored names sections of a body file the ask has no place for.
	Ignored []string
}

// parseBody reads the markdown body of an ask: the first paragraph is the
// question, "## Background" is the big picture and "## Details" the technical
// details. A leading "# Title" line is ignored, and text between the question
// and the first "##" heading joins the background. Any other "##" section is
// reported in Ignored rather than silently lost.
func parseBody(text string) (askText, error) {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	i := 0
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i < len(lines) && strings.HasPrefix(lines[i], "# ") {
		i++
	}
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	var question []string
	for ; i < len(lines) && strings.TrimSpace(lines[i]) != "" && !isSectionHeading(lines[i]); i++ {
		question = append(question, strings.TrimSpace(lines[i]))
	}
	if len(question) == 0 {
		return askText{}, fmt.Errorf("the body file has no question: its first paragraph is the question")
	}
	out := askText{Question: strings.Join(question, " ")}

	var preamble []string
	section := ""
	sections := map[string][]string{}
	for ; i < len(lines); i++ {
		if isSectionHeading(lines[i]) {
			section = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(lines[i], "## ")))
			if section != "background" && section != "details" {
				out.Ignored = append(out.Ignored, strings.TrimSpace(strings.TrimPrefix(lines[i], "## ")))
			}
			continue
		}
		if section == "" {
			preamble = append(preamble, lines[i])
		} else {
			sections[section] = append(sections[section], lines[i])
		}
	}
	background := strings.TrimSpace(strings.TrimSpace(strings.Join(preamble, "\n")) + "\n\n" + strings.TrimSpace(strings.Join(sections["background"], "\n")))
	out.BigPicture = background
	out.TechnicalDetails = strings.TrimSpace(strings.Join(sections["details"], "\n"))
	return out, nil
}

func isSectionHeading(line string) bool { return strings.HasPrefix(line, "## ") }

// readBody reads --body-file; "-" reads standard input.
func (s *service) readBody(path string) (askText, error) {
	var data []byte
	var err error
	if path == "-" {
		reader := s.stdin
		if reader == nil {
			reader = os.Stdin
		}
		data, err = io.ReadAll(reader)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return askText{}, fmt.Errorf("read --body-file: %w", err)
	}
	return parseBody(string(data))
}

// coverText joins the flags and the body file. The same part given twice is a
// mistake to report, not a choice to make silently.
func coverText(question, bigPicture, technicalDetails string, body *askText) (askText, error) {
	text := askText{Question: question, BigPicture: bigPicture, TechnicalDetails: technicalDetails}
	if body == nil {
		return text, nil
	}
	text.Ignored = body.Ignored
	for _, part := range []struct {
		name       string
		flag, file string
		dest       *string
	}{
		{"--question and the first paragraph", question, body.Question, &text.Question},
		{"--big-picture and ## Background", bigPicture, body.BigPicture, &text.BigPicture},
		{"--technical-details and ## Details", technicalDetails, body.TechnicalDetails, &text.TechnicalDetails},
	} {
		if part.flag != "" && part.file != "" {
			return askText{}, fmt.Errorf("%s of --body-file both give the same text; use one", part.name)
		}
		if part.file != "" {
			*part.dest = part.file
		}
	}
	return text, nil
}

// resolveOptions is the options of the ask: --yes-no, or the repeated --option values.
func resolveOptions(values []string, yesNo bool) ([]askOption, error) {
	if !yesNo {
		options, err := parseOptions(values)
		if err != nil {
			return nil, fmt.Errorf("%w; or use --yes-no for a yes/no question", err)
		}
		return options, nil
	}
	if len(values) > 0 {
		return nil, fmt.Errorf("--yes-no means the two options yes and no; drop --option, or drop --yes-no to list your own")
	}
	return []askOption{{Key: "yes", Label: "Yes"}, {Key: "no", Label: "No"}}, nil
}

// derivedKeyPrefix marks an idempotency key the CLI derived.
const derivedKeyPrefix = "ask-"

// deriveIdempotencyKey names one send of a request, so that a retry of it is
// recognised and returns the same ask.
//
// An ask without a subject key is named by what it says: the same terms always
// give the same key, so sending it again returns the same ask. The expiry takes
// part as the agent wrote it ("6h", or the RFC 3339 time), never as the absolute
// time it resolves to: a duration resolves to a different time on every send,
// which would make every resend look like a new ask, while leaving the expiry
// out would let a resend with a different expiry return the old ask unchanged.
//
// A keyed ask is named by the send instead: a fresh key each time. The subject
// already makes a repeat of identical terms change nothing, and a key made from
// the terms would be an earlier revision's key whenever an agent returns to
// earlier wording (A, then B, then A again) in any client that did not know the
// Hub treats only the latest request's key as a retry.
func deriveIdempotencyKey(req *v1.AskRequest, expires string) string {
	if req.GetSubjectKey() != "" {
		nonce := make([]byte, 10)
		if _, err := rand.Read(nonce); err == nil {
			return derivedKeyPrefix + hex.EncodeToString(nonce)
		}
		// Without randomness fall back to the terms: a repeat of earlier wording
		// is then a replay the Hub reads as a retry, which is worse than a fresh
		// key but never wrong.
	}
	terms := proto.Clone(req).(*v1.AskRequest)
	terms.IdempotencyKey = ""
	terms.Deadline = ""
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(terms)
	if err != nil {
		// A request that cannot be encoded cannot be sent either; the caller's
		// send reports it. Hash what can be seen so the key stays defined.
		encoded = []byte(terms.String())
	}
	sum := sha256.Sum256(append(encoded, []byte("\x00expires="+strings.TrimSpace(expires))...))
	return derivedKeyPrefix + hex.EncodeToString(sum[:])[:20]
}
