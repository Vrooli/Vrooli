package componentrevisions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const maxCandidateBytes = 1 << 20

// Spec supplies the typed Connect operations for one immutable candidate family.
// Shared parsing, bounds, paging, and reports stay identical across domains.
type Spec struct {
	Group, Service, CandidateName, SaveMethod, GetMethod, HeadMethod, RevisionsMethod string
	CandidatesMethod                                                                  string
	Save                                                                              func(context.Context, proto.Message, uint32) (proto.Message, error)
	Get                                                                               func(context.Context, string, uint32) (proto.Message, error)
	Head                                                                              func(context.Context, string) (proto.Message, error)
	Revisions                                                                         func(context.Context, string, uint32, uint32) (proto.Message, error)
	Candidates                                                                        func(context.Context, string, uint32) (proto.Message, error)
	CandidateRows                                                                     func(proto.Message) (rows []proto.Message, next string)
	RevisionRows                                                                      func(proto.Message) (rows []proto.Message, next uint32)
	DescribeRevision                                                                  func(proto.Message) string
	Candidate                                                                         func() proto.Message
	RevisionFromGet                                                                   func(proto.Message) proto.Message
	RevisionFromHead                                                                  func(proto.Message) proto.Message
	RevisionFromSave                                                                  func(proto.Message) proto.Message
	RevisionID                                                                        func(proto.Message) string
	RevisionNumber                                                                    func(proto.Message) uint32
	AdditionalHandlers                                                                map[string]cliapp.PrimitiveHandler
}

func Register(core *cliapp.ScenarioApp, manifest []byte, s Spec) (cliapp.SubcommandGroup, error) {
	if s.Candidate == nil || s.Save == nil || s.Get == nil || s.Head == nil || s.Revisions == nil || s.RevisionRows == nil || s.RevisionFromSave == nil || s.RevisionFromGet == nil || s.RevisionFromHead == nil || s.RevisionID == nil || s.RevisionNumber == nil || s.DescribeRevision == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: incomplete candidate CLI specification", s.Group)
	}
	bindings := map[string]cliapp.PrimitiveHandler{
		s.Service + "." + s.SaveMethod: cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (proto.Message, error) {
			return saveCall(ctx, s)
		}, func(_ cliapp.OperationContext, response proto.Message) cliapp.MutationReport {
			r := s.RevisionFromSave(response)
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("Saved candidate revision: %s", s.DescribeRevision(r))}, Changes: []string{s.DescribeRevision(r)}, NextCommand: []string{fmt.Sprintf("`%s get %s --revision %d` — inspect this immutable candidate revision", s.Group, s.RevisionID(r), s.RevisionNumber(r))}}
		}),
		s.Service + "." + s.GetMethod: cliapp.ProtoList(func(ctx cliapp.OperationContext) (proto.Message, error) {
			return getCall(ctx, s)
		}, func(_ cliapp.OperationContext, response proto.Message) cliapp.ListReport {
			r := s.RevisionFromGet(response)
			return cliapp.ListReport{Summary: []string{"Loaded immutable candidate revision."}, ResultsHeading: s.CandidateName + " revision", Results: []string{s.DescribeRevision(r)}}
		}),
		s.Service + "." + s.HeadMethod: cliapp.ProtoList(func(ctx cliapp.OperationContext) (proto.Message, error) {
			return headCall(ctx, s)
		}, func(_ cliapp.OperationContext, response proto.Message) cliapp.ListReport {
			r := s.RevisionFromHead(response)
			return cliapp.ListReport{Summary: []string{"Current candidate head; qualification is separate."}, ResultsHeading: s.CandidateName + " head", Results: []string{s.DescribeRevision(r)}}
		}),
		s.Service + "." + s.RevisionsMethod: cliapp.ProtoList(func(ctx cliapp.OperationContext) (proto.Message, error) {
			return revisionsCall(ctx, s)
		}, func(_ cliapp.OperationContext, response proto.Message) cliapp.ListReport {
			rows, next := s.RevisionRows(response)
			formatted := make([]string, 0, len(rows))
			for _, row := range rows {
				formatted = append(formatted, s.DescribeRevision(row))
			}
			hints := []string{}
			if next > 0 {
				hints = append(hints, fmt.Sprintf("`%s revisions %s --after-revision %d` — next page", s.Group, rowsID(s, rows), next))
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d immutable revision(s).", len(rows))}, ResultsHeading: s.CandidateName + " revisions", Results: formatted, RetrievalHints: hints}
		}),
	}
	if s.CandidatesMethod != "" {
		if s.Candidates == nil || s.CandidateRows == nil {
			return cliapp.SubcommandGroup{}, fmt.Errorf("%s: incomplete candidate list CLI specification", s.Group)
		}
		bindings[s.Service+"."+s.CandidatesMethod] = cliapp.ProtoList(func(ctx cliapp.OperationContext) (proto.Message, error) {
			return candidatesCall(ctx, s)
		}, func(_ cliapp.OperationContext, response proto.Message) cliapp.ListReport {
			rows, next := s.CandidateRows(response)
			formatted := make([]string, 0, len(rows))
			for _, row := range rows {
				formatted = append(formatted, s.DescribeRevision(row))
			}
			hints := []string{}
			if next != "" {
				hints = append(hints, fmt.Sprintf("`%s list --after-id %s` — next page", s.Group, next))
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d current candidate head(s). Qualification is separate.", len(rows))}, ResultsHeading: s.CandidateName + " candidates", Results: formatted, RetrievalHints: hints}
		})
	}
	for key, handler := range s.AdditionalHandlers {
		if strings.TrimSpace(key) == "" {
			return cliapp.SubcommandGroup{}, fmt.Errorf("%s: additional handler key is required", s.Group)
		}
		if _, exists := bindings[key]; exists {
			return cliapp.SubcommandGroup{}, fmt.Errorf("%s: duplicate additional handler %q", s.Group, key)
		}
		bindings[key] = handler
	}
	g, err := cliapp.LoadFromManifestPrimitives(manifest, s.Group, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: load manifest: %w", s.Group, err)
	}
	return g, nil
}

func readCandidate(path string, newMessage func() proto.Message) (proto.Message, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("candidate is required; pass a strict proto-JSON file path or - for stdin")
	}
	var reader io.Reader
	var file *os.File
	if path == "-" {
		reader = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open candidate: %w", err)
		}
		file, reader = f, f
		defer file.Close()
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxCandidateBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read candidate: %w", err)
	}
	if len(data) > maxCandidateBytes {
		return nil, fmt.Errorf("candidate exceeds %d-byte input limit", maxCandidateBytes)
	}
	msg := newMessage()
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, msg); err != nil {
		return nil, fmt.Errorf("parse strict candidate proto JSON: %w", err)
	}
	return msg, nil
}

func saveCall(ctx cliapp.OperationContext, s Spec) (proto.Message, error) {
	expected, err := parseUint32(ctx.Flag("expected-head"), "expected-head", true)
	if err != nil {
		return nil, err
	}
	candidate, err := readCandidate(ctx.Flag("candidate"), s.Candidate)
	if err != nil {
		return nil, err
	}
	response, err := s.Save(context.Background(), candidate, expected)
	if err != nil {
		return nil, err
	}
	if !hasRevision(response, s.RevisionFromSave) {
		return nil, fmt.Errorf("server returned no saved candidate revision")
	}
	return response, nil
}
func getCall(ctx cliapp.OperationContext, s Spec) (proto.Message, error) {
	rev, e := parseUint32(ctx.Flag("revision"), "revision", false)
	if e != nil {
		return nil, e
	}
	response, err := s.Get(context.Background(), ctx.Positional("id"), rev)
	if err != nil {
		return nil, err
	}
	if !hasRevision(response, s.RevisionFromGet) {
		return nil, fmt.Errorf("server returned no immutable candidate revision")
	}
	return response, nil
}
func headCall(ctx cliapp.OperationContext, s Spec) (proto.Message, error) {
	response, err := s.Head(context.Background(), ctx.Positional("id"))
	if err != nil {
		return nil, err
	}
	if !hasRevision(response, s.RevisionFromHead) {
		return nil, fmt.Errorf("server returned no candidate head")
	}
	return response, nil
}

func hasRevision(response proto.Message, project func(proto.Message) proto.Message) bool {
	if response == nil || !response.ProtoReflect().IsValid() {
		return false
	}
	revision := project(response)
	return revision != nil && revision.ProtoReflect().IsValid()
}
func revisionsCall(ctx cliapp.OperationContext, s Spec) (proto.Message, error) {
	after, e := parseUint32Default(ctx.Flag("after-revision"), "after-revision", 0)
	if e != nil {
		return nil, e
	}
	size, e := boundedPageSize(ctx.Flag("page-size"))
	if e != nil {
		return nil, e
	}
	response, err := s.Revisions(context.Background(), ctx.Positional("id"), after, size)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("server returned no candidate revision page")
	}
	return response, nil
}

func candidatesCall(ctx cliapp.OperationContext, s Spec) (proto.Message, error) {
	after := ctx.Flag("after-id")
	if after != "" && !validStableID(after) {
		return nil, fmt.Errorf("after-id must be a stable identifier of at most 128 characters")
	}
	page, err := boundedPageSize(ctx.Flag("page-size"))
	if err != nil {
		return nil, err
	}
	response, err := s.Candidates(context.Background(), after, page)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("server returned no candidate page")
	}
	return response, nil
}

func validStableID(value string) bool {
	if len(value) == 0 || len(value) > 128 || !asciiAlphaNumeric(value[0]) {
		return false
	}
	for i := 1; i < len(value); i++ {
		ch := value[i]
		if !asciiAlphaNumeric(ch) && ch != '.' && ch != '_' && ch != '-' {
			return false
		}
	}
	return true
}

func asciiAlphaNumeric(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9'
}

func boundedPageSize(raw string) (uint32, error) {
	size, err := parseUint32Default(raw, "page-size", 50)
	if err != nil {
		return 0, err
	}
	if size < 1 || size > 100 {
		return 0, fmt.Errorf("page-size must be between 1 and 100")
	}
	return size, nil
}

func APIError(action, group string, err error) error {
	var ce *connect.Error
	if errors.As(err, &ce) && ce.Code() == connect.CodeAborted {
		return fmt.Errorf("%s: candidate head changed; run `%s head <id>`, review the current candidate, then retry with its revision as --expected-head", action, group)
	}
	return cliapp.WrapAPIError(action, err, nil)
}
func parseUint32(raw, name string, allowZero bool) (uint32, error) {
	n, e := strconv.ParseUint(strings.TrimSpace(raw), 10, 32)
	if e != nil || (!allowZero && n == 0) {
		if allowZero {
			return 0, fmt.Errorf("%s must be a non-negative 32-bit integer", name)
		}
		return 0, fmt.Errorf("%s must be a positive 32-bit integer", name)
	}
	return uint32(n), nil
}
func parseUint32Default(raw, name string, def uint32) (uint32, error) {
	if strings.TrimSpace(raw) == "" {
		return def, nil
	}
	return parseUint32(raw, name, true)
}

func rowsID(s Spec, rows []proto.Message) string {
	if len(rows) == 0 {
		return "<id>"
	}
	return s.RevisionID(rows[0])
}
