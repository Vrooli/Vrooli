// Package discovery owns the device-control DNS-SD service catalog while the
// shared mdns-go package remains vendor- and service-agnostic.
package discovery

import (
	"context"

	mdns "github.com/vrooli/mdns-go"
)

type (
	Options         = mdns.Options
	ServiceInstance = mdns.ServiceInstance
)

var (
	AndroidTVRemoteServices = []string{"_androidtvremote2._tcp", "_androidtvremote._tcp"}
	GoogleCastServices      = []string{"_googlecast._tcp"}
)

func Browse(ctx context.Context, services []string, options Options) ([]ServiceInstance, error) {
	return mdns.Browse(ctx, services, options)
}
