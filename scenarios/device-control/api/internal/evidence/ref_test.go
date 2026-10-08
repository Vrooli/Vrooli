package evidence

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvidenceSinkRefusesUnverifiedCapture(t *testing.T) {
	sink := NewEvidenceSink(DefaultPolicy)
	_, err := sink.Put("capture-1", []byte("private"), Result{Policy: DefaultPolicy})
	require.ErrorContains(t, err, "not been verified")
}

func TestEvidenceSinkRefusesCaptureVerifiedUnderDifferentPolicy(t *testing.T) {
	sink := NewEvidenceSink(DefaultPolicy)
	_, err := sink.Put("capture-1", []byte("private"), Result{Verified: true, Policy: Policy{}})
	require.ErrorContains(t, err, "policy does not match")
}

func TestDefaultPolicyIsFailClosed(t *testing.T) {
	_, err := RedactCapture([]byte("capture"), "text/plain", Policy{}, false, "operator")
	require.ErrorContains(t, err, "default-deny")
}
