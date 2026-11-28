package codingjobs

import (
	"testing"
	"time"

	"client-rehearsal/internal/codingjobs"
	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/codingjobs"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestToProtoPreservesOptimisticRevision(t *testing.T) {
	item := codingjobs.Job{ID: "job-1", Revision: 4, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	wire := toProto(item)
	require.Equal(t, uint64(4), wire.GetRevision())
	encoded, err := protojson.Marshal(wire)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"revision":"4"`)
	var decoded v1.CodingJob
	require.NoError(t, protojson.Unmarshal(encoded, &decoded))
	require.Equal(t, uint64(4), decoded.GetRevision())
}
