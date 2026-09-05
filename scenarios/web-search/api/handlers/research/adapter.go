package research

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	researchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/research"

	internalresearch "web-search/internal/research"
)

// briefToProto projects an internal research Brief onto the wire shape.
func briefToProto(b internalresearch.Brief) *researchv1.Brief {
	out := &researchv1.Brief{
		Query:   b.Query,
		Level:   b.Level,
		Summary: b.Summary,
	}
	for _, c := range b.Citations {
		out.Citations = append(out.Citations, &researchv1.Citation{
			ResultIndex: int32(c.ResultIndex),
			Url:         c.URL,
			Title:       c.Title,
		})
	}
	return out
}

// rfc3339ToProto parses an RFC3339 timestamp string into a proto timestamp,
// returning nil for an empty/unparseable value.
func rfc3339ToProto(s string) *timestamppb.Timestamp {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return timestamppb.New(t)
}
