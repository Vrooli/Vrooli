package accounts

import "context"

type Repository interface {
	Link(context.Context, AccountLink) (AccountLink, error)
	ListAccounts(context.Context, string) ([]AccountLink, error)
	AddAddress(context.Context, Address) (Address, error)
	ListAddresses(context.Context, string) ([]Address, error)
	GetAddress(context.Context, string, string) (Address, error)
	AddObligation(context.Context, Obligation) (Obligation, error)
	ListObligations(context.Context, string) ([]Obligation, error)
	GetObligation(context.Context, string) (Obligation, error)
	CancelObligation(context.Context, string) (Obligation, error)
}
