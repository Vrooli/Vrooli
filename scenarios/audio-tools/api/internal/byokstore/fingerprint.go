package byokstore

// Fingerprint returns a deterministic, human-recognisable redacted
// representation of a secret suitable for display in lists/logs.
// Format: "<first2>***<last4>" when the secret is long enough, else
// "***". Callers should never log the raw secret.
func Fingerprint(secret string) string {
	if len(secret) < 6 {
		return "***"
	}
	if len(secret) < 10 {
		return secret[:1] + "***" + secret[len(secret)-2:]
	}
	return secret[:2] + "***" + secret[len(secret)-4:]
}
