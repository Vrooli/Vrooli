//go:build !linux

package hostwireless

import (
	"context"
	"crypto/x509"
)

func evaluateKernelCertificate(_ context.Context, cert *x509.Certificate) KernelTrustEvidence {
	return KernelTrustEvidence{CertificateSHA256: digest(cert.Raw), State: "unsupported", Issue: "Linux running-kernel key verification is unavailable on this platform"}
}
