package sketch

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"react-component-library/internal/designinference"
	internal "react-component-library/internal/sketch"
)

// TestConnectErrorClassification pins the error taxonomy the studio's retry
// policy depends on: client mistakes stay client errors, while an unexpected
// disk/permission/parse fault must not masquerade as InvalidArgument.
func TestConnectErrorClassification(t *testing.T) {
	handler := NewConnectHandler(Deps{})

	cases := []struct {
		name string
		err  error
		want connect.Code
	}{
		{
			name: "disk permission failure is a server fault",
			err:  fmt.Errorf("read sketch: %w", os.ErrPermission),
			want: connect.CodeInternal,
		},
		{
			name: "parse failure is a server fault",
			err:  errors.New("decode sketch: unexpected end of JSON input"),
			want: connect.CodeInternal,
		},
		{
			name: "explicit validation keeps invalid argument",
			err:  connect.NewError(connect.CodeInvalidArgument, errors.New("region is required")),
			want: connect.CodeInvalidArgument,
		},
		{
			name: "unavailable dependency keeps unavailable",
			err:  connect.NewError(connect.CodeUnavailable, errors.New("catalog unavailable")),
			want: connect.CodeUnavailable,
		},
		{
			name: "missing sketch is not found",
			err:  fmt.Errorf("read sketch: %w", sql.ErrNoRows),
			want: connect.CodeNotFound,
		},
		{
			name: "missing file is not found",
			err:  fmt.Errorf("read design source: %w", os.ErrNotExist),
			want: connect.CodeNotFound,
		},
		{
			name: "locked region is failed precondition",
			err:  &internal.LockedRegionError{Region: "main"},
			want: connect.CodeFailedPrecondition,
		},
		{
			name: "content conflict is aborted",
			err:  &internal.ConflictError{Expected: "a", Current: "b"},
			want: connect.CodeAborted,
		},
		{
			name: "missing expected revision is failed precondition",
			err:  internal.ErrExpectedRevision,
			want: connect.CodeFailedPrecondition,
		},
		{
			name: "duplicate inference is already exists",
			err:  designinference.ErrConflict,
			want: connect.CodeAlreadyExists,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := handler.connectError("Test", tc.err)
			require.Equal(t, tc.want, connect.CodeOf(err), "unexpected code for %q", tc.err)
		})
	}
}
