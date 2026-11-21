package integrations

import (
	"log"
	"strings"
	"sync"

	"notification-hub/internal/hub"
)

// contextLinkDrops counts the context links the event path dropped, by producer.
var contextLinkDrops = struct {
	sync.Mutex
	bySource map[string]uint64
}{bySource: map[string]uint64{}}

// DroppedContextLinks is how many event context links were dropped for failing
// hub.ValidateContextURL since the Hub started, by producing scenario.
func DroppedContextLinks() map[string]uint64 {
	contextLinkDrops.Lock()
	defer contextLinkDrops.Unlock()
	out := make(map[string]uint64, len(contextLinkDrops.bySource))
	for source, n := range contextLinkDrops.bySource {
		out[source] = n
	}
	return out
}

// eventContextURL is the context link an event may carry. The operator taps it,
// so a link that fails hub.ValidateContextURL (javascript:, data:, //host,
// embedded credentials, ...) is not stored. The notification still is: a bad link
// must never lose a notice, so the link is dropped, counted against the producer
// and logged with the reason. The link itself is not logged; it can hold a token.
func eventContextURL(source, eventType, raw string) string {
	link := strings.TrimSpace(raw)
	err := hub.ValidateContextURL(link)
	if err == nil {
		return link
	}
	contextLinkDrops.Lock()
	contextLinkDrops.bySource[source]++
	contextLinkDrops.Unlock()
	log.Printf("notification-hub: dropped the context_url of a %s event from %s and kept the notification: %v", eventType, source, err)
	return ""
}
