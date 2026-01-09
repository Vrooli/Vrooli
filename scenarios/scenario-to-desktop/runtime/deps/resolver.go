// Package deps provides dependency resolution for services.
package deps

import (
	"errors"

	"scenario-to-desktop-runtime/manifest"
)

// ErrCyclicDependency indicates a cycle was detected in service dependencies.
var ErrCyclicDependency = errors.New("cycle detected in dependencies")

// TopoSort performs topological sort on services based on their dependencies.
// Returns service IDs in dependency order (dependencies first).
//
// This is a pure algorithm function suitable for testing in isolation.
// It uses Kahn's algorithm for topological sorting.
func TopoSort(services []manifest.Service) ([]string, error) {
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	for _, svc := range services {
		graph[svc.ID] = append(graph[svc.ID], svc.Dependencies...)
		if _, ok := inDegree[svc.ID]; !ok {
			inDegree[svc.ID] = 0
		}
		for _, dep := range svc.Dependencies {
			inDegree[svc.ID]++
			if _, ok := inDegree[dep]; !ok {
				inDegree[dep] = 0
			}
		}
	}

	queue := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	order := make([]string, 0, len(inDegree))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		order = append(order, id)
		for _, svc := range services {
			for _, dep := range svc.Dependencies {
				if dep == id {
					inDegree[svc.ID]--
					if inDegree[svc.ID] == 0 {
						queue = append(queue, svc.ID)
					}
				}
			}
		}
	}

	if len(order) != len(inDegree) {
		return nil, ErrCyclicDependency
	}
	return order, nil
}

// FindService looks up a service by ID from a slice.
// Returns nil if the service is not found.
func FindService(services []manifest.Service, id string) *manifest.Service {
	for i := range services {
		if services[i].ID == id {
			return &services[i]
		}
	}
	return nil
}
