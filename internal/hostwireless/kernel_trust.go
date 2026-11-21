package hostwireless

import "context"

// KernelTrustEvidence separates enrollment/SKID observations from a running
// kernel cryptographic operation. Verified never claims a module was loaded.
type KernelTrustEvidence struct {
	CertificateSHA256 string `json:"certificate_der_sha256"`
	State             string `json:"state"`
	Ring              string `json:"ring,omitempty"`
	KeySerial         int    `json:"key_serial,omitempty"`
	Algorithm         string `json:"algorithm,omitempty"`
	Revocation        string `json:"certificate_revocation"`
	Issue             string `json:"issue,omitempty"`
}

// EvaluateKernelTrust reads only a public certificate and existing kernel keys.
// It neither creates keys nor links keyrings nor loads a trial module.
func EvaluateKernelTrust(ctx context.Context, certificatePath string) (KernelTrustEvidence, error) {
	cert, err := readPublicCertificate(certificatePath)
	if err != nil {
		return KernelTrustEvidence{}, err
	}
	return evaluateKernelCertificate(ctx, cert), nil
}
