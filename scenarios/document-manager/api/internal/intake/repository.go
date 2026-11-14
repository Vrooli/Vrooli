package intake

import "fmt"

type ErrNotFound struct{ Key string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("document %q not found", e.Key) }
