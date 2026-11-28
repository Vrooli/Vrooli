//go:build linux

package hostreqkit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// DiscoverLinuxOpenRGBDevices reads kernel topology and character-node metadata.
// It never opens a hardware node or asks the lighting backend to detect devices.
func DiscoverLinuxOpenRGBDevices(ctx context.Context) ([]OpenRGBLinuxNode, error) {
	return newOpenRGBLinuxProbe(nil).discover(ctx)
}

func (p openRGBLinuxProbe) discover(ctx context.Context) ([]OpenRGBLinuxNode, error) {
	entries, err := os.ReadDir(p.dev)
	if err != nil {
		return nil, err
	}
	var out []OpenRGBLinuxNode
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		kind := ""
		if numberedOpenRGBNode(entry.Name(), "i2c-") {
			kind = "i2c"
		} else if numberedOpenRGBNode(entry.Name(), "hidraw") {
			kind = "hidraw"
		} else {
			continue
		}
		path := filepath.Join(p.dev, entry.Name())
		var stat unix.Stat_t
		if err := p.stat(path, &stat); err != nil || stat.Mode&unix.S_IFMT != unix.S_IFCHR {
			return nil, errors.Join(fmt.Errorf("OpenRGB candidate %s is not a verified character node", path), err)
		}
		device, err := p.nodeDevice(entry.Name(), kind, stat.Rdev)
		if err != nil {
			return nil, err
		}
		device.Detector = openRGBKnownDetector(device)
		if device.Detector == "" {
			continue
		}
		class := kind
		if kind == "i2c" {
			class = "i2c-dev"
		}
		sysPath, err := filepath.EvalSymlinks(filepath.Join(p.sys, "class", class, entry.Name()))
		if err != nil {
			return nil, err
		}
		if err := p.confirmNode(path, stat); err != nil {
			return nil, err
		}
		out = append(out, OpenRGBLinuxNode{Selector: device, Node: path, SysPath: sysPath, MajorMinor: fmt.Sprintf("%d:%d", unix.Major(stat.Rdev), unix.Minor(stat.Rdev))})
	}
	return out, nil
}

func ResolveLinuxOpenRGBDevices(ctx context.Context, devices []OpenRGBLinuxDevice) ([]OpenRGBLinuxNode, error) {
	nodes, err := DiscoverLinuxOpenRGBDevices(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]OpenRGBLinuxNode, 0, len(devices))
	for _, selected := range devices {
		var match *OpenRGBLinuxNode
		for _, node := range nodes {
			if node.Selector == selected {
				if match != nil {
					return nil, errors.New("selected OpenRGB physical route matched multiple nodes")
				}
				copy := node
				match = &copy
			}
		}
		if match == nil {
			return nil, errors.New("selected OpenRGB physical route is absent or changed")
		}
		out = append(out, *match)
	}
	return out, nil
}
