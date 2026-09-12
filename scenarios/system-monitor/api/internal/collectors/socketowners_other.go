//go:build !linux

package collectors

import "context"

// attributeSocketOwners is Linux-only. Darwin and Windows can attribute sockets
// to processes, but only through backends this collector deliberately avoids on
// a per-cycle budget (lsof forks; GetExtendedTcpTable needs a cgo/syscall path).
// Reporting unsupported keeps the absence visible instead of empty-and-green.
func attributeSocketOwners(_ context.Context, established int, _ int) SocketAttribution {
	return SocketAttribution{
		Supported: false,
		Reason:    "per-process socket attribution has no native backend on this operating system",
		Total:     established,
	}
}
