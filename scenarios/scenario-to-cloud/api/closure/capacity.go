package closure

import (
	"fmt"
	"scenario-to-cloud/domain"
	"sort"
	"strings"
)

func (r *resolver) capacity(scenarios []*ScenarioDeclaration, resources []*ResourceDeclaration) domain.ClosureCapacity {
	capacity := domain.ClosureCapacity{Fit: "unknown"}
	add := func(component string, requirements *Requirements) {
		if requirements == nil {
			return
		}
		contribution := domain.ClosureCapacityContribution{
			Component:   component,
			CPU:         requirements.CPUCores,
			MemoryBytes: uint64(requirements.RAMMB * mebibyte),
			DiskBytes:   uint64(requirements.DiskMB * mebibyte),
		}
		capacity.CPU += contribution.CPU
		capacity.MemoryBytes += contribution.MemoryBytes
		capacity.DiskBytes += contribution.DiskBytes
		capacity.Contributions = append(capacity.Contributions, contribution)
	}
	for _, scenario := range scenarios {
		add("scenario:"+scenario.ID, scenario.Requirements)
	}
	for _, resource := range resources {
		add("resource:"+resource.ID, resource.Requirements)
	}
	sort.Slice(capacity.Contributions, func(i, j int) bool { return capacity.Contributions[i].Component < capacity.Contributions[j].Component })

	var largest uint64
	if len(r.in.ReleaseArtifactBytes) > 0 {
		for _, size := range r.in.ReleaseArtifactBytes {
			if size > largest {
				largest = size
			}
		}
		capacity.HeadroomBasis = "release_artifacts"
	} else {
		for _, contribution := range capacity.Contributions {
			if contribution.DiskBytes > largest {
				largest = contribution.DiskBytes
			}
		}
		capacity.HeadroomBasis = "largest_declared_disk_footprint"
	}
	capacity.TransientUpdateHeadroomBytes = 2*largest + StagingHeadroomBytes

	if target := r.in.TargetCapacity; target != nil {
		var shortfalls []string
		if target.CPU < capacity.CPU {
			shortfalls = append(shortfalls, fmt.Sprintf("cpu %.2f < %.2f", target.CPU, capacity.CPU))
		}
		if target.MemoryBytes < capacity.MemoryBytes {
			shortfalls = append(shortfalls, fmt.Sprintf("memory %d < %d bytes", target.MemoryBytes, capacity.MemoryBytes))
		}
		if needed := capacity.DiskBytes + capacity.TransientUpdateHeadroomBytes; target.DiskBytes < needed {
			shortfalls = append(shortfalls, fmt.Sprintf("disk %d < %d bytes (including %d bytes transient update headroom)", target.DiskBytes, needed, capacity.TransientUpdateHeadroomBytes))
		}
		if len(shortfalls) == 0 {
			capacity.Fit = "fits"
		} else {
			capacity.Fit = "insufficient"
			r.addUnsupported("capacity", ReasonInsufficientCapacity, strings.Join(shortfalls, "; "))
		}
	}
	return capacity
}

var kindOrder = map[domain.ClosureComponentKind]int{
	domain.ClosureKindScenario:             0,
	domain.ClosureKindResource:             1,
	domain.ClosureKindPackage:              2,
	domain.ClosureKindNativeArtifact:       3,
	domain.ClosureKindTool:                 4,
	domain.ClosureKindSafeguard:            5,
	domain.ClosureKindCredentialDescriptor: 6,
}

func sortComponents(components []domain.ClosureComponent) {
	sort.SliceStable(components, func(i, j int) bool {
		a, b := components[i], components[j]
		if kindOrder[a.Kind] != kindOrder[b.Kind] {
			return kindOrder[a.Kind] < kindOrder[b.Kind]
		}
		return a.ID < b.ID
	})
}

func normalizeReasons(reasons []domain.ClosureReason) []domain.ClosureReason {
	seen := map[domain.ClosureReason]struct{}{}
	out := make([]domain.ClosureReason, 0, len(reasons))
	for _, reason := range reasons {
		if _, ok := seen[reason]; ok {
			continue
		}
		seen[reason] = struct{}{}
		out = append(out, reason)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.From != b.From {
			return a.From < b.From
		}
		return a.Detail < b.Detail
	})
	return out
}
