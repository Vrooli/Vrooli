package agents

import "fmt"

type (
	Binding          struct{ AgentID, ChannelID, ThreadKey, Address string }
	AmbiguousBinding struct {
		Key   string
		Count int
	}
)

func (e AmbiguousBinding) Error() string {
	return fmt.Sprintf("ambiguous binding for %s (%d matches)", e.Key, e.Count)
}

func Resolve(bindings []Binding, channel, thread, address string) (Binding, error) {
	matches := make([]Binding, 0)
	for _, b := range bindings {
		if b.ChannelID == channel && b.ThreadKey == thread && b.Address == address {
			matches = append(matches, b)
		}
	}
	if len(matches) > 1 {
		return Binding{}, AmbiguousBinding{Key: channel + ":" + thread + ":" + address, Count: len(matches)}
	}
	if len(matches) == 0 {
		return Binding{}, fmt.Errorf("no binding for %s:%s:%s", channel, thread, address)
	}
	return matches[0], nil
}
