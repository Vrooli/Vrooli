package devicegraph

import (
	"path/filepath"
	"strings"
)

// builder accumulates devices while keeping a resolved-path index. The index is
// what turns a flat list into a graph with real edges: a block device finds its
// controller, a sensor finds the part it measures, and a USB device finds its
// hub, all by walking sysfs parent directories rather than by matching names.
type builder struct {
	env    Env
	graph  *Graph
	grader grader
	byPath map[string]string
}

func newBuilder(env Env, graph *Graph, at grader) *builder {
	return &builder{env: env, graph: graph, grader: at, byPath: map[string]string{}}
}

func (b *builder) add(device Device) {
	if device.SysPath != "" {
		b.byPath[filepath.Clean(device.SysPath)] = device.ID
	}
	b.graph.addDevice(device)
}

// ownerOf walks outward from a resolved sysfs path and returns the identity of
// the nearest already-enumerated device, which is that path's real parent in
// the topology. The optional skip set lets a caller ignore its own node.
func (b *builder) ownerOf(resolved string, skipSelf bool) string {
	chain := b.env.ancestors(resolved)
	for index, candidate := range chain {
		if index == 0 && skipSelf {
			continue
		}
		if id, ok := b.byPath[filepath.Clean(candidate)]; ok {
			return id
		}
	}
	return ""
}

// setParents fills in every parent link once all devices are enumerated, so
// enumeration order cannot decide whether an edge exists.
func (b *builder) setParents() {
	for index := range b.graph.Devices {
		device := &b.graph.Devices[index]
		if device.ParentID != "" || device.SysPath == "" {
			continue
		}
		device.ParentID = b.ownerOf(device.SysPath, true)
	}
}

// dropDanglingParents removes parent links whose target was not enumerated.
// A parent reference that points nowhere is worse than no reference: it makes
// the graph claim an edge it cannot show.
func (b *builder) dropDanglingParents() {
	known := make(map[string]struct{}, len(b.graph.Devices))
	for _, device := range b.graph.Devices {
		known[device.ID] = struct{}{}
	}
	for index := range b.graph.Devices {
		device := &b.graph.Devices[index]
		if device.ParentID == "" {
			continue
		}
		if _, ok := known[device.ParentID]; !ok {
			b.graph.warn("device %s referenced parent %s which was not enumerated", device.ID, device.ParentID)
			device.ParentID = ""
		}
	}
}

func setAttribute(device *Device, key, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if device.Attributes == nil {
		device.Attributes = map[string]string{}
	}
	device.Attributes[key] = value
}

func setReading(device *Device, key string, value float64) {
	if device.Readings == nil {
		device.Readings = map[string]float64{}
	}
	device.Readings[key] = value
}
