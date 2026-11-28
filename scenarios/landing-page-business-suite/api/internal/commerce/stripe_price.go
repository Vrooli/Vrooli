package commerce

// StripePriceImport is the normalized Stripe price data consumed by catalog
// reconciliation. It is deliberately transport-neutral: Stripe HTTP decoding
// belongs to API composition, while plan policy consumes this stable contract.
type StripePriceImport struct {
	PriceID       string `json:"price_id"`
	LookupKey     string `json:"lookup_key,omitempty"`
	Currency      string `json:"currency"`
	AmountCents   int64  `json:"amount_cents"`
	Interval      string `json:"interval,omitempty"`
	ProductID     string `json:"product_id"`
	ProductName   string `json:"product_name"`
	Active        bool   `json:"active"`
	ExistsLocally bool   `json:"exists_locally"`
}
