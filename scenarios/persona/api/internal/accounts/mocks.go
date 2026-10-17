package accounts

import "context"

type FakeRepository struct {
	LinkFunc             func(context.Context, AccountLink) (AccountLink, error)
	ListAccountsFunc     func(context.Context, string) ([]AccountLink, error)
	AddAddressFunc       func(context.Context, Address) (Address, error)
	ListAddressesFunc    func(context.Context, string) ([]Address, error)
	GetAddressFunc       func(context.Context, string, string) (Address, error)
	AddObligationFunc    func(context.Context, Obligation) (Obligation, error)
	ListObligationsFunc  func(context.Context, string) ([]Obligation, error)
	GetObligationFunc    func(context.Context, string) (Obligation, error)
	CancelObligationFunc func(context.Context, string) (Obligation, error)
}

var _ Repository = FakeRepository{}

func (f FakeRepository) Link(ctx context.Context, a AccountLink) (AccountLink, error) {
	return f.LinkFunc(ctx, a)
}

func (f FakeRepository) ListAccounts(ctx context.Context, id string) ([]AccountLink, error) {
	return f.ListAccountsFunc(ctx, id)
}

func (f FakeRepository) AddAddress(ctx context.Context, a Address) (Address, error) {
	return f.AddAddressFunc(ctx, a)
}

func (f FakeRepository) ListAddresses(ctx context.Context, id string) ([]Address, error) {
	return f.ListAddressesFunc(ctx, id)
}

func (f FakeRepository) GetAddress(ctx context.Context, personaID, id string) (Address, error) {
	return f.GetAddressFunc(ctx, personaID, id)
}

func (f FakeRepository) AddObligation(ctx context.Context, o Obligation) (Obligation, error) {
	return f.AddObligationFunc(ctx, o)
}

func (f FakeRepository) ListObligations(ctx context.Context, id string) ([]Obligation, error) {
	return f.ListObligationsFunc(ctx, id)
}

func (f FakeRepository) GetObligation(ctx context.Context, id string) (Obligation, error) {
	return f.GetObligationFunc(ctx, id)
}

func (f FakeRepository) CancelObligation(ctx context.Context, id string) (Obligation, error) {
	return f.CancelObligationFunc(ctx, id)
}
