package journal

import "context"

type Repository interface {
	Append(context.Context, Entry) (Entry, error)
	List(context.Context, string, int) ([]Entry, error)
}
