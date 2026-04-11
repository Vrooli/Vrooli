//go:build !linux

package network

func ListenersForPort(port int) ([]PortListener, error) {
	return nil, nil
}
