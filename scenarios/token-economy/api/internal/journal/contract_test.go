package journal_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/journal"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// [REQ:TKE-P0-004] [REQ:TKE-P0-010] [REQ:TKE-P0-011] The generated event
// contract carries the immutable projection input, correction reason, and
// actor verification receipt.
func TestGeneratedEventContract(t *testing.T) {
	file := journalv1.File_token_economy_v1_journal_journal_proto
	event := file.Messages().ByName("Event")
	require.NotNil(t, event)
	wantFields := []string{
		"id", "token_type_id", "holder_id", "amount", "kind", "cause_reference",
		"created_at", "actor_identity", "reason", "actor_kind",
		"actor_verification_status", "actor_run_id",
	}
	require.Equal(t, len(wantFields), event.Fields().Len())
	for _, name := range wantFields {
		require.NotNil(t, event.Fields().ByName(protoreflectName(name)), "missing Event.%s", name)
	}

	kind := file.Enums().ByName("EventKind")
	require.NotNil(t, kind)
	for _, name := range []string{
		"EVENT_KIND_MINT", "EVENT_KIND_CREDIT", "EVENT_KIND_DEBIT",
		"EVENT_KIND_REVERSAL", "EVENT_KIND_EXPIRY",
	} {
		require.NotNil(t, kind.Values().ByName(protoreflectName(name)), "missing EventKind.%s", name)
	}
	verification := file.Enums().ByName("VerificationStatus")
	require.NotNil(t, verification)
	for _, name := range []string{
		"VERIFICATION_STATUS_VERIFIED", "VERIFICATION_STATUS_UNAVAILABLE",
		"VERIFICATION_STATUS_INVALID", "VERIFICATION_STATUS_ABSENT",
	} {
		require.NotNil(t, verification.Values().ByName(protoreflectName(name)), "missing VerificationStatus.%s", name)
	}
}

func protoreflectName(value string) protoreflect.Name { return protoreflect.Name(value) }
