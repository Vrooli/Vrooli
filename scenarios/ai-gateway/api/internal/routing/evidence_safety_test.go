package routing

import "testing"

func TestAddAttachmentBytesSaturatesWithoutSignedOverflow(t *testing.T) {
	const maxInt64 = int64(1<<63 - 1)
	for _, tc := range []struct {
		name  string
		total int64
		bytes uint64
		want  int64
	}{
		{name: "ordinary", total: 12, bytes: 30, want: 42},
		{name: "exact limit", total: maxInt64 - 1, bytes: 1, want: maxInt64},
		{name: "sum overflow", total: maxInt64 - 1, bytes: 2, want: maxInt64},
		{name: "wire value exceeds signed range", total: 0, bytes: uint64(maxInt64) + 1, want: maxInt64},
		{name: "negative current value", total: -1, bytes: 1, want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := addAttachmentBytes(tc.total, tc.bytes); got != tc.want {
				t.Fatalf("addAttachmentBytes(%d, %d) = %d, want %d", tc.total, tc.bytes, got, tc.want)
			}
		})
	}
}
