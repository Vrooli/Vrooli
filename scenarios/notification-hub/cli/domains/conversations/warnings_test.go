package conversations

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

func render(warnings ...*v1.PacketWarning) string {
	var out bytes.Buffer
	writePacketWarnings(&out, warnings, true)
	return out.String()
}

func TestWarningsAreSilentForACleanPacket(t *testing.T) {
	require.Empty(t, render())
}

func TestWarningsListDroppedPartsBeforeAdvice(t *testing.T) {
	got := render(
		&v1.PacketWarning{Code: "long_question", Field: "question", Message: "Shorten it.", Severity: "advice"},
		&v1.PacketWarning{Code: "attachment_blob_missing", Field: "attachments[1]", Message: "The Hub does not hold shot.png.", Severity: "dropped"},
		&v1.PacketWarning{Code: "missing_big_picture", Message: "Add a big picture.", Severity: "advice"},
	)
	require.Equal(t, `Packet check: 1 left out, 2 advice. The ask was sent; nothing here blocked it.
  Left out of the packet:
    - attachments[1]: The Hub does not hold shot.png. (attachment_blob_missing)
  Advice:
    - question: Shorten it. (long_question)
    - Add a big picture. (missing_big_picture)
`, got)
}

func TestWarningsTreatAnUnknownSeverityAsAdvice(t *testing.T) {
	got := render(&v1.PacketWarning{Code: "new_rule", Message: "Something new.", Severity: "future"})
	require.Contains(t, got, "Advice:")
	require.NotContains(t, got, "Left out")
}

func TestWarningsFoldWhatTheOpenAskAlreadyHadIntoOneLine(t *testing.T) {
	got := render(
		&v1.PacketWarning{Code: "long_question", Field: "question", Message: "Shorten it.", Severity: "advice"},
		&v1.PacketWarning{Code: "missing_big_picture", Field: "big_picture", Message: "Add a big picture.", Severity: "advice", CarriedOver: true},
		&v1.PacketWarning{Code: "unattached_path", Field: "technical_details", Message: "Attach it.", Severity: "advice", CarriedOver: true},
	)
	require.Equal(t, `Packet check: 1 advice. The ask was sent; nothing here blocked it.
  Advice:
    - question: Shorten it. (long_question)
  2 earlier warnings still apply (same as before; --json lists them).
`, got)

	require.Equal(t, `Packet check: no new findings. The ask was sent; nothing here blocked it.
  1 earlier warning still applies (same as before; --json lists them).
`, render(&v1.PacketWarning{Code: "missing_big_picture", Message: "Add a big picture.", Severity: "advice", CarriedOver: true}))
}
