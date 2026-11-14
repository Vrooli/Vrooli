// Package render holds what every CLI domain shares: one RPC call shape and
// one way to print a derived value with its source.
package render

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/types/known/timestamppb"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/shared"
)

// Call runs one RPC and turns transport errors into the CLI's API error. A
// refusal's command-line next step (a CommandHint detail; the message names
// none) follows the message.
func Call[Req, Resp any](what string, fn func(context.Context, *connect.Request[Req]) (*connect.Response[Resp], error), req *Req) (*Resp, error) {
	resp, err := fn(context.Background(), connect.NewRequest(req))
	if err != nil {
		wrapped := cliapp.WrapAPIError(what, err, nil)
		if hint := Hint(err); hint != "" {
			return nil, fmt.Errorf("%w (next: %s)", wrapped, hint)
		}
		return nil, wrapped
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("%s: the server returned no response", what)
	}
	return resp.Msg, nil
}

// Hint is the command-line next step a desk refusal carries, or "".
func Hint(err error) string {
	var cerr *connect.Error
	if !errors.As(err, &cerr) {
		return ""
	}
	for _, d := range cerr.Details() {
		if v, derr := d.Value(); derr == nil {
			if h, ok := v.(*sharedv1.CommandHint); ok {
				return h.GetCommand()
			}
		}
	}
	return ""
}

// When prints a time in local time, or "-".
func When(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return "-"
	}
	return ts.AsTime().Local().Format("2006-01-02 15:04")
}

// Evidence renders "<source>" for a known value and "unknown: <reason>" otherwise.
func Evidence(e *sharedv1.Evidence) string {
	switch {
	case e == nil:
		return "unknown"
	case !e.Known:
		return "unknown: " + e.Reason
	case e.Reason != "":
		return e.Source + " (" + e.Reason + ")"
	default:
		return e.Source
	}
}

// Unreadable renders files and folders that exist but could not be read:
// what they hold is unknown, not empty.
func Unreadable(list []*sharedv1.Evidence) []string {
	out := make([]string, 0, len(list))
	for _, e := range list {
		out = append(out, "unreadable, so unknown: "+e.GetReason())
	}
	return out
}

// Time renders a derived time with its source.
func Time(t *sharedv1.DerivedTime) string {
	if t == nil || t.Evidence == nil || !t.Evidence.Known {
		return Evidence(t.GetEvidence())
	}
	return When(t.Value) + "  [" + t.Evidence.Source + "]"
}

// Text renders a derived text with its source.
func Text(t *sharedv1.DerivedText) string {
	if t == nil || t.Evidence == nil || !t.Evidence.Known {
		return Evidence(t.GetEvidence())
	}
	return t.Value + "  [" + t.Evidence.Source + "]"
}

// Flag renders a derived yes/no with its source.
func Flag(f *sharedv1.DerivedFlag, yes, no string) string {
	if f == nil || f.Evidence == nil || !f.Evidence.Known {
		return Evidence(f.GetEvidence())
	}
	word := no
	if f.Value {
		word = yes
	}
	return word + "  [" + f.Evidence.Source + "]"
}

// Entry renders one operator-file entry heading and its first line.
func Entry(e *sharedv1.OperatorEntry) string {
	if e == nil {
		return "-"
	}
	subject := e.Kind
	if e.Subject != "" {
		subject += " " + e.Subject
	}
	return fmt.Sprintf("%s · %s · via %s", When(e.At), subject, e.Via)
}
