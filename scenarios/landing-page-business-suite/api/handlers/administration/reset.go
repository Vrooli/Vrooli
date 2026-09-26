package administration

import (
	"context"
	"time"
)

type ResetDependencies struct {
	Reset    func(context.Context) error
	Now      func() time.Time
	LogError func(string, map[string]any)
}
