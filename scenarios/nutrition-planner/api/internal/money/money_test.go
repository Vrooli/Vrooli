package money

import "testing"

func TestMoneyCurrencyAndExponent(t *testing.T) {
	a, _ := New(123, "USD", 2)
	b, _ := New(77, "USD", 2)
	got, err := Add(a, b)
	if err != nil || got.Minor != 200 {
		t.Fatalf("got %#v, %v", got, err)
	}
	eur, _ := New(1, "EUR", 2)
	if _, err := Add(a, eur); err == nil {
		t.Fatal("mixed currency accepted")
	}
	if got.String() != "2.00 USD" {
		t.Fatal(got.String())
	}
}
