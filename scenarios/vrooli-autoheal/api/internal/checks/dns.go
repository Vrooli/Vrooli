package checks

// DNSResolver abstracts DNS resolution for infrastructure checks.
type DNSResolver interface {
	LookupHost(host string) ([]string, error)
}
