package threads

import "switchboard/internal/channels"

func ShouldRespond(e channels.Envelope, group bool, addressed bool) bool {
	if e.AuthorKind == channels.AuthorAgent {
		return false
	}
	return !group || addressed
}
