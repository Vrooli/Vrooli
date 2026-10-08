package redemption_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
)

// [REQ:TKE-P0-013] Every catalog declaration carries an explicit immediate or
// requires-approval posture; unspecified cannot be treated as a valid policy.
func TestCatalogApprovalPostureIsExplicitInRedemptionContract(t *testing.T) {
	message := accessv1.File_token_economy_v1_access_access_proto.Messages().ByName("CatalogEntry")
	require.NotNil(t, message)
	field := message.Fields().ByName("approval_posture")
	require.NotNil(t, field)
	values := field.Enum().Values()
	require.Equal(t, 3, values.Len())
	require.Equal(t, "APPROVAL_POSTURE_UNSPECIFIED", string(values.Get(0).Name()))
	require.Equal(t, "APPROVAL_POSTURE_IMMEDIATE", string(values.Get(1).Name()))
	require.Equal(t, "APPROVAL_POSTURE_REQUIRES_APPROVAL", string(values.Get(2).Name()))
}
