package synthesisruntime

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestNormalizeRole(t *testing.T) {
	if got := NormalizeRole("  "); got != DefaultRole {
		t.Fatalf("NormalizeRole(blank) = %q, want %q", got, DefaultRole)
	}
	if got := NormalizeRole("  search.custom  "); got != "search.custom" {
		t.Fatalf("NormalizeRole(custom) = %q", got)
	}
}

func TestRunUsesInjectedRunnerWithoutChangingArgumentsOrInput(t *testing.T) {
	wantArgs := []string{"gateway", "generate", "--role", "summarize.default", "--prompt-stdin"}
	var gotArgs []string
	var gotInput string
	wantOutput := []byte(`{"response":"ok"}`)
	got, err := Run(context.Background(), func(_ context.Context, args []string, input string) ([]byte, error) {
		gotArgs = append([]string(nil), args...)
		gotInput = input
		return wantOutput, nil
	}, wantArgs, "prompt text")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !reflect.DeepEqual(gotArgs, wantArgs) || gotInput != "prompt text" || !reflect.DeepEqual(got, wantOutput) {
		t.Fatalf("Run() args/input/output = %#v/%q/%q", gotArgs, gotInput, got)
	}
}

func TestRunReturnsInjectedRunnerError(t *testing.T) {
	want := errors.New("runner unavailable")
	_, err := Run(context.Background(), func(context.Context, []string, string) ([]byte, error) {
		return nil, want
	}, nil, "")
	if !errors.Is(err, want) {
		t.Fatalf("Run() error = %v, want wrapped identity %v", err, want)
	}
}

func TestFirstJSONObject(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{name: "no object", in: `plain text`, want: ""},
		{name: "nested object", in: `prefix {"items":[{"x":1}]} suffix`, want: `{"items":[{"x":1}]}`},
		{name: "braces and escaped quote in string", in: `{"text":"brace } and \\\" quote {"}`, want: `{"text":"brace } and \\\" quote {"}`},
		{name: "first of multiple objects", in: `{"first":true} {"second":true}`, want: `{"first":true}`},
		{name: "unbalanced first object", in: `{"unfinished":true {"later":true}`, want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := FirstJSONObject(tc.in); got != tc.want {
				t.Fatalf("FirstJSONObject(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
