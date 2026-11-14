//go:build !linux

package target

import "fmt"

type unixCDPBridge struct{ socketPath string }

func prepareOfflineNetworkRoot(string) (string, bool, error) {
	return "", false, fmt.Errorf("offline Electron network isolation is only supported on Linux")
}

func pathWithin(string, string) bool { return false }

func startUnixCDPBridge(string, int) (*unixCDPBridge, error) {
	return nil, fmt.Errorf("offline Electron CDP bridge is only supported on Linux")
}

func (*unixCDPBridge) Port() int        { return 0 }
func (*unixCDPBridge) Endpoint() string { return "" }
func (*unixCDPBridge) Close() error     { return nil }

func isolatedElectronCommand(string, string, string, string, int, string, []string) (string, []string, error) {
	return "", nil, fmt.Errorf("offline Electron network isolation is only supported on Linux")
}
