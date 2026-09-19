// Package money stores monetary values as integer minor units with currency metadata.
package money

import (
	"errors"
	"fmt"
)

type Money struct {
	Minor    int64
	Currency string
	Exponent int
	Known    bool
}

func Unknown(currency string, exponent int) Money {
	return Money{Currency: currency, Exponent: exponent}
}

func New(minor int64, currency string, exponent int) (Money, error) {
	if len(currency) != 3 || exponent < 0 || exponent > 9 {
		return Money{}, errors.New("invalid currency metadata")
	}
	return Money{Minor: minor, Currency: currency, Exponent: exponent, Known: true}, nil
}

func Add(a, b Money) (Money, error) {
	if a.Currency != b.Currency || a.Exponent != b.Exponent {
		return Money{}, fmt.Errorf("cannot add mixed currencies")
	}
	if !a.Known || !b.Known {
		return Unknown(a.Currency, a.Exponent), nil
	}
	a.Minor += b.Minor
	return a, nil
}
func (m Money) IsUnknown() bool { return !m.Known }
func (m Money) String() string {
	if !m.Known {
		return "unknown"
	}
	sign := ""
	n := m.Minor
	if n < 0 {
		sign = "-"
		n = -n
	}
	base := int64(1)
	for i := 0; i < m.Exponent; i++ {
		base *= 10
	}
	whole := n / base
	frac := n % base
	if m.Exponent == 0 {
		return fmt.Sprintf("%s%s %s", sign, fmt.Sprint(whole), m.Currency)
	}
	return fmt.Sprintf("%s%d.%0*d %s", sign, whole, m.Exponent, frac, m.Currency)
}
