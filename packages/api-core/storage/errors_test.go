package storage

import (
	"errors"
	"testing"
)

func TestErrorUnwrap(t *testing.T) {
	t.Parallel()

	base := errors.New("boom")
	err := &Error{Kind: ErrIO, Message: "io", Err: base}
	if !errors.Is(err, base) {
		t.Fatalf("expected wrapped error to match base")
	}
	if err.Unwrap() != base {
		t.Fatalf("unwrap mismatch")
	}

	var nilErr *Error
	if nilErr.Unwrap() != nil {
		t.Fatalf("nil unwrap should return nil")
	}
}

func TestErrorStringNilReceiver(t *testing.T) {
	t.Parallel()

	var err *Error
	if got := err.Error(); got != "<nil>" {
		t.Fatalf("nil Error() = %q, want <nil>", got)
	}
}

func TestPathsForClassAll(t *testing.T) {
	t.Parallel()

	p := Paths{
		ConfigDir: "/c",
		DataDir:   "/d",
		CacheDir:  "/k",
		LogsDir:   "/l",
		StateDir:  "/s",
	}

	cases := []struct {
		class Class
		want  string
	}{
		{ClassConfig, "/c"},
		{ClassData, "/d"},
		{ClassCache, "/k"},
		{ClassLogs, "/l"},
		{ClassState, "/s"},
	}
	for _, tc := range cases {
		got, err := p.ForClass(tc.class)
		if err != nil {
			t.Fatalf("ForClass(%s) error = %v", tc.class, err)
		}
		if got != tc.want {
			t.Fatalf("ForClass(%s) = %q, want %q", tc.class, got, tc.want)
		}
	}
}
