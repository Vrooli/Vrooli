package hub

// Severity values for a PacketWarning.
const (
	// WarningAdvice reports a packet that was accepted but could be better.
	WarningAdvice = "advice"
	// WarningDropped reports content that was removed from the packet so the
	// ask could still be accepted.
	WarningDropped = "dropped"
)

// PacketWarning is something the producer should know about the packet it
// sent. Warnings never fail an ask; they ride back on the AskResult.
type PacketWarning struct {
	// Code is a stable machine-readable identifier (for example
	// "attachment_blob_missing").
	Code string
	// Field names the request field the warning is about (for example
	// "attachments[2]").
	Field string
	// Message is a human-readable sentence naming the problem and the fix.
	Message string
	// Severity is WarningAdvice or WarningDropped.
	Severity string
	// CarriedOver is set on a revision's finding that the open ask already had,
	// word for word, about a part the revision did not touch.
	CarriedOver bool
}

// withoutWarning returns warnings without the ones with code.
func withoutWarning(warnings []PacketWarning, code string) []PacketWarning {
	kept := warnings[:0:0]
	for _, warning := range warnings {
		if warning.Code != code {
			kept = append(kept, warning)
		}
	}
	return kept
}
