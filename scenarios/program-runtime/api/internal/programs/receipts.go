package programs

import (
	"program-runtime/internal/corpuscodec"
	"program-runtime/internal/receipts"
)

var resumeReceiptPattern = receipts.PlaintextPattern

// Program corpus receipts use the same protection as task records. Keys are
// scoped to each owner instance and are intentionally not durable.
type receiptSealer struct{ delegate receipts.Sealer }

func newReceiptSealer() receiptSealer                     { return receiptSealer{delegate: receipts.New()} }
func (s receiptSealer) seal(value string) (string, error) { return s.delegate.Seal(value) }
func (s receiptSealer) open(value string) string          { return s.delegate.Open(value) }

func (s receiptSealer) encode(value string) (string, error) {
	return corpuscodec.Encode(s.delegate, value)
}
func (s receiptSealer) decode(value string, format int) (string, error) {
	out, _, err := corpuscodec.Decode(&s.delegate, value, format)
	return out, err
}
