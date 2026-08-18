// Package mdns implements the read-only DNS-SD browse protocol.
//
// It intentionally contains no service-specific knowledge. Callers provide a
// service type (for example _example._tcp); the package only performs PTR
// enumeration and resolves the SRV, TXT, A, and AAAA records belonging to the
// returned instances.
package mdns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

const (
	MulticastIPv4 = "224.0.0.251"
	MulticastIPv6 = "ff02::fb"
	MulticastPort = 5353
	DefaultWindow = 3 * time.Second
	maxPacketSize = 64 * 1024
)

type ServiceInstance struct {
	// Instance is the normalized DNS-SD instance name.
	Instance string `json:"instance"`
	Name     string `json:"name"`
	Service  string `json:"service"`
	Host     string `json:"host,omitempty"`
	// Addrs contains the resolved A and AAAA addresses. Addresses is retained
	// as the device-control-facing JSON spelling and is populated identically.
	Addrs     []net.IP          `json:"addrs,omitempty"`
	Addresses []net.IP          `json:"addresses,omitempty"`
	Port      int               `json:"port"`
	TXT       map[string]string `json:"txt,omitempty"`
	// Interface identifies the multicast interface that observed this
	// instance. It is needed to dial link-local IPv6 endpoints, whose zone is
	// otherwise lost when the AAAA record is represented as net.IP.
	Interface  string    `json:"interface,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
}

type BrowseError struct {
	Interface string
	Err       error
}

func (e *BrowseError) Error() string {
	if e.Interface == "" {
		return "mDNS browse: " + e.Err.Error()
	}
	return fmt.Sprintf("mDNS browse on interface %s: %v", e.Interface, e.Err)
}

func (e *BrowseError) Unwrap() error { return e.Err }

type Browser struct {
	Window     time.Duration
	Interfaces []*net.Interface
	IPv4Only   bool
	Listen     func(*net.Interface) (packetConn, error)
}

// Options controls a bounded browse. When IPv4Only is false, the browser joins
// both the IPv4 and IPv6 DNS-SD multicast groups on every selected interface.
type Options struct {
	Window     time.Duration
	Interfaces []*net.Interface
	IPv4Only   bool
}

type packetConn interface {
	ReadFrom([]byte) (int, net.Addr, error)
	WriteTo([]byte, net.Addr) (int, error)
	Close() error
}

func NewBrowser() *Browser { return &Browser{Window: DefaultWindow} }

func (b *Browser) Browse(ctx context.Context, service string) ([]ServiceInstance, error) {
	service = normalizeService(service)
	if service == "" {
		return nil, errors.New("mDNS service type is required")
	}
	interfaces := b.Interfaces
	if len(interfaces) == 0 {
		available, err := net.Interfaces()
		if err != nil {
			return nil, fmt.Errorf("enumerate network interfaces: %w", err)
		}
		interfaces = make([]*net.Interface, 0, len(available))
		for i := range available {
			interfaces = append(interfaces, &available[i])
		}
	}
	window := b.Window
	if window <= 0 {
		window = DefaultWindow
	}
	listen := b.Listen
	if listen == nil {
		listen = listenMulticast
		if b.IPv4Only {
			listen = listenMulticastIPv4
		}
	}

	query := dnsQuery(service, dnsmessage.TypePTR)
	deadline := time.Now().Add(window)
	type result struct {
		instances []ServiceInstance
		err       error
	}
	results := make(chan result, len(interfaces))
	var started int
	for _, iface := range interfaces {
		if iface == nil || iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		started++
		go func(iface *net.Interface) {
			conn, err := listen(iface)
			if err != nil {
				results <- result{err: &BrowseError{Interface: iface.Name, Err: err}}
				return
			}
			defer conn.Close()
			if err := writeBrowseQuery(conn, query, b.IPv4Only); err != nil {
				results <- result{err: &BrowseError{Interface: iface.Name, Err: err}}
				return
			}
			instances, err := browseConn(ctx, conn, service, deadline, b.IPv4Only, iface.Name)
			results <- result{instances: instances, err: err}
		}(iface)
	}
	if started == 0 {
		return nil, errors.New("mDNS browse has no usable multicast interface")
	}

	merged := map[string]ServiceInstance{}
	var firstErr error
	mergeResult := func(r result) {
		if r.err != nil && firstErr == nil {
			firstErr = r.err
		}
		for _, instance := range r.instances {
			merged[serviceInstanceKey(instance)] = mergeInstance(merged[serviceInstanceKey(instance)], instance)
		}
	}
	received := 0
	for received < started {
		select {
		case r := <-results:
			mergeResult(r)
			received++
		case <-ctx.Done():
			// Workers return any instances collected before observing cancellation.
			// Give those bounded workers a short drain window so a caller that
			// cancels after the first response does not lose that observation.
			timer := time.NewTimer(100 * time.Millisecond)
			for received < started {
				select {
				case r := <-results:
					mergeResult(r)
					received++
				case <-timer.C:
					return sortedInstances(merged), ctx.Err()
				}
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return sortedInstances(merged), ctx.Err()
		}
	}
	if len(merged) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return sortedInstances(merged), nil
}

// Browse performs one bounded browse per requested service type and merges
// results that arrive across interfaces and packets.
func Browse(ctx context.Context, services []string, option Options) ([]ServiceInstance, error) {
	if len(services) == 0 {
		return nil, errors.New("mDNS service type is required")
	}
	merged := map[string]ServiceInstance{}
	for _, service := range services {
		browser := NewBrowser()
		browser.Window = option.Window
		browser.Interfaces = option.Interfaces
		browser.IPv4Only = option.IPv4Only
		instances, err := browser.Browse(ctx, service)
		if err != nil {
			return nil, err
		}
		for _, instance := range instances {
			merged[serviceInstanceKey(instance)] = mergeInstance(merged[serviceInstanceKey(instance)], instance)
		}
	}
	return values(merged), nil
}

func browseConn(ctx context.Context, conn packetConn, service string, deadline time.Time, ipv4Only bool, iface string) ([]ServiceInstance, error) {
	instances := map[string]ServiceInstance{}
	var lastParseErr error
	buf := make([]byte, maxPacketSize)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return values(instances), ctx.Err()
		}
		_ = setReadDeadline(conn, minTime(deadline, time.Now().Add(100*time.Millisecond)))
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			if isTimeout(err) {
				continue
			}
			return nil, err
		}
		parsed, err := parsePacket(buf[:n], service)
		if err != nil {
			if errors.Is(err, dnsmessage.ErrSectionDone) {
				continue
			}
			lastParseErr = err
			continue
		}
		for name, item := range parsed {
			item.Interface = iface
			parsed[name] = item
			instances[name] = mergeInstance(instances[name], item)
		}
		// A PTR response identifies instances, but the additional section often
		// already carries every record needed to resolve them. Ask explicitly
		// only for records that are still missing.
		for _, item := range parsed {
			if item.Host == "" || item.Port == 0 || len(item.TXT) == 0 || len(item.Addresses) == 0 {
				_ = writeBrowseQuery(conn, dnsQuery(item.Name, dnsmessage.TypeSRV), ipv4Only)
				_ = writeBrowseQuery(conn, dnsQuery(item.Name, dnsmessage.TypeTXT), ipv4Only)
				if item.Host != "" {
					_ = writeBrowseQuery(conn, dnsQuery(item.Host, dnsmessage.TypeA), ipv4Only)
					_ = writeBrowseQuery(conn, dnsQuery(item.Host, dnsmessage.TypeAAAA), ipv4Only)
				}
			}
		}
	}
	if len(instances) == 0 && lastParseErr != nil {
		return nil, fmt.Errorf("parse mDNS response: %w", lastParseErr)
	}
	return values(instances), nil
}

func writeBrowseQuery(conn packetConn, query []byte, ipv4Only bool) error {
	if len(query) == 0 {
		return errors.New("mDNS query is empty")
	}
	targets := []*net.UDPAddr{{IP: net.ParseIP(MulticastIPv4), Port: MulticastPort}}
	if !ipv4Only {
		targets = append(targets, &net.UDPAddr{IP: net.ParseIP(MulticastIPv6), Port: MulticastPort})
	}
	var firstErr error
	for _, target := range targets {
		if _, err := conn.WriteTo(query, target); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func parsePacket(data []byte, service string) (map[string]ServiceInstance, error) {
	var parser dnsmessage.Parser
	_, err := parser.Start(data)
	if err != nil {
		return nil, err
	}
	if err := parser.SkipAllQuestions(); err != nil && !errors.Is(err, dnsmessage.ErrSectionDone) {
		return nil, fmt.Errorf("questions: %w", err)
	}
	items := map[string]ServiceInstance{}
	knownInstances := map[string]bool{}
	consume := func(section int) error {
		for {
			var recordHeader dnsmessage.ResourceHeader
			var err error
			switch section {
			case 0:
				recordHeader, err = parser.AnswerHeader()
			case 1:
				recordHeader, err = parser.AuthorityHeader()
			case 2:
				recordHeader, err = parser.AdditionalHeader()
			default:
				return nil
			}
			if errors.Is(err, dnsmessage.ErrSectionDone) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("section %d header: %w", section, err)
			}
			target := strings.TrimSuffix(strings.ToLower(recordHeader.Name.String()), ".")
			switch recordHeader.Type {
			case dnsmessage.TypePTR:
				value, err := parser.PTRResource()
				if err != nil {
					return fmt.Errorf("PTR resource: %w", err)
				}
				if normalizeService(target) == service {
					instanceName := strings.TrimSuffix(strings.ToLower(value.PTR.String()), ".")
					knownInstances[instanceName] = true
					item := items[instanceName]
					item.Instance = instanceName
					item.Name = instanceName
					item.Service = service
					items[serviceInstanceKey(item)] = item
				}
			case dnsmessage.TypeSRV:
				value, err := parser.SRVResource()
				if err != nil {
					return fmt.Errorf("SRV resource %q: %w", target, err)
				}
				item := items[target]
				if serviceInstanceKey(item) == "" {
					item.Instance = target
					item.Name = target
				}
				item.Host = strings.TrimSuffix(value.Target.String(), ".")
				item.Port = int(value.Port)
				item.Service = service
				items[serviceInstanceKey(item)] = item
			case dnsmessage.TypeTXT:
				value, err := parser.TXTResource()
				if err != nil {
					return fmt.Errorf("TXT resource %q: %w", target, err)
				}
				item := items[target]
				if serviceInstanceKey(item) == "" {
					item.Instance = target
					item.Name = target
				}
				item.Service = service
				if item.TXT == nil {
					item.TXT = map[string]string{}
				}
				for _, raw := range value.TXT {
					key, val, ok := strings.Cut(raw, "=")
					if ok {
						item.TXT[key] = val
					} else {
						item.TXT[raw] = ""
					}
				}
				items[serviceInstanceKey(item)] = item
			case dnsmessage.TypeA:
				value, err := parser.AResource()
				if err != nil {
					return fmt.Errorf("A resource %q: %w", target, err)
				}
				for key, item := range items {
					if strings.EqualFold(item.Host, target) {
						item.Addresses = appendUniqueIP(item.Addresses, net.IP(value.A[:]))
						items[key] = item
					}
				}
			case dnsmessage.TypeAAAA:
				value, err := parser.AAAAResource()
				if err != nil {
					return fmt.Errorf("AAAA resource %q: %w", target, err)
				}
				for key, item := range items {
					if strings.EqualFold(item.Host, target) {
						item.Addresses = appendUniqueIP(item.Addresses, net.IP(value.AAAA[:]))
						items[key] = item
					}
				}
			default:
				var skipErr error
				switch section {
				case 0:
					skipErr = parser.SkipAnswer()
				case 1:
					skipErr = parser.SkipAuthority()
				case 2:
					skipErr = parser.SkipAdditional()
				}
				if skipErr != nil {
					return fmt.Errorf("unknown resource %q type=%d: %w", target, recordHeader.Type, skipErr)
				}
			}
		}
	}
	if err := consume(0); err != nil {
		return nil, err
	}
	if err := consume(1); err != nil {
		return nil, err
	}
	if err := consume(2); err != nil {
		return nil, err
	}
	observedAt := time.Now().UTC()
	for name, item := range items {
		if !knownInstances[name] {
			delete(items, name)
			continue
		}
		if item.ObservedAt.IsZero() {
			item.ObservedAt = observedAt
		}
		item.Instance = serviceInstanceKey(item)
		item.Name = item.Instance
		item.Addrs = append([]net.IP(nil), item.Addresses...)
		items[name] = item
	}
	return items, nil
}

func dnsQuery(name string, typ dnsmessage.Type) []byte {
	if !strings.HasSuffix(name, ".") {
		name += "."
	}
	q, err := dnsmessage.NewName(name)
	if err != nil {
		return nil
	}
	builder := dnsmessage.NewBuilder(nil, dnsmessage.Header{})
	if err := builder.StartQuestions(); err != nil {
		return nil
	}
	// Use a multicast question (QM). A QU question asks responders to send
	// directly to the ephemeral source port used by the Linux Avahi fallback;
	// some receivers do not honor QU for browse traffic. The listener joins
	// 224.0.0.251 on every selected interface, so the multicast response is
	// valid for both the shared 5353 socket and the fallback socket.
	if err := builder.Question(dnsmessage.Question{Name: q, Type: typ, Class: dnsmessage.ClassINET}); err != nil {
		return nil
	}
	packed, _ := builder.Finish()
	return packed
}

func normalizeService(value string) string {
	value = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
	if value != "" && !strings.HasSuffix(value, ".local") {
		value += ".local"
	}
	return value
}

func appendUniqueIP(values []net.IP, candidate net.IP) []net.IP {
	for _, value := range values {
		if value.Equal(candidate) {
			return values
		}
	}
	return append(values, candidate)
}

func mergeInstance(a, b ServiceInstance) ServiceInstance {
	if serviceInstanceKey(a) == "" {
		a = b
		a.Instance = serviceInstanceKey(b)
		a.Name = a.Instance
		if a.TXT != nil {
			a.TXT = cloneTXT(a.TXT)
		}
		return a
	}
	if a.Instance == "" {
		a.Instance = serviceInstanceKey(a)
	}
	if a.Name == "" {
		a.Name = a.Instance
	}
	if a.Host == "" {
		a.Host = b.Host
	}
	if a.Port == 0 {
		a.Port = b.Port
	}
	if a.Service == "" {
		a.Service = b.Service
	}
	if a.Interface == "" {
		a.Interface = b.Interface
	}
	if a.ObservedAt.IsZero() || (!b.ObservedAt.IsZero() && b.ObservedAt.Before(a.ObservedAt)) {
		a.ObservedAt = b.ObservedAt
	}
	for _, ip := range b.Addrs {
		a.Addresses = appendUniqueIP(a.Addresses, ip)
	}
	for _, ip := range b.Addresses {
		a.Addresses = appendUniqueIP(a.Addresses, ip)
	}
	a.Addrs = append([]net.IP(nil), a.Addresses...)
	if a.TXT == nil {
		a.TXT = map[string]string{}
	}
	for k, v := range b.TXT {
		a.TXT[k] = v
	}
	return a
}

func serviceInstanceKey(instance ServiceInstance) string {
	if instance.Instance != "" {
		return instance.Instance
	}
	return instance.Name
}

func cloneTXT(input map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range input {
		out[k] = v
	}
	return out
}

func values(input map[string]ServiceInstance) []ServiceInstance {
	out := make([]ServiceInstance, 0, len(input))
	for _, v := range input {
		out = append(out, v)
	}
	return out
}

func sortedInstances(input map[string]ServiceInstance) []ServiceInstance {
	out := values(input)
	sort.Slice(out, func(i, j int) bool { return serviceInstanceKey(out[i]) < serviceInstanceKey(out[j]) })
	return out
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
func isTimeout(err error) bool { var ne net.Error; return errors.As(err, &ne) && ne.Timeout() }
func setReadDeadline(conn packetConn, deadline time.Time) error {
	if c, ok := conn.(interface{ SetReadDeadline(time.Time) error }); ok {
		return c.SetReadDeadline(deadline)
	}
	return nil
}
