package commitments

import "context"

type Repository interface {
	List(context.Context) ([]Commitment, error)
	Create(context.Context, Commitment) (Commitment, error)
	UpdateState(context.Context, string, string, int64) (Commitment, error)
	Revise(context.Context, ReviseInput) (Commitment, Revision, error)
}
