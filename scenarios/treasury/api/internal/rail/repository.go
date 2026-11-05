package rail

import "context"

type Registration struct {
	Name    string
	Enabled bool
}

type Repository interface {
	List(context.Context) ([]Registration, error)
}
