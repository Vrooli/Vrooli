// Package topics exposes the topics transport boundary.
package topics

import domain "prompt-manager/internal/topics"

type (
	MatchedTopic   = domain.MatchedTopic
	TopicMatchFunc = domain.TopicMatchFunc
)

var NewHandlers = domain.NewHandlers
