package obs

import "testing"

func TestRecoverToFailureCapturesPanicAndStack(t *testing.T) {
	var got PanicFailure
	func() {
		defer RecoverToFailure("test worker", func(failure PanicFailure) { got = failure })
		panic("boom")
	}()

	if got.Operation != "test worker" {
		t.Fatalf("operation = %q, want test worker", got.Operation)
	}
	if got.Value != "boom" {
		t.Fatalf("value = %#v, want boom", got.Value)
	}
	if got.Stack == "" {
		t.Fatal("stack was not captured")
	}
	if got.Error() != "panic in test worker: boom" {
		t.Fatalf("error = %q", got.Error())
	}
}
