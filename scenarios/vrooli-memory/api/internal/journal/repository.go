package journal

import "context"

// Repository deliberately exposes append and reads only. The journal has no
// update or delete operation because entries are permanent evidence.
type Repository interface {
	Append(context.Context, Entry, []string) (Entry, error)
	Get(context.Context, string) (Entry, error)
	List(context.Context, int) ([]Entry, error)
	FindByImportKey(context.Context, string) (Entry, bool, error)
}
