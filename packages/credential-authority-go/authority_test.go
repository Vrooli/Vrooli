package credentialauthority

import (
	"errors"
	"testing"
)

func TestFailureSentinelsRemainDistinguishable(t *testing.T) {
	if ErrUnconfigured == ErrProviderUnavailable || ErrUnconfigured == ErrProviderAbsent || ErrProviderUnavailable == ErrProviderAbsent {
		t.Fatal("credential failure sentinels must remain distinct")
	}
	wrapped := []struct {
		name string
		err  error
		want error
	}{
		{"unconfigured", errors.Join(errors.New("context"), ErrUnconfigured), ErrUnconfigured},
		{"unavailable", errors.Join(errors.New("context"), ErrProviderUnavailable), ErrProviderUnavailable},
		{"absent", errors.Join(errors.New("context"), ErrProviderAbsent), ErrProviderAbsent},
	}
	for _, testCase := range wrapped {
		t.Run(testCase.name, func(t *testing.T) {
			if !errors.Is(testCase.err, testCase.want) {
				t.Fatalf("errors.Is(%v, %v) = false", testCase.err, testCase.want)
			}
		})
	}
}
