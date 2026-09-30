package viewport

import "testing"

func TestResolve(t *testing.T) {
	got, err := Resolve("mobile")
	if err != nil {
		t.Fatal(err)
	}
	if got != (Dimensions{Width: 390, Height: 844}) {
		t.Fatalf("mobile = %+v", got)
	}
	if _, err := Resolve("watch"); err == nil {
		t.Fatal("expected invalid preset error")
	}
}
