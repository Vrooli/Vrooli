package onboard

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// rejectSelfTarget prevents the Bridge control-plane host from being silently
// treated as a fleet target. A self-SSH can work technically, but it is not a
// remote onboarding proof and is an easy operator mistake when the CLI is run
// from an SSH session on the control-plane machine.
func rejectSelfTarget(host string) error {
	target := strings.TrimSpace(strings.TrimSuffix(strings.ToLower(host), "."))
	if target == "" {
		return nil
	}
	localHost, _ := os.Hostname()
	localHost = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(localHost)), ".")
	if target == "localhost" || target == "localhost.localdomain" || target == localHost || target == localHost+".local" {
		return selfTargetError(host, localHost)
	}
	parsed := net.ParseIP(target)
	if parsed != nil && (parsed.IsLoopback() || localInterfaceHasIP(parsed)) {
		return selfTargetError(host, localHost)
	}
	resolved, err := net.LookupIP(strings.TrimSuffix(host, "."))
	if err != nil {
		return nil
	}
	for _, ip := range resolved {
		if ip.IsLoopback() || localInterfaceHasIP(ip) {
			return selfTargetError(host, localHost)
		}
	}
	return nil
}

func localInterfaceHasIP(target net.IP) bool {
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch value := addr.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			if ip != nil && ip.Equal(target) {
				return true
			}
		}
	}
	return false
}

func selfTargetError(host, localHost string) error {
	if localHost == "" {
		localHost = "this machine"
	}
	return fmt.Errorf("refusing to onboard %q: it resolves to the local Bridge control-plane host %q; choose the remote target hostname or IP", host, localHost)
}
