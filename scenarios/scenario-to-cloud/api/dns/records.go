package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	mdns "github.com/miekg/dns"

	"scenario-to-cloud/domain"
)

// LookupRecordSet returns common DNS records for a domain using system resolvers.
func LookupRecordSet(ctx context.Context, domainName string) (domain.DNSRecordSet, error) {
	host := normalizeHost(domainName)
	if host == "" {
		return domain.DNSRecordSet{}, fmt.Errorf("domain is empty")
	}
	fqdn := mdns.Fqdn(host)

	serverAddr, err := resolverAddress()
	if err != nil {
		return domain.DNSRecordSet{}, err
	}

	client := &mdns.Client{Timeout: 5 * time.Second}
	recordSet := domain.DNSRecordSet{Domain: host}

	for _, qtype := range []uint16{mdns.TypeA, mdns.TypeAAAA, mdns.TypeCNAME, mdns.TypeMX, mdns.TypeTXT, mdns.TypeNS} {
		msg := new(mdns.Msg)
		msg.SetQuestion(fqdn, qtype)
		resp, _, err := client.ExchangeContext(ctx, msg, serverAddr)
		if err != nil {
			continue
		}
		for _, ans := range resp.Answer {
			switch rr := ans.(type) {
			case *mdns.A:
				recordSet.A = append(recordSet.A, domain.DNSRecordValue{Value: rr.A.String(), TTL: rr.Hdr.Ttl})
			case *mdns.AAAA:
				recordSet.AAAA = append(recordSet.AAAA, domain.DNSRecordValue{Value: rr.AAAA.String(), TTL: rr.Hdr.Ttl})
			case *mdns.CNAME:
				recordSet.CNAME = append(recordSet.CNAME, domain.DNSRecordValue{Value: strings.TrimSuffix(rr.Target, "."), TTL: rr.Hdr.Ttl})
			case *mdns.MX:
				recordSet.MX = append(recordSet.MX, domain.DNSMXRecord{Host: strings.TrimSuffix(rr.Mx, "."), Priority: rr.Preference, TTL: rr.Hdr.Ttl})
			case *mdns.TXT:
				recordSet.TXT = append(recordSet.TXT, domain.DNSRecordValue{Value: strings.Join(rr.Txt, " "), TTL: rr.Hdr.Ttl})
			case *mdns.NS:
				recordSet.NS = append(recordSet.NS, domain.DNSRecordValue{Value: strings.TrimSuffix(rr.Ns, "."), TTL: rr.Hdr.Ttl})
			}
		}
	}

	return recordSet, nil
}

func resolverAddress() (string, error) {
	config, err := mdns.ClientConfigFromFile("/etc/resolv.conf")
	if err == nil && len(config.Servers) > 0 {
		return net.JoinHostPort(config.Servers[0], config.Port), nil
	}
	return "", fmt.Errorf("no resolvers available")
}
