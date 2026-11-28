package personas

import "context"

// Repository is the persistence seam for persona records.
type Repository interface {
	Create(context.Context, Persona) (Persona, error)
	Get(context.Context, string) (Persona, error)
	List(context.Context, bool, int) ([]Persona, error)
	Archive(context.Context, string) (Persona, error)
}
