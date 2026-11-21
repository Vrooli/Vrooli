package ledger

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
)

// flags reads a command's typed flags strictly and keeps the first refusal.
// An amount that does not parse exactly ("1,000", "12abc", "9e99") is refused
// rather than sent as a different number: a ledger must never record an
// amount the operator did not type. An omitted optional flag reads as zero.
type flags struct {
	c   cliapp.OperationContext
	err error
}

func readFlags(c cliapp.OperationContext) *flags { return &flags{c: c} }

func (f *flags) value(name string) string { return strings.TrimSpace(f.c.Flag(name)) }

func (f *flags) refuse(err error) {
	if f.err == nil {
		f.err = err
	}
}

// Int64 reads a whole number such as an amount in minor units.
func (f *flags) Int64(name string) int64 {
	v := f.value(name)
	if v == "" {
		return 0
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		f.refuse(fmt.Errorf("--%s must be a whole number (minor units, no separators), not %q", name, v))
		return 0
	}
	return n
}

// Int32 reads a small whole number such as a count or a limit.
func (f *flags) Int32(name string) int32 {
	v := f.value(name)
	if v == "" {
		return 0
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		f.refuse(fmt.Errorf("--%s must be a whole number between %d and %d, not %q", name, math.MinInt32, math.MaxInt32, v))
		return 0
	}
	return int32(n)
}

// Float reads a finite decimal such as a ratio.
func (f *flags) Float(name string) float64 {
	v := f.value(name)
	if v == "" {
		return 0
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		f.refuse(fmt.Errorf("--%s must be a finite decimal number, not %q", name, v))
		return 0
	}
	return n
}

// Basis reads how a figure is known. Omitted means authoritative, the
// adapter's own record; any other value must name a basis exactly, so a typo
// never upgrades an operator's estimate to an authoritative figure.
func (f *flags) Basis(name string) sharedpb.Basis {
	v := strings.ReplaceAll(strings.ToLower(f.value(name)), "_", "-")
	switch v {
	case "", "authoritative":
		return sharedpb.Basis_BASIS_AUTHORITATIVE
	case "derived":
		return sharedpb.Basis_BASIS_DERIVED
	case "operator-asserted":
		return sharedpb.Basis_BASIS_OPERATOR_ASSERTED
	}
	f.refuse(fmt.Errorf("--%s must be authoritative, derived or operator-asserted, not %q", name, f.value(name)))
	return sharedpb.Basis_BASIS_UNSPECIFIED
}

// Err is the first refusal, if any flag did not parse.
func (f *flags) Err() error { return f.err }
