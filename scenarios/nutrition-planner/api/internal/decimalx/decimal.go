// Package decimalx provides exact decimal-string arithmetic for domain values.
package decimalx

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

var numberPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)

// Decimal is a canonical finite decimal. Unknown is deliberately separate
// from a known zero and is never silently substituted during arithmetic.
type Decimal struct {
	value   *big.Rat
	unknown bool
}

var Unknown = Decimal{unknown: true}

func Parse(s string) (Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Unknown, errors.New("decimal is empty")
	}
	if s == "unknown" {
		return Unknown, nil
	}
	if !numberPattern.MatchString(s) {
		return Unknown, fmt.Errorf("invalid decimal %q", s)
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return Unknown, fmt.Errorf("invalid decimal %q", s)
	}
	return Decimal{value: r}, nil
}

func KnownInt(n int64) Decimal    { return Decimal{value: big.NewRat(n, 1)} }
func (d Decimal) IsUnknown() bool { return d.unknown }
func (d Decimal) IsZero() bool    { return !d.unknown && d.value.Sign() == 0 }
func (d Decimal) String() string {
	if d.unknown {
		return "unknown"
	}
	return canonical(d.value)
}
func (d Decimal) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }
func (d *Decimal) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	v, err := Parse(s)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

func Add(a, b Decimal) (Decimal, error) {
	return binary(a, b, func(x, y *big.Rat) *big.Rat { return new(big.Rat).Add(x, y) })
}

func Sub(a, b Decimal) (Decimal, error) {
	return binary(a, b, func(x, y *big.Rat) *big.Rat { return new(big.Rat).Sub(x, y) })
}

func Mul(a, b Decimal) (Decimal, error) {
	return binary(a, b, func(x, y *big.Rat) *big.Rat { return new(big.Rat).Mul(x, y) })
}

func Div(a, b Decimal) (Decimal, error) {
	if b.IsUnknown() || b.IsZero() {
		return Unknown, errors.New("division by unknown or zero")
	}
	return binary(a, b, func(x, y *big.Rat) *big.Rat { return new(big.Rat).Quo(x, y) })
}

// Compare returns -1, 0, or 1 for known values. Unknown values cannot be ordered.
func Compare(a, b Decimal) (int, error) {
	if a.IsUnknown() || b.IsUnknown() {
		return 0, errors.New("cannot compare unknown decimals")
	}
	return a.value.Cmp(b.value), nil
}

// Ceil returns the smallest known integer decimal greater than or equal to d.
func Ceil(d Decimal) (Decimal, error) {
	if d.IsUnknown() {
		return Unknown, errors.New("cannot ceil unknown decimal")
	}
	n := new(big.Int).Quo(d.value.Num(), d.value.Denom())
	if d.value.Num().Sign() > 0 && new(big.Int).Mod(d.value.Num(), d.value.Denom()).Sign() != 0 {
		n.Add(n, big.NewInt(1))
	}
	return Decimal{value: new(big.Rat).SetInt(n)}, nil
}

func binary(a, b Decimal, op func(*big.Rat, *big.Rat) *big.Rat) (Decimal, error) {
	if a.IsUnknown() || b.IsUnknown() {
		return Unknown, nil
	}
	return Decimal{value: op(a.value, b.value)}, nil
}

func canonical(r *big.Rat) string {
	if r.Sign() == 0 {
		return "0"
	}
	neg := r.Sign() < 0
	n := new(big.Int).Abs(r.Num())
	d := r.Denom()
	whole := new(big.Int)
	rem := new(big.Int)
	whole.QuoRem(n, d, rem)
	if rem.Sign() == 0 {
		if neg {
			return "-" + whole.String()
		}
		return whole.String()
	}
	var frac strings.Builder
	for rem.Sign() != 0 {
		rem.Mul(rem, big.NewInt(10))
		digit := new(big.Int)
		digit.QuoRem(rem, d, rem)
		frac.WriteString(digit.String())
	}
	result := whole.String() + "." + frac.String()
	if neg {
		return "-" + result
	}
	return result
}
